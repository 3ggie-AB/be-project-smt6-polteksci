package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
	"project_smt6/repository"

	probing "github.com/go-ping/ping"
)

type MetricSink interface {
	WritePing(context.Context, domain.PingMetric)
	WriteTCP(context.Context, domain.TCPMetric)
	WriteAP(context.Context, domain.APMetric)
	WriteAnomaly(context.Context, domain.AnomalyMetric)
	WriteSyslog(context.Context, domain.SyslogEvent)
}

type EventPublisher interface {
	Publish(domain.RealtimeEvent)
}

type FeatureRecorder interface {
	AddPing(domain.PingMetric) domain.FeatureVector
	AddAP(domain.APMetric) domain.FeatureVector
}

type ActiveEngine struct {
	devices  repository.DeviceRepository
	sink     MetricSink
	events   EventPublisher
	features FeatureRecorder
	cfg      config.MonitoringConfig
	logger   *slog.Logger
}

func NewActiveEngine(
	devices repository.DeviceRepository,
	sink MetricSink,
	events EventPublisher,
	features FeatureRecorder,
	cfg config.MonitoringConfig,
	logger *slog.Logger,
) *ActiveEngine {
	return &ActiveEngine{
		devices:  devices,
		sink:     sink,
		events:   events,
		features: features,
		cfg:      cfg,
		logger:   logger,
	}
}

func (e *ActiveEngine) Run(ctx context.Context) {
	pingWorkers := positive(e.cfg.PingWorkers, 512)
	tcpWorkers := positive(e.cfg.TCPWorkers, 256)
	pingJobs := make(chan domain.Device, pingWorkers*2)
	tcpJobs := make(chan domain.Device, tcpWorkers*2)

	var wg sync.WaitGroup
	for i := 0; i < pingWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.pingWorker(ctx, pingJobs)
		}()
	}
	for i := 0; i < tcpWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.tcpWorker(ctx, tcpJobs)
		}()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		e.schedule(ctx, "ping", e.cfg.PingInterval, pingJobs)
	}()
	go func() {
		defer wg.Done()
		e.schedule(ctx, "tcp", e.cfg.TCPInterval, tcpJobs)
	}()

	e.logger.Info("[OK] Active monitoring engine started", "ping_workers", pingWorkers, "tcp_workers", tcpWorkers)
	<-ctx.Done()
	wg.Wait()
	e.logger.Info("[OK] Active monitoring engine stopped")
}

func (e *ActiveEngine) schedule(ctx context.Context, kind string, interval time.Duration, jobs chan<- domain.Device) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.enqueueActiveDevices(ctx, kind, jobs)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.enqueueActiveDevices(ctx, kind, jobs)
		}
	}
}

func (e *ActiveEngine) enqueueActiveDevices(ctx context.Context, kind string, jobs chan<- domain.Device) {
	devices, err := e.devices.ListActive(ctx)
	if err != nil {
		e.logger.Error("Active device list failed", "kind", kind, "detail", err)
		return
	}

	for _, device := range devices {
		select {
		case <-ctx.Done():
			return
		case jobs <- device:
		}
	}
}

func (e *ActiveEngine) pingWorker(ctx context.Context, jobs <-chan domain.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		case device := <-jobs:
			metric := e.ping(ctx, device)
			e.sink.WritePing(ctx, metric)
			if e.features != nil {
				vector := e.features.AddPing(metric)
				e.maybePublishAnomaly(ctx, metric, vector)
			}
			e.publishPingEvents(metric, device)
			if metric.StatusUp {
				_ = e.devices.MarkSeen(ctx, device.ID)
			}
		}
	}
}

func (e *ActiveEngine) tcpWorker(ctx context.Context, jobs <-chan domain.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		case device := <-jobs:
			metric := e.tcpCheck(ctx, device)
			e.sink.WriteTCP(ctx, metric)
			if !metric.Success {
				e.events.Publish(domain.RealtimeEvent{
					Type:      "tcp.service_down",
					Severity:  "critical",
					Workspace: metric.Workspace,
					DeviceID:  metric.DeviceID,
					IP:        metric.IP,
					Title:     "TCP service down",
					Message:   fmt.Sprintf("%s:%d cannot be reached", metric.IP, metric.Port),
					Attributes: map[string]any{
						"port":    metric.Port,
						"timeout": metric.Timeout,
						"error":   metric.Error,
					},
				})
			}
		}
	}
}

func (e *ActiveEngine) ping(ctx context.Context, device domain.Device) domain.PingMetric {
	started := time.Now()
	metric := domain.PingMetric{
		DeviceID:  device.ID,
		Workspace: workspaceSlug(device),
		IP:        device.IPAddress,
		Timestamp: started,
	}

	pinger, err := probing.NewPinger(device.IPAddress)
	if err != nil {
		metric.PacketLoss = 100
		return metric
	}
	pinger.Count = positive(e.cfg.PingCount, 3)
	pinger.Timeout = durationOr(e.cfg.PingTimeout, 3*time.Second)
	pinger.SetPrivileged(os.Getuid() == 0)

	done := make(chan error, 1)
	go func() {
		done <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		metric.PacketLoss = 100
		return metric
	case err := <-done:
		if err != nil {
			up, latency := tcpFallbackProbe(ctx, device.IPAddress, e.cfg.DefaultTCPPort, e.cfg.PingTimeout)
			metric.StatusUp = up
			if up {
				metric.LatencyMS = latency
				metric.ResponseTime = latency
			} else {
				metric.PacketLoss = 100
			}
			return metric
		}
	}

	stats := pinger.Statistics()
	metric.StatusUp = stats.PacketsRecv > 0
	metric.PacketLoss = stats.PacketLoss
	if stats.PacketsRecv > 0 {
		metric.LatencyMS = roundMS(stats.AvgRtt)
		metric.ResponseTime = roundMS(time.Since(started))
	}
	return metric
}

func (e *ActiveEngine) tcpCheck(ctx context.Context, device domain.Device) domain.TCPMetric {
	port := device.EffectiveTCPPort(e.cfg.DefaultTCPPort)
	timeout := durationOr(e.cfg.TCPTimeout, 3*time.Second)
	started := time.Now()

	metric := domain.TCPMetric{
		DeviceID:  device.ID,
		Workspace: workspaceSlug(device),
		IP:        device.IPAddress,
		Port:      port,
		Timestamp: started,
	}

	dialer := net.Dialer{Timeout: timeout}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(device.IPAddress, fmt.Sprintf("%d", port)))
	metric.ConnectDurationMS = roundMS(time.Since(started))
	if err != nil {
		metric.Timeout = errors.Is(dialCtx.Err(), context.DeadlineExceeded)
		metric.Error = err.Error()
		return metric
	}
	_ = conn.Close()
	metric.Success = true
	return metric
}

func (e *ActiveEngine) publishPingEvents(metric domain.PingMetric, device domain.Device) {
	if !metric.StatusUp {
		e.events.Publish(domain.RealtimeEvent{
			Type:      "ap.down",
			Severity:  "critical",
			Workspace: metric.Workspace,
			DeviceID:  metric.DeviceID,
			IP:        metric.IP,
			Title:     "Device down",
			Message:   fmt.Sprintf("%s is not responding", device.Name),
		})
		return
	}

	if metric.LatencyMS >= e.cfg.HighLatencyMS {
		e.events.Publish(domain.RealtimeEvent{
			Type:      "latency.high",
			Severity:  "warning",
			Workspace: metric.Workspace,
			DeviceID:  metric.DeviceID,
			IP:        metric.IP,
			Title:     "High latency",
			Message:   fmt.Sprintf("%s latency is %.2f ms", device.Name, metric.LatencyMS),
			Attributes: map[string]any{
				"latency_ms": metric.LatencyMS,
			},
		})
	}

	if metric.PacketLoss/100 >= e.cfg.HighPacketLossRatio {
		e.events.Publish(domain.RealtimeEvent{
			Type:      "packet_loss.high",
			Severity:  "warning",
			Workspace: metric.Workspace,
			DeviceID:  metric.DeviceID,
			IP:        metric.IP,
			Title:     "Packet loss high",
			Message:   fmt.Sprintf("%s packet loss is %.2f%%", device.Name, metric.PacketLoss),
			Attributes: map[string]any{
				"packet_loss": metric.PacketLoss,
			},
		})
	}
}

func (e *ActiveEngine) maybePublishAnomaly(ctx context.Context, metric domain.PingMetric, vector domain.FeatureVector) {
	score := vector.TrafficAnomalyScore
	if vector.PacketLossRatio >= e.cfg.HighPacketLossRatio {
		score += vector.PacketLossRatio
	}
	if vector.LatencyRollingAvgMS >= e.cfg.HighLatencyMS {
		score += vector.LatencyRollingAvgMS / e.cfg.HighLatencyMS
	}
	if score < 2.5 {
		return
	}
	anomaly := domain.AnomalyMetric{
		DeviceID:            metric.DeviceID,
		Workspace:           metric.Workspace,
		IP:                  metric.IP,
		Score:               score,
		LatencyRollingAvgMS: vector.LatencyRollingAvgMS,
		PacketLossRatio:     vector.PacketLossRatio,
		APLoadScore:         vector.APLoadScore,
		RoamingFrequency:    vector.RoamingFrequency,
		TrafficAnomalyScore: vector.TrafficAnomalyScore,
		Model:               "rules_v1_export_ready_onnx",
		Timestamp:           metric.Timestamp,
	}
	e.sink.WriteAnomaly(ctx, anomaly)
	e.events.Publish(domain.RealtimeEvent{
		Type:      "anomaly.detected",
		Severity:  "warning",
		Workspace: metric.Workspace,
		DeviceID:  metric.DeviceID,
		IP:        metric.IP,
		Title:     "Network anomaly detected",
		Message:   fmt.Sprintf("Anomaly score %.2f", score),
		Attributes: map[string]any{
			"score":      score,
			"onnx_input": vector.ONNXInput(),
		},
	})
}

func tcpFallbackProbe(ctx context.Context, ip string, port int, timeout time.Duration) (bool, float64) {
	if port == 0 {
		port = 443
	}
	started := time.Now()
	dialer := net.Dialer{Timeout: timeout}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(ip, fmt.Sprintf("%d", port)))
	if err != nil {
		return false, 0
	}
	_ = conn.Close()
	return true, roundMS(time.Since(started))
}

func workspaceSlug(device domain.Device) string {
	if device.Workspace.Slug != "" {
		return device.Workspace.Slug
	}
	return fmt.Sprintf("%d", device.WorkspaceID)
}

func positive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func durationOr(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

func roundMS(duration time.Duration) float64 {
	return math.Round(float64(duration.Microseconds())/10) / 100
}

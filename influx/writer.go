package influx

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type Writer struct {
	cfg      config.InfluxConfig
	logger   *slog.Logger
	enabled  bool
	client   influxdb2.Client
	writeAPI influxapi.WriteAPIBlocking
	queue    chan *write.Point
	wg       sync.WaitGroup
}

var ErrDisabled = fmt.Errorf("influx writer disabled")

func NewWriter(cfg config.InfluxConfig, logger *slog.Logger) *Writer {
	w := &Writer{
		cfg:     cfg,
		logger:  logger,
		enabled: cfg.URL != "" && cfg.Token != "" && cfg.Org != "" && cfg.Bucket != "",
	}
	if !w.enabled {
		logger.Warn("[WARN] InfluxDB writer disabled", "reason", "configure INFLUX_URL, INFLUX_TOKEN, INFLUX_ORG, and INFLUX_BUCKET to persist time-series metrics")
		return w
	}

	if w.cfg.BatchSize <= 0 {
		w.cfg.BatchSize = 500
	}
	if w.cfg.QueueSize <= 0 {
		w.cfg.QueueSize = 10000
	}
	if w.cfg.FlushInterval <= 0 {
		w.cfg.FlushInterval = 2 * time.Second
	}
	if w.cfg.MaxRetries <= 0 {
		w.cfg.MaxRetries = 3
	}
	if w.cfg.RetryInterval <= 0 {
		w.cfg.RetryInterval = 500 * time.Millisecond
	}

	w.client = influxdb2.NewClient(cfg.URL, cfg.Token)
	w.writeAPI = w.client.WriteAPIBlocking(cfg.Org, cfg.Bucket)
	w.queue = make(chan *write.Point, w.cfg.QueueSize)
	return w
}

func (w *Writer) Enabled() bool {
	return w.enabled
}

func (w *Writer) Ping(ctx context.Context) error {
	if !w.enabled {
		return ErrDisabled
	}
	ok, err := w.client.Ping(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("influx ping returned unhealthy")
	}
	return nil
}

func (w *Writer) Start(ctx context.Context) {
	if !w.enabled {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()
}

func (w *Writer) Close(ctx context.Context) error {
	if !w.enabled {
		return nil
	}
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		w.client.Close()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Writer) run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]*write.Point, 0, w.cfg.BatchSize)
	flush := func(flushCtx context.Context) {
		if len(batch) == 0 {
			return
		}
		if err := w.writeWithRetry(flushCtx, batch); err != nil {
			w.logger.Error("InfluxDB batch write failed", "detail", err, "points", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			flush(flushCtx)
			cancel()
			return
		case point := <-w.queue:
			if point == nil {
				continue
			}
			batch = append(batch, point)
			if len(batch) >= w.cfg.BatchSize {
				flush(ctx)
			}
		case <-ticker.C:
			flush(ctx)
		}
	}
}

func (w *Writer) writeWithRetry(ctx context.Context, points []*write.Point) error {
	var lastErr error
	for attempt := 0; attempt <= w.cfg.MaxRetries; attempt++ {
		if err := w.writeAPI.WritePoint(ctx, points...); err != nil {
			lastErr = err
			sleep := time.Duration(attempt+1) * w.cfg.RetryInterval
			select {
			case <-time.After(sleep):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	return fmt.Errorf("influx write exhausted retries: %w", lastErr)
}

func (w *Writer) enqueue(point *write.Point) {
	if !w.enabled || point == nil {
		return
	}
	select {
	case w.queue <- point:
	default:
		w.logger.Warn("InfluxDB queue full, dropping metric", "measurement", point.Name())
	}
}

func (w *Writer) WritePing(ctx context.Context, metric domain.PingMetric) {
	point := influxdb2.NewPoint(
		"ping_metrics",
		baseTags(metric.DeviceID, metric.TargetID, metric.Workspace, metric.IP, ""),
		map[string]any{
			"latency":       metric.LatencyMS,
			"packet_loss":   metric.PacketLoss,
			"response_time": metric.ResponseTime,
			"status_up":     metric.StatusUp,
		},
		metric.Timestamp,
	)
	w.enqueue(point)
}

func (w *Writer) WriteTCP(ctx context.Context, metric domain.TCPMetric) {
	point := influxdb2.NewPoint(
		"tcp_metrics",
		withTag(baseTags(metric.DeviceID, metric.TargetID, metric.Workspace, metric.IP, ""), "port", strconv.Itoa(metric.Port)),
		map[string]any{
			"connect_duration": metric.ConnectDurationMS,
			"success":          metric.Success,
			"timeout":          metric.Timeout,
			"error":            metric.Error,
		},
		metric.Timestamp,
	)
	w.enqueue(point)
}

func (w *Writer) WriteAP(ctx context.Context, metric domain.APMetric) {
	fields := map[string]any{
		"client_count": metric.ClientCount,
		"cpu":          metric.CPU,
		"memory":       metric.Memory,
		"rssi":         metric.RSSI,
		"throughput":   metric.ThroughputBPS,
		"online":       metric.Online,
	}
	if metric.UptimeSeconds > 0 {
		fields["uptime"] = metric.UptimeSeconds
	}

	point := influxdb2.NewPoint(
		"ap_metrics",
		withTag(baseTags(metric.DeviceID, 0, metric.Workspace, metric.IP, metric.APName), "source", metric.Source),
		fields,
		metric.Timestamp,
	)
	w.enqueue(point)
}

func (w *Writer) WriteAnomaly(ctx context.Context, metric domain.AnomalyMetric) {
	point := influxdb2.NewPoint(
		"anomaly_metrics",
		baseTags(metric.DeviceID, metric.TargetID, metric.Workspace, metric.IP, ""),
		map[string]any{
			"score":                 metric.Score,
			"latency_rolling_avg":   metric.LatencyRollingAvgMS,
			"packet_loss_ratio":     metric.PacketLossRatio,
			"ap_load_score":         metric.APLoadScore,
			"roaming_frequency":     metric.RoamingFrequency,
			"traffic_anomaly_score": metric.TrafficAnomalyScore,
			"model":                 metric.Model,
		},
		metric.Timestamp,
	)
	w.enqueue(point)
}

func (w *Writer) WriteSyslog(ctx context.Context, event domain.SyslogEvent) {
	point := influxdb2.NewPoint(
		"syslog_events",
		withTags(baseTags(event.DeviceID, 0, event.Workspace, event.IP, ""), map[string]string{
			"facility": event.Facility,
			"severity": event.Severity,
			"hostname": event.Hostname,
		}),
		map[string]any{
			"message": event.Message,
		},
		event.Timestamp,
	)
	w.enqueue(point)
}

func baseTags(deviceID, targetID uint, workspace, ip, apName string) map[string]string {
	tags := map[string]string{
		"workspace": workspace,
		"ip":        ip,
	}
	if deviceID > 0 {
		tags["device_id"] = strconv.FormatUint(uint64(deviceID), 10)
	}
	if targetID > 0 {
		tags["target_id"] = strconv.FormatUint(uint64(targetID), 10)
	}
	if apName != "" {
		tags["ap_name"] = apName
	}
	return tags
}

func withTag(tags map[string]string, key, value string) map[string]string {
	tags[key] = value
	return tags
}

func withTags(tags map[string]string, extra map[string]string) map[string]string {
	for key, value := range extra {
		tags[key] = value
	}
	return tags
}

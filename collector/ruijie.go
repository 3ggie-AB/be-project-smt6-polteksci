package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
)

type RuijieCollector struct {
	cfg      config.RuijieConfig
	sink     MetricSink
	events   EventPublisher
	features FeatureRecorder
	client   *http.Client
	logger   *slog.Logger
}

type ruijieAPTelemetry struct {
	DeviceID      uint    `json:"device_id"`
	Workspace     string  `json:"workspace"`
	APName        string  `json:"ap_name"`
	Name          string  `json:"name"`
	IP            string  `json:"ip"`
	ClientCount   int     `json:"client_count"`
	CPU           float64 `json:"cpu"`
	Memory        float64 `json:"memory"`
	RSSI          float64 `json:"rssi"`
	ThroughputBPS float64 `json:"throughput_bps"`
	Throughput    float64 `json:"throughput"`
	Online        bool    `json:"online"`
	Status        string  `json:"status"`
}

func NewRuijieCollector(cfg config.RuijieConfig, sink MetricSink, events EventPublisher, features FeatureRecorder, logger *slog.Logger) *RuijieCollector {
	return &RuijieCollector{
		cfg:      cfg,
		sink:     sink,
		events:   events,
		features: features,
		client:   &http.Client{Timeout: cfg.RequestTimeout},
		logger:   logger,
	}
}

func (c *RuijieCollector) Run(ctx context.Context) {
	if c.cfg.BaseURL == "" {
		c.logger.Warn("[WARN] Ruijie collector skipped", "reason", "RUIJIE_BASE_URL is empty")
		return
	}
	if c.cfg.PollInterval <= 0 {
		c.cfg.PollInterval = 30 * time.Second
	}
	if c.cfg.RequestTimeout <= 0 {
		c.cfg.RequestTimeout = 10 * time.Second
	}

	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()

	c.logger.Info("[OK] Ruijie collector started", "interval", c.cfg.PollInterval.String())
	c.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("[OK] Ruijie collector stopped")
			return
		case <-ticker.C:
			c.poll(ctx)
		}
	}
}

func (c *RuijieCollector) poll(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+c.cfg.Endpoint, nil)
	if err != nil {
		c.logger.Error("Ruijie request creation failed", "detail", err)
		return
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("Ruijie telemetry poll failed", "detail", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		c.logger.Error("Ruijie response read failed", "detail", err)
		return
	}
	if resp.StatusCode >= 300 {
		c.logger.Error("Ruijie telemetry returned error", "status", resp.Status, "body", string(bytes.TrimSpace(body)))
		return
	}
	if c.cfg.DebugRawJSON {
		c.logger.Debug("ruijie raw telemetry", "payload", string(body))
	}

	items, err := decodeRuijieAPs(body)
	if err != nil {
		c.logger.Error("Ruijie telemetry decode failed", "detail", err)
		return
	}

	now := time.Now()
	for _, item := range items {
		name := item.APName
		if name == "" {
			name = item.Name
		}
		throughput := item.ThroughputBPS
		if throughput == 0 {
			throughput = item.Throughput
		}
		online := item.Online || item.Status == "online" || item.Status == "up"
		metric := domain.APMetric{
			DeviceID:      item.DeviceID,
			Workspace:     item.Workspace,
			APName:        name,
			IP:            item.IP,
			ClientCount:   item.ClientCount,
			CPU:           item.CPU,
			Memory:        item.Memory,
			RSSI:          item.RSSI,
			ThroughputBPS: throughput,
			Online:        online,
			Source:        "ruijie_api",
			Timestamp:     now,
		}
		c.sink.WriteAP(ctx, metric)
		if c.features != nil {
			vector := c.features.AddAP(metric)
			c.sink.WriteAnomaly(ctx, domain.AnomalyMetric{
				DeviceID:            metric.DeviceID,
				Workspace:           metric.Workspace,
				IP:                  metric.IP,
				Score:               vector.TrafficAnomalyScore,
				LatencyRollingAvgMS: vector.LatencyRollingAvgMS,
				PacketLossRatio:     vector.PacketLossRatio,
				APLoadScore:         vector.APLoadScore,
				RoamingFrequency:    vector.RoamingFrequency,
				TrafficAnomalyScore: vector.TrafficAnomalyScore,
				Model:               "feature_export_v1",
				Timestamp:           now,
			})
		}
		if !metric.Online {
			c.events.Publish(domain.RealtimeEvent{
				Type:      "ap.down",
				Severity:  "critical",
				Workspace: metric.Workspace,
				DeviceID:  metric.DeviceID,
				IP:        metric.IP,
				Title:     "AP down",
				Message:   fmt.Sprintf("%s is offline", metric.APName),
			})
		}
	}
}

func decodeRuijieAPs(body []byte) ([]ruijieAPTelemetry, error) {
	var direct []ruijieAPTelemetry
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct, nil
	}

	var wrapped struct {
		Data  []ruijieAPTelemetry `json:"data"`
		Items []ruijieAPTelemetry `json:"items"`
		APs   []ruijieAPTelemetry `json:"aps"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	switch {
	case len(wrapped.Data) > 0:
		return wrapped.Data, nil
	case len(wrapped.Items) > 0:
		return wrapped.Items, nil
	default:
		return wrapped.APs, nil
	}
}

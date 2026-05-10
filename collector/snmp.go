package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
	"project_smt6/repository"

	"github.com/gosnmp/gosnmp"
)

const (
	oidSysUptime   = ".1.3.6.1.2.1.1.3.0"
	oidIfInOctets  = ".1.3.6.1.2.1.2.2.1.10"
	oidIfOutOctets = ".1.3.6.1.2.1.2.2.1.16"
)

type SNMPCollector struct {
	cfg      config.SNMPConfig
	devices  repository.DeviceRepository
	sink     MetricSink
	features FeatureRecorder
	logger   *slog.Logger
}

func NewSNMPCollector(cfg config.SNMPConfig, devices repository.DeviceRepository, sink MetricSink, features FeatureRecorder, logger *slog.Logger) *SNMPCollector {
	return &SNMPCollector{
		cfg:      cfg,
		devices:  devices,
		sink:     sink,
		features: features,
		logger:   logger,
	}
}

func (c *SNMPCollector) Run(ctx context.Context) {
	if !c.cfg.Enabled {
		c.logger.Warn("[WARN] SNMP collector disabled")
		return
	}
	if c.cfg.PollInterval <= 0 {
		c.cfg.PollInterval = time.Minute
	}
	if c.cfg.Port == 0 {
		c.cfg.Port = 161
	}
	if c.cfg.Timeout <= 0 {
		c.cfg.Timeout = 3 * time.Second
	}

	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()

	c.logger.Info("[OK] SNMP collector started", "interval", c.cfg.PollInterval.String())
	c.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("[OK] SNMP collector stopped")
			return
		case <-ticker.C:
			c.poll(ctx)
		}
	}
}

func (c *SNMPCollector) poll(ctx context.Context) {
	devices, err := c.devices.ListActive(ctx)
	if err != nil {
		c.logger.Error("SNMP active device list failed", "detail", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 64)
	for _, device := range devices {
		if device.SNMPCommunity == "" {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(device domain.Device) {
			defer wg.Done()
			defer func() { <-sem }()
			c.pollDevice(ctx, device)
		}(device)
	}
	wg.Wait()
}

func (c *SNMPCollector) pollDevice(ctx context.Context, device domain.Device) {
	client := &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      c.cfg.Port,
		Community: device.SNMPCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   c.cfg.Timeout,
		Retries:   c.cfg.Retries,
	}
	if err := client.Connect(); err != nil {
		c.logger.Warn("SNMP device connection failed", "device_id", device.ID, "ip", device.IPAddress, "detail", err)
		return
	}
	defer client.Conn.Close()

	now := time.Now()
	totalIn, _ := c.walkCounter(client, oidIfInOctets)
	totalOut, _ := c.walkCounter(client, oidIfOutOctets)
	uptime := c.getScalar(client, oidSysUptime)
	cpu := 0.0
	memory := 0.0
	if c.cfg.CPUOID != "" {
		cpu = c.getScalar(client, c.cfg.CPUOID)
	}
	if c.cfg.MemoryOID != "" {
		memory = c.getScalar(client, c.cfg.MemoryOID)
	}

	metric := domain.APMetric{
		DeviceID:      device.ID,
		Workspace:     workspaceSlug(device),
		APName:        device.Name,
		IP:            device.IPAddress,
		CPU:           cpu,
		Memory:        memory,
		ThroughputBPS: totalIn + totalOut,
		Online:        true,
		UptimeSeconds: uptime / 100,
		Source:        "snmp",
		Timestamp:     now,
	}
	c.sink.WriteAP(ctx, metric)
	if c.features != nil {
		vector := c.features.AddAP(metric)
		if vector.TrafficAnomalyScore >= 3 {
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
				Model:               "snmp_rules_v1",
				Timestamp:           now,
			})
		}
	}
}

func (c *SNMPCollector) walkCounter(client *gosnmp.GoSNMP, oid string) (float64, error) {
	var total float64
	err := client.Walk(oid, func(pdu gosnmp.SnmpPDU) error {
		total += pduToFloat64(pdu.Value)
		return nil
	})
	return total, err
}

func (c *SNMPCollector) getScalar(client *gosnmp.GoSNMP, oid string) float64 {
	result, err := client.Get([]string{oid})
	if err != nil || len(result.Variables) == 0 {
		return 0
	}
	return pduToFloat64(result.Variables[0].Value)
}

func pduToFloat64(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case uint:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case float64:
		return v
	case []byte:
		parsed, _ := strconv.ParseFloat(string(v), 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return parsed
	}
}

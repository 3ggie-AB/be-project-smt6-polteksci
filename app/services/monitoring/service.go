package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"project_smt6/app/models"
	"project_smt6/config"

	"github.com/gosnmp/gosnmp"
	"gorm.io/gorm"
)

var (
	packetLossPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)%\s+packet loss`)
	rttPattern        = regexp.MustCompile(`(?:rtt|round-trip)[^=]*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)`)
)

type Service struct {
	db     *gorm.DB
	cfg    config.MonitoringConfig
	logger *slog.Logger
}

type deviceJob struct {
	device models.Device
	config models.MonitoringConfig
}

type pingResult struct {
	latencyMS  float64
	packetLoss float64
	success    bool
}

type snmpResult struct {
	cpuUsage    *float64
	memoryUsage *float64
}

func NewService(db *gorm.DB, cfg config.MonitoringConfig, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, cfg: normalizeConfig(cfg), logger: logger}
}

func (s *Service) Start(ctx context.Context) {
	if !s.cfg.Enabled {
		s.logger.Info("monitoring worker disabled")
		return
	}
	if !s.cfg.Ping.Enabled && !s.snmpReady() {
		s.logger.Info("monitoring worker has no enabled probes")
		return
	}

	go s.loop(ctx)
}

func (s *Service) loop(ctx context.Context) {
	interval := s.pollInterval()
	timer := time.NewTimer(0)
	defer timer.Stop()

	nextSNMPPoll := time.Time{}
	s.logger.Info(
		"monitoring worker started",
		"interval", interval,
		"ping_enabled", s.cfg.Ping.Enabled,
		"snmp_enabled", s.snmpReady(),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("monitoring worker stopped")
			return
		case <-timer.C:
			now := time.Now()
			includeSNMP := s.snmpReady() && !now.Before(nextSNMPPoll)
			s.runOnce(ctx, includeSNMP)
			if includeSNMP {
				nextSNMPPoll = time.Now().Add(s.cfg.SNMP.Interval)
			}
			timer.Reset(interval)
		}
	}
}

func (s *Service) runOnce(ctx context.Context, includeSNMP bool) {
	devices, configs, err := s.loadTargets(ctx)
	if err != nil {
		s.logger.Error("failed to load monitoring targets", "error", err)
		return
	}
	if len(devices) == 0 {
		s.logger.Debug("no devices to monitor")
		return
	}

	jobs := make(chan deviceJob)
	workers := s.workerCount(len(devices), includeSNMP)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				s.monitorDevice(ctx, job.device, job.config, includeSNMP)
			}
		}()
	}

	for _, device := range devices {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- deviceJob{device: device, config: configForDevice(configs, device.ID)}:
		}
	}
	close(jobs)
	wg.Wait()
}

func (s *Service) loadTargets(ctx context.Context) ([]models.Device, map[uint64]models.MonitoringConfig, error) {
	var devices []models.Device
	if err := s.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return nil, nil, err
	}

	var rows []models.MonitoringConfig
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	configs := make(map[uint64]models.MonitoringConfig, len(rows))
	for _, row := range rows {
		configs[row.DeviceID] = row
	}
	return devices, configs, nil
}

func (s *Service) monitorDevice(ctx context.Context, device models.Device, cfg models.MonitoringConfig, includeSNMP bool) {
	status := models.DeviceStatus{
		DeviceID: device.ID,
		LastSeen: time.Now(),
	}

	measured := false
	var health *models.DeviceStatusValue

	if s.cfg.Ping.Enabled && cfg.PingEnabled {
		result, err := runPing(ctx, device.IP, s.cfg.Ping.Count, s.cfg.Ping.Timeout)
		if err != nil {
			s.logger.Warn("ping probe failed", "device_id", device.ID, "ip", device.IP, "error", err)
		}
		status.Latency = result.latencyMS
		status.PacketLoss = result.packetLoss
		measured = true
		health = s.healthFromPing(result)
	}

	if includeSNMP {
		result, err := s.pollSNMP(ctx, device.IP)
		if err != nil {
			s.logger.Warn("snmp probe failed", "device_id", device.ID, "ip", device.IP, "error", err)
		} else {
			if result.cpuUsage != nil {
				status.CPUUsage = *result.cpuUsage
				measured = true
			}
			if result.memoryUsage != nil {
				status.MemoryUsage = *result.memoryUsage
				measured = true
			}
			if health == nil && (result.cpuUsage != nil || result.memoryUsage != nil) {
				value := models.DeviceStatusOnline
				health = &value
			}
		}
	}

	if !measured {
		return
	}
	if err := s.db.WithContext(ctx).Create(&status).Error; err != nil {
		s.logger.Error("failed to save device status", "device_id", device.ID, "error", err)
		return
	}
	if health != nil {
		if err := s.db.WithContext(ctx).
			Model(&models.Device{}).
			Where("id = ?", device.ID).
			Update("status", *health).Error; err != nil {
			s.logger.Error("failed to update device health", "device_id", device.ID, "error", err)
		}
	}
}

func runPing(ctx context.Context, target string, count int, timeout time.Duration) (pingResult, error) {
	if strings.TrimSpace(target) == "" {
		return pingResult{packetLoss: 100}, fmt.Errorf("target IP is empty")
	}

	deadline := time.Duration(count)*timeout + time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	timeoutSeconds := int(math.Ceil(timeout.Seconds()))
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}

	cmd := exec.CommandContext(
		cmdCtx,
		"ping",
		"-c", strconv.Itoa(count),
		"-W", strconv.Itoa(timeoutSeconds),
		target,
	)
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	result := parsePingOutput(output)
	if cmdCtx.Err() != nil {
		return result, cmdCtx.Err()
	}
	if err != nil && !result.success {
		return result, fmt.Errorf("%w: %s", err, strings.TrimSpace(output))
	}
	return result, nil
}

func parsePingOutput(output string) pingResult {
	result := pingResult{packetLoss: 100}

	if matches := packetLossPattern.FindStringSubmatch(output); len(matches) == 2 {
		if parsed, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.packetLoss = parsed
		}
	}
	if matches := rttPattern.FindStringSubmatch(output); len(matches) == 4 {
		if parsed, err := strconv.ParseFloat(matches[2], 64); err == nil {
			result.latencyMS = parsed
		}
	}
	result.success = result.packetLoss < 100
	return result
}

func (s *Service) healthFromPing(result pingResult) *models.DeviceStatusValue {
	value := models.DeviceStatusOffline
	if result.success && !s.pingWarning(result) {
		value = models.DeviceStatusOnline
	} else if result.success {
		value = models.DeviceStatusWarning
	}
	return &value
}

func (s *Service) pingWarning(result pingResult) bool {
	if result.packetLoss > 0 {
		if s.cfg.Ping.WarningPacketLossPercent <= 0 {
			return true
		}
		if result.packetLoss >= s.cfg.Ping.WarningPacketLossPercent {
			return true
		}
	}
	return s.cfg.Ping.WarningLatencyMS > 0 && result.latencyMS >= s.cfg.Ping.WarningLatencyMS
}

func (s *Service) pollSNMP(ctx context.Context, target string) (snmpResult, error) {
	if err := ctx.Err(); err != nil {
		return snmpResult{}, err
	}
	if strings.TrimSpace(target) == "" {
		return snmpResult{}, fmt.Errorf("target IP is empty")
	}

	oids := make([]string, 0, 2)
	cpuOID := strings.TrimSpace(s.cfg.SNMP.CPUOID)
	memoryOID := strings.TrimSpace(s.cfg.SNMP.MemoryOID)
	if cpuOID != "" {
		oids = append(oids, cpuOID)
	}
	if memoryOID != "" {
		oids = append(oids, memoryOID)
	}
	if len(oids) == 0 {
		return snmpResult{}, nil
	}

	client := &gosnmp.GoSNMP{
		Target:    target,
		Port:      s.cfg.SNMP.Port,
		Community: s.cfg.SNMP.Community,
		Version:   snmpVersion(s.cfg.SNMP.Version),
		Timeout:   s.cfg.SNMP.Timeout,
		Retries:   s.cfg.SNMP.Retries,
	}
	if err := client.Connect(); err != nil {
		return snmpResult{}, err
	}
	defer client.Conn.Close()

	packet, err := client.Get(oids)
	if err != nil {
		return snmpResult{}, err
	}

	result := snmpResult{}
	for i, variable := range packet.Variables {
		value, ok := numericSNMPValue(variable.Value)
		if !ok {
			continue
		}
		if oids[i] == cpuOID {
			result.cpuUsage = &value
		}
		if oids[i] == memoryOID {
			result.memoryUsage = &value
		}
	}
	return result, ctx.Err()
}

func numericSNMPValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil
	case []byte:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func snmpVersion(raw string) gosnmp.SnmpVersion {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "v1":
		return gosnmp.Version1
	default:
		return gosnmp.Version2c
	}
}

func (s *Service) snmpReady() bool {
	return s.cfg.SNMP.Enabled &&
		(strings.TrimSpace(s.cfg.SNMP.CPUOID) != "" || strings.TrimSpace(s.cfg.SNMP.MemoryOID) != "")
}

func (s *Service) pollInterval() time.Duration {
	if s.cfg.Ping.Enabled && s.snmpReady() && s.cfg.SNMP.Interval < s.cfg.Ping.Interval {
		return s.cfg.SNMP.Interval
	}
	if s.cfg.Ping.Enabled {
		return s.cfg.Ping.Interval
	}
	if s.snmpReady() {
		return s.cfg.SNMP.Interval
	}
	return time.Minute
}

func (s *Service) workerCount(deviceCount int, includeSNMP bool) int {
	workers := s.cfg.Ping.Workers
	if includeSNMP && s.cfg.SNMP.Workers > workers {
		workers = s.cfg.SNMP.Workers
	}
	if workers < 1 {
		workers = 1
	}
	if deviceCount > 0 && workers > deviceCount {
		return deviceCount
	}
	return workers
}

func configForDevice(configs map[uint64]models.MonitoringConfig, deviceID uint64) models.MonitoringConfig {
	if cfg, ok := configs[deviceID]; ok {
		return cfg
	}
	return models.MonitoringConfig{
		DeviceID:     deviceID,
		PingEnabled:  true,
		PingInterval: 5,
		TCPInterval:  30,
	}
}

func normalizeConfig(cfg config.MonitoringConfig) config.MonitoringConfig {
	if cfg.Ping.Interval <= 0 {
		cfg.Ping.Interval = 5 * time.Second
	}
	if cfg.Ping.Timeout <= 0 {
		cfg.Ping.Timeout = 3 * time.Second
	}
	if cfg.Ping.Count <= 0 {
		cfg.Ping.Count = 3
	}
	if cfg.Ping.Workers <= 0 {
		cfg.Ping.Workers = 64
	}
	if cfg.SNMP.Interval <= 0 {
		cfg.SNMP.Interval = time.Minute
	}
	if cfg.SNMP.Timeout <= 0 {
		cfg.SNMP.Timeout = 3 * time.Second
	}
	if cfg.SNMP.Retries < 0 {
		cfg.SNMP.Retries = 0
	}
	if cfg.SNMP.Community == "" {
		cfg.SNMP.Community = "public"
	}
	if cfg.SNMP.Port == 0 {
		cfg.SNMP.Port = 161
	}
	if cfg.SNMP.Workers <= 0 {
		cfg.SNMP.Workers = 64
	}
	return cfg
}

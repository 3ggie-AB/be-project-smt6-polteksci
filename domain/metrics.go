package domain

import "time"

type PingMetric struct {
	DeviceID     uint      `json:"device_id"`
	TargetID     uint      `json:"target_id"`
	Workspace    string    `json:"workspace"`
	IP           string    `json:"ip"`
	LatencyMS    float64   `json:"latency_ms"`
	PacketLoss   float64   `json:"packet_loss"`
	ResponseTime float64   `json:"response_time_ms"`
	StatusUp     bool      `json:"status_up"`
	Timestamp    time.Time `json:"timestamp"`
}

type TCPMetric struct {
	DeviceID          uint      `json:"device_id"`
	TargetID          uint      `json:"target_id"`
	Workspace         string    `json:"workspace"`
	IP                string    `json:"ip"`
	Port              int       `json:"port"`
	ConnectDurationMS float64   `json:"connect_duration_ms"`
	Success           bool      `json:"success"`
	Timeout           bool      `json:"timeout"`
	Error             string    `json:"error,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

type APMetric struct {
	DeviceID      uint      `json:"device_id"`
	Workspace     string    `json:"workspace"`
	APName        string    `json:"ap_name"`
	IP            string    `json:"ip"`
	ClientCount   int       `json:"client_count"`
	CPU           float64   `json:"cpu"`
	Memory        float64   `json:"memory"`
	RSSI          float64   `json:"rssi"`
	ThroughputBPS float64   `json:"throughput_bps"`
	Online        bool      `json:"online"`
	UptimeSeconds float64   `json:"uptime_seconds,omitempty"`
	Source        string    `json:"source"`
	Timestamp     time.Time `json:"timestamp"`
}

type AnomalyMetric struct {
	DeviceID            uint      `json:"device_id"`
	TargetID            uint      `json:"target_id"`
	Workspace           string    `json:"workspace"`
	IP                  string    `json:"ip"`
	Score               float64   `json:"score"`
	LatencyRollingAvgMS float64   `json:"latency_rolling_avg_ms"`
	PacketLossRatio     float64   `json:"packet_loss_ratio"`
	APLoadScore         float64   `json:"ap_load_score"`
	RoamingFrequency    float64   `json:"roaming_frequency"`
	TrafficAnomalyScore float64   `json:"traffic_anomaly_score"`
	Model               string    `json:"model"`
	Timestamp           time.Time `json:"timestamp"`
}

type SyslogEvent struct {
	DeviceID  uint      `json:"device_id"`
	Workspace string    `json:"workspace"`
	IP        string    `json:"ip"`
	Facility  string    `json:"facility"`
	Severity  string    `json:"severity"`
	Hostname  string    `json:"hostname"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type RealtimeEvent struct {
	Type       string         `json:"type"`
	Severity   string         `json:"severity"`
	Workspace  string         `json:"workspace"`
	DeviceID   uint           `json:"device_id,omitempty"`
	TargetID   uint           `json:"target_id,omitempty"`
	IP         string         `json:"ip,omitempty"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	Attributes map[string]any `json:"attributes,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type FeatureVector struct {
	DeviceID            uint      `json:"device_id"`
	TargetID            uint      `json:"target_id"`
	Workspace           string    `json:"workspace"`
	LatencyRollingAvgMS float64   `json:"latency_rolling_avg_ms"`
	PacketLossRatio     float64   `json:"packet_loss_ratio"`
	APLoadScore         float64   `json:"ap_load_score"`
	RoamingFrequency    float64   `json:"roaming_frequency"`
	TrafficAnomalyScore float64   `json:"traffic_anomaly_score"`
	Timestamp           time.Time `json:"timestamp"`
}

func (v FeatureVector) ONNXInput() []float32 {
	return []float32{
		float32(v.LatencyRollingAvgMS),
		float32(v.PacketLossRatio),
		float32(v.APLoadScore),
		float32(v.RoamingFrequency),
		float32(v.TrafficAnomalyScore),
	}
}

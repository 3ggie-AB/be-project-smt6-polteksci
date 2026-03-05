package models

import (
	"time"
)

// PingResult menyimpan hasil ping ke setiap IP
type PingResult struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	IPAddress   string    `json:"ip_address" gorm:"index;not null"`
	Label       string    `json:"label"`
	IsReachable bool      `json:"is_reachable"`
	LatencyMs   float64   `json:"latency_ms"`
	PacketLoss  float64   `json:"packet_loss"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}

// Target adalah daftar IP yang akan di-ping
type Target struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	IPAddress string    `json:"ip_address" gorm:"uniqueIndex;not null"`
	Label     string    `json:"label"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
}

// Survey adalah form angket kepuasan jaringan
type Survey struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	RespondentName  string    `json:"respondent_name"`
	Location        string    `json:"location"`
	// Pertanyaan 1-5 (skala Likert 1-5)
	Q1Speed         int       `json:"q1_speed"`         // Kecepatan internet memadai?
	Q2Stability     int       `json:"q2_stability"`     // Koneksi stabil?
	Q3Latency       int       `json:"q3_latency"`       // Latensi terasa rendah?
	Q4Availability  int       `json:"q4_availability"`  // Internet selalu tersedia?
	Q5Satisfaction  int       `json:"q5_satisfaction"`  // Kepuasan keseluruhan?
	Comment         string    `json:"comment"`
	AvgScore        float64   `json:"avg_score" gorm:"->"`
	CreatedAt       time.Time `json:"created_at" gorm:"index"`
}

// CorrelationResult adalah hasil perhitungan korelasi
type CorrelationResult struct {
	Period          string  `json:"period"`
	AvgLatency      float64 `json:"avg_latency"`
	AvgPacketLoss   float64 `json:"avg_packet_loss"`
	UptimePercent   float64 `json:"uptime_percent"`
	AvgSatisfaction float64 `json:"avg_satisfaction"`
	SurveyCount     int     `json:"survey_count"`
	PingCount       int     `json:"ping_count"`
	// Korelasi Pearson
	CorrelationLatencySatisfaction   float64 `json:"correlation_latency_satisfaction"`
	CorrelationUptimeSatisfaction    float64 `json:"correlation_uptime_satisfaction"`
	CorrelationPacketlossSatisfaction float64 `json:"correlation_packetloss_satisfaction"`
}
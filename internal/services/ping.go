package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	probing "github.com/go-ping/ping"
)

type PingResult struct {
	DeviceID        uint      `json:"device_id"`
	DeviceName      string    `json:"device_name"`
	IPAddress       string    `json:"ip_address"`
	PacketsSent     int       `json:"packets_sent"`
	PacketsReceived int       `json:"packets_received"`
	PacketLoss      float64   `json:"packet_loss"`
	MinRTT          float64   `json:"min_rtt_ms"`
	AvgRTT          float64   `json:"avg_rtt_ms"`
	MaxRTT          float64   `json:"max_rtt_ms"`
	Status          string    `json:"status"`
	CheckedAt       time.Time `json:"checked_at"`
	Error           string    `json:"error,omitempty"`
}

type PingService struct {
	ch driver.Conn
	db string
}

func NewPingService(ch driver.Conn, db string) *PingService {
	return &PingService{ch: ch, db: db}
}

func (s *PingService) Ping(deviceID uint, deviceName, ip string, count int, checkedBy uint) (*PingResult, error) {
	result := &PingResult{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		IPAddress:  ip,
		CheckedAt:  time.Now(),
	}

	if count <= 0 {
		count = 4
	}

	if net.ParseIP(ip) == nil {
		ips, err := net.LookupHost(ip)
		if err != nil || len(ips) == 0 {
			result.Status = "down"
			result.Error = fmt.Sprintf("Tidak dapat resolve host: %s", ip)
			return result, nil
		}
		ip = ips[0]
	}

	pinger, err := probing.NewPinger(ip)
	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result, nil
	}

	pinger.Count = count
	pinger.Timeout = 10 * time.Second
	pinger.SetPrivileged(true)

	if err := pinger.Run(); err != nil {
		pinger.SetPrivileged(false)
		if err2 := pinger.Run(); err2 != nil {
			result.Status = "down"
			result.Error = err.Error()
			return result, nil
		}
	}

	stats := pinger.Statistics()
	result.PacketsSent = stats.PacketsSent
	result.PacketsReceived = stats.PacketsRecv
	result.PacketLoss = stats.PacketLoss
	result.MinRTT = float64(stats.MinRtt.Microseconds()) / 1000.0
	result.AvgRTT = float64(stats.AvgRtt.Microseconds()) / 1000.0
	result.MaxRTT = float64(stats.MaxRtt.Microseconds()) / 1000.0

	if stats.PacketsRecv > 0 {
		result.Status = "up"
	} else {
		result.Status = "down"
	}

	if err := s.saveToCH(result, checkedBy); err != nil {
		log.Printf("⚠️  Gagal simpan ping result ke ClickHouse: %v", err)
	}

	return result, nil
}

func (s *PingService) saveToCH(r *PingResult, checkedBy uint) error {
	ctx := context.Background()
	query := fmt.Sprintf(`
		INSERT INTO %s.ping_results
		(device_id, device_name, ip_address, packets_sent, packets_received, packet_loss,
		 min_rtt, avg_rtt, max_rtt, status, checked_by, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.db)

	return s.ch.Exec(ctx, query,
		r.DeviceID, r.DeviceName, r.IPAddress,
		r.PacketsSent, r.PacketsReceived, r.PacketLoss,
		r.MinRTT, r.AvgRTT, r.MaxRTT,
		r.Status, checkedBy, r.CheckedAt,
	)
}

func (s *PingService) GetHistory(deviceID uint, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT id, device_id, device_name, ip_address, packets_sent, packets_received,
		       packet_loss, min_rtt, avg_rtt, max_rtt, status, checked_by, checked_at
		FROM %s.ping_results
		WHERE device_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`, s.db)

	rows, err := s.ch.Query(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var (
			id, deviceName, ipAddr, status string
			dID, checkedBy                  uint64
			sent, received                  uint8
			loss, minR, avgR, maxR          float64
			checkedAt                       time.Time
		)
		if err := rows.Scan(&id, &dID, &deviceName, &ipAddr, &sent, &received,
			&loss, &minR, &avgR, &maxR, &status, &checkedBy, &checkedAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id": id, "device_id": dID, "device_name": deviceName,
			"ip_address": ipAddr, "packets_sent": sent, "packets_received": received,
			"packet_loss": loss, "min_rtt_ms": minR, "avg_rtt_ms": avgR, "max_rtt_ms": maxR,
			"status": status, "checked_by": checkedBy, "checked_at": checkedAt,
		})
	}
	return results, nil
}

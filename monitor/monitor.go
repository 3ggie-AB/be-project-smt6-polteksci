package monitor

import (
	"log"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"project_smt6/db"
	"project_smt6/models"

	probing "github.com/go-ping/ping"
)

type MonitorService struct {
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewMonitorService() *MonitorService {
	return &MonitorService{
		stopChan: make(chan struct{}),
	}
}

// Start memulai goroutine ping setiap 5 detik
func (m *MonitorService) Start() {
	log.Println("🚀 Ping monitor dimulai (interval: 5 detik)")
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Langsung ping pertama kali
		m.pingAllTargets()

		for {
			select {
			case <-ticker.C:
				m.pingAllTargets()
			case <-m.stopChan:
				log.Println("🛑 Ping monitor dihentikan")
				return
			}
		}
	}()
}

func (m *MonitorService) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

func (m *MonitorService) pingAllTargets() {
	var targets []models.Target
	db.DB.Where("is_active = ?", true).Find(&targets)

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t models.Target) {
			defer wg.Done()
			result := pingHost(t)
			if err := db.DB.Create(&result).Error; err != nil {
				log.Printf("⚠️ Gagal simpan ping result untuk %s: %v", t.IPAddress, err)
			}
		}(target)
	}
	wg.Wait()
}

func pingHost(target models.Target) models.PingResult {
	result := models.PingResult{
		IPAddress: target.IPAddress,
		Label:     target.Label,
		CreatedAt: time.Now(),
	}

	// Cek apakah berjalan sebagai root (diperlukan untuk ICMP raw socket)
	isRoot := os.Getuid() == 0

	if isRoot {
		// Gunakan ICMP ping proper
		pinger, err := probing.NewPinger(target.IPAddress)
		if err != nil {
			result.IsReachable = false
			return result
		}

		pinger.Count = 3
		pinger.Timeout = 3 * time.Second
		pinger.SetPrivileged(true)

		err = pinger.Run()
		if err != nil {
			result.IsReachable = false
			return result
		}

		stats := pinger.Statistics()
		result.IsReachable = stats.PacketsRecv > 0
		result.PacketLoss = stats.PacketLoss
		if stats.PacketsRecv > 0 {
			result.LatencyMs = math.Round(float64(stats.AvgRtt.Microseconds())/1000*100) / 100
		}
	} else {
		// Fallback: TCP dial ke port 80/443
		result.IsReachable, result.LatencyMs = tcpProbe(target.IPAddress)
	}

	return result
}

func tcpProbe(ip string) (bool, float64) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", ip+":80", 3*time.Second)
	elapsed := time.Since(start)
	if err != nil {
		// Coba port 443
		start = time.Now()
		conn, err = net.DialTimeout("tcp", ip+":443", 3*time.Second)
		elapsed = time.Since(start)
		if err != nil {
			return false, 0
		}
	}
	conn.Close()
	return true, math.Round(float64(elapsed.Milliseconds())*100) / 100
}
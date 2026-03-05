package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"project_smt6/db"
	"project_smt6/models"

	"github.com/gin-gonic/gin"
)

// ─── TARGETS ────────────────────────────────────────────────────────────────

type CombinedData struct {
	Date            string  `json:"date"`
	AvgLatency      float64 `json:"avg_latency"`
	AvgPacketLoss   float64 `json:"avg_packet_loss"`
	UptimePercent   float64 `json:"uptime_percent"`
	AvgSatisfaction float64 `json:"avg_satisfaction"`
	SurveyCount     int     `json:"survey_count"`
}

func GetTargets(c *gin.Context) {
	var targets []models.Target
	db.DB.Find(&targets)
	c.JSON(http.StatusOK, targets)
}

func CreateTarget(c *gin.Context) {
	var target models.Target
	if err := c.ShouldBindJSON(&target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target.IsActive = true
	if err := db.DB.Create(&target).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah target"})
		return
	}
	c.JSON(http.StatusCreated, target)
}

func DeleteTarget(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Target{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Target dihapus"})
}

// ─── PING RESULTS ────────────────────────────────────────────────────────────

func GetLatestPings(c *gin.Context) {
	// Ambil ping terbaru per IP (untuk dashboard real-time)
	var results []models.PingResult
	db.DB.
		Order("created_at DESC").
		Limit(100).
		Find(&results)
	c.JSON(http.StatusOK, results)
}

func GetPingHistory(c *gin.Context) {
	ip := c.Query("ip")
	hours := c.DefaultQuery("hours", "1")
	h, _ := strconv.Atoi(hours)

	since := time.Now().Add(-time.Duration(h) * time.Hour)

	var results []models.PingResult
	query := db.DB.Where("created_at >= ?", since).Order("created_at ASC")
	if ip != "" {
		query = query.Where("ip_address = ?", ip)
	}
	query.Find(&results)
	c.JSON(http.StatusOK, results)
}

func GetPingSummary(c *gin.Context) {
	// Summary per IP: uptime%, avg latency, total pings dalam 1 jam terakhir
	since := time.Now().Add(-1 * time.Hour)

	type Summary struct {
		IPAddress     string  `json:"ip_address"`
		Label         string  `json:"label"`
		TotalPings    int64   `json:"total_pings"`
		ReachablePings int64  `json:"reachable_pings"`
		UptimePercent float64 `json:"uptime_percent"`
		AvgLatencyMs  float64 `json:"avg_latency_ms"`
		LastSeen      *time.Time `json:"last_seen"`
		LastStatus    bool    `json:"last_status"`
	}

	var targets []models.Target
	db.DB.Where("is_active = ?", true).Find(&targets)

	summaries := make([]Summary, 0)
	for _, t := range targets {
		var total, reachable int64
		db.DB.Model(&models.PingResult{}).
			Where("ip_address = ? AND created_at >= ?", t.IPAddress, since).
			Count(&total)
		db.DB.Model(&models.PingResult{}).
			Where("ip_address = ? AND created_at >= ? AND is_reachable = ?", t.IPAddress, since, true).
			Count(&reachable)

		var avgLatency float64
		db.DB.Model(&models.PingResult{}).
			Where("ip_address = ? AND created_at >= ? AND is_reachable = ?", t.IPAddress, since, true).
			Select("COALESCE(AVG(latency_ms), 0)").
			Scan(&avgLatency)

		var lastPing models.PingResult
		db.DB.Where("ip_address = ?", t.IPAddress).
			Order("created_at DESC").
			First(&lastPing)

		uptime := 0.0
		if total > 0 {
			uptime = math.Round(float64(reachable)/float64(total)*10000) / 100
		}

		summaries = append(summaries, Summary{
			IPAddress:      t.IPAddress,
			Label:          t.Label,
			TotalPings:     total,
			ReachablePings: reachable,
			UptimePercent:  uptime,
			AvgLatencyMs:   math.Round(avgLatency*100) / 100,
			LastSeen:       &lastPing.CreatedAt,
			LastStatus:     lastPing.IsReachable,
		})
	}

	c.JSON(http.StatusOK, summaries)
}

// ─── SURVEY ──────────────────────────────────────────────────────────────────

func SubmitSurvey(c *gin.Context) {
	var survey models.Survey
	if err := c.ShouldBindJSON(&survey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi skor 1-5
	scores := []int{survey.Q1Speed, survey.Q2Stability, survey.Q3Latency, survey.Q4Availability, survey.Q5Satisfaction}
	for _, s := range scores {
		if s < 1 || s > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Skor harus antara 1-5"})
			return
		}
	}

	survey.CreatedAt = time.Now()
	if err := db.DB.Create(&survey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan angket"})
		return
	}

	avg := float64(survey.Q1Speed+survey.Q2Stability+survey.Q3Latency+survey.Q4Availability+survey.Q5Satisfaction) / 5.0
	c.JSON(http.StatusCreated, gin.H{
		"message":   "Terima kasih atas penilaian Anda!",
		"survey":    survey,
		"avg_score": math.Round(avg*100) / 100,
	})
}

func GetSurveys(c *gin.Context) {
	var surveys []models.Survey
	db.DB.Order("created_at DESC").Find(&surveys)

	// Hitung avg_score untuk setiap survey
	type SurveyResponse struct {
		models.Survey
		AvgScore float64 `json:"avg_score"`
	}
	responses := make([]SurveyResponse, len(surveys))
	for i, s := range surveys {
		avg := float64(s.Q1Speed+s.Q2Stability+s.Q3Latency+s.Q4Availability+s.Q5Satisfaction) / 5.0
		responses[i] = SurveyResponse{Survey: s, AvgScore: math.Round(avg*100) / 100}
	}

	c.JSON(http.StatusOK, responses)
}

// ─── KORELASI ────────────────────────────────────────────────────────────────

func GetCorrelation(c *gin.Context) {
	days := c.DefaultQuery("days", "7")
	d, _ := strconv.Atoi(days)
	since := time.Now().AddDate(0, 0, -d)

	// Ambil data ping harian
	type DailyPing struct {
		Date          string  `json:"date"`
		AvgLatency    float64 `json:"avg_latency"`
		AvgPacketLoss float64 `json:"avg_packet_loss"`
		UptimePercent float64 `json:"uptime_percent"`
	}

	var dailyPings []DailyPing
	db.DB.Raw(`
		SELECT 
			DATE(created_at AT TIME ZONE 'Asia/Jakarta') as date,
			COALESCE(AVG(CASE WHEN is_reachable THEN latency_ms ELSE NULL END), 0) as avg_latency,
			COALESCE(AVG(packet_loss), 0) as avg_packet_loss,
			ROUND(
				COUNT(CASE WHEN is_reachable THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0), 2
			) as uptime_percent
		FROM ping_results
		WHERE created_at >= ?
		GROUP BY DATE(created_at AT TIME ZONE 'Asia/Jakarta')
		ORDER BY date ASC
	`, since).Scan(&dailyPings)

	// Ambil data survey harian
	type DailySurvey struct {
		Date            string  `json:"date"`
		AvgSatisfaction float64 `json:"avg_satisfaction"`
		Count           int     `json:"count"`
	}

	var dailySurveys []DailySurvey
	db.DB.Raw(`
		SELECT 
			DATE(created_at AT TIME ZONE 'Asia/Jakarta') as date,
			AVG((q1_speed + q2_stability + q3_latency + q4_availability + q5_satisfaction) / 5.0) as avg_satisfaction,
			COUNT(*) as count
		FROM surveys
		WHERE created_at >= ?
		GROUP BY DATE(created_at AT TIME ZONE 'Asia/Jakarta')
		ORDER BY date ASC
	`, since).Scan(&dailySurveys)

	// Gabungkan data berdasarkan tanggal
	surveyMap := make(map[string]DailySurvey)
	for _, s := range dailySurveys {
		surveyMap[s.Date] = s
	}

	combined := make([]CombinedData, 0)
	for _, p := range dailyPings {
		s := surveyMap[p.Date]
		combined = append(combined, CombinedData{
			Date:            p.Date,
			AvgLatency:      math.Round(p.AvgLatency*100) / 100,
			AvgPacketLoss:   math.Round(p.AvgPacketLoss*100) / 100,
			UptimePercent:   p.UptimePercent,
			AvgSatisfaction: math.Round(s.AvgSatisfaction*100) / 100,
			SurveyCount:     s.Count,
		})
	}

	// Hitung Pearson Correlation
	corrLatency := pearsonCorrelation(combined, "latency")
	corrUptime := pearsonCorrelation(combined, "uptime")
	corrPacketloss := pearsonCorrelation(combined, "packetloss")

	c.JSON(http.StatusOK, gin.H{
		"period_days":     d,
		"data":            combined,
		"correlations": gin.H{
			"latency_vs_satisfaction":    math.Round(corrLatency*1000) / 1000,
			"uptime_vs_satisfaction":     math.Round(corrUptime*1000) / 1000,
			"packetloss_vs_satisfaction": math.Round(corrPacketloss*1000) / 1000,
		},
		"interpretation": interpretCorrelations(corrLatency, corrUptime, corrPacketloss),
	})
}

func pearsonCorrelation(data []CombinedData, metric string) float64 {
	n := 0
	var xVals, yVals []float64
	for _, d := range data {
		if d.AvgSatisfaction == 0 {
			continue
		}
		var x float64
		switch metric {
		case "latency":
			x = d.AvgLatency
		case "uptime":
			x = d.UptimePercent
		case "packetloss":
			x = d.AvgPacketLoss
		}
		xVals = append(xVals, x)
		yVals = append(yVals, d.AvgSatisfaction)
		n++
	}

	if n < 2 {
		return 0
	}

	// Hitung mean
	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += xVals[i]
		sumY += yVals[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Hitung korelasi Pearson
	var num, denX, denY float64
	for i := 0; i < n; i++ {
		dx := xVals[i] - meanX
		dy := yVals[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}

	denominator := math.Sqrt(denX * denY)
	if denominator == 0 {
		return 0
	}
	return num / denominator
}

func interpretCorrelations(latency, uptime, packetloss float64) gin.H {
	return gin.H{
		"latency": interpretR(latency, true),
		"uptime":  interpretR(uptime, false),
		"packetloss": interpretR(packetloss, true),
	}
}

func interpretR(r float64, inverse bool) string {
	abs := math.Abs(r)
	var strength string
	switch {
	case abs >= 0.7:
		strength = "kuat"
	case abs >= 0.4:
		strength = "sedang"
	case abs >= 0.2:
		strength = "lemah"
	default:
		return "Tidak ada korelasi signifikan"
	}

	var direction string
	if (r < 0 && inverse) || (r > 0 && !inverse) {
		direction = "positif"
	} else {
		direction = "negatif"
	}

	return "Korelasi " + strength + " " + direction
}
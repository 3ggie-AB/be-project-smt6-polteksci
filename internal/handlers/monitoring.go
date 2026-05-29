package handlers

import (
	"network-monitor/internal/middleware"
	"network-monitor/internal/models"
	"network-monitor/internal/services"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MonitoringHandler struct {
	db          *gorm.DB
	pingService *services.PingService
	snmpService *services.SNMPService
}

func NewMonitoringHandler(db *gorm.DB, ping *services.PingService, snmp *services.SNMPService) *MonitoringHandler {
	return &MonitoringHandler{db: db, pingService: ping, snmpService: snmp}
}

func (h *MonitoringHandler) ListDevices(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")
	deviceType := c.Query("type", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	query := h.db.Model(&models.Device{}).Preload("CreatedBy.Role")

	if search != "" {
		query = query.Where("name LIKE ? OR ip_address LIKE ? OR location LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if deviceType != "" {
		query = query.Where("type = ?", deviceType)
	}

	var total int64
	query.Count(&total)

	var devices []models.Device
	if err := query.Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil data perangkat"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    devices,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *MonitoringHandler) GetDevice(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var device models.Device
	if err := h.db.Preload("CreatedBy.Role").First(&device, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Perangkat tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"success": true, "data": device})
}

func (h *MonitoringHandler) CreateDevice(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Name          string `json:"name"`
		IPAddress     string `json:"ip_address"`
		Type          string `json:"type"`
		Location      string `json:"location"`
		Description   string `json:"description"`
		SNMPCommunity string `json:"snmp_community"`
		SNMPVersion   string `json:"snmp_version"`
		SNMPPort      int    `json:"snmp_port"`
		IsActive      *bool  `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Name == "" || req.IPAddress == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Nama dan IP address wajib diisi"})
	}

	var existing models.Device
	if err := h.db.Where("ip_address = ?", req.IPAddress).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "message": "IP address sudah terdaftar"})
	}

	if req.SNMPCommunity == "" {
		req.SNMPCommunity = "public"
	}
	if req.SNMPVersion == "" {
		req.SNMPVersion = "2c"
	}
	if req.SNMPPort == 0 {
		req.SNMPPort = 161
	}
	if req.Type == "" {
		req.Type = "other"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	device := models.Device{
		Name:          req.Name,
		IPAddress:     req.IPAddress,
		Type:          req.Type,
		Location:      req.Location,
		Description:   req.Description,
		SNMPCommunity: req.SNMPCommunity,
		SNMPVersion:   req.SNMPVersion,
		SNMPPort:      req.SNMPPort,
		IsActive:      isActive,
		CreatedByID:   userID,
	}

	if err := h.db.Create(&device).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menambah perangkat"})
	}

	h.db.Preload("CreatedBy").First(&device, device.ID)
	return c.Status(201).JSON(fiber.Map{"success": true, "message": "Perangkat berhasil ditambahkan", "data": device})
}

func (h *MonitoringHandler) UpdateDevice(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var device models.Device
	if err := h.db.First(&device, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Perangkat tidak ditemukan"})
	}

	var req struct {
		Name          string `json:"name"`
		IPAddress     string `json:"ip_address"`
		Type          string `json:"type"`
		Location      string `json:"location"`
		Description   string `json:"description"`
		SNMPCommunity string `json:"snmp_community"`
		SNMPVersion   string `json:"snmp_version"`
		SNMPPort      *int   `json:"snmp_port"`
		IsActive      *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Name != "" {
		device.Name = req.Name
	}
	if req.IPAddress != "" {
		device.IPAddress = req.IPAddress
	}
	if req.Type != "" {
		device.Type = req.Type
	}
	if req.Location != "" {
		device.Location = req.Location
	}
	if req.Description != "" {
		device.Description = req.Description
	}
	if req.SNMPCommunity != "" {
		device.SNMPCommunity = req.SNMPCommunity
	}
	if req.SNMPVersion != "" {
		device.SNMPVersion = req.SNMPVersion
	}
	if req.SNMPPort != nil {
		device.SNMPPort = *req.SNMPPort
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}

	if err := h.db.Save(&device).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengupdate perangkat"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Perangkat berhasil diupdate", "data": device})
}

func (h *MonitoringHandler) DeleteDevice(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var device models.Device
	if err := h.db.First(&device, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Perangkat tidak ditemukan"})
	}

	h.db.Delete(&device)
	return c.JSON(fiber.Map{"success": true, "message": "Perangkat berhasil dihapus"})
}

func (h *MonitoringHandler) PingDevice(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var req struct {
		Count int `json:"count"`
	}
	c.BodyParser(&req)
	if req.Count == 0 {
		req.Count = 4
	}

	var device models.Device
	if err := h.db.First(&device, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Perangkat tidak ditemukan"})
	}

	userID := middleware.GetUserID(c)
	result, err := h.pingService.Ping(device.ID, device.Name, device.IPAddress, req.Count, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menjalankan ping", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Ping selesai",
		"data":    result,
	})
}

func (h *MonitoringHandler) PingCustom(c *fiber.Ctx) error {
	var req struct {
		IPAddress string `json:"ip_address"`
		Count     int    `json:"count"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}
	if req.IPAddress == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ip_address wajib diisi"})
	}
	if req.Count == 0 {
		req.Count = 4
	}

	userID := middleware.GetUserID(c)
	result, err := h.pingService.Ping(0, "Custom", req.IPAddress, req.Count, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menjalankan ping"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Ping selesai", "data": result})
}

func (h *MonitoringHandler) GetPingHistory(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit > 500 {
		limit = 500
	}

	history, err := h.pingService.GetHistory(uint(id), limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil histori ping"})
	}

	return c.JSON(fiber.Map{"success": true, "data": history})
}

func (h *MonitoringHandler) SNMPDevice(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var req struct {
		OIDs []string `json:"oids"`
	}
	c.BodyParser(&req)

	var device models.Device
	if err := h.db.First(&device, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Perangkat tidak ditemukan"})
	}

	userID := middleware.GetUserID(c)
	results, err := h.snmpService.Walk(
		device.ID, device.Name, device.IPAddress,
		device.SNMPCommunity, device.SNMPVersion, device.SNMPPort,
		req.OIDs, userID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menjalankan SNMP", "error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "SNMP berhasil", "data": results})
}

func (h *MonitoringHandler) GetSNMPHistory(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit > 500 {
		limit = 500
	}

	history, err := h.snmpService.GetHistory(uint(id), limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil histori SNMP"})
	}

	return c.JSON(fiber.Map{"success": true, "data": history})
}

func (h *MonitoringHandler) GetOIDs(c *fiber.Ctx) error {
	oids := h.snmpService.GetAvailableOIDs()
	list := make([]fiber.Map, 0, len(oids))
	for oid, name := range oids {
		list = append(list, fiber.Map{"oid": oid, "name": name})
	}
	return c.JSON(fiber.Map{"success": true, "data": list})
}

package handlers

import (
	"network-monitor/internal/middleware"
	"network-monitor/internal/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type FeedbackHandler struct {
	db *gorm.DB
}

func NewFeedbackHandler(db *gorm.DB) *FeedbackHandler {
	return &FeedbackHandler{db: db}
}

func (h *FeedbackHandler) ListFeedbacks(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	status := c.Query("status", "")
	category := c.Query("category", "")
	priority := c.Query("priority", "")
	search := c.Query("search", "")
	myOnly := c.Query("my_only", "false")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	query := h.db.Model(&models.Feedback{}).
		Preload("CreatedBy.Role").
		Preload("AssignedTo.Role").
		Preload("RespondedBy.Role")

	if role == "karyawan" || myOnly == "true" {
		query = query.Where("created_by_id = ?", userID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var feedbacks []models.Feedback
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil data feedback"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    feedbacks,
		"meta": fiber.Map{
			"total":    total,
			"page":     page,
			"limit":    limit,
			"pages":    (total + int64(limit) - 1) / int64(limit),
			"statuses": []string{"open", "in_progress", "resolved", "closed"},
		},
	})
}

func (h *FeedbackHandler) GetFeedback(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	var feedback models.Feedback
	query := h.db.Preload("CreatedBy.Role").
		Preload("AssignedTo.Role").
		Preload("RespondedBy.Role").
		First(&feedback, id)

	if query.Error != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Feedback tidak ditemukan"})
	}

	if role == "karyawan" && feedback.CreatedByID != userID {
		return c.Status(403).JSON(fiber.Map{"success": false, "message": "Anda tidak bisa melihat feedback milik orang lain"})
	}

	return c.JSON(fiber.Map{"success": true, "data": feedback})
}

func (h *FeedbackHandler) CreateFeedback(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Title       string                  `json:"title"`
		Description string                  `json:"description"`
		Category    models.FeedbackCategory `json:"category"`
		Priority    models.FeedbackPriority `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Title == "" || req.Description == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Judul dan deskripsi wajib diisi"})
	}

	if req.Category == "" {
		req.Category = models.CategoryKeluhan
	}
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}

	feedback := models.Feedback{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Priority:    req.Priority,
		Status:      models.StatusOpen,
		CreatedByID: userID,
	}

	if err := h.db.Create(&feedback).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat feedback"})
	}

	h.db.Preload("CreatedBy.Role").First(&feedback, feedback.ID)
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Feedback berhasil dikirim. Tim IT akan segera menindaklanjuti.",
		"data":    feedback,
	})
}

func (h *FeedbackHandler) UpdateFeedback(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	var feedback models.Feedback
	if err := h.db.First(&feedback, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Feedback tidak ditemukan"})
	}

	if role == "karyawan" && feedback.CreatedByID != userID {
		return c.Status(403).JSON(fiber.Map{"success": false, "message": "Anda tidak bisa mengedit feedback milik orang lain"})
	}

	if role == "karyawan" && (feedback.Status == models.StatusResolved || feedback.Status == models.StatusClosed) {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Feedback yang sudah selesai tidak bisa diedit"})
	}

	var req struct {
		Title       string                  `json:"title"`
		Description string                  `json:"description"`
		Category    models.FeedbackCategory `json:"category"`
		Priority    models.FeedbackPriority `json:"priority"`
		Status      models.FeedbackStatus   `json:"status"`
		AssignedToID *uint                  `json:"assigned_to_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Title != "" {
		feedback.Title = req.Title
	}
	if req.Description != "" {
		feedback.Description = req.Description
	}
	if req.Category != "" {
		feedback.Category = req.Category
	}

	if role == "atasan" || role == "teknisi_it" || role == "staff" {
		if req.Priority != "" {
			feedback.Priority = req.Priority
		}
		if req.Status != "" {
			feedback.Status = req.Status
		}
		if req.AssignedToID != nil {
			feedback.AssignedToID = req.AssignedToID
		}
	}

	if err := h.db.Save(&feedback).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengupdate feedback"})
	}

	h.db.Preload("CreatedBy.Role").Preload("AssignedTo.Role").First(&feedback, feedback.ID)
	return c.JSON(fiber.Map{"success": true, "message": "Feedback berhasil diupdate", "data": feedback})
}

func (h *FeedbackHandler) RespondFeedback(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var req struct {
		Response string                `json:"response"`
		Status   models.FeedbackStatus `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil || req.Response == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Response wajib diisi"})
	}

	var feedback models.Feedback
	if err := h.db.First(&feedback, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Feedback tidak ditemukan"})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	feedback.Response = req.Response
	feedback.RespondedByID = &userID
	feedback.RespondedAt = &now

	if req.Status != "" {
		feedback.Status = req.Status
	} else {
		feedback.Status = models.StatusResolved
	}

	if err := h.db.Save(&feedback).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menyimpan respons"})
	}

	h.db.Preload("CreatedBy.Role").Preload("AssignedTo.Role").Preload("RespondedBy.Role").First(&feedback, feedback.ID)
	return c.JSON(fiber.Map{"success": true, "message": "Respons berhasil dikirim", "data": feedback})
}

func (h *FeedbackHandler) DeleteFeedback(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var feedback models.Feedback
	if err := h.db.First(&feedback, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Feedback tidak ditemukan"})
	}

	h.db.Delete(&feedback)
	return c.JSON(fiber.Map{"success": true, "message": "Feedback berhasil dihapus"})
}

func (h *FeedbackHandler) GetStats(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	type StatRow struct {
		Status string
		Count  int64
	}

	var stats []StatRow
	query := h.db.Model(&models.Feedback{}).Select("status, COUNT(*) as count").Group("status")
	if role == "karyawan" {
		query = query.Where("created_by_id = ?", userID)
	}
	query.Scan(&stats)

	statusMap := map[string]int64{
		"open": 0, "in_progress": 0, "resolved": 0, "closed": 0,
	}
	var total int64
	for _, s := range stats {
		statusMap[s.Status] = s.Count
		total += s.Count
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total":       total,
			"open":        statusMap["open"],
			"in_progress": statusMap["in_progress"],
			"resolved":    statusMap["resolved"],
			"closed":      statusMap["closed"],
		},
	})
}

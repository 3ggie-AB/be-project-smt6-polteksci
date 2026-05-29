package handlers

import (
	"network-monitor/internal/middleware"
	"network-monitor/internal/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	search := c.Query("search", "")
	roleFilter := c.Query("role", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	query := h.db.Model(&models.User{}).Preload("Role")

	if search != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if roleFilter != "" {
		query = query.Joins("JOIN roles ON roles.id = users.role_id").Where("roles.name = ?", roleFilter)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil data user"})
	}

	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var user models.User
	if err := h.db.Preload("Role.Permissions").First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"success": true, "data": user.ToResponse()})
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		RoleID     uint   `json:"role_id"`
		Phone      string `json:"phone"`
		Department string `json:"department"`
		IsActive   *bool  `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" || req.RoleID == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Nama, email, password, dan role_id wajib diisi"})
	}

	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "message": "Email sudah digunakan"})
	}

	var role models.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Role tidak ditemukan"})
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user := models.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   string(hashed),
		RoleID:     req.RoleID,
		Phone:      req.Phone,
		Department: req.Department,
		IsActive:   isActive,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat user"})
	}

	h.db.Preload("Role").First(&user, user.ID)
	return c.Status(201).JSON(fiber.Map{"success": true, "message": "User berhasil dibuat", "data": user.ToResponse()})
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	currentUserID := middleware.GetUserID(c)
	currentRole := middleware.GetRole(c)

	if currentRole != "atasan" && uint(id) != currentUserID {
		return c.Status(403).JSON(fiber.Map{"success": false, "message": "Anda hanya bisa mengedit profil sendiri"})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	var req struct {
		Name       string `json:"name"`
		Phone      string `json:"phone"`
		Department string `json:"department"`
		RoleID     *uint  `json:"role_id"`
		IsActive   *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Department != "" {
		user.Department = req.Department
	}

	if currentRole == "atasan" {
		if req.RoleID != nil {
			var role models.Role
			if err := h.db.First(&role, *req.RoleID).Error; err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "Role tidak ditemukan"})
			}
			user.RoleID = *req.RoleID
		}
		if req.IsActive != nil {
			user.IsActive = *req.IsActive
		}
	}

	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengupdate user"})
	}

	h.db.Preload("Role").First(&user, user.ID)
	return c.JSON(fiber.Map{"success": true, "message": "User berhasil diupdate", "data": user.ToResponse()})
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	currentUserID := middleware.GetUserID(c)
	if uint(id) == currentUserID {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Tidak bisa menghapus akun sendiri"})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	h.db.Delete(&user)
	return c.JSON(fiber.Map{"success": true, "message": "User berhasil dihapus"})
}

func (h *UserHandler) ListRoles(c *fiber.Ctx) error {
	var roles []models.Role
	if err := h.db.Preload("Permissions").Find(&roles).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil data role"})
	}
	return c.JSON(fiber.Map{"success": true, "data": roles})
}

func (h *UserHandler) GetRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}
	var role models.Role
	if err := h.db.Preload("Permissions").First(&role, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Role tidak ditemukan"})
	}
	return c.JSON(fiber.Map{"success": true, "data": role})
}

func (h *UserHandler) UpdateRolePermissions(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var req struct {
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "Role tidak ditemukan"})
	}

	var perms []models.Permission
	h.db.Where("id IN ?", req.PermissionIDs).Find(&perms)
	h.db.Model(&role).Association("Permissions").Replace(perms)

	h.db.Preload("Permissions").First(&role, id)
	return c.JSON(fiber.Map{"success": true, "message": "Permission role berhasil diupdate", "data": role})
}

func (h *UserHandler) ListPermissions(c *fiber.Ctx) error {
	var permissions []models.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal mengambil data permission"})
	}
	return c.JSON(fiber.Map{"success": true, "data": permissions})
}

func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "ID tidak valid"})
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil || req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "new_password wajib diisi"})
	}
	if len(req.NewPassword) < 8 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Password minimal 8 karakter"})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	h.db.Model(&user).Update("password", string(hashed))

	return c.JSON(fiber.Map{"success": true, "message": "Password berhasil direset"})
}

package handlers

import (
	"network-monitor/internal/config"
	"network-monitor/internal/middleware"
	"network-monitor/internal/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Email dan password wajib diisi"})
	}

	var user models.User
	if err := h.db.Preload("Role.Permissions").Where("email = ? AND is_active = ?", req.Email, true).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "message": "Email atau password salah"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "message": "Email atau password salah"})
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, user.Role.Name, h.cfg.JWTSecret, h.cfg.JWTExpireHours)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat token"})
	}

	permissions := make([]string, 0)
	for _, p := range user.Role.Permissions {
		permissions = append(permissions, p.Name)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Login berhasil",
		"data": fiber.Map{
			"token":       token,
			"expire_in":   h.cfg.JWTExpireHours * 3600,
			"user":        user.ToResponse(),
			"permissions": permissions,
		},
	})
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Nama, email, dan password wajib diisi"})
	}

	if len(req.Password) < 8 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Password minimal 8 karakter"})
	}

	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "message": "Email sudah terdaftar"})
	}

	var karyawanRole models.Role
	if err := h.db.Where("name = ?", "karyawan").First(&karyawanRole).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Konfigurasi role tidak ditemukan"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal memproses password"})
	}

	user := models.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   string(hashed),
		RoleID:     karyawanRole.ID,
		Phone:      req.Phone,
		Department: req.Department,
		IsActive:   true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat akun"})
	}

	h.db.Preload("Role").First(&user, user.ID)

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Akun berhasil dibuat. Anda terdaftar sebagai Karyawan.",
		"data":    user.ToResponse(),
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var user models.User
	if err := h.db.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	permissions := make([]string, 0)
	for _, p := range user.Role.Permissions {
		permissions = append(permissions, p.Name)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user":        user.ToResponse(),
			"permissions": permissions,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Password lama dan baru wajib diisi"})
	}
	if len(req.NewPassword) < 8 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "Password baru minimal 8 karakter"})
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "User tidak ditemukan"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "message": "Password lama tidak sesuai"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal memproses password"})
	}

	h.db.Model(&user).Update("password", string(hashed))

	return c.JSON(fiber.Map{"success": true, "message": "Password berhasil diubah"})
}

package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	httpauth "project_smt6/app/http/auth"
	"project_smt6/app/models"
	"project_smt6/config"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	DB     *gorm.DB
	Config config.AppConfig
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthController(db *gorm.DB, cfg config.AppConfig) AuthController {
	return AuthController{DB: db, Config: cfg}
}

func (ctl AuthController) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		return errorResponse(c, fiber.StatusUnauthorized, "email or password is invalid", nil)
	}

	var user models.User
	err := ctl.DB.Where("email = ?", req.Email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		createdUser, createErr := ctl.bootstrapFirstUser(req)
		if createErr != nil {
			return errorResponse(c, fiber.StatusUnauthorized, "email or password is invalid", nil)
		}
		user = *createdUser
	} else if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to login", err)
	} else if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "email or password is invalid", nil)
	}

	token, err := randomToken()
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to create session token", err)
	}
	hashedToken := httpauth.TokenHash(token)

	session := models.Session{
		UserID:    user.ID,
		Token:     hashedToken,
		TokenHash: hashedToken,
		ExpiredAt: time.Now().Add(ctl.Config.SessionTTL),
	}
	if err := ctl.DB.Create(&session).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to create session", err)
	}

	return c.JSON(fiber.Map{
		"message": "login berhasil",
		"token":   token,
		"session": session,
		"user":    user,
	})
}

func (ctl AuthController) Logout(c *fiber.Ctx) error {
	token := httpauth.BearerToken(c)
	if token == "" {
		return errorResponse(c, fiber.StatusUnauthorized, "token is required", nil)
	}
	if err := ctl.DB.Where("token_hash = ?", httpauth.TokenHash(token)).Delete(&models.Session{}).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to logout", err)
	}
	return c.JSON(fiber.Map{"message": "logout berhasil"})
}

func (ctl AuthController) Me(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return errorResponse(c, fiber.StatusUnauthorized, "unauthenticated", nil)
	}
	return c.JSON(fiber.Map{"data": user})
}

func (ctl AuthController) bootstrapFirstUser(req loginRequest) (*models.User, error) {
	var count int64
	if err := ctl.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, gorm.ErrRecordNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	name := strings.Split(req.Email, "@")[0]
	user := models.User{
		Name:     name,
		Email:    req.Email,
		Password: string(hash),
		Role:     models.RoleSuperAdmin,
	}
	if err := ctl.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

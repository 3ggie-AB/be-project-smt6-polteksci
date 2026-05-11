package controllers

import (
	"strings"

	"project_smt6/app/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

type userPayload struct {
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     models.UserRole `json:"role"`
}

func NewUserController(db *gorm.DB) UserController {
	return UserController{DB: db}
}

func (ctl UserController) Index(c *fiber.Ctx) error {
	var users []models.User
	if err := ctl.DB.Find(&users).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to list users", err)
	}
	return c.JSON(fiber.Map{"data": users})
}

func (ctl UserController) Store(c *fiber.Ctx) error {
	var payload userPayload
	if err := c.BodyParser(&payload); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	user, err := buildUser(payload)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid user payload", err)
	}
	if err := ctl.DB.Create(&user).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to create user", err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": user})
}

func (ctl UserController) Show(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}
	var user models.User
	if err := ctl.DB.First(&user, id).Error; err != nil {
		return notFoundOrError(c, "user", err)
	}
	return c.JSON(fiber.Map{"data": user})
}

func (ctl UserController) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}
	var user models.User
	if err := ctl.DB.First(&user, id).Error; err != nil {
		return notFoundOrError(c, "user", err)
	}

	var payload userPayload
	if err := c.BodyParser(&payload); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	updates := map[string]any{}
	if strings.TrimSpace(payload.Name) != "" {
		updates["name"] = strings.TrimSpace(payload.Name)
	}
	if strings.TrimSpace(payload.Email) != "" {
		updates["email"] = strings.ToLower(strings.TrimSpace(payload.Email))
	}
	if payload.Role != "" {
		updates["role"] = payload.Role
	}
	if payload.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
		if err != nil {
			return errorResponse(c, fiber.StatusBadRequest, "failed to hash password", err)
		}
		updates["password"] = string(hash)
	}
	if err := ctl.DB.Model(&user).Updates(updates).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to update user", err)
	}
	if err := ctl.DB.First(&user, id).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to reload user", err)
	}
	return c.JSON(fiber.Map{"data": user})
}

func (ctl UserController) Destroy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}
	if err := ctl.DB.Delete(&models.User{}, id).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to delete user", err)
	}
	return c.JSON(fiber.Map{"message": "user deleted"})
}

func buildUser(payload userPayload) (models.User, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Email = strings.ToLower(strings.TrimSpace(payload.Email))
	if payload.Name == "" || payload.Email == "" || payload.Password == "" {
		return models.User{}, fiber.NewError(fiber.StatusBadRequest, "name, email, and password are required")
	}
	if payload.Role == "" {
		payload.Role = models.RoleUser
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	return models.User{
		Name:     payload.Name,
		Email:    payload.Email,
		Password: string(hash),
		Role:     payload.Role,
	}, nil
}

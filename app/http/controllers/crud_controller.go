package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResourceController[T any] struct {
	DB   *gorm.DB
	Name string
}

func NewResourceController[T any](db *gorm.DB, name string) ResourceController[T] {
	return ResourceController[T]{DB: db, Name: name}
}

func (ctl ResourceController[T]) Index(c *fiber.Ctx) error {
	var rows []T
	if err := ctl.DB.Preload(clause.Associations).Find(&rows).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to list "+ctl.Name, err)
	}
	return c.JSON(fiber.Map{"data": rows})
}

func (ctl ResourceController[T]) Store(c *fiber.Ctx) error {
	var row T
	if err := c.BodyParser(&row); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := ctl.DB.Create(&row).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to create "+ctl.Name, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": row})
}

func (ctl ResourceController[T]) Show(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}

	var row T
	if err := ctl.DB.Preload(clause.Associations).First(&row, id).Error; err != nil {
		return notFoundOrError(c, ctl.Name, err)
	}
	return c.JSON(fiber.Map{"data": row})
}

func (ctl ResourceController[T]) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}

	var row T
	if err := ctl.DB.First(&row, id).Error; err != nil {
		return notFoundOrError(c, ctl.Name, err)
	}

	var payload map[string]any
	if err := c.BodyParser(&payload); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	delete(payload, "id")
	delete(payload, "created_at")

	if err := ctl.DB.Model(&row).Updates(payload).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to update "+ctl.Name, err)
	}
	if err := ctl.DB.Preload(clause.Associations).First(&row, id).Error; err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "failed to reload "+ctl.Name, err)
	}
	return c.JSON(fiber.Map{"data": row})
}

func (ctl ResourceController[T]) Destroy(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "invalid id", err)
	}

	var row T
	if err := ctl.DB.First(&row, id).Error; err != nil {
		return notFoundOrError(c, ctl.Name, err)
	}
	if err := ctl.DB.Delete(&row).Error; err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "failed to delete "+ctl.Name, err)
	}
	return c.JSON(fiber.Map{"message": ctl.Name + " deleted"})
}

func parseID(c *fiber.Ctx) (uint64, error) {
	return strconv.ParseUint(c.Params("id"), 10, 64)
}

func notFoundOrError(c *fiber.Ctx, name string, err error) error {
	if err == gorm.ErrRecordNotFound {
		return errorResponse(c, fiber.StatusNotFound, name+" not found", nil)
	}
	return errorResponse(c, fiber.StatusInternalServerError, "failed to get "+name, err)
}

func errorResponse(c *fiber.Ctx, status int, message string, err error) error {
	body := fiber.Map{"error": message}
	if err != nil {
		body["detail"] = err.Error()
	}
	return c.Status(status).JSON(body)
}

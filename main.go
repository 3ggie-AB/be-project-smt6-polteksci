package main

import (
	"fmt"
	"log"
	"network-monitor/internal/config"
	"network-monitor/internal/database"
	"network-monitor/internal/models"
	"network-monitor/internal/routes"
	"network-monitor/internal/seeder"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  File .env tidak ditemukan, menggunakan environment variable sistem")
	}

	cfg := config.Load()

	log.Println("========================================")
	log.Println("  Network Monitor API — v1.0.0")
	log.Println("  Golang Fiber + MySQL + ClickHouse")
	log.Println("========================================")

	database.ConnectMySQL(cfg)

	database.ConnectClickHouse(cfg)
	database.MigrateClickHouse(cfg)

	db := database.GetDB()

	log.Println("🔄 Menjalankan migrasi database MySQL...")
	if err := db.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.User{},
		&models.Device{},
		&models.Feedback{},
	); err != nil {
		log.Fatalf("❌ Migrasi gagal: %v", err)
	}
	log.Println("✅ Migrasi MySQL selesai")

	seeder.Run(db, cfg)

	app := fiber.New(fiber.Config{
		AppName:           "Network Monitor API v1.0.0",
		EnablePrintRoutes: cfg.AppEnv == "development",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		},
	})

	app.Use(recover.New())
	app.Use(compress.New())
	app.Use(fiberLogger.New(fiberLogger.Config{
		Format: "[${time}] ${status} ${method} ${path} — ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("CORS_ORIGINS"),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))

	routes.Setup(app, cfg)

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Route %s %s tidak ditemukan", c.Method(), c.Path()),
		})
	})

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("🚀 Server berjalan di http://localhost%s", addr)
	log.Printf("📌 Env: %s", cfg.AppEnv)
	log.Printf("📋 Docs: lihat README.md untuk daftar endpoint")

	if err := app.Listen(addr); err != nil {
		log.Fatalf("❌ Server gagal dijalankan: %v", err)
	}
}

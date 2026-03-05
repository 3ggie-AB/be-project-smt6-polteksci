package db

import (
	"fmt"
	"log"
	"os"

	"project_smt6/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "netmonitor"),
		getEnv("DB_PORT", "5432"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke database: %v", err)
	}

	log.Println("✅ Database terhubung!")
	migrate()
}

func migrate() {
	err := DB.AutoMigrate(
		&models.Target{},
		&models.PingResult{},
		&models.Survey{},
	)
	if err != nil {
		log.Fatalf("❌ Migrasi gagal: %v", err)
	}

	// Seed target default jika kosong
	var count int64
	DB.Model(&models.Target{}).Count(&count)
	if count == 0 {
		defaults := []models.Target{
			{IPAddress: "8.8.8.8", Label: "Google DNS", IsActive: true},
			{IPAddress: "1.1.1.1", Label: "Cloudflare DNS", IsActive: true},
			{IPAddress: "192.168.1.1", Label: "Gateway Lokal", IsActive: true},
		}
		DB.Create(&defaults)
		log.Println("✅ Target default berhasil di-seed")
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
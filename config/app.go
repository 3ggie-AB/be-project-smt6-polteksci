package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Name       string
	Env        string
	Port       string
	JWTSecret  string
	SessionTTL time.Duration
	CORS       []string
	MySQL      MySQLConfig
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Location string
}

func Load() (AppConfig, error) {
	_ = godotenv.Load()

	cfg := AppConfig{
		Name:       env("APP_NAME", "NetMonitor API"),
		Env:        env("APP_ENV", "development"),
		Port:       env("PORT", "8080"),
		JWTSecret:  env("JWT_SECRET", "change-me-in-production"),
		SessionTTL: duration("SESSION_TTL", 24*time.Hour),
		CORS:       csv(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		MySQL: MySQLConfig{
			Host:     env("MYSQL_HOST", env("DB_HOST", "localhost")),
			Port:     env("MYSQL_PORT", env("DB_PORT", "3306")),
			User:     env("MYSQL_USER", env("DB_USER", "netmonitor")),
			Password: env("MYSQL_PASSWORD", env("DB_PASSWORD", "netmonitor")),
			Database: env("MYSQL_DATABASE", env("DB_NAME", "netmonitor")),
			Location: env("MYSQL_LOCATION", "Asia%2FJakarta"),
		},
	}

	if cfg.Env == "production" && cfg.JWTSecret == "change-me-in-production" {
		return AppConfig{}, fmt.Errorf("JWT_SECRET must be configured in production")
	}
	if cfg.MySQL.Database == "" {
		return AppConfig{}, fmt.Errorf("MYSQL_DATABASE/DB_NAME is required")
	}
	return cfg, nil
}

func (c MySQLConfig) ServerDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Location,
	)
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Location,
	)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func csv(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func EnvBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

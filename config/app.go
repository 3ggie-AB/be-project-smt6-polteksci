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
	Monitoring MonitoringConfig
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Location string
}

type MonitoringConfig struct {
	Enabled bool
	Ping    PingConfig
	SNMP    SNMPConfig
}

type PingConfig struct {
	Enabled                  bool
	Interval                 time.Duration
	Timeout                  time.Duration
	Count                    int
	Workers                  int
	WarningLatencyMS         float64
	WarningPacketLossPercent float64
}

type SNMPConfig struct {
	Enabled   bool
	Interval  time.Duration
	Port      uint16
	Timeout   time.Duration
	Retries   int
	Community string
	Version   string
	CPUOID    string
	MemoryOID string
	Workers   int
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
		Monitoring: MonitoringConfig{
			Enabled: EnvBool("MONITORING_ENABLED", true),
			Ping: PingConfig{
				Enabled:                  EnvBool("PING_ENABLED", true),
				Interval:                 duration("PING_INTERVAL", 5*time.Second),
				Timeout:                  duration("PING_TIMEOUT", 3*time.Second),
				Count:                    intEnv("PING_COUNT", 3),
				Workers:                  intEnv("PING_WORKERS", 64),
				WarningLatencyMS:         floatEnv("HIGH_LATENCY_MS", 0),
				WarningPacketLossPercent: packetLossPercentEnv("HIGH_PACKET_LOSS_RATIO", 0),
			},
			SNMP: SNMPConfig{
				Enabled:   EnvBool("SNMP_ENABLED", false),
				Interval:  duration("SNMP_POLL_INTERVAL", 60*time.Second),
				Port:      uint16Env("SNMP_PORT", 161),
				Timeout:   duration("SNMP_TIMEOUT", 3*time.Second),
				Retries:   intEnv("SNMP_RETRIES", 1),
				Community: env("SNMP_COMMUNITY", "public"),
				Version:   env("SNMP_VERSION", "2c"),
				CPUOID:    env("SNMP_CPU_OID", ""),
				MemoryOID: env("SNMP_MEMORY_OID", ""),
				Workers:   intEnv("SNMP_WORKERS", 64),
			},
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

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func uint16Env(key string, fallback uint16) uint16 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(raw, 10, 16)
	if err != nil || parsed == 0 {
		return fallback
	}
	return uint16(parsed)
}

func floatEnv(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func packetLossPercentEnv(key string, fallback float64) float64 {
	parsed := floatEnv(key, fallback)
	if parsed > 0 && parsed <= 1 {
		return parsed * 100
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

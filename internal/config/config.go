package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	MySQL      MySQLConfig
	Influx     InfluxConfig
	Auth       AuthConfig
	Monitoring MonitoringConfig
	Ruijie     RuijieConfig
	Syslog     SyslogConfig
	SNMP       SNMPConfig
}

type ServerConfig struct {
	Port              string
	Environment       string
	AllowedOrigins    []string
	ReadHeaderTimeout time.Duration
}

type MySQLConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Database  string
	ParseTime bool
	Location  string
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=%t&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.ParseTime,
		c.Location,
	)
}

func (c MySQLConfig) ServerDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=%t&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.ParseTime,
		c.Location,
	)
}

type InfluxConfig struct {
	URL           string
	Token         string
	Org           string
	Bucket        string
	BatchSize     int
	FlushInterval time.Duration
	QueueSize     int
	MaxRetries    int
	RetryInterval time.Duration
}

type AuthConfig struct {
	JWTSecret              string
	JWTIssuer              string
	JWTTTL                 time.Duration
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	BootstrapAdminName     string
}

type MonitoringConfig struct {
	PingInterval        time.Duration
	PingTimeout         time.Duration
	PingCount           int
	PingWorkers         int
	TCPInterval         time.Duration
	TCPTimeout          time.Duration
	TCPWorkers          int
	DefaultTCPPort      int
	HighLatencyMS       float64
	HighPacketLossRatio float64
}

type RuijieConfig struct {
	BaseURL        string
	APIKey         string
	Endpoint       string
	PollInterval   time.Duration
	RequestTimeout time.Duration
	DebugRawJSON   bool
}

type SyslogConfig struct {
	Enabled bool
	Address string
}

type SNMPConfig struct {
	Enabled      bool
	PollInterval time.Duration
	Port         uint16
	Timeout      time.Duration
	Retries      int
	CPUOID       string
	MemoryOID    string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Server: ServerConfig{
			Port:              getEnv("PORT", "8080"),
			Environment:       getEnv("APP_ENV", "development"),
			AllowedOrigins:    splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
			ReadHeaderTimeout: getDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		},
		MySQL: MySQLConfig{
			Host:      getEnv("MYSQL_HOST", getEnv("DB_HOST", "localhost")),
			Port:      getEnv("MYSQL_PORT", getEnv("DB_PORT", "3306")),
			User:      getEnv("MYSQL_USER", getEnv("DB_USER", "netmonitor")),
			Password:  getEnv("MYSQL_PASSWORD", getEnv("DB_PASSWORD", "netmonitor")),
			Database:  getEnv("MYSQL_DATABASE", getEnv("DB_NAME", "netmonitor")),
			ParseTime: true,
			Location:  getEnv("MYSQL_LOCATION", "Asia%2FJakarta"),
		},
		Influx: InfluxConfig{
			URL:           getEnv("INFLUX_URL", "http://localhost:8086"),
			Token:         getEnv("INFLUX_TOKEN", ""),
			Org:           getEnv("INFLUX_ORG", "netmonitor"),
			Bucket:        getEnv("INFLUX_BUCKET", "network_metrics"),
			BatchSize:     getInt("INFLUX_BATCH_SIZE", 500),
			FlushInterval: getDuration("INFLUX_FLUSH_INTERVAL", 2*time.Second),
			QueueSize:     getInt("INFLUX_QUEUE_SIZE", 10000),
			MaxRetries:    getInt("INFLUX_MAX_RETRIES", 3),
			RetryInterval: getDuration("INFLUX_RETRY_INTERVAL", 500*time.Millisecond),
		},
		Auth: AuthConfig{
			JWTSecret:              getEnv("JWT_SECRET", "change-me-in-production"),
			JWTIssuer:              getEnv("JWT_ISSUER", "netmonitor"),
			JWTTTL:                 getDuration("JWT_TTL", 24*time.Hour),
			BootstrapAdminEmail:    getEnv("ADMIN_EMAIL", "admin@netmonitor.local"),
			BootstrapAdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
			BootstrapAdminName:     getEnv("ADMIN_NAME", "NetMonitor Admin"),
		},
		Monitoring: MonitoringConfig{
			PingInterval:        getDuration("PING_INTERVAL", 5*time.Second),
			PingTimeout:         getDuration("PING_TIMEOUT", 3*time.Second),
			PingCount:           getInt("PING_COUNT", 3),
			PingWorkers:         getInt("PING_WORKERS", 512),
			TCPInterval:         getDuration("TCP_INTERVAL", 30*time.Second),
			TCPTimeout:          getDuration("TCP_TIMEOUT", 3*time.Second),
			TCPWorkers:          getInt("TCP_WORKERS", 256),
			DefaultTCPPort:      getInt("DEFAULT_TCP_PORT", 443),
			HighLatencyMS:       getFloat("HIGH_LATENCY_MS", 150),
			HighPacketLossRatio: getFloat("HIGH_PACKET_LOSS_RATIO", 0.2),
		},
		Ruijie: RuijieConfig{
			BaseURL:        strings.TrimRight(getEnv("RUIJIE_BASE_URL", ""), "/"),
			APIKey:         getEnv("RUIJIE_API_KEY", ""),
			Endpoint:       getEnv("RUIJIE_AP_ENDPOINT", "/api/telemetry/aps"),
			PollInterval:   getDuration("RUIJIE_POLL_INTERVAL", 30*time.Second),
			RequestTimeout: getDuration("RUIJIE_REQUEST_TIMEOUT", 10*time.Second),
			DebugRawJSON:   getBool("RUIJIE_DEBUG_RAW_JSON", false),
		},
		Syslog: SyslogConfig{
			Enabled: getBool("SYSLOG_ENABLED", true),
			Address: getEnv("SYSLOG_ADDRESS", ":5514"),
		},
		SNMP: SNMPConfig{
			Enabled:      getBool("SNMP_ENABLED", true),
			PollInterval: getDuration("SNMP_POLL_INTERVAL", 60*time.Second),
			Port:         uint16(getInt("SNMP_PORT", 161)),
			Timeout:      getDuration("SNMP_TIMEOUT", 3*time.Second),
			Retries:      getInt("SNMP_RETRIES", 1),
			CPUOID:       getEnv("SNMP_CPU_OID", ""),
			MemoryOID:    getEnv("SNMP_MEMORY_OID", ""),
		},
	}

	if cfg.Auth.JWTSecret == "change-me-in-production" && cfg.Server.Environment == "production" {
		return Config{}, fmt.Errorf("JWT_SECRET must be configured in production")
	}
	if cfg.Auth.BootstrapAdminPassword == "" && cfg.Server.Environment == "production" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must be configured in production")
	}

	return cfg, nil
}

func NewLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	return slog.New(newCleanHandler(os.Stdout, level))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

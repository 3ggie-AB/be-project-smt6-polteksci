package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	AppPort string
	AppEnv  string
	JWTSecret string
	JWTExpireHours int

	// MySQL
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string

	// ClickHouse
	ClickHouseHost     string
	ClickHousePort     string
	ClickHouseUser     string
	ClickHousePassword string
	ClickHouseDatabase string

	// Default Users (seeded on startup)
	AdminName     string
	AdminEmail    string
	AdminPassword string

	TeknisiName     string
	TeknisiEmail    string
	TeknisiPassword string

	StaffName     string
	StaffEmail    string
	StaffPassword string

	KaryawanName     string
	KaryawanEmail    string
	KaryawanPassword string
}

func Load() *Config {
	jwtExpire, err := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	if err != nil {
		jwtExpire = 24
	}

	return &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		AppEnv:         getEnv("APP_ENV", "development"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-jwt-key-change-in-production"),
		JWTExpireHours: jwtExpire,

		MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLUser:     getEnv("MYSQL_USER", "root"),
		MySQLPassword: getEnv("MYSQL_PASSWORD", ""),
		MySQLDatabase: getEnv("MYSQL_DATABASE", "network_monitor"),

		ClickHouseHost:     getEnv("CLICKHOUSE_HOST", "localhost"),
		ClickHousePort:     getEnv("CLICKHOUSE_PORT", "9000"),
		ClickHouseUser:     getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),
		ClickHouseDatabase: getEnv("CLICKHOUSE_DATABASE", "network_monitor"),

		AdminName:     getEnv("ADMIN_NAME", "Administrator"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@company.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "Admin@1234"),

		TeknisiName:     getEnv("TEKNISI_NAME", "Budi Santoso"),
		TeknisiEmail:    getEnv("TEKNISI_EMAIL", "teknisi@company.com"),
		TeknisiPassword: getEnv("TEKNISI_PASSWORD", "Teknisi@1234"),

		StaffName:     getEnv("STAFF_NAME", "Siti Rahayu"),
		StaffEmail:    getEnv("STAFF_EMAIL", "staff@company.com"),
		StaffPassword: getEnv("STAFF_PASSWORD", "Staff@1234"),

		KaryawanName:     getEnv("KARYAWAN_NAME", "Agus Pratama"),
		KaryawanEmail:    getEnv("KARYAWAN_EMAIL", "karyawan@company.com"),
		KaryawanPassword: getEnv("KARYAWAN_PASSWORD", "Karyawan@1234"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

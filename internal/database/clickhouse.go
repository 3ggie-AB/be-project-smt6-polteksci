package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"network-monitor/internal/config"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var CH driver.Conn

func ConnectClickHouse(cfg *config.Config) {
	ensureClickHouseDatabase(cfg)

	conn, err := openClickHouse(cfg, cfg.ClickHouseDatabase)
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke ClickHouse: %v", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatalf("❌ ClickHouse tidak merespon: %v", err)
	}

	CH = conn
	log.Println("✅ Terhubung ke ClickHouse")
}

func openClickHouse(cfg *config.Config, databaseName string) (driver.Conn, error) {
	port, err := strconv.Atoi(strings.TrimSpace(cfg.ClickHousePort))
	if err != nil {
		return nil, fmt.Errorf("port ClickHouse tidak valid: %w", err)
	}
	if err := validateClickHouseNativePort(port); err != nil {
		return nil, err
	}

	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouseHost, port)},
		Auth: clickhouse.Auth{
			Database: databaseName,
			Username: cfg.ClickHouseUser,
			Password: cfg.ClickHousePassword,
		},
		TLS: clickHouseTLSConfig(port),
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:  30 * time.Second,
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	})
}

func validateClickHouseNativePort(port int) error {
	switch port {
	case 8123, 8443:
		return fmt.Errorf("port %d adalah port HTTP ClickHouse; aplikasi memakai native driver, gunakan port 9000", port)
	}
	return nil
}

func clickHouseTLSConfig(port int) *tls.Config {
	switch port {
	case 9440:
		return &tls.Config{InsecureSkipVerify: true}
	default:
		return nil
	}
}

func ensureClickHouseDatabase(cfg *config.Config) {
	conn, err := openClickHouse(cfg, "default")
	if err != nil {
		log.Fatalf("❌ Gagal membuka koneksi ClickHouse untuk membuat database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("❌ ClickHouse tidak merespon: %v", err)
	}

	if err := conn.Exec(
		ctx,
		"CREATE DATABASE IF NOT EXISTS {database:Identifier}",
		clickhouse.Named("database", cfg.ClickHouseDatabase),
	); err != nil {
		log.Fatalf("❌ Gagal membuat database ClickHouse: %v", err)
	}

	log.Printf("✅ Database ClickHouse siap: %s", cfg.ClickHouseDatabase)
}

func GetCH() driver.Conn {
	return CH
}

func MigrateClickHouse(cfg *config.Config) {
	ctx := context.Background()

	pingTable := `
		CREATE TABLE IF NOT EXISTS {database:Identifier}.ping_results (
			id          UUID DEFAULT generateUUIDv4(),
			device_id   UInt64,
			device_name String,
			ip_address  String,
			packets_sent     UInt8,
			packets_received UInt8,
			packet_loss      Float32,
			min_rtt     Float64,
			avg_rtt     Float64,
			max_rtt     Float64,
			status      String,
			checked_by  UInt64,
			checked_at  DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (checked_at, device_id)
		TTL checked_at + INTERVAL 90 DAY
	`

	if err := CH.Exec(ctx, pingTable, clickhouse.Named("database", cfg.ClickHouseDatabase)); err != nil {
		log.Fatalf("❌ Gagal membuat tabel ping_results: %v", err)
	}

	snmpTable := `
		CREATE TABLE IF NOT EXISTS {database:Identifier}.snmp_results (
			id          UUID DEFAULT generateUUIDv4(),
			device_id   UInt64,
			device_name String,
			ip_address  String,
			oid         String,
			oid_name    String,
			value       String,
			value_type  String,
			checked_by  UInt64,
			checked_at  DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (checked_at, device_id)
		TTL checked_at + INTERVAL 90 DAY
	`

	if err := CH.Exec(ctx, snmpTable, clickhouse.Named("database", cfg.ClickHouseDatabase)); err != nil {
		log.Fatalf("❌ Gagal membuat tabel snmp_results: %v", err)
	}

	log.Println("✅ Migrasi ClickHouse selesai")
}

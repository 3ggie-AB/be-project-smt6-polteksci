package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"network-monitor/internal/config"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var CH driver.Conn

func ConnectClickHouse(cfg *config.Config) {
	port := uint16(9000)
	fmt.Sscanf(cfg.ClickHousePort, "%d", &port)

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouseHost, port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDatabase,
			Username: cfg.ClickHouseUser,
			Password: cfg.ClickHousePassword,
		},
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:  30,
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke ClickHouse: %v", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatalf("❌ ClickHouse tidak merespon: %v", err)
	}

	CH = conn
	log.Println("✅ Terhubung ke ClickHouse")
}

func GetCH() driver.Conn {
	return CH
}

func MigrateClickHouse(cfg *config.Config) {
	ctx := context.Background()

	if err := CH.Exec(ctx, fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS %s`, cfg.ClickHouseDatabase)); err != nil {
		log.Fatalf("❌ Gagal membuat database ClickHouse: %v", err)
	}

	pingTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.ping_results (
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
	`, cfg.ClickHouseDatabase)

	if err := CH.Exec(ctx, pingTable); err != nil {
		log.Fatalf("❌ Gagal membuat tabel ping_results: %v", err)
	}

	snmpTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.snmp_results (
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
	`, cfg.ClickHouseDatabase)

	if err := CH.Exec(ctx, snmpTable); err != nil {
		log.Fatalf("❌ Gagal membuat tabel snmp_results: %v", err)
	}

	log.Println("✅ Migrasi ClickHouse selesai")
}

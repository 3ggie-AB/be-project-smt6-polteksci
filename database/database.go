package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project_smt6/app/models"
	"project_smt6/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(ctx context.Context, cfg config.MySQLConfig) (*gorm.DB, error) {
	if err := ensureDatabase(ctx, cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	if err := db.WithContext(ctx).AutoMigrate(models.ModelsForMigration()...); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}

func ensureDatabase(ctx context.Context, cfg config.MySQLConfig) error {
	serverDB, err := gorm.Open(mysql.Open(cfg.ServerDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("connect mysql server: %w", err)
	}

	sqlDB, err := serverDB.DB()
	if err != nil {
		return fmt.Errorf("get mysql server db: %w", err)
	}
	defer sqlDB.Close()

	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		quoteIdentifier(cfg.Database),
	)
	if err := serverDB.WithContext(ctx).Exec(stmt).Error; err != nil {
		return fmt.Errorf("create database %q: %w", cfg.Database, err)
	}
	return nil
}

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

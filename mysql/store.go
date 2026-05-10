package mysql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
	"project_smt6/repository"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Store struct {
	db     *gorm.DB
	logger *slog.Logger
}

func New(ctx context.Context, cfg config.MySQLConfig, log *slog.Logger) (*Store, error) {
	if err := ensureDatabase(ctx, cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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

	store := &Store{db: db, logger: log}
	if err := store.Migrate(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func ensureDatabase(ctx context.Context, cfg config.MySQLConfig) error {
	if strings.TrimSpace(cfg.Database) == "" {
		return errors.New("mysql database name is required")
	}

	serverDB, err := gorm.Open(mysql.Open(cfg.ServerDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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
		return fmt.Errorf("create mysql database %q: %w", cfg.Database, err)
	}
	return nil
}

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	if err := s.db.WithContext(ctx).AutoMigrate(
		&domain.Workspace{},
		&domain.Role{},
		&domain.User{},
		&domain.DeviceGroup{},
		&domain.Device{},
		&domain.Notification{},
		&domain.Survey{},
	); err != nil {
		return fmt.Errorf("auto migrate mysql: %w", err)
	}

	for _, roleName := range []domain.RoleName{domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleUser} {
		role := domain.Role{Name: roleName}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&role).Error; err != nil {
			return fmt.Errorf("seed role %s: %w", roleName, err)
		}
	}

	defaultWorkspace := domain.Workspace{Name: "Default Workspace", Slug: "default"}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&defaultWorkspace).Error; err != nil {
		return fmt.Errorf("seed default workspace: %w", err)
	}
	return nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&domain.User{}).Count(&count).Error
	return count, err
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := s.db.WithContext(ctx).Preload("Role").Preload("Workspace").Where("email = ?", email).First(&user).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	return nil, err
}

func (s *Store) CreateLocalUser(ctx context.Context, user *domain.User, roleName domain.RoleName) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role domain.Role
		if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
			return err
		}

		var workspace domain.Workspace
		if err := tx.Where("slug = ?", "default").First(&workspace).Error; err != nil {
			return err
		}

		now := time.Now()
		user.WorkspaceID = &workspace.ID
		user.RoleID = role.ID
		user.IsActive = true
		user.LastLoginAt = &now
		return tx.Create(user).Error
	})
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Preload("Role").Preload("Workspace").First(user, user.ID).Error
}

func (s *Store) TouchLastLogin(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("last_login_at", &now).Error
}

func (s *Store) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := s.db.WithContext(ctx).Preload("Role").Preload("Workspace").First(&user, id).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	return nil, err
}

func (s *Store) List(ctx context.Context, workspaceID *uint) ([]domain.User, error) {
	var users []domain.User
	query := s.db.WithContext(ctx).Preload("Role").Preload("Workspace").Order("created_at DESC")
	if workspaceID != nil {
		query = query.Where("workspace_id = ?", *workspaceID)
	}
	return users, query.Find(&users).Error
}

func (s *Store) ListDevices(ctx context.Context, workspaceID *uint) ([]domain.Device, error) {
	var devices []domain.Device
	query := s.db.WithContext(ctx).Preload("Workspace").Order("name ASC")
	if workspaceID != nil {
		query = query.Where("workspace_id = ?", *workspaceID)
	}
	return devices, query.Find(&devices).Error
}

func (s *Store) ListActive(ctx context.Context) ([]domain.Device, error) {
	var devices []domain.Device
	err := s.db.WithContext(ctx).
		Preload("Workspace").
		Where("is_active = ?", true).
		Find(&devices).Error
	return devices, err
}

func (s *Store) FindDeviceByID(ctx context.Context, id uint) (*domain.Device, error) {
	var device domain.Device
	if err := s.db.WithContext(ctx).Preload("Workspace").First(&device, id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (s *Store) CreateDevice(ctx context.Context, device *domain.Device) error {
	if device.WorkspaceID == 0 {
		var workspace domain.Workspace
		if err := s.db.WithContext(ctx).Where("slug = ?", "default").First(&workspace).Error; err != nil {
			return err
		}
		device.WorkspaceID = workspace.ID
	}
	if device.TCPPort == 0 {
		device.TCPPort = 443
	}
	if device.PingIntervalSec == 0 {
		device.PingIntervalSec = 5
	}
	if device.TCPIntervalSec == 0 {
		device.TCPIntervalSec = 30
	}
	return s.db.WithContext(ctx).Create(device).Error
}

func (s *Store) DeleteDevice(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&domain.Device{}, id).Error
}

func (s *Store) MarkDeviceSeen(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&domain.Device{}).Where("id = ?", id).Update("last_seen_at", &now).Error
}

func (s *Store) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	return s.db.WithContext(ctx).Create(notification).Error
}

func (s *Store) ListUnreadNotifications(ctx context.Context, workspaceID uint, userID *uint) ([]domain.Notification, error) {
	var notifications []domain.Notification
	query := s.db.WithContext(ctx).
		Preload("Device").
		Where("workspace_id = ? AND read_at IS NULL", workspaceID).
		Order("created_at DESC").
		Limit(100)
	if userID != nil {
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	}
	return notifications, query.Find(&notifications).Error
}

func (s *Store) MarkNotificationRead(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&domain.Notification{}).Where("id = ?", id).Update("read_at", &now).Error
}

type DeviceRepositoryAdapter struct {
	*Store
}

func (a DeviceRepositoryAdapter) List(ctx context.Context, workspaceID *uint) ([]domain.Device, error) {
	return a.Store.ListDevices(ctx, workspaceID)
}

func (a DeviceRepositoryAdapter) FindByID(ctx context.Context, id uint) (*domain.Device, error) {
	return a.Store.FindDeviceByID(ctx, id)
}

func (a DeviceRepositoryAdapter) Create(ctx context.Context, device *domain.Device) error {
	return a.Store.CreateDevice(ctx, device)
}

func (a DeviceRepositoryAdapter) Delete(ctx context.Context, id uint) error {
	return a.Store.DeleteDevice(ctx, id)
}

func (a DeviceRepositoryAdapter) MarkSeen(ctx context.Context, id uint) error {
	return a.Store.MarkDeviceSeen(ctx, id)
}

type NotificationRepositoryAdapter struct {
	*Store
}

func (a NotificationRepositoryAdapter) Create(ctx context.Context, notification *domain.Notification) error {
	return a.Store.CreateNotification(ctx, notification)
}

func (a NotificationRepositoryAdapter) ListUnread(ctx context.Context, workspaceID uint, userID *uint) ([]domain.Notification, error) {
	return a.Store.ListUnreadNotifications(ctx, workspaceID, userID)
}

func (a NotificationRepositoryAdapter) MarkRead(ctx context.Context, id uint) error {
	return a.Store.MarkNotificationRead(ctx, id)
}

var _ repository.UserRepository = (*Store)(nil)
var _ repository.DeviceRepository = DeviceRepositoryAdapter{}
var _ repository.NotificationRepository = NotificationRepositoryAdapter{}

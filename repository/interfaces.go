package repository

import (
	"context"
	"errors"

	"project_smt6/domain"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Count(ctx context.Context) (int64, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateLocalUser(ctx context.Context, user *domain.User, role domain.RoleName) error
	TouchLastLogin(ctx context.Context, id uint) error
	List(ctx context.Context, workspaceID *uint) ([]domain.User, error)
}

type DeviceRepository interface {
	List(ctx context.Context, workspaceID *uint) ([]domain.Device, error)
	ListActive(ctx context.Context) ([]domain.Device, error)
	FindByID(ctx context.Context, id uint) (*domain.Device, error)
	Create(ctx context.Context, device *domain.Device) error
	Delete(ctx context.Context, id uint) error
	MarkSeen(ctx context.Context, id uint) error
}

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	ListUnread(ctx context.Context, workspaceID uint, userID *uint) ([]domain.Notification, error)
	MarkRead(ctx context.Context, id uint) error
}

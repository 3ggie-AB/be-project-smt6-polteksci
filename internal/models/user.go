package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string      `gorm:"uniqueIndex;size:50;not null" json:"name"`
	DisplayName string      `gorm:"size:100" json:"display_name"`
	Description string      `gorm:"type:text" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Permission struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Module      string    `gorm:"size:50" json:"module"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:150;not null" json:"name"`
	Email     string         `gorm:"uniqueIndex;size:200;not null" json:"email"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	RoleID    uint           `gorm:"not null" json:"role_id"`
	Role      Role           `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	Phone     string         `gorm:"size:20" json:"phone"`
	Department string        `gorm:"size:100" json:"department"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UserResponse struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	RoleID     uint      `json:"role_id"`
	Role       Role      `json:"role"`
	IsActive   bool      `json:"is_active"`
	Phone      string    `json:"phone"`
	Department string    `json:"department"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		RoleID:     u.RoleID,
		Role:       u.Role,
		IsActive:   u.IsActive,
		Phone:      u.Phone,
		Department: u.Department,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

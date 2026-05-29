package models

import (
	"time"

	"gorm.io/gorm"
)

type Device struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:150;not null" json:"name"`
	IPAddress     string         `gorm:"size:50;not null" json:"ip_address"`
	Type          string         `gorm:"size:50;default:'server'" json:"type"`
	Location      string         `gorm:"size:150" json:"location"`
	Description   string         `gorm:"type:text" json:"description"`
	SNMPCommunity string         `gorm:"size:100;default:'public'" json:"snmp_community"`
	SNMPVersion   string         `gorm:"size:10;default:'2c'" json:"snmp_version"`
	SNMPPort      int            `gorm:"default:161" json:"snmp_port"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedByID   uint           `json:"created_by_id"`
	CreatedBy     User           `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type DeviceType string

const (
	DeviceTypeRouter  DeviceType = "router"
	DeviceTypeSwitch  DeviceType = "switch"
	DeviceTypeServer  DeviceType = "server"
	DeviceTypeFirewall DeviceType = "firewall"
	DeviceTypeAP      DeviceType = "access_point"
	DeviceTypeOther   DeviceType = "other"
)

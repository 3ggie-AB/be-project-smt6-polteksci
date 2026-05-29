package models

import "time"

type UserRole string

const (
	RoleSuperAdmin UserRole = "SUPER_ADMIN"
	RoleAdmin      UserRole = "ADMIN"
	RoleUser       UserRole = "USER"
)

type DeviceType string

const (
	DeviceTypeAP      DeviceType = "AP"
	DeviceTypeService DeviceType = "SERVICE"
)

type DeviceStatusValue string

const (
	DeviceStatusOnline  DeviceStatusValue = "ONLINE"
	DeviceStatusOffline DeviceStatusValue = "OFFLINE"
	DeviceStatusWarning DeviceStatusValue = "WARNING"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityCritical AlertSeverity = "CRITICAL"
)

type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "ACTIVE"
	AlertStatusResolved AlertStatus = "RESOLVED"
)

type User struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"type:varchar(160);not null"`
	Email     string    `json:"email" gorm:"type:varchar(255);not null;uniqueIndex"`
	Password  string    `json:"-" gorm:"type:varchar(255);not null"`
	Role      UserRole  `json:"role" gorm:"type:varchar(32);not null;default:USER;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Session struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `json:"user_id" gorm:"not null;index"`
	User      User      `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token     string    `json:"-" gorm:"column:token;type:text;not null"`
	TokenHash string    `json:"-" gorm:"column:token_hash;type:char(64);not null;index"`
	ExpiredAt time.Time `json:"expired_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Device struct {
	ID        uint64            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string            `json:"name" gorm:"type:varchar(160);not null;index"`
	IP        string            `json:"ip" gorm:"column:ip;type:varchar(64);not null;index"`
	Type      DeviceType        `json:"type" gorm:"type:enum('AP','SERVICE');not null;index"`
	Vendor    string            `json:"vendor" gorm:"type:varchar(120)"`
	Location  string            `json:"location" gorm:"type:varchar(180);index"`
	Status    DeviceStatusValue `json:"status" gorm:"type:enum('ONLINE','OFFLINE','WARNING');not null;default:OFFLINE;index"`
	CreatedAt time.Time         `json:"created_at" gorm:"autoCreateTime"`
}

type DeviceStatus struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID    uint64    `json:"device_id" gorm:"not null;index"`
	Device      Device    `json:"device,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Latency     float64   `json:"latency" gorm:"not null;default:0"`
	PacketLoss  float64   `json:"packet_loss" gorm:"not null;default:0"`
	CPUUsage    float64   `json:"cpu_usage" gorm:"not null;default:0"`
	MemoryUsage float64   `json:"memory_usage" gorm:"not null;default:0"`
	LastSeen    time.Time `json:"last_seen" gorm:"index"`
}

func (DeviceStatus) TableName() string {
	return "device_status"
}

type MonitoringConfig struct {
	ID            uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID      uint64    `json:"device_id" gorm:"not null;index"`
	Device        Device    `json:"device,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PingEnabled   bool      `json:"ping_enabled" gorm:"not null;default:true"`
	TCPEnabled    bool      `json:"tcp_enabled" gorm:"not null;default:false"`
	PingInterval  int       `json:"ping_interval" gorm:"not null;default:5"`
	TCPInterval   int       `json:"tcp_interval" gorm:"not null;default:30"`
	MonitoredPort int       `json:"monitored_port" gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Alert struct {
	ID        uint64        `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID  uint64        `json:"device_id" gorm:"not null;index"`
	Device    Device        `json:"device,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Severity  AlertSeverity `json:"severity" gorm:"type:enum('INFO','WARNING','CRITICAL');not null;index"`
	Message   string        `json:"message" gorm:"type:text;not null"`
	Status    AlertStatus   `json:"status" gorm:"type:enum('ACTIVE','RESOLVED');not null;default:ACTIVE;index"`
	CreatedAt time.Time     `json:"created_at" gorm:"autoCreateTime"`
}

type Notification struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `json:"user_id" gorm:"not null;index"`
	User      User      `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlertID   uint64    `json:"alert_id" gorm:"not null;index"`
	Alert     Alert     `json:"alert,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Title     string    `json:"title" gorm:"type:varchar(180);not null"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	IsRead    bool      `json:"is_read" gorm:"not null;default:false;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type ActivityLog struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64    `json:"user_id" gorm:"not null;index"`
	User        User      `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Action      string    `json:"action" gorm:"type:varchar(160);not null;index"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type NetworkTopology struct {
	ID             uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceDeviceID uint64    `json:"source_device_id" gorm:"not null;index"`
	SourceDevice   Device    `json:"source_device,omitempty" gorm:"foreignKey:SourceDeviceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TargetDeviceID uint64    `json:"target_device_id" gorm:"not null;index"`
	TargetDevice   Device    `json:"target_device,omitempty" gorm:"foreignKey:TargetDeviceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	RelationType   string    `json:"relation_type" gorm:"type:varchar(120);not null;index"`
	Status         string    `json:"status" gorm:"type:varchar(80);not null;index"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (NetworkTopology) TableName() string {
	return "network_topology"
}

func ModelsForMigration() []any {
	return []any{
		&User{},
		&Session{},
		&Device{},
		&DeviceStatus{},
		&MonitoringConfig{},
		&Alert{},
		&Notification{},
		&ActivityLog{},
		&NetworkTopology{},
	}
}

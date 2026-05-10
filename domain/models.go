package domain

import "time"

type RoleName string

const (
	RoleSuperAdmin RoleName = "SUPER_ADMIN"
	RoleAdmin      RoleName = "ADMIN"
	RoleUser       RoleName = "USER"
)

type Workspace struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"size:160;not null;uniqueIndex"`
	Slug      string    `json:"slug" gorm:"size:160;not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Role struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      RoleName  `json:"name" gorm:"size:32;not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID  *uint      `json:"workspace_id" gorm:"index"`
	Workspace    *Workspace `json:"workspace,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	RoleID       uint       `json:"role_id" gorm:"not null;index"`
	Role         Role       `json:"role" gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Email        string     `json:"email" gorm:"size:255;not null;uniqueIndex"`
	PasswordHash string     `json:"-" gorm:"size:255;not null"`
	Name         string     `json:"name" gorm:"size:160;not null"`
	AvatarURL    string     `json:"avatar_url" gorm:"size:512"`
	IsActive     bool       `json:"is_active" gorm:"not null;default:true;index"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type DeviceGroup struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID uint      `json:"workspace_id" gorm:"not null;index"`
	Workspace   Workspace `json:"workspace" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Name        string    `json:"name" gorm:"size:160;not null;index"`
	Description string    `json:"description" gorm:"size:500"`
	Devices     []Device  `json:"devices,omitempty" gorm:"many2many:device_group_members;constraint:OnDelete:CASCADE;"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Device struct {
	ID               uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID      uint          `json:"workspace_id" gorm:"not null;index:idx_device_workspace_ip,priority:1"`
	Workspace        Workspace     `json:"workspace" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Groups           []DeviceGroup `json:"groups,omitempty" gorm:"many2many:device_group_members;"`
	Name             string        `json:"name" gorm:"size:160;not null;index"`
	IPAddress        string        `json:"ip_address" gorm:"size:64;not null;index:idx_device_workspace_ip,priority:2"`
	MACAddress       string        `json:"mac_address" gorm:"size:32;index"`
	Vendor           string        `json:"vendor" gorm:"size:80;index"`
	Model            string        `json:"model" gorm:"size:120"`
	Location         string        `json:"location" gorm:"size:180;index"`
	DeviceType       string        `json:"device_type" gorm:"size:50;not null;default:network;index"`
	SNMPCommunity    string        `json:"-" gorm:"size:120"`
	SNMPVersion      string        `json:"snmp_version" gorm:"size:10;default:v2c"`
	RuijieExternalID string        `json:"ruijie_external_id" gorm:"size:160;index"`
	IsActive         bool          `json:"is_active" gorm:"not null;default:true;index"`
	LastSeenAt       *time.Time    `json:"last_seen_at" gorm:"index"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type CheckType string

const (
	CheckTypePing CheckType = "ping"
	CheckTypeTCP  CheckType = "tcp"
)

type MonitoringTarget struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID   uint       `json:"workspace_id" gorm:"not null;index:idx_target_workspace_host,priority:1"`
	Workspace     Workspace  `json:"workspace" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Name          string     `json:"name" gorm:"size:160;not null;index"`
	Host          string     `json:"host" gorm:"size:255;not null;index:idx_target_workspace_host,priority:2"`
	CheckType     CheckType  `json:"check_type" gorm:"size:16;not null;index:idx_target_check_active,priority:1"`
	Port          int        `json:"port" gorm:"not null;default:0"`
	IntervalSec   int        `json:"interval_sec" gorm:"not null;default:0"`
	TimeoutSec    int        `json:"timeout_sec" gorm:"not null;default:0"`
	Description   string     `json:"description" gorm:"size:500"`
	IsActive      bool       `json:"is_active" gorm:"not null;default:true;index:idx_target_check_active,priority:2"`
	LastCheckedAt *time.Time `json:"last_checked_at" gorm:"index"`
	LastStatus    *bool      `json:"last_status" gorm:"index"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Notification struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID uint       `json:"workspace_id" gorm:"not null;index"`
	Workspace   Workspace  `json:"workspace" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID      *uint      `json:"user_id" gorm:"index"`
	User        *User      `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	DeviceID    *uint      `json:"device_id" gorm:"index"`
	Device      *Device    `json:"device,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Type        string     `json:"type" gorm:"size:80;not null;index"`
	Severity    string     `json:"severity" gorm:"size:24;not null;default:info;index"`
	Title       string     `json:"title" gorm:"size:180;not null"`
	Message     string     `json:"message" gorm:"size:1000"`
	ReadAt      *time.Time `json:"read_at" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at" gorm:"index"`
}

type Survey struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkspaceID    *uint     `json:"workspace_id" gorm:"index"`
	RespondentName string    `json:"respondent_name" gorm:"size:160"`
	Location       string    `json:"location" gorm:"size:180;index"`
	Q1Speed        int       `json:"q1_speed"`
	Q2Stability    int       `json:"q2_stability"`
	Q3Latency      int       `json:"q3_latency"`
	Q4Availability int       `json:"q4_availability"`
	Q5Satisfaction int       `json:"q5_satisfaction"`
	Comment        string    `json:"comment" gorm:"size:1000"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
}

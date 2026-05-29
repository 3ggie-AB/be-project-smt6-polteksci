package models

import (
	"time"

	"gorm.io/gorm"
)

type FeedbackStatus string
type FeedbackPriority string
type FeedbackCategory string

const (
	StatusOpen       FeedbackStatus = "open"
	StatusInProgress FeedbackStatus = "in_progress"
	StatusResolved   FeedbackStatus = "resolved"
	StatusClosed     FeedbackStatus = "closed"

	PriorityLow      FeedbackPriority = "low"
	PriorityMedium   FeedbackPriority = "medium"
	PriorityHigh     FeedbackPriority = "high"
	PriorityCritical FeedbackPriority = "critical"

	CategoryKeluhan  FeedbackCategory = "keluhan"
	CategorySaran    FeedbackCategory = "saran"
	CategoryPertanyaan FeedbackCategory = "pertanyaan"
	CategoryInsiden  FeedbackCategory = "insiden"
)

type Feedback struct {
	ID           uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	Title        string           `gorm:"size:255;not null" json:"title"`
	Description  string           `gorm:"type:text;not null" json:"description"`
	Category     FeedbackCategory `gorm:"size:50;default:'keluhan'" json:"category"`
	Status       FeedbackStatus   `gorm:"size:50;default:'open'" json:"status"`
	Priority     FeedbackPriority `gorm:"size:50;default:'medium'" json:"priority"`
	CreatedByID  uint             `gorm:"not null" json:"created_by_id"`
	CreatedBy    User             `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	AssignedToID *uint            `json:"assigned_to_id"`
	AssignedTo   *User            `gorm:"foreignKey:AssignedToID" json:"assigned_to,omitempty"`
	Response     string           `gorm:"type:text" json:"response"`
	RespondedByID *uint           `json:"responded_by_id"`
	RespondedBy  *User            `gorm:"foreignKey:RespondedByID" json:"responded_by,omitempty"`
	RespondedAt  *time.Time       `json:"responded_at"`
	Attachment   string           `gorm:"size:500" json:"attachment"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	DeletedAt    gorm.DeletedAt   `gorm:"index" json:"-"`
}

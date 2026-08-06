package model

import (
	"time"

	"gorm.io/gorm"
)

// PermissionGroup status values.
const (
	PermissionGroupStatusActive   = "active"
	PermissionGroupStatusDisabled = "disabled"
)

// PermissionGroup is the unit of service authorization. Users gain access to an
// MCP service by being a member of a group that grants that service. A group's
// effective grants are ignored while Status is PermissionGroupStatusDisabled.
type PermissionGroup struct {
	gorm.Model

	// Override gorm.Model fields with json tags for API serialization.
	ID        uint `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name        string `json:"name" gorm:"type:varchar(255);uniqueIndex;not null"`
	DisplayName string `json:"display_name" gorm:"type:varchar(255)"`
	Description string `json:"description" gorm:"type:text"`

	// Status is PermissionGroupStatusActive|Disabled.
	Status string `json:"status" gorm:"type:varchar(20);not null;default:active"`

	CreatedByID uint `json:"created_by_id" gorm:"not null"`
}

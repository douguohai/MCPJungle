package model

import (
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

	Name        string `gorm:"type:varchar(255);uniqueIndex;not null"`
	DisplayName string `gorm:"type:varchar(255)"`
	Description string `gorm:"type:text"`

	// Status is PermissionGroupStatusActive|Disabled.
	Status string `gorm:"type:varchar(20);not null;default:active"`

	CreatedByID uint `gorm:"not null"`
}

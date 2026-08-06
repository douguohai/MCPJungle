package model

import (
	"time"

	"gorm.io/gorm"
)

// PermissionGroupMember links a user to a permission group.
type PermissionGroupMember struct {
	gorm.Model

	PermissionGroupID uint `gorm:"not null;uniqueIndex:idx_permission_group_member,priority:1"`
	UserID            uint `gorm:"not null;uniqueIndex:idx_permission_group_member,priority:2"`
	AssignedByID      uint `gorm:"not null"`
	AssignedAt        time.Time `gorm:"not null"`
}

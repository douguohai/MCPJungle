package model

import (
	"time"

	"gorm.io/gorm"
)

// PermissionGroupMember links a user to a permission group.
type PermissionGroupMember struct {
	gorm.Model

	// Override gorm.Model fields with json tags for API serialization.
	ID        uint `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	PermissionGroupID uint `json:"permission_group_id" gorm:"not null;uniqueIndex:idx_permission_group_member,priority:1"`
	UserID            uint `json:"user_id" gorm:"not null;uniqueIndex:idx_permission_group_member,priority:2"`
	AssignedByID      uint `json:"assigned_by_id" gorm:"not null"`
	AssignedAt        time.Time `json:"assigned_at" gorm:"not null"`
}

package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// Role represents a role entity (for seeding)
type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r"`
	Role          string    `bun:"role,pk" json:"role"`
	Description   string    `bun:"description" json:"description"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// UserRole represents a user's role assignment (many-to-many)
type UserRole struct {
	bun.BaseModel `bun:"table:user_roles,alias:ur"`
	UserID        string    `bun:"user_id,pk" json:"user_id"`
	Role          string    `bun:"role,pk" json:"role"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// RolePermission represents a role's permission (composite PK: role + permission)
type RolePermission struct {
	bun.BaseModel `bun:"table:role_permissions,alias:rp"`
	Role          string     `bun:"role,pk" json:"role"`
	Permission    string     `bun:"permission,pk" json:"permission"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	ExpiresAt     *time.Time `bun:"expires_at" json:"expires_at,omitempty"`
}

// UserPermission represents a user's permission override (composite PK: user_id + permission)
type UserPermission struct {
	bun.BaseModel `bun:"table:user_permissions,alias:up"`
	UserID        string     `bun:"user_id,pk" json:"user_id"`
	Permission    string     `bun:"permission,pk" json:"permission"`
	IsGranted     bool       `bun:"is_granted,notnull" json:"is_granted"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	ExpiresAt     *time.Time `bun:"expires_at" json:"expires_at,omitempty"`
}

// PermissionChangesLog represents an audit log entry
type PermissionChangesLog struct {
	bun.BaseModel `bun:"table:permission_changes_log,alias:pcl"`
	ID            string     `bun:"id,pk" json:"id"`
	Action        string     `bun:"action,notnull" json:"action"`           // grant, revoke
	TargetType    string     `bun:"target_type,notnull" json:"target_type"` // role, user_permission
	TargetRole    *string    `bun:"target_role" json:"target_role,omitempty"`
	TargetUserID  *string    `bun:"target_user_id" json:"target_user_id,omitempty"`
	Permission    string     `bun:"permission,notnull" json:"permission"`
	IsGranted     bool       `bun:"is_granted,notnull" json:"is_granted"`
	ChangedBy     string     `bun:"changed_by,notnull" json:"changed_by"`
	Reason        *string    `bun:"reason" json:"reason,omitempty"`
	IPAddress     *string    `bun:"ip_address" json:"ip_address,omitempty"`
	UserAgent     *string    `bun:"user_agent" json:"user_agent,omitempty"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	ExpiresAt     *time.Time `bun:"expires_at" json:"expires_at,omitempty"`
}

// PermissionQueryFilter defines filters for querying permission changes
type PermissionQueryFilter struct {
	TargetType   *string
	TargetRole   *string
	TargetUserID *string
	Permission   *string
	ChangedBy    *string
	Action       *string
	FromDate     *time.Time
	ToDate       *time.Time
	Limit        int
	Offset       int
}

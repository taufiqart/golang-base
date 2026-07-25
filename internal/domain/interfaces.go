package domain

import (
	"context"
	"time"
)

// UserRepository defines user repository operations
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Count(ctx context.Context) (int, error)
}

// RolePermissionRepository defines role permission operations
type RolePermissionRepository interface {
	Grant(ctx context.Context, role string, permission string, expiresAt ...*time.Time) error
	Revoke(ctx context.Context, role string, permission string) error
	GetByRole(ctx context.Context, role string) ([]*RolePermission, error)
}

// UserPermissionRepository defines user permission operations
type UserPermissionRepository interface {
	Grant(ctx context.Context, userID string, permission string, isGranted bool, expiresAt *time.Time) error
	Revoke(ctx context.Context, userID string, permission string) error
	GetByUser(ctx context.Context, userID string) ([]*UserPermission, error)
}

// PermissionChangesLogRepository defines audit log operations
type PermissionChangesLogRepository interface {
	Log(ctx context.Context, log *PermissionChangesLog) error
	Query(ctx context.Context, filter *PermissionQueryFilter) ([]*PermissionChangesLog, error)
}

// AuthService defines authentication operations
type AuthService interface {
	CreateUser(ctx context.Context, email, password, name string, roles []string) (*User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (accessToken string, err error)
}

// UserService defines user business logic operations
type UserService interface {
	GetProfile(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, limit, offset int) ([]*User, int, error)
}

// End of base interfaces

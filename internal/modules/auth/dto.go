package auth

import (
	"golang-base/internal/domain"
)

// Auth DTOs

type RegisterRequest struct {
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required"`
	Name     string   `json:"name" validate:"required"`
	Roles    []string `json:"roles"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string        `json:"access_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int           `json:"expires_in"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	User         *UserResponse `json:"user"`
}

// NewAuthResponse creates AuthResponse with defaults
func NewAuthResponse(accessToken, refreshToken string, user *UserResponse) AuthResponse {
	return AuthResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
		RefreshToken: refreshToken,
		User:         user,
	}
}

type UserResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	IsActive    bool     `json:"is_active"`
	CreatedAt   string   `json:"created_at"`
	Permissions []string `json:"permissions,omitempty"`
}

type CreateUserRequest struct {
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required"`
	Name     string   `json:"name" validate:"required"`
	Roles    []string `json:"roles" validate:"required"`
}

type UpdateUserRequest struct {
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	IsActive *bool    `json:"is_active"`
}

type RoleRequest struct {
	Role        string `json:"role"`
	Description string `json:"description"`
}

type RoleResponse struct {
	Role        string `json:"role"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type UpdatePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type GrantPermissionRequest struct {
	Permission string `json:"permission"`
}

type UserPermissionRequest struct {
	Granted   *bool   `json:"granted"`
	ExpiresAt *string `json:"expires_at"`
	Reason    *string `json:"reason"`
}

type UserFilter struct {
	Search   *string `query:"search"`
	Role     *string `query:"role"`
	IsActive *bool   `query:"is_active"`
	Limit    int     `query:"limit"`
	Offset   int     `query:"offset"`
}

type AuditLogQuery struct {
	TargetType   *string `query:"target_type"`
	TargetRole   *string `query:"target_role"`
	TargetUserID *string `query:"target_user_id"`
	Permission   *string `query:"permission"`
	ChangedBy    *string `query:"changed_by"`
	Action       *string `query:"action"`
	FromDate     *string `query:"from_date"`
	ToDate       *string `query:"to_date"`
	Limit        int     `query:"limit"`
	Offset       int     `query:"offset"`
}

type PermissionMatrixResponse struct {
	Permissions []string                   `json:"permissions"`
	Roles       []string                   `json:"roles"`
	Matrix      map[string]map[string]bool `json:"matrix"`
}

type PermissionGroupResponse struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// Helper functions

func ToUserResponse(user *domain.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Roles:     user.Roles,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ToUserResponsePtr converts *User to *UserResponse
func ToUserResponsePtr(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}
	resp := ToUserResponse(user)
	return &resp
}

// ToUserResponseWithPermissions converts *User with computed permissions
func ToUserResponseWithPermissions(user *domain.User, permissions []string) UserResponse {
	resp := ToUserResponse(user)
	resp.Permissions = permissions
	return resp
}

func ToRoleResponse(role *domain.Role) RoleResponse {
	return RoleResponse{
		Role:        role.Role,
		Description: role.Description,
		CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// RolePermissionsResponse represents role permissions from database
type RolePermissionsResponse struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// PermissionListResponse represents the /permissions/list response
type PermissionListResponse struct {
	Permissions []domain.PermissionDefinition `json:"permissions"`
	Roles       []string                      `json:"roles"`
}

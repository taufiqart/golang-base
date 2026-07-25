package user

import (
	"golang-base/internal/domain"
)

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Email    string   `json:"email" validate:"required,email"`
	Name     string   `json:"name" validate:"required"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email string   `json:"email"`
	Name  string   `json:"name" validate:"required"`
	Roles []string `json:"roles"`
}

// UserResponse represents the response body for user data
type UserResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

// ToResponse converts a User entity to UserResponse DTO
func ToResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

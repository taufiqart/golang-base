package seeders

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang-base/internal/domain"
	"golang-base/internal/modules/auth"

	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type SuperAdminSeed struct {
	email    string
	password string
	role     string
}

func (s *SuperAdminSeed) Name() string { return "SuperAdminSeeder" }

func (s *SuperAdminSeed) Order() int { return 2 }

func (s *SuperAdminSeed) SetArgs(args []string) {
	// Usage: SuperAdminSeeder [email] [password] [role]
	if len(args) > 0 && args[0] != "" {
		s.email = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		s.password = args[1]
	}
	if len(args) > 2 && args[2] != "" {
		s.role = args[2]
	}
}

func (s *SuperAdminSeed) Run(db *bun.DB) error {
	email := s.email
	if email == "" {
		email = os.Getenv("SUPER_ADMIN_EMAIL")
	}
	if email == "" {
		email = "admin@example.com"
	}

	password := s.password
	if password == "" {
		password = os.Getenv("SUPER_ADMIN_PASSWORD")
	}
	if password == "" {
		return fmt.Errorf("password required: pass via ARGS or SUPER_ADMIN_PASSWORD env")
	}

	role := s.role
	if role == "" {
		role = string(domain.RoleSuperAdmin)
	}

	ctx := context.Background()
	authRepo := auth.NewRepository(db)

	// Check if user already exists
	existingUser, err := authRepo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		// Hash password for update
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		hashedStr := string(hashedPassword)

		existingUser.Roles = []string{role}
		existingUser.Password = &hashedStr
		existingUser.UpdatedAt = time.Now()

		if err := authRepo.UpdateUser(ctx, existingUser); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		fmt.Printf("  Updated %s (role=%q, password refreshed)\n", email, role)
		return nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashedStr := string(hashedPassword)
	user := &domain.User{
		Email:     email,
		Password:  &hashedStr,
		Name:      "Super Admin",
		Roles:     []string{role},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := authRepo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create super admin: %w", err)
	}

	fmt.Printf("  Created user %s with role %q\n", email, role)
	return nil
}

func init() { Register(&SuperAdminSeed{}) }

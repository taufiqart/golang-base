package seeders

import (
	"context"
	"fmt"

	"golang-base/internal/domain"
	"golang-base/internal/modules/auth"

	"github.com/uptrace/bun"
)

type RolePermissionsSeed struct{}

func (s *RolePermissionsSeed) Name() string { return "RolePermissionsSeeder" }

func (s *RolePermissionsSeed) Order() int { return 1 }

func (s *RolePermissionsSeed) Run(db *bun.DB) error {
	ctx := context.Background()

	allPermissions := domain.AllPermissions()

	fmt.Printf("  Found %d permissions in code\n", len(allPermissions))
	fmt.Printf("  Found %d roles in code\n", len(PermissionMatrix))

	// Count existing rows
	var count int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM role_permissions").Scan(&count)
	fmt.Printf("  Current role_permissions count: %d\n", count)

	// Clear existing role_permissions
	if count > 0 {
		_, err := db.ExecContext(ctx, "TRUNCATE TABLE role_permissions CASCADE")
		if err != nil {
			return fmt.Errorf("failed to truncate role_permissions: %w", err)
		}
		fmt.Println("  Cleared existing role_permissions")
	}

	// Insert permissions for each role from code
	totalInserted := 0
	authRepo := auth.NewRepository(db)
	for role, permissions := range PermissionMatrix {
		// Create role first to avoid foreign key constraints
		_ = authRepo.CreateRole(ctx, &domain.Role{
			Role:        role,
			Description: fmt.Sprintf("Role: %s", role),
		})

		fmt.Printf("  Seeding %d permissions for role: %s\n", len(permissions), role)
		for _, perm := range permissions {
			err := authRepo.GrantRolePermission(ctx, role, perm)
			if err != nil {
				// Skip duplicates or errors
				continue
			}
			totalInserted++
		}
	}

	fmt.Printf("  Seeded %d role_permissions\n", totalInserted)
	return nil
}

// PermissionMatrix defines default permissions per role
var PermissionMatrix = map[string][]string{
	domain.RoleSuperAdmin: domain.AllPermissions(),
	domain.RoleAdmin: {
		"user.create", "user.edit", "user.view",
		"role.view",
	},
	domain.RoleUser: {
		"user.view",
	},
}

func init() { Register(&RolePermissionsSeed{}) }

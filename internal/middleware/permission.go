package middleware

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"golang-base/internal/database"
	"golang-base/internal/domain"
	"golang-base/internal/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// CachedAuthRepository adds Redis caching to auth operations
type CachedAuthRepository struct {
	GetUserByID        func(ctx context.Context, id string) (*domain.User, error)
	GetUserPermissions func(ctx context.Context, userID string) ([]*domain.UserPermission, error)
	GetRolePermissions func(ctx context.Context, role string) ([]*domain.RolePermission, error)
	redis              *redis.Client
}

// NewCachedAuthRepository creates a cached auth repository
func NewCachedAuthRepository(
	getUserByID func(ctx context.Context, id string) (*domain.User, error),
	getUserPermissions func(ctx context.Context, userID string) ([]*domain.UserPermission, error),
	getRolePermissions func(ctx context.Context, role string) ([]*domain.RolePermission, error),
	redisClient *redis.Client,
) *CachedAuthRepository {
	if redisClient == nil {
		redisClient = database.Redis
	}
	return &CachedAuthRepository{
		GetUserByID:        getUserByID,
		GetUserPermissions: getUserPermissions,
		GetRolePermissions: getRolePermissions,
		redis:              redisClient,
	}
}

// CachedGetRolePermissions retrieves role permissions from cache or database
func (r *CachedAuthRepository) CachedGetRolePermissions(ctx context.Context, role string) ([]*domain.RolePermission, error) {
	if r.redis != nil {
		cacheKey := domain.CacheRolePermissionsKey + role
		cached, err := r.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var perms []*domain.RolePermission
			if json.Unmarshal(cached, &perms) == nil {
				return perms, nil
			}
		}
	}

	perms, err := r.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	if r.redis != nil && len(perms) > 0 {
		if data, err := json.Marshal(perms); err == nil {
			r.redis.Set(ctx, domain.CacheRolePermissionsKey+role, data, domain.CacheTTL)
		}
	}

	return perms, nil
}

// CachedGetUserPermissions retrieves user permissions from cache or database
func (r *CachedAuthRepository) CachedGetUserPermissions(ctx context.Context, userID string) ([]*domain.UserPermission, error) {
	if r.redis != nil {
		cacheKey := domain.CacheUserPermissionsKey + userID
		cached, err := r.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var perms []*domain.UserPermission
			if json.Unmarshal(cached, &perms) == nil {
				return perms, nil
			}
		}
	}

	perms, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	if r.redis != nil && len(perms) > 0 {
		if data, err := json.Marshal(perms); err == nil {
			r.redis.Set(ctx, domain.CacheUserPermissionsKey+userID, data, domain.CacheTTL)
		}
	}

	return perms, nil
}

// InvalidateUserPermissionsCache removes user permissions from cache
func (r *CachedAuthRepository) InvalidateUserPermissionsCache(ctx context.Context, userID string) error {
	if r.redis != nil {
		return r.redis.Del(ctx, domain.CacheUserPermissionsKey+userID).Err()
	}
	return nil
}

// InvalidateRolePermissionsCache removes role permissions from cache
func (r *CachedAuthRepository) InvalidateRolePermissionsCache(ctx context.Context, role string) error {
	if r.redis != nil {
		return r.redis.Del(ctx, domain.CacheRolePermissionsKey+role).Err()
	}
	return nil
}

// matchPermission checks if user permission matches any allowed pattern
// Supports wildcard: "master.*" matches "master.view", "master.edit", etc.
func matchPermission(userPerm string, allowedPatterns []string) bool {
	for _, pattern := range allowedPatterns {
		// Exact match
		if userPerm == pattern {
			return true
		}
		// Wildcard match: "master.*" matches "master.view"
		if prefix, ok := strings.CutSuffix(pattern, ".*"); ok {
			if strings.HasPrefix(userPerm, prefix+".") {
				return true
			}
		}
	}
	return false
}

// AllowedPermissions creates middleware requiring any of the specified permissions
// Supports wildcard patterns: "master.*" matches "master.view", "master.edit", etc.
// Super admin always bypasses.
//
// Usage:
//
//	middleware.AllowedPermissions("master.view")
//	middleware.AllowedPermissions("master.*")
//	middleware.AllowedPermissions("master.view", "master.edit")
//	middleware.AllowedPermissions("master.*", "pricing.view")
func AllowedPermissions(permissions ...string) fiber.Handler {
	if cachedRepo == nil {
		return func(c *fiber.Ctx) error { return c.Next() }
	}

	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(string)
		if !ok || userID == "" {
			return response.Unauthorized(c, "unauthorized")
		}

		ctx := c.Context()
		user, err := cachedRepo.GetUserByID(ctx, userID)
		if err != nil {
			return response.Unauthorized(c, "user not found")
		}

		// Super admin bypass
		if slices.Contains(user.Roles, domain.RoleSuperAdmin) {
			return c.Next()
		}

		// Check role permissions (from cache or DB)
		for _, role := range user.Roles {
			rolePerms, err := cachedRepo.CachedGetRolePermissions(ctx, role)
			if err == nil {
				for _, p := range rolePerms {
					if matchPermission(p.Permission, permissions) {
						return c.Next()
					}
				}
			}
		}

		// Check user permission overrides
		userPerms, err := cachedRepo.CachedGetUserPermissions(ctx, userID)
		if err == nil {
			for _, up := range userPerms {
				if matchPermission(up.Permission, permissions) {
					return c.Next()
				}
			}
		}

		return response.Forbidden(c, "insufficient permissions")
	}
}

// Global cached auth repository (initialized via InitPermissionMiddleware)
var cachedRepo *CachedAuthRepository

// InitPermissionMiddleware initializes the global permission middleware with auth repository functions
func InitPermissionMiddleware(
	getUserByID func(ctx context.Context, id string) (*domain.User, error),
	getUserPermissions func(ctx context.Context, userID string) ([]*domain.UserPermission, error),
	getRolePermissions func(ctx context.Context, role string) ([]*domain.RolePermission, error),
) {
	cachedRepo = NewCachedAuthRepository(getUserByID, getUserPermissions, getRolePermissions, nil)
}

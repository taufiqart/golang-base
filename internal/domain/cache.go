package domain

import "time"

// Cache key prefixes and TTL for permission caching
const (
	// CacheUserPermissionsKey is the Redis key prefix for user permissions
	CacheUserPermissionsKey = "user:permissions:"
	// CacheRolePermissionsKey is the Redis key prefix for role permissions
	CacheRolePermissionsKey = "role:permissions:"
	// CacheTTL is the TTL for permissions cache (1 day)
	CacheTTL = 24 * time.Hour
)

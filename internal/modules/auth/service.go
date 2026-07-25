package auth

import (
	"context"
	"slices"
	"sort"
	"time"

	"golang-base/internal/database"
	"golang-base/internal/domain"
	"golang-base/internal/pkg/db"
	jwtpkg "golang-base/internal/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

// Domain errors re-exported for convenience
var (
	ErrInvalidCredentials = domain.ErrInvalidCredentials
	ErrUserExists         = domain.ErrUserExists
	ErrUserNotFound       = domain.ErrUserNotFound
	ErrRoleNotFound       = domain.ErrRoleNotFound
)

type service struct {
	repo *repository
	jwt  *jwtpkg.JWT
}

func NewService(repo *repository, jwtInstance *jwtpkg.JWT) *service {
	if jwtInstance == nil {
		jwtInstance = jwtpkg.NewJWT()
	}
	return &service{
		repo: repo,
		jwt:  jwtInstance,
	}
}

// Auth Service

func (s *service) Register(ctx context.Context, email, password, name string, roles []string) (*domain.User, error) {
	existing, _ := s.repo.GetUserByEmail(ctx, email)
	if existing != nil {
		return nil, ErrUserExists
	}

	var hashedPassword *string
	if password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedStr := string(hashed)
		hashedPassword = &hashedStr
	}

	user := &domain.User{
		ID:        "",
		Email:     email,
		Password:  hashedPassword,
		Name:      name,
		Roles:     roles,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, db.MapDBError(err)
	}

	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, string, *domain.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	if user.Password == nil {
		return "", "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return "", "", nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := s.jwt.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	newAccessToken, err := s.jwt.GenerateAccessToken(claims.UserID)
	if err != nil {
		return "", err
	}

	return newAccessToken, nil
}

// User Service

func (s *service) CreateUser(ctx context.Context, email, password, name string, roles []string) (*domain.User, error) {
	existing, _ := s.repo.GetUserByEmail(ctx, email)
	if existing != nil {
		return nil, ErrUserExists
	}

	var hashedPassword *string
	if password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedStr := string(hashed)
		hashedPassword = &hashedStr
	}

	user := &domain.User{
		ID:        "",
		Email:     email,
		Password:  hashedPassword,
		Name:      name,
		Roles:     roles,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, db.MapDBError(err)
	}

	return user, nil
}

// User Service

func (s *service) GetUser(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *service) UpdateUser(ctx context.Context, id string, name *string, roles []string, isActive *bool) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if name != nil {
		user.Name = *name
	}
	if len(roles) > 0 {
		user.Roles = roles
	}
	if isActive != nil {
		user.IsActive = *isActive
	}
	user.UpdatedAt = time.Now()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, db.MapDBError(err)
	}

	return user, nil
}

func (s *service) DeleteUser(ctx context.Context, id string) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) ListUsers(ctx context.Context, filter *UserFilter) ([]*domain.User, int, error) {
	users, err := s.repo.ListUsers(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountUsers(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

// Role Service

func (s *service) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	perms, err := s.repo.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(perms))
	now := time.Now()
	for _, p := range perms {
		if p.ExpiresAt != nil && p.ExpiresAt.Before(now) {
			continue // skip expired permissions
		}
		result = append(result, p.Permission)
	}
	return result, nil
}

func (s *service) UpdateRolePermissions(ctx context.Context, role string, permissions []string) error {
	existing, err := s.repo.GetRolePermissions(ctx, role)
	if err != nil {
		return err
	}

	existingMap := make(map[string]bool)
	for _, p := range existing {
		existingMap[p.Permission] = true
	}

	for _, perm := range permissions {
		if !existingMap[perm] {
			if err := s.repo.GrantRolePermission(ctx, role, perm); err != nil {
				return err
			}
		}
	}

	newMap := make(map[string]bool)
	for _, p := range permissions {
		newMap[p] = true
	}

	for _, p := range existing {
		if !newMap[p.Permission] {
			if err := s.repo.RevokeRolePermission(ctx, role, p.Permission); err != nil {
				return err
			}
		}
	}

	// Invalidate role permission cache after bulk update
	s.invalidateRolePermissionCache(ctx, role)

	return nil
}

// Permission Service

// GetPermissionMatrix returns permission matrix from hardcoded definitions + DB role permissions
func (s *service) GetPermissionMatrix(ctx context.Context) (*PermissionMatrixResponse, error) {
	// Get all permissions and roles from hardcoded definitions
	allPerms := domain.AllPermissions()
	allRoles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		allRoles = []string{}
	}

	// Get role permissions from database
	dbRolePerms, err := s.repo.GetAllRolePermissions(ctx)
	if err != nil {
		dbRolePerms = make(map[string][]string)
	}

	// Build matrix: hardcoded roles x permissions, check DB for granted permissions
	matrix := make(map[string]map[string]bool)

	for _, role := range allRoles {
		matrix[role] = make(map[string]bool)
		// Initialize all permissions to false
		for _, perm := range allPerms {
			matrix[role][perm] = false
		}
		// Check DB: if role has permissions in DB, mark them as true
		if dbPerms, ok := dbRolePerms[role]; ok {
			for _, perm := range dbPerms {
				matrix[role][perm] = true
			}
		}
	}

	return &PermissionMatrixResponse{
		Permissions: allPerms,
		Roles:       allRoles,
		Matrix:      matrix,
	}, nil
}

// ListPermissions returns all available permission definitions (static) and role names (from DB)
func (s *service) ListPermissions(ctx context.Context) (*PermissionListResponse, error) {
	allPerms := domain.AllPermissions()
	allRoles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		allRoles = []string{}
	}

	definitions := make([]domain.PermissionDefinition, 0, len(allPerms))
	for _, p := range allPerms {
		definitions = append(definitions, domain.PermissionDefinition{
			Key:         p,
			Description: domain.GetPermissionDescription(p),
			Category:    domain.GetPermissionCategory(p),
		})
	}

	return &PermissionListResponse{
		Permissions: definitions,
		Roles:       allRoles,
	}, nil
}

// GetAllRolesWithPermissions returns all roles with their permissions from DB
func (s *service) GetAllRolesWithPermissions(ctx context.Context) (map[string][]string, error) {
	var rolePermissions []*domain.RolePermission
	err := s.repo.db.NewSelect().Model(&rolePermissions).Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, rp := range rolePermissions {
		result[rp.Role] = append(result[rp.Role], rp.Permission)
	}

	// Sort permissions for each role
	for role, perms := range result {
		sort.Strings(perms)
		result[role] = perms
	}

	return result, nil
}

func (s *service) GetUserPermissions(ctx context.Context, userID string) (map[string]bool, error) {
	perms, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	now := time.Now()
	for _, p := range perms {
		if p.ExpiresAt != nil && p.ExpiresAt.Before(now) {
			continue // skip expired permissions
		}
		result[p.Permission] = p.IsGranted
	}
	return result, nil
}

// GetComputedPermissions returns all permissions for a user (role permissions + user overrides)
func (s *service) GetComputedPermissions(ctx context.Context, userID string, roles []string) ([]string, error) {
	var permissions []string

	// Super admin has all permissions
	if slices.Contains(roles, domain.RoleSuperAdmin) {
		permissions = append(permissions, "*")
		return permissions, nil
	}

	// Get role permissions
	permSet := make(map[string]bool)
	for _, role := range roles {
		rolePerms, err := s.GetRolePermissions(ctx, role)
		if err != nil {
			continue
		}
		for _, p := range rolePerms {
			permSet[p] = true
		}
	}

	// Get user permission overrides
	userPerms, err := s.GetUserPermissions(ctx, userID)
	if err == nil {
		for perm, granted := range userPerms {
			if granted {
				permSet[perm] = true
			} else {
				delete(permSet, perm) // Explicitly revoked
			}
		}
	}

	// Convert to sorted slice
	permissions = make([]string, 0, len(permSet))
	for p := range permSet {
		permissions = append(permissions, p)
	}
	sort.Strings(permissions)

	return permissions, nil
}

// HasPermission checks if a user has a specific permission (role permissions + user overrides)
func (s *service) HasPermission(ctx context.Context, userID, permission string) bool {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return false
	}

	// Super admin has all permissions
	if slices.Contains(user.Roles, domain.RoleSuperAdmin) {
		return true
	}

	// Check user permission overrides first (explicit grant or explicit revoke)
	userPerms, err := s.GetUserPermissions(ctx, userID)
	if err == nil {
		if granted, ok := userPerms[permission]; ok {
			return granted
		}
	}

	// Check role permissions from DB
	for _, role := range user.Roles {
		rolePerms, err := s.GetRolePermissions(ctx, role)
		if err == nil && slices.Contains(rolePerms, permission) {
			return true
		}
	}

	return false
}

func (s *service) GrantUserPermission(ctx context.Context, targetType string, targetRole *string, targetUserID *string, permission string, isGranted bool, expiresAt *time.Time, actorID string, reason *string, ipAddress *string, userAgent *string) error {
	if targetType == "role" && targetRole != nil {
		if err := s.repo.GrantRolePermission(ctx, *targetRole, permission, expiresAt); err != nil {
			return err
		}
		// Invalidate role permission cache
		s.invalidateRolePermissionCache(ctx, *targetRole)
	} else if targetType == "user_permission" && targetUserID != nil {
		if err := s.repo.GrantUserPermission(ctx, *targetUserID, permission, isGranted, expiresAt); err != nil {
			return err
		}
		// Invalidate user permission cache
		s.invalidateUserPermissionCache(ctx, *targetUserID)
	}

	action := "grant"
	if !isGranted {
		action = "revoke"
	}

	log := &domain.PermissionChangesLog{
		ID:           "",
		Action:       action,
		TargetType:   targetType,
		TargetRole:   targetRole,
		TargetUserID: targetUserID,
		Permission:   permission,
		IsGranted:    isGranted,
		ChangedBy:    actorID,
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		ExpiresAt:    expiresAt,
	}

	return s.repo.LogPermissionChange(ctx, log)
}

func (s *service) RevokeUserPermission(ctx context.Context, targetType string, targetRole *string, targetUserID *string, permission string, actorID string, reason *string, ipAddress *string, userAgent *string) error {
	if targetType == "role" && targetRole != nil {
		if err := s.repo.RevokeRolePermission(ctx, *targetRole, permission); err != nil {
			return err
		}
		// Invalidate role permission cache
		s.invalidateRolePermissionCache(ctx, *targetRole)
	} else if targetType == "user_permission" && targetUserID != nil {
		if err := s.repo.RevokeUserPermission(ctx, *targetUserID, permission); err != nil {
			return err
		}
		// Invalidate user permission cache
		s.invalidateUserPermissionCache(ctx, *targetUserID)
	}

	log := &domain.PermissionChangesLog{
		ID:           "",
		Action:       "revoke",
		TargetType:   targetType,
		TargetRole:   targetRole,
		TargetUserID: targetUserID,
		Permission:   permission,
		ChangedBy:    actorID,
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
	}

	return s.repo.LogPermissionChange(ctx, log)
}

// Audit Service

func (s *service) QueryPermissionChanges(ctx context.Context, filter *domain.PermissionQueryFilter) ([]*domain.PermissionChangesLog, error) {
	return s.repo.QueryPermissionChanges(ctx, filter)
}

func (s *service) CountPermissionChanges(ctx context.Context, filter *domain.PermissionQueryFilter) (int, error) {
	return s.repo.CountPermissionChanges(ctx, filter)
}

// invalidateUserPermissionCache removes user permissions from Redis cache
func (s *service) invalidateUserPermissionCache(ctx context.Context, userID string) {
	if database.Redis == nil {
		return
	}
	key := domain.CacheUserPermissionsKey + userID
	database.Redis.Del(ctx, key)
}

// invalidateRolePermissionCache removes role permissions from Redis cache
func (s *service) invalidateRolePermissionCache(ctx context.Context, role string) {
	if database.Redis == nil {
		return
	}
	key := domain.CacheRolePermissionsKey + role
	database.Redis.Del(ctx, key)
}

// GetAllRoleNames returns all role names from database
func (s *service) GetAllRoleNames(ctx context.Context) ([]string, error) {
	return s.repo.GetAllRoles(ctx)
}

package auth

import (
	"context"
	"time"

	"golang-base/internal/database"
	"golang-base/internal/domain"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *repository {
	if db == nil {
		db = database.DB
	}
	return &repository{db: db}
}

// User Repository

func (r *repository) CreateUser(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		user.ID = id.String()
	}
	_, err := r.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return err
	}
	if len(user.Roles) > 0 {
		var userRoles []domain.UserRole
		for _, role := range user.Roles {
			userRoles = append(userRoles, domain.UserRole{UserID: user.ID, Role: role})
		}
		_, err = r.db.NewInsert().Model(&userRoles).Exec(ctx)
	}
	return err
}

func (r *repository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	var roles []string
	r.db.NewSelect().Model((*domain.UserRole)(nil)).Column("role").Where("user_id = ?", user.ID).Scan(ctx, &roles)
	user.Roles = roles
	return &user, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.NewSelect().Model(&user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		return nil, err
	}
	var roles []string
	r.db.NewSelect().Model((*domain.UserRole)(nil)).Column("role").Where("user_id = ?", user.ID).Scan(ctx, &roles)
	user.Roles = roles
	return &user, nil
}

func (r *repository) UpdateUser(ctx context.Context, user *domain.User) error {
	_, err := r.db.NewUpdate().Model(user).Where("id = ?", user.ID).Exec(ctx)
	if err != nil {
		return err
	}
	r.db.NewDelete().Model((*domain.UserRole)(nil)).Where("user_id = ?", user.ID).Exec(ctx)
	if len(user.Roles) > 0 {
		var userRoles []domain.UserRole
		for _, role := range user.Roles {
			userRoles = append(userRoles, domain.UserRole{UserID: user.ID, Role: role})
		}
		_, err = r.db.NewInsert().Model(&userRoles).Exec(ctx)
	}
	return err
}

func (r *repository) DeleteUser(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&domain.User{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *repository) ListUsers(ctx context.Context, filter *UserFilter) ([]*domain.User, error) {
	var users []*domain.User
	query := r.db.NewSelect().Model(&users)

	if filter.Search != nil && *filter.Search != "" {
		s := "%" + *filter.Search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", s, s)
	}
	if filter.Role != nil && *filter.Role != "" {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	query = query.Order("created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Scan(ctx)
	return users, err
}

func (r *repository) CountUsers(ctx context.Context, filter *UserFilter) (int, error) {
	query := r.db.NewSelect().Model((*domain.User)(nil))

	if filter.Search != nil && *filter.Search != "" {
		s := "%" + *filter.Search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", s, s)
	}
	if filter.Role != nil && *filter.Role != "" {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	count, err := query.Count(ctx)
	return count, err
}

// Role Repository

func (r *repository) CreateRole(ctx context.Context, role *domain.Role) error {
	_, err := r.db.NewInsert().Model(role).Ignore().Exec(ctx)
	return err
}

func (r *repository) GrantRolePermission(ctx context.Context, role string, permission string, expiresAt ...*time.Time) error {
	var exp *time.Time
	if len(expiresAt) > 0 {
		exp = expiresAt[0]
	}
	rp := &domain.RolePermission{
		Role:       role,
		Permission: permission,
		CreatedAt:  time.Now(),
		ExpiresAt:  exp,
	}
	_, err := r.db.NewInsert().Model(rp).
		On("CONFLICT (role, permission) DO UPDATE").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx)
	return err
}

func (r *repository) RevokeRolePermission(ctx context.Context, role string, permission string) error {
	_, err := r.db.NewDelete().Model(&domain.RolePermission{}).
		Where("role = ? AND permission = ?", role, permission).Exec(ctx)
	return err
}

func (r *repository) GetRolePermissions(ctx context.Context, role string) ([]*domain.RolePermission, error) {
	var permissions []*domain.RolePermission
	err := r.db.NewSelect().Model(&permissions).
		Where("role = ?", role).Scan(ctx)
	return permissions, err
}

// UserPermission Repository

func (r *repository) GrantUserPermission(ctx context.Context, userID string, permission string, isGranted bool, expiresAt *time.Time) error {
	up := &domain.UserPermission{
		UserID:     userID,
		Permission: permission,
		IsGranted:  isGranted,
		CreatedAt:  time.Now(),
		ExpiresAt:  expiresAt,
	}
	_, err := r.db.NewInsert().Model(up).
		On("CONFLICT (user_id, permission) DO UPDATE").
		Set("is_granted = EXCLUDED.is_granted").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx)
	return err
}

func (r *repository) RevokeUserPermission(ctx context.Context, userID string, permission string) error {
	_, err := r.db.NewDelete().Model(&domain.UserPermission{}).
		Where("user_id = ? AND permission = ?", userID, permission).Exec(ctx)
	return err
}

func (r *repository) GetUserPermissions(ctx context.Context, userID string) ([]*domain.UserPermission, error) {
	var permissions []*domain.UserPermission
	err := r.db.NewSelect().Model(&permissions).
		Where("user_id = ?", userID).Scan(ctx)
	return permissions, err
}

// PermissionChangesLog Repository

func (r *repository) LogPermissionChange(ctx context.Context, log *domain.PermissionChangesLog) error {
	if log.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		log.ID = id.String()
	}
	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	return err
}

func (r *repository) QueryPermissionChanges(ctx context.Context, filter *domain.PermissionQueryFilter) ([]*domain.PermissionChangesLog, error) {
	var logs []*domain.PermissionChangesLog
	query := r.db.NewSelect().Model(&logs)

	if filter.TargetType != nil {
		query = query.Where("target_type = ?", *filter.TargetType)
	}
	if filter.TargetRole != nil {
		query = query.Where("target_role = ?", *filter.TargetRole)
	}
	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}
	if filter.Permission != nil {
		query = query.Where("permission = ?", *filter.Permission)
	}
	if filter.ChangedBy != nil {
		query = query.Where("changed_by = ?", *filter.ChangedBy)
	}
	if filter.Action != nil {
		query = query.Where("action = ?", *filter.Action)
	}
	if filter.FromDate != nil {
		query = query.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("created_at <= ?", *filter.ToDate)
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Scan(ctx)
	return logs, err
}

func (r *repository) CountPermissionChanges(ctx context.Context, filter *domain.PermissionQueryFilter) (int, error) {
	query := r.db.NewSelect().Model((*domain.PermissionChangesLog)(nil))

	if filter.TargetType != nil {
		query = query.Where("target_type = ?", *filter.TargetType)
	}
	if filter.TargetRole != nil {
		query = query.Where("target_role = ?", *filter.TargetRole)
	}
	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}
	if filter.Permission != nil {
		query = query.Where("permission = ?", *filter.Permission)
	}
	if filter.ChangedBy != nil {
		query = query.Where("changed_by = ?", *filter.ChangedBy)
	}
	if filter.Action != nil {
		query = query.Where("action = ?", *filter.Action)
	}
	if filter.FromDate != nil {
		query = query.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("created_at <= ?", *filter.ToDate)
	}

	count, err := query.Count(ctx)
	return count, err
}

// GetAllRolePermissions returns all role permissions grouped by role
func (r *repository) GetAllRolePermissions(ctx context.Context) (map[string][]string, error) {
	var rolePermissions []*domain.RolePermission
	err := r.db.NewSelect().Model(&rolePermissions).Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, rp := range rolePermissions {
		result[rp.Role] = append(result[rp.Role], rp.Permission)
	}
	return result, nil
}

// GetAllRoles returns all role names from database
func (r *repository) GetAllRoles(ctx context.Context) ([]string, error) {
	var roles []*domain.Role
	err := r.db.NewSelect().Model(&roles).Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(roles))
	for i, r := range roles {
		result[i] = r.Role
	}
	return result, nil
}

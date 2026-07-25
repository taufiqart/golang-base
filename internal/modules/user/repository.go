package user

import (
	"context"

	"golang-base/internal/database"
	"golang-base/internal/domain"

	"github.com/uptrace/bun"
)

type repository struct {
	db *bun.DB
}

// NewRepository creates a new user repository instance
func NewRepository() domain.UserRepository {
	return &repository{
		db: database.DB,
	}
}

func (r *repository) GetByID(ctx context.Context, id string) (*domain.User, error) {
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

func (r *repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
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

func (r *repository) Create(ctx context.Context, user *domain.User) error {
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

func (r *repository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	var users []*domain.User
	err := r.db.NewSelect().
		Model(&users).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *repository) Update(ctx context.Context, user *domain.User) error {
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

func (r *repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&domain.User{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *repository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().Model(&domain.User{}).Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

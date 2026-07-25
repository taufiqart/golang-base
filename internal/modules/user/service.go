package user

import (
	"context"
	"errors"

	"golang-base/internal/domain"
)

type service struct {
	repo domain.UserRepository
}

// NewService creates a new user service instance
func NewService(repo domain.UserRepository) domain.UserService {
	return &service{
		repo: repo,
	}
}

func (s *service) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx)
	if err != nil {
		return users, len(users), nil // Return partial data
	}
	return users, total, nil
}

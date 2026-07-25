package auth

import (
	"golang-base/internal/database"
	"golang-base/internal/middleware"
	jwtpkg "golang-base/internal/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Register(router fiber.Router) {
	repo := NewRepository(database.DB)

	// Initialize permission middleware with auth repository functions
	middleware.InitPermissionMiddleware(
		repo.GetUserByID,
		repo.GetUserPermissions,
		repo.GetRolePermissions,
	)

	svc := NewService(repo, jwtpkg.NewJWT())
	handler := NewHandler(svc)

	handler.RegisterRoutes(router)
}

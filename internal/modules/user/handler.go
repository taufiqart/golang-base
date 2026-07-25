package user

import (
	"log"

	"golang-base/internal/domain"
	"golang-base/internal/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type handler struct {
	service domain.UserService
}

// NewHandler creates a new user handler instance
func NewHandler(service domain.UserService) *handler {
	return &handler{
		service: service,
	}
}

// RegisterRoutes registers all user routes
func (h *handler) RegisterRoutes(router fiber.Router) {
	users := router.Group("/users")
	users.Get("/", h.ListUsers)
	users.Get("/:id", h.GetProfile)
}

// ListUsers handles GET /users
func (h *handler) ListUsers(c *fiber.Ctx) error {
	page, limit, offset := response.ParsePaginationParams(c)

	users, total, err := h.service.List(c.Context(), limit, offset)
	if err != nil {
		log.Printf("error: %v", err)
		return response.InternalError(c, "internal server error")
	}

	userResponses := make([]UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = ToResponse(u)
	}

	return response.PaginatedResponse(c, userResponses, page, limit, total)
}

// GetProfile handles GET /users/:id
func (h *handler) GetProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	if id == "" {
		return response.BadRequest(c, "id is required")
	}

	user, err := h.service.GetProfile(c.Context(), id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}

	return response.OK(c, ToResponse(user))
}

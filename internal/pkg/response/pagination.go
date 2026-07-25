package response

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// PaginationMeta represents pagination metadata per OpenAPI spec
type PaginationMeta struct {
	TotalItems  int `json:"total_items"`
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

// NewPaginationMeta creates pagination meta from page, limit, and total
func NewPaginationMeta(page, limit, total int) PaginationMeta {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return PaginationMeta{
		TotalItems:  total,
		TotalPages:  totalPages,
		CurrentPage: page,
		Limit:       limit,
	}
}

// ParsePaginationParams extracts page and limit from Fiber context with defaults
func ParsePaginationParams(c *fiber.Ctx) (page, limit, offset int) {
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page = 1
	limit = 10

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100 // max limit
	}

	offset = (page - 1) * limit
	return
}

// PaginatedResponse sends a paginated response with meta
func PaginatedResponse(c *fiber.Ctx, data interface{}, page, limit, total int) error {
	meta := NewPaginationMeta(page, limit, total)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": data,
		"meta": meta,
	})
}

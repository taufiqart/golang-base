package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestDataResponse(t *testing.T) {
	resp := DataResponse{
		Data: map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"data"`)
	assert.Contains(t, string(data), `"key":"value"`)
}

func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse{
		Message: "validation error",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"message":"validation error"`)
}

func TestErrorResponseWithDetails(t *testing.T) {
	resp := ErrorResponse{
		Message: "Validation failed",
		Errors: []ErrorDetail{
			{Field: "email", Message: "The email address is already in use."},
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"message":"Validation failed"`)
	assert.Contains(t, string(data), `"field":"email"`)
}

func TestResponseHelpers(t *testing.T) {
	// Test that helper functions are available
	assert.NotNil(t, OK)
	assert.NotNil(t, Created)
	assert.NotNil(t, BadRequest)
	assert.NotNil(t, Unauthorized)
	assert.NotNil(t, NotFound)
	assert.NotNil(t, Forbidden)
	assert.NotNil(t, InternalError)
}

func TestOKResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return OK(c, "test data")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestOKResponseWithMeta(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return OKWithMeta(c, []string{"item1", "item2"}, map[string]int{"page": 1, "limit": 10})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCreatedResponse(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		return Created(c, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestBadRequestResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(c, "bad input")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestBadRequestValidationResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequestValidation(c, []ErrorDetail{
			{Field: "email", Message: "Invalid email format"},
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestNotFoundResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound(c, "resource not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestUnauthorizedResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Unauthorized(c, "not authenticated")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestForbiddenResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Forbidden(c, "access denied")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestConflictResponse(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		return Conflict(c, "resource already exists")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)
}

func TestInternalErrorResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return InternalError(c, "server error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestNoContentResponse(t *testing.T) {
	app := fiber.New()
	app.Delete("/test", func(c *fiber.Ctx) error {
		return NoContent(c)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

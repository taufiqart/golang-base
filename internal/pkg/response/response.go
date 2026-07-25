package response

import "github.com/gofiber/fiber/v2"

// DataResponse represents standard successful API response per OpenAPI spec
// Response format: { "data": ..., "meta": ... }
type DataResponse struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
}

// ErrorDetail represents field-level validation error
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents standard error API response per OpenAPI spec
// Response format: { "message": "...", "errors": [...] }
type ErrorResponse struct {
	Message string        `json:"message"`
	Errors  []ErrorDetail `json:"errors,omitempty"`
}

// OK sends a 200 successful response with data
func OK(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(DataResponse{
		Data: data,
	})
}

// OKWithMeta sends a 200 successful response with data and pagination meta
func OKWithMeta(c *fiber.Ctx, data interface{}, meta interface{}) error {
	return c.Status(fiber.StatusOK).JSON(DataResponse{
		Data: data,
		Meta: meta,
	})
}

// Created sends a 201 created response
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(DataResponse{
		Data: data,
	})
}

// BadRequest sends a 400 error response
func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Message: message,
	})
}

// BadRequestValidation sends a 400 with field-level errors
func BadRequestValidation(c *fiber.Ctx, errs []ErrorDetail) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Message: "Validation failed",
		Errors:  errs,
	})
}

// Unprocessable sends a 422 Unprocessable Entity with field-level errors
func Unprocessable(c *fiber.Ctx, errs []ErrorDetail) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
		Message: "Validation failed",
		Errors:  errs,
	})
}

// NotFound sends a 404 error response
func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Message: message,
	})
}

// Unauthorized sends a 401 error response
func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Message: message,
	})
}

// Forbidden sends a 403 error response
func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
		Message: message,
	})
}

// Conflict sends a 409 error response (for duplicate resources like email)
func Conflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
		Message: message,
	})
}

// InternalError sends a 500 error response
func InternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Message: message,
	})
}

// NoContent sends a 204 response (for DELETE operations)
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

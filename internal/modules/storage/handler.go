package storage

import (
	"fmt"
	"strconv"
	"time"

	"golang-base/internal/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service       Service
	maxUploadSize int64
}

func NewHandler(service Service, maxUploadSize int64) *Handler {
	if maxUploadSize <= 0 {
		maxUploadSize = 10 * 1024 * 1024 // 10MB default
	}
	return &Handler{
		service:       service,
		maxUploadSize: maxUploadSize,
	}
}

// Upload handles multipart file upload
// POST /api/v1/storage/upload
func (h *Handler) Upload(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file field is required in form-data")
	}

	if fileHeader.Size > h.maxUploadSize {
		return response.BadRequest(c, fmt.Sprintf("file size exceeds maximum allowed limit (%d bytes)", h.maxUploadSize))
	}

	f, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "failed to open uploaded file")
	}
	defer f.Close()

	folder := c.FormValue("folder")
	contentType := fileHeader.Header.Get("Content-Type")

	result, err := h.service.Upload(c.Context(), f, fileHeader.Filename, contentType, folder)
	if err != nil {
		return response.InternalError(c, fmt.Sprintf("upload failed: %v", err))
	}

	return response.Created(c, fiber.Map{
		"object_key": result.ObjectKey,
		"size":       result.Size,
		"url":        result.URL,
	})
}

// GetPresignedURL generates a temporary presigned URL for downloading/viewing an object
// GET /api/v1/storage/presigned?key=...&expiry_minutes=...
func (h *Handler) GetPresignedURL(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return response.BadRequest(c, "query parameter 'key' is required")
	}

	expiryMinutes, _ := strconv.Atoi(c.Query("expiry_minutes", "15"))
	if expiryMinutes <= 0 || expiryMinutes > 1440 {
		expiryMinutes = 15
	}

	url, err := h.service.PresignedURL(c.Context(), key, time.Duration(expiryMinutes)*time.Minute)
	if err != nil {
		return response.InternalError(c, fmt.Sprintf("failed to generate presigned URL: %v", err))
	}

	return response.OK(c, fiber.Map{
		"key":                key,
		"url":                url,
		"expires_in_minutes": expiryMinutes,
	})
}

// Delete removes an object from storage
// DELETE /api/v1/storage?key=...
func (h *Handler) Delete(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return response.BadRequest(c, "query parameter 'key' is required")
	}

	if err := h.service.Delete(c.Context(), key); err != nil {
		return response.InternalError(c, fmt.Sprintf("failed to delete file: %v", err))
	}

	return response.OK(c, fiber.Map{
		"message": "file deleted successfully",
		"key":     key,
	})
}

// Ping checks storage connectivity
// GET /api/v1/storage/ping
func (h *Handler) Ping(c fiber.Ctx) error {
	if err := h.service.Ping(c.Context()); err != nil {
		return response.InternalError(c, fmt.Sprintf("storage ping failed: %v", err))
	}

	return response.OK(c, fiber.Map{
		"status":  "connected",
		"service": "storage",
	})
}

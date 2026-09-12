package storage_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	modstorage "golang-base/internal/modules/storage"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(mock *mockProvider, maxUploadSize int64) *fiber.App {
	app := fiber.New()
	svc := modstorage.NewService(mock)
	handler := modstorage.NewHandler(svc, maxUploadSize)

	storageGroup := app.Group("/api/v1/storage")
	storageGroup.Get("/ping", handler.Ping)
	storageGroup.Post("/upload", handler.Upload)
	storageGroup.Get("/presigned", handler.GetPresignedURL)
	storageGroup.Delete("", handler.Delete)

	return app
}

func TestHandler_Ping(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 1024*1024)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_Upload_Success(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 1024*1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = io.WriteString(part, "test file content")
	require.NoError(t, err)

	_ = writer.WriteField("folder", "custom-folder")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var res map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	assert.NotNil(t, res["data"])
}

func TestHandler_Upload_MissingFile(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 1024*1024)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/upload", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandler_Upload_Oversized(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 10) // 10 bytes max

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "oversized.txt")
	require.NoError(t, err)
	_, err = io.WriteString(part, "this is definitely longer than 10 bytes")
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandler_GetPresignedURL(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 1024*1024)

	// Missing key
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/presigned", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Valid key
	req = httptest.NewRequest(http.MethodGet, "/api/v1/storage/presigned?key=uploads/test.txt&expiry_minutes=30", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_Delete(t *testing.T) {
	mock := &mockProvider{}
	app := setupTestApp(mock, 1024*1024)

	// Missing key
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Valid key
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/storage?key=uploads/test.txt", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, mock.deleted)
}

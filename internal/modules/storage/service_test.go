package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	modstorage "golang-base/internal/modules/storage"
	pkgstorage "golang-base/internal/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	uploadedKey string
	downloaded  bool
	deleted     bool
	pingErr     error
}

func (m *mockProvider) Upload(ctx context.Context, objectKey string, file io.Reader, contentType string) (*pkgstorage.UploadResult, error) {
	m.uploadedKey = objectKey
	data, _ := io.ReadAll(file)
	url := "http://example.com/" + objectKey
	return &pkgstorage.UploadResult{
		ObjectKey: objectKey,
		URL:       &url,
		Size:      int64(len(data)),
	}, nil
}

func (m *mockProvider) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	m.downloaded = true
	return io.NopCloser(bytes.NewReader([]byte("mock content"))), nil
}

func (m *mockProvider) Delete(ctx context.Context, objectKey string) error {
	m.deleted = true
	return nil
}

func (m *mockProvider) PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return "http://example.com/presigned/" + objectKey, nil
}

func (m *mockProvider) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestStorageService_Upload(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	content := []byte("hello world")
	res, err := svc.Upload(ctx, bytes.NewReader(content), "avatar.png", "image/png", "users/avatars")
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Contains(t, res.ObjectKey, "users/avatars/")
	assert.Contains(t, res.ObjectKey, "avatar.png")
	assert.Equal(t, int64(len(content)), res.Size)
}

func TestStorageService_Upload_DefaultFolder(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	content := []byte("default folder test")
	res, err := svc.Upload(ctx, bytes.NewReader(content), "doc.pdf", "", "")
	require.NoError(t, err)
	assert.Contains(t, res.ObjectKey, "uploads/")
	assert.Contains(t, res.ObjectKey, "doc.pdf")
}

func TestStorageService_Download(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	body, err := svc.Download(ctx, "test/file.txt")
	require.NoError(t, err)
	defer body.Close()

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, []byte("mock content"), data)
	assert.True(t, mock.downloaded)
}

func TestStorageService_Delete(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	err := svc.Delete(ctx, "test/file.txt")
	assert.NoError(t, err)
	assert.True(t, mock.deleted)
}

func TestStorageService_PresignedURL(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	url, err := svc.PresignedURL(ctx, "test/file.txt", 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/presigned/test/file.txt", url)
}

func TestStorageService_Ping(t *testing.T) {
	mock := &mockProvider{}
	svc := modstorage.NewService(mock)
	ctx := context.Background()

	err := svc.Ping(ctx)
	assert.NoError(t, err)

	mock.pingErr = os.ErrNotExist
	err = svc.Ping(ctx)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

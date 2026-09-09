package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	pkgstorage "golang-base/internal/pkg/storage"

	"github.com/google/uuid"
)

type Service interface {
	Upload(ctx context.Context, file io.Reader, originalFilename string, contentType string, folder string) (*pkgstorage.UploadResult, error)
	Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectKey string) error
	PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	Ping(ctx context.Context) error
}

type service struct {
	provider pkgstorage.Provider
}

func NewService(provider pkgstorage.Provider) Service {
	return &service{
		provider: provider,
	}
}

func (s *service) Upload(ctx context.Context, file io.Reader, originalFilename string, contentType string, folder string) (*pkgstorage.UploadResult, error) {
	ext := filepath.Ext(originalFilename)
	base := strings.TrimSuffix(filepath.Base(originalFilename), ext)
	base = strings.ReplaceAll(base, " ", "-")
	uid := uuid.New().String()

	folder = strings.Trim(folder, "/")
	var objectKey string
	if folder != "" {
		objectKey = fmt.Sprintf("%s/%s-%s%s", folder, uid, base, ext)
	} else {
		objectKey = fmt.Sprintf("uploads/%s/%s-%s%s", time.Now().Format("2006/01"), uid, base, ext)
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return s.provider.Upload(ctx, objectKey, file, contentType)
}

func (s *service) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return s.provider.Download(ctx, objectKey)
}

func (s *service) Delete(ctx context.Context, objectKey string) error {
	return s.provider.Delete(ctx, objectKey)
}

func (s *service) PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	return s.provider.PresignedURL(ctx, objectKey, expiry)
}

func (s *service) Ping(ctx context.Context) error {
	return s.provider.Ping(ctx)
}

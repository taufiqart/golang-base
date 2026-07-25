package storage

import (
	"context"
	"io"
	"time"
)

// UploadResult contains metadata about a completed upload.
type UploadResult struct {
	ObjectKey string
	URL       *string
	Size      int64
}

// Provider defines the interface for storage backends.
type Provider interface {
	Upload(ctx context.Context, objectKey string, file io.Reader, contentType string) (*UploadResult, error)
	Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectKey string) error
	PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
}

package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrStorageDisabled = errors.New("storage is disabled")

// DisabledProvider returns errors for all operations.
// Used when no storage backend is configured so the attachment module
// can still register routes (e.g., external link operations).
type DisabledProvider struct{}

func (p *DisabledProvider) Upload(_ context.Context, _ string, _ io.Reader, _ string) (*UploadResult, error) {
	return nil, ErrStorageDisabled
}

func (p *DisabledProvider) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, ErrStorageDisabled
}

func (p *DisabledProvider) Delete(_ context.Context, _ string) error {
	return ErrStorageDisabled
}

func (p *DisabledProvider) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", ErrStorageDisabled
}

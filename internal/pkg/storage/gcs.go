package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSProvider stores files in Google Cloud Storage.
type GCSProvider struct {
	client *storage.Client
	bucket string
}

func NewGCSProvider(cfg GCSConfig) (*GCSProvider, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("gcs: bucket is required")
	}

	ctx := context.Background()
	var opts []option.ClientOption
	if cfg.CredentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcs: create client: %w", err)
	}

	return &GCSProvider{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (p *GCSProvider) Upload(ctx context.Context, objectKey string, file io.Reader, contentType string) (*UploadResult, error) {
	obj := p.client.Bucket(p.bucket).Object(objectKey)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType

	size, err := io.Copy(writer, file)
	if err != nil {
		writer.Close() // nolint: errcheck
		return nil, fmt.Errorf("gcs upload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("gcs upload close: %w", err)
	}

	return &UploadResult{
		ObjectKey: objectKey,
		Size:      size,
	}, nil
}

func (p *GCSProvider) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	reader, err := p.client.Bucket(p.bucket).Object(objectKey).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs download: %w", err)
	}
	return reader, nil
}

func (p *GCSProvider) Delete(ctx context.Context, objectKey string) error {
	if err := p.client.Bucket(p.bucket).Object(objectKey).Delete(ctx); err != nil {
		return fmt.Errorf("gcs delete: %w", err)
	}
	return nil
}

func (p *GCSProvider) PresignedURL(ctx context.Context, _ string, expiry time.Duration) (string, error) {
	// GCS signed URLs are generated per-request; pass objectKey through caller.
	// Since we don't store the signing key in this provider, use environment-based
	// signing (GOOGLE_APPLICATION_CREDENTIALS or default service account).
	// For production, Pre-signed URL should be generated per-request in the handler.
	return "", fmt.Errorf("gcs presigned url not implemented: use Download endpoint instead")
}

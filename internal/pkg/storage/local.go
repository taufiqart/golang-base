package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrLocalProviderDisabled = errors.New("local storage provider is not configured")

type Config struct {
	Provider      string
	LocalBasePath string
	PublicBaseURL string
	S3            S3Config
	GCS           GCSConfig
}

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
}

type GCSConfig struct {
	Bucket          string
	CredentialsJSON string
}

type LocalProvider struct {
	basePath  string
	publicURL string
}

func NewLocalProvider(basePath, publicURL string) (*LocalProvider, error) {
	if basePath == "" {
		return nil, ErrLocalProviderDisabled
	}
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid local storage path: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("cannot create local storage directory: %w", err)
	}
	return &LocalProvider{basePath: abs, publicURL: publicURL}, nil
}

func (p *LocalProvider) Upload(ctx context.Context, objectKey string, file io.Reader, _ string) (*UploadResult, error) {
	fullPath, err := p.safePath(objectKey)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create directory: %w", err)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		return nil, fmt.Errorf("cannot write file: %w", err)
	}

	downloadURL := p.publicURL + "/" + objectKey
	return &UploadResult{
		ObjectKey: objectKey,
		URL:       &downloadURL,
		Size:      size,
	}, nil
}

func (p *LocalProvider) Download(_ context.Context, objectKey string) (io.ReadCloser, error) {
	fullPath, err := p.safePath(objectKey)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", objectKey)
		}
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	return f, nil
}

func (p *LocalProvider) Delete(_ context.Context, objectKey string) error {
	fullPath, err := p.safePath(objectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete file: %w", err)
	}
	return nil
}

func (p *LocalProvider) PresignedURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	if _, err := p.safePath(objectKey); err != nil {
		return "", err
	}
	return p.publicURL + "/" + objectKey, nil
}

func (p *LocalProvider) safePath(objectKey string) (string, error) {
	cleaned := filepath.Clean(objectKey)
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid object key: %s", objectKey)
	}
	fullPath := filepath.Join(p.basePath, cleaned)
	rel, err := filepath.Rel(p.basePath, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("object key escapes storage root: %s", objectKey)
	}
	return fullPath, nil
}

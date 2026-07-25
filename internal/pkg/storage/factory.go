package storage

import (
	"errors"
)

var ErrUnsupportedProvider = errors.New("unsupported storage provider")

// Storage provider types
const (
	ProviderLocal = "local"
	ProviderS3    = "s3"
	ProviderGCS   = "gcs"
)

// NewProvider creates a storage provider based on the given configuration.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderLocal:
		return NewLocalProvider(cfg.LocalBasePath, cfg.PublicBaseURL)
	case ProviderS3:
		return NewS3Provider(cfg.S3)
	case ProviderGCS:
		return NewGCSProvider(cfg.GCS)
	case "":
		// Default to local with sensible dev defaults
		return NewLocalProvider("./storage/app/public", "/storage")
	default:
		return nil, ErrUnsupportedProvider
	}
}

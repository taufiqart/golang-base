package storage

import (
	"errors"

	"golang-base/config"
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

// NewProviderFromAppConfig creates a Provider using the application config.Config.
func NewProviderFromAppConfig(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return NewProvider(Config{})
	}
	return NewProvider(Config{
		Provider:      cfg.StorageProvider,
		LocalBasePath: cfg.StorageLocalBasePath,
		PublicBaseURL: cfg.StoragePublicBaseURL,
		S3: S3Config{
			Bucket:          cfg.S3Bucket,
			Region:          cfg.S3Region,
			Endpoint:        cfg.S3Endpoint,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			ForcePathStyle:  cfg.S3ForcePathStyle,
		},
		GCS: GCSConfig{
			Bucket:          cfg.GCSBucket,
			CredentialsJSON: cfg.GCSCredentialsJSON,
		},
	})
}

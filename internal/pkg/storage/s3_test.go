package storage_test

import (
	"context"
	"testing"
	"time"

	"golang-base/internal/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3Provider_MissingBucket(t *testing.T) {
	cfg := storage.S3Config{
		Bucket: "",
	}

	provider, err := storage.NewS3Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "bucket is required")
}

func TestNewS3Provider_ValidConfig(t *testing.T) {
	cfg := storage.S3Config{
		Bucket:          "my-test-bucket",
		Region:          "ap-southeast-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		ForcePathStyle:  true,
	}

	provider, err := storage.NewS3Provider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Verify implements storage.Provider interface
	var _ storage.Provider = provider
}

func TestNewProvider_S3Factory(t *testing.T) {
	cfg := storage.Config{
		Provider: storage.ProviderS3,
		S3: storage.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		},
	}

	p, err := storage.NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.IsType(t, &storage.S3Provider{}, p)
}

func TestNewProvider_Unsupported(t *testing.T) {
	cfg := storage.Config{
		Provider: "azure_blob",
	}

	p, err := storage.NewProvider(cfg)
	assert.ErrorIs(t, err, storage.ErrUnsupportedProvider)
	assert.Nil(t, p)
}

func TestDisabledProvider(t *testing.T) {
	p := &storage.DisabledProvider{}
	ctx := context.Background()

	_, err := p.Upload(ctx, "test.txt", nil, "text/plain")
	assert.ErrorIs(t, err, storage.ErrStorageDisabled)

	_, err = p.Download(ctx, "test.txt")
	assert.ErrorIs(t, err, storage.ErrStorageDisabled)

	err = p.Delete(ctx, "test.txt")
	assert.ErrorIs(t, err, storage.ErrStorageDisabled)

	_, err = p.PresignedURL(ctx, "test.txt", time.Minute)
	assert.ErrorIs(t, err, storage.ErrStorageDisabled)

	err = p.Ping(ctx)
	assert.ErrorIs(t, err, storage.ErrStorageDisabled)
}

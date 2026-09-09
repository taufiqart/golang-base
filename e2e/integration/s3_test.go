package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"golang-base/internal/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestS3Provider(t *testing.T) *storage.S3Provider {
	t.Helper()

	if TestConfig == nil {
		t.Skip("TestConfig not initialized")
	}

	bucket := TestConfig.S3Bucket
	if bucket == "" {
		bucket = os.Getenv("S3_BUCKET")
	}
	if bucket == "" || bucket == "your-bucket" {
		t.Skip("S3_BUCKET not configured, skipping S3 integration tests")
	}

	accessKey := TestConfig.S3AccessKeyID
	if accessKey == "" {
		accessKey = os.Getenv("S3_ACCESS_KEY_ID")
		if accessKey == "" {
			accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		}
	}
	secretKey := TestConfig.S3SecretAccessKey
	if secretKey == "" {
		secretKey = os.Getenv("S3_SECRET_ACCESS_KEY")
		if secretKey == "" {
			secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}
	}

	if accessKey == "" || secretKey == "" {
		t.Skip("S3 credentials not configured, skipping S3 integration tests")
	}

	cfg := storage.S3Config{
		Bucket:          bucket,
		Region:          TestConfig.S3Region,
		Endpoint:        TestConfig.S3Endpoint,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		ForcePathStyle:  TestConfig.S3ForcePathStyle,
	}

	p, err := storage.NewS3Provider(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	return p
}

func TestS3_E2E_Ping(t *testing.T) {
	p := getTestS3Provider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := p.Ping(ctx)
	assert.NoError(t, err, "S3 bucket ping/head should succeed")
}

func TestS3_E2E_UploadDownloadDelete(t *testing.T) {
	p := getTestS3Provider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testKey := fmt.Sprintf("e2e-tests/test-%d.txt", time.Now().UnixNano())
	testContent := []byte("Hello from Golang-Base S3 E2E Test!")

	// 1. Upload
	res, err := p.Upload(ctx, testKey, bytes.NewReader(testContent), "text/plain")
	require.NoError(t, err, "upload should succeed")
	assert.Equal(t, testKey, res.ObjectKey)
	assert.Equal(t, int64(len(testContent)), res.Size)

	// 2. Download
	body, err := p.Download(ctx, testKey)
	require.NoError(t, err, "download should succeed")
	defer body.Close()

	downloaded, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloaded)

	// 3. Presigned URL
	presignedURL, err := p.PresignedURL(ctx, testKey, 15*time.Minute)
	assert.NoError(t, err, "generating presigned URL should succeed")
	assert.NotEmpty(t, presignedURL)

	// 4. Delete
	err = p.Delete(ctx, testKey)
	assert.NoError(t, err, "delete should succeed")
}

func TestS3_E2E_HealthEndpoint(t *testing.T) {
	_ = getTestS3Provider(t)

	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/health",
	})
	AssertOK(t, resp)
	assert.Equal(t, "connected", resp.Body["storage"])
}

func TestS3_E2E_APIRoutes(t *testing.T) {
	_ = getTestS3Provider(t)
	seedAdminUser(t)
	token := AdminToken(t)

	// 1. Storage Ping API
	resp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   "/api/v1/storage/ping",
	})
	AssertOK(t, resp)

	// 2. Multipart file upload
	bodyBuf := &bytes.Buffer{}
	mpWriter := multipart.NewWriter(bodyBuf)
	part, err := mpWriter.CreateFormFile("file", "api-test.txt")
	require.NoError(t, err)
	_, err = io.WriteString(part, "file content uploaded via api")
	require.NoError(t, err)
	_ = mpWriter.WriteField("folder", "e2e-api-tests")
	mpWriter.Close()

	httpReq, err := http.NewRequest(http.MethodPost, "/api/v1/storage/upload", bodyBuf)
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", mpWriter.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+token)

	rawResp, err := TestApp.Test(httpReq, 10000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rawResp.StatusCode)

	var uploadResp map[string]interface{}
	err = json.NewDecoder(rawResp.Body).Decode(&uploadResp)
	require.NoError(t, err)

	dataMap, ok := uploadResp["data"].(map[string]interface{})
	require.True(t, ok)
	objectKey, ok := dataMap["object_key"].(string)
	require.True(t, ok)
	assert.Contains(t, objectKey, "e2e-api-tests/")

	// 3. Get Presigned URL
	presignResp := Do(t, TestApp, Request{
		Method: "GET",
		Path:   fmt.Sprintf("/api/v1/storage/presigned?key=%s&expiry_minutes=30", url.QueryEscape(objectKey)),
		Token:  token,
	})
	AssertOK(t, presignResp)
	presignData := AssertDataEnvelope(t, presignResp)
	assert.NotEmpty(t, presignData["url"])

	// 4. Delete file
	delResp := Do(t, TestApp, Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/api/v1/storage?key=%s", url.QueryEscape(objectKey)),
		Token:  token,
	})
	AssertOK(t, delResp)
}

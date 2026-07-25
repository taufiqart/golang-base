package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang-base/internal/database"
)

type Request struct {
	Method string
	Path   string
	Token  string
	Body   any
}

type Response struct {
	Status  int
	Headers http.Header
	Body    map[string]interface{}
	Raw     []byte
}

func Do(t *testing.T, app *fiber.App, req Request) *Response {
	t.Helper()

	var bodyReader io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		require.NoError(t, err, "marshal body")
		bodyReader = bytes.NewReader(jsonBody)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	require.NoError(t, err, "create http request")
	httpReq.Header.Set("Content-Type", "application/json")
	if req.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	}

	resp, err := app.Test(httpReq, 10000) // 10s timeout for migrations
	require.NoError(t, err, "app.Test")

	var result map[string]interface{}
	raw := make([]byte, 0)
	if resp.Body != nil {
		raw, _ = io.ReadAll(resp.Body)
		json.Unmarshal(raw, &result)
	}

	return &Response{
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Body:    result,
		Raw:     raw,
	}
}

func AssertOK(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 200, resp.Status, "expected 200 OK")
}

func AssertCreated(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 201, resp.Status, "expected 201 Created")
}

func AssertNoContent(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 204, resp.Status, "expected 204 No Content")
}

func AssertBadRequest(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 400, resp.Status, "expected 400 Bad Request")
}

func AssertUnprocessable(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 422, resp.Status, "expected 422 Unprocessable Entity")
}

func AssertUnauthorized(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 401, resp.Status, "expected 401 Unauthorized")
}

func AssertForbidden(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 403, resp.Status, "expected 403 Forbidden")
}

func AssertNotFound(t *testing.T, resp *Response) {
	t.Helper()
	assert.Equal(t, 404, resp.Status, "expected 404 Not Found")
}

func AssertJSON(t *testing.T, resp *Response) {
	t.Helper()
	assert.Contains(t, resp.Headers.Get("Content-Type"), "application/json", "response should be JSON")
}

func AssertDataEnvelope(t *testing.T, resp *Response) map[string]interface{} {
	t.Helper()
	require.NotNil(t, resp.Body, "response body is nil")
	data, ok := resp.Body["data"]
	if !ok {
		t.Logf("no 'data' envelope, keys: %v", mapKeys(resp.Body))
	}
	require.True(t, ok, "response should have 'data' envelope")
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		t.Logf("'data' is %T, not object", data)
	}
	require.True(t, ok, "'data' should be an object")
	return dataMap
}

func AssertDataList(t *testing.T, resp *Response) []interface{} {
	t.Helper()
	require.NotNil(t, resp.Body, "response body is nil")
	data, ok := resp.Body["data"]
	require.True(t, ok, "response should have 'data' envelope")
	arr, ok := data.([]interface{})
	assert.True(t, ok, "'data' should be an array")
	return arr
}

func AssertMessageEnvelope(t *testing.T, resp *Response) {
	t.Helper()
	require.NotNil(t, resp.Body, "response body is nil")
	_, ok := resp.Body["message"]
	if !ok {
		t.Logf("no 'message' envelope, keys: %v", mapKeys(resp.Body))
	}
	assert.True(t, ok, "error response should have 'message' envelope")
}

func AssertPaginationMeta(t *testing.T, resp *Response) map[string]interface{} {
	t.Helper()
	require.NotNil(t, resp.Body)
	meta, ok := resp.Body["meta"].(map[string]interface{})
	require.True(t, ok, "response should have 'meta' object")
	for _, field := range []string{"total_items", "total_pages", "current_page", "limit"} {
		_, ok := meta[field]
		assert.True(t, ok, "meta should have '%s'", field)
	}
	return meta
}

func TruncateTables(t *testing.T, tables ...string) {
	t.Helper()
	ctx := context.Background()
	for _, tbl := range tables {
		_, err := database.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tbl))
		require.NoError(t, err, "truncate %s", tbl)
	}
}

func AdminToken(t *testing.T) string {
	t.Helper()

	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body: map[string]any{
			"email":    "admin@test.com",
			"password": "Test@1234",
		},
	})
	AssertOK(t, resp)
	data := AssertDataEnvelope(t, resp)
	token, ok := data["access_token"].(string)
	require.True(t, ok, "access_token missing")
	return token
}

func newUUID(t *testing.T) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	return id.String()
}

func loginAs(t *testing.T, email, password string) string {
	t.Helper()
	resp := Do(t, TestApp, Request{
		Method: "POST",
		Path:   "/api/v1/auth/login",
		Body:   map[string]any{"email": email, "password": password},
	})
	if resp.Status != 200 {
		return ""
	}
	data := AssertDataEnvelope(t, resp)
	token, _ := data["access_token"].(string)
	return token
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

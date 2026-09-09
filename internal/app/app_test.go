package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang-base/config"
	"golang-base/internal/app"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Health_ServiceNameFromConfig(t *testing.T) {
	cfg := &config.Config{
		AppService: "my-custom-service",
	}

	fiberApp := app.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "my-custom-service", body["service"])
}

func TestApp_Health_DefaultServiceName(t *testing.T) {
	fiberApp := app.New(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "golang-base", body["service"])
}

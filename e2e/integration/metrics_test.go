package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MetricsToken is the bearer token TestMain configures for the scrape endpoint.
const MetricsToken = "e2e-metrics-token"

// generateTraffic drives at least one non-scrape request, because a labelled
// collector publishes nothing at all until it has one observation.
func generateTraffic(t *testing.T) {
	t.Helper()

	resp := Do(t, TestApp, Request{Method: http.MethodGet, Path: "/health"})
	require.Equal(t, http.StatusOK, resp.Status)
}

func scrape(t *testing.T, token string) *Response {
	t.Helper()
	return Do(t, TestApp, Request{Method: http.MethodGet, Path: "/metrics", Token: token})
}

func TestMetricsEndpointRejectsAnonymousScrape(t *testing.T) {
	resp := scrape(t, "")
	assert.Equal(t, http.StatusUnauthorized, resp.Status, "anonymous scrapes must not read metrics")
	assert.NotContains(t, string(resp.Raw), "http_server_requests_total")
}

func TestMetricsEndpointRejectsWrongToken(t *testing.T) {
	resp := scrape(t, "not-the-real-token")
	assert.Equal(t, http.StatusUnauthorized, resp.Status)
	assert.NotContains(t, string(resp.Raw), "app_info")
}

func TestMetricsEndpointServesExpositionWithToken(t *testing.T) {
	generateTraffic(t)

	resp := scrape(t, MetricsToken)
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Raw))

	body := string(resp.Raw)
	assert.Contains(t, resp.Headers.Get("Content-Type"), "text/plain")

	for _, want := range []string{
		"app_info{",
		"http_server_requests_total",
		"http_server_request_duration_seconds_bucket",
		"go_goroutines",
		"process_cpu_seconds_total",
	} {
		assert.Contains(t, body, want, "exposition must include %q", want)
	}
}

// TestMetricsReportsDatabasePool covers the pool wiring that only exists once a
// real *sql.DB is registered, which unit tests cannot exercise.
func TestMetricsReportsDatabasePool(t *testing.T) {
	generateTraffic(t)

	resp := scrape(t, MetricsToken)
	require.Equal(t, http.StatusOK, resp.Status)

	body := string(resp.Raw)
	for _, want := range []string{"go_sql_max_open_connections", "go_sql_in_use_connections"} {
		assert.Contains(t, body, want, "database pool metrics must be published")
	}
}

// TestMetricsUsesRouteTemplatesAndHidesRawPaths guards the label cardinality
// contract against the live router: path parameters must collapse onto the
// registered template, and a random 404 path must never reach a label.
func TestMetricsUsesRouteTemplatesAndHidesRawPaths(t *testing.T) {
	TruncateTables(t, "users")
	seedAdminUser(t)
	token := AdminToken(t)

	for _, id := range []string{"00000000-0000-7000-8000-000000000001", "00000000-0000-7000-8000-000000000002"} {
		Do(t, TestApp, Request{Method: http.MethodGet, Path: "/api/v1/users/" + id, Token: token})
	}

	const canary = "c4rd1n4l-7est-9f8e7d6c"
	missing := Do(t, TestApp, Request{Method: http.MethodGet, Path: "/api/v1/nope/" + canary})
	require.Equal(t, http.StatusNotFound, missing.Status)

	body := string(scrape(t, MetricsToken).Raw)
	assert.Contains(t, body, `route="/api/v1/users/:id"`,
		"path parameters must be templated, not labelled per id")
	assert.NotContains(t, body, canary, "an unmatched URL must never become a metric label")
}

// TestResponsesCarryRequestID checks the correlation header the whole logging
// and tracing chain depends on.
func TestResponsesCarryRequestID(t *testing.T) {
	resp := Do(t, TestApp, Request{Method: http.MethodGet, Path: "/health"})
	require.Equal(t, http.StatusOK, resp.Status)

	id := resp.Headers.Get("X-Request-ID")
	assert.NotEmpty(t, id, "every response must carry a correlation id")
	assert.Len(t, id, 36, "a generated id is a UUID")
	assert.Equal(t, byte('7'), id[14], "generated ids must be UUIDv7")

	inbound := Do(t, TestApp, Request{
		Method:  http.MethodGet,
		Path:    "/health",
		Headers: map[string]string{"X-Request-ID": "e2e-inbound-id"},
	})
	assert.Equal(t, "e2e-inbound-id", inbound.Headers.Get("X-Request-ID"),
		"a caller-supplied id must be honoured so traces can be joined across services")
}

func TestMetricsPathIsExcludedFromRequestCounters(t *testing.T) {
	generateTraffic(t)

	before := counterSeries(t)
	require.Positive(t, before, "the test needs an existing series to be meaningful")

	require.Equal(t, http.StatusOK, scrape(t, MetricsToken).Status)
	require.Equal(t, http.StatusOK, scrape(t, MetricsToken).Status)

	assert.Equal(t, before, counterSeries(t),
		"scrape traffic must not feed the request-rate metric it is measured by")
}

// counterSeries counts the label sets present on http_server_requests_total.
func counterSeries(t *testing.T) int {
	t.Helper()

	body := string(scrape(t, MetricsToken).Raw)
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "http_server_requests_total{") {
			count++
		}
	}
	return count
}

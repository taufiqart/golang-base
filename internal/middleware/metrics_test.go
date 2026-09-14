package middleware

import (
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/metrics"
)

// seriesByRoute returns the route label values recorded for one metric family.
func metricLabelValues(t *testing.T, m *metrics.Metrics, family, label string) []string {
	t.Helper()

	families, err := m.Registry.Gather()
	require.NoError(t, err)

	var values []string
	for _, f := range families {
		if f.GetName() != family {
			continue
		}
		for _, metric := range f.GetMetric() {
			for _, lp := range metric.GetLabel() {
				if lp.GetName() == label {
					values = append(values, lp.GetValue())
				}
			}
		}
	}

	return values
}

func newMetricsApp(t *testing.T) (*fiber.App, *metrics.Metrics) {
	t.Helper()

	m := metrics.New("test-service", "test-version")

	app := fiber.New()
	app.Use(Metrics(m))
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/users/:id", func(c fiber.Ctx) error { return c.SendString("user") })

	return app, m
}

func TestMetricsMiddlewareRecordsRouteTemplate(t *testing.T) {
	app, m := newMetricsApp(t)

	for _, id := range []string{"1", "2", "3"} {
		res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/users/"+id, nil), fiber.TestConfig{Timeout: 10000})
		require.NoError(t, err)
		assert.NoError(t, res.Body.Close())
	}

	// Prometheus keeps one series per label set, so three requests must collapse
	// into a single /users/:id series whose counter reads three.
	routes := metricLabelValues(t, m, "http_server_requests_total", "route")
	assert.Equal(t, []string{"/users/:id"}, routes,
		"path parameters must collapse into the registered route template")
	assert.Equal(t, []string{"3"}, counterValues(t, m, "http_server_requests_total", "route", "/users/:id"))
}

func counterValues(t *testing.T, m *metrics.Metrics, family, routeLabel, route string) []string {
	t.Helper()

	families, err := m.Registry.Gather()
	require.NoError(t, err)

	var values []string
	for _, f := range families {
		if f.GetName() != family {
			continue
		}
		for _, metric := range f.GetMetric() {
			matches := false
			for _, lp := range metric.GetLabel() {
				if lp.GetName() == routeLabel && lp.GetValue() == route {
					matches = true
				}
			}
			if matches {
				values = append(values, strconv.FormatFloat(metric.GetCounter().GetValue(), 'f', 0, 64))
			}
		}
	}

	return values
}

func TestMetricsMiddlewareDoesNotLeakUnmatchedPaths(t *testing.T) {
	app, m := newMetricsApp(t)

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/probe/a1b2c3d4e5", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	routes := metricLabelValues(t, m, "http_server_requests_total", "route")
	assert.Equal(t, []string{metrics.UnmatchedRoute}, routes,
		"unknown paths must share one label value instead of creating a series each")

	// The raw URL is the thing that must never reach a label.
	for _, series := range routes {
		assert.NotContains(t, series, "a1b2c3d4e5")
	}
}

func TestMetricsMiddlewareBalancesInFlight(t *testing.T) {
	inFlight := make(chan struct{})
	release := make(chan struct{})

	m := metrics.New("test-service", "test-version")

	app := fiber.New()
	app.Use(Metrics(m))
	app.Get("/slow", func(c fiber.Ctx) error {
		close(inFlight)
		<-release
		return c.SendString("done")
	})

	go func() {
		res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/slow", nil), fiber.TestConfig{Timeout: 10000})
		if err == nil {
			_ = res.Body.Close()
		}
	}()

	<-inFlight
	assertGauge(t, m, "http_server_requests_in_flight", 1)

	close(release)

	// Give the middleware time to decrement after the handler resumes.
	require.Eventually(t, func() bool {
		value, ok := gaugeValue(m, "http_server_requests_in_flight")
		return ok && value == 0
	}, 5*time.Second, 10*time.Millisecond, "in-flight gauge must return to zero once the request finishes")
}

func assertGauge(t *testing.T, m *metrics.Metrics, family string, want float64) {
	t.Helper()

	got, ok := gaugeValue(m, family)
	require.True(t, ok, "missing gauge family %s", family)
	assert.Equal(t, want, got)
}

func gaugeValue(m *metrics.Metrics, family string) (float64, bool) {
	families, err := m.Registry.Gather()
	if err != nil {
		return 0, false
	}

	for _, f := range families {
		if f.GetName() != family || len(f.GetMetric()) == 0 {
			continue
		}
		return f.GetMetric()[0].GetGauge().GetValue(), true
	}

	return 0, false
}

func TestMetricsMiddlewareExcludesScrapePath(t *testing.T) {
	m := metrics.New("test-service", "test-version")

	app := fiber.New()
	app.Use(Metrics(m))
	app.Get(MetricsPath, func(c fiber.Ctx) error { return c.SendString("scrape") })

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, MetricsPath, nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	assert.Empty(t, metricLabelValues(t, m, "http_server_requests_total", "route"),
		"scrape traffic must not feed the request-rate metric")
}

func TestMetricsMiddlewareCountsHandlerErrors(t *testing.T) {
	m := metrics.New("test-service", "test-version")

	app := fiber.New()
	app.Use(Metrics(m))
	app.Get("/boom", func(c fiber.Ctx) error { return fiber.NewError(fiber.StatusBadGateway, "upstream down") })

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/boom", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	assert.Equal(t, []string{"502"}, metricLabelValues(t, m, "http_server_requests_total", "status"),
		"the status the client receives must be the status recorded")
}

func TestMetricsAuth(t *testing.T) {
	tests := []struct {
		name        string
		configured  string
		header      string
		wantStatus  int
		wantReached bool
	}{
		{name: "unconfigured endpoint is not reachable", configured: "", wantStatus: fiber.StatusNotFound},
		{name: "missing header rejected", configured: "sekrit", wantStatus: fiber.StatusUnauthorized},
		{name: "wrong scheme rejected", configured: "sekrit", header: "Basic sekrit", wantStatus: fiber.StatusUnauthorized},
		{name: "wrong token rejected", configured: "sekrit", header: "Bearer nope", wantStatus: fiber.StatusUnauthorized},
		{name: "correct token accepted", configured: "sekrit", header: "Bearer sekrit", wantStatus: fiber.StatusOK, wantReached: true},
		{name: "scheme is case insensitive", configured: "sekrit", header: "bearer sekrit", wantStatus: fiber.StatusOK, wantReached: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false

			app := fiber.New()
			app.Get(MetricsPath, MetricsAuth(tc.configured), func(c fiber.Ctx) error {
				reached = true
				return c.SendString("exposed")
			})

			req := httptest.NewRequest(fiber.MethodGet, MetricsPath, nil)
			if tc.header != "" {
				req.Header.Set(fiber.HeaderAuthorization, tc.header)
			}

			res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
			require.NoError(t, err)
			assert.NoError(t, res.Body.Close())

			assert.Equal(t, tc.wantStatus, res.StatusCode)
			assert.Equal(t, tc.wantReached, reached)
		})
	}
}

func TestMetricsAuthDoesNotServeBodyOnFailure(t *testing.T) {
	m := metrics.New("svc", "v")

	app := fiber.New()
	app.Get(MetricsPath, MetricsAuth("sekrit"), m.Handler())

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, MetricsPath, nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "app_info", "an unauthorized scrape must not expose metrics")
}

func TestMetricsEndpointServesExpositionWithToken(t *testing.T) {
	m := metrics.New("svc", "v")
	m.ObserveRequest("/ok", "GET", 200, 0.01)

	app := fiber.New()
	app.Get(MetricsPath, MetricsAuth("sekrit"), m.Handler())

	req := httptest.NewRequest(fiber.MethodGet, MetricsPath, nil)
	req.Header.Set(fiber.HeaderAuthorization, "Bearer sekrit")

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get(fiber.HeaderContentType), "text/plain")

	// One Read is not enough: the exposition is far larger than the buffer and
	// http_server_* sorts after the go_* families.
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	exposition := string(raw)
	assert.Contains(t, exposition, "http_server_requests_total")
	assert.Contains(t, exposition, `app_info{service="svc",version="v"} 1`)
}

// TestMetricsSurvivesHandlerPanic pins the two guarantees that only hold because
// the in-flight decrement and the observation are deferred: a panicking handler
// must not leave the gauge elevated, and the 500 it becomes must still show up
// in RED.
func TestMetricsSurvivesHandlerPanic(t *testing.T) {
	m := metrics.New("test-service", "test-version")

	app := fiber.New()
	app.Use(recover.New())
	app.Use(Metrics(m))
	app.Get("/panic", func(c fiber.Ctx) error { panic("boom") })
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("ok") })

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/panic", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	res, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/ok", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	assertGauge(t, m, "http_server_requests_in_flight", 0)
	assert.ElementsMatch(t, []string{"200", "500"},
		metricLabelValues(t, m, "http_server_requests_total", "status"))
	assert.ElementsMatch(t, []string{"/ok", "/panic"},
		metricLabelValues(t, m, "http_server_requests_total", "route"))
	assert.Equal(t, []string{"1"}, counterValues(t, m, "http_server_requests_total", "route", "/panic"),
		"the panic must be counted once as a server error")
}

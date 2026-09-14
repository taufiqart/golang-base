package metrics

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func familyNames(t *testing.T, m *Metrics) map[string]*dto.MetricFamily {
	t.Helper()

	families, err := m.Registry.Gather()
	require.NoError(t, err)

	out := make(map[string]*dto.MetricFamily, len(families))
	for _, f := range families {
		out[f.GetName()] = f
	}

	return out
}

func TestNewRegistersExpectedCollectors(t *testing.T) {
	m := New("svc-test", "v1.2.3")

	names := familyNames(t, m)
	for _, want := range []string{
		"app_info",
		"http_server_requests_in_flight",
		"go_goroutines",
		"process_cpu_seconds_total",
	} {
		assert.Contains(t, names, want, "registry must expose %q", want)
	}

	info := names["app_info"].GetMetric()[0]
	labels := map[string]string{}
	for _, lp := range info.GetLabel() {
		labels[lp.GetName()] = lp.GetValue()
	}
	assert.Equal(t, map[string]string{"service": "svc-test", "version": "v1.2.3"}, labels)
	assert.Equal(t, float64(1), info.GetGauge().GetValue())
}

func TestRegistryIsIsolatedPerInstance(t *testing.T) {
	first := New("a", "1")
	second := New("b", "2")

	assert.NotSame(t, first.Registry, second.Registry)

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, f := range families {
		assert.NotContains(t, []string{"app_info", "http_server_requests_total"}, f.GetName(),
			"package-level collectors must not register into the default registry")
	}
}

func TestObserveRequestPopulatesCountersAndHistogram(t *testing.T) {
	m := New("svc", "v")

	m.ObserveRequest("/api/v1/users/:id", "GET", 200, 0.012)
	m.ObserveRequest("/api/v1/users/:id", "GET", 200, 0.4)
	m.ObserveRequest("/api/v1/users/:id", "GET", 500, 1.5)

	names := familyNames(t, m)

	total := names["http_server_requests_total"]
	require.NotNil(t, total)
	counts := map[string]float64{}
	for _, metric := range total.GetMetric() {
		labels := map[string]string{}
		for _, lp := range metric.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		counts[labels["status"]] = metric.GetCounter().GetValue()
		assert.Equal(t, "/api/v1/users/:id", labels["route"])
	}
	assert.Equal(t, map[string]float64{"200": 2, "500": 1}, counts)

	hist := names["http_server_request_duration_seconds"]
	require.NotNil(t, hist)
	require.Len(t, hist.GetMetric(), 1, "one series per route+method pair")
	assert.Equal(t, uint64(3), hist.GetMetric()[0].GetHistogram().GetSampleCount())
}

func TestStatusCodeLabel(t *testing.T) {
	assert.Equal(t, "200", statusCodeLabel(200))
	assert.Equal(t, "404", statusCodeLabel(404))
	assert.Equal(t, "invalid", statusCodeLabel(0))
	assert.Equal(t, "invalid", statusCodeLabel(600))
	assert.Equal(t, "invalid", statusCodeLabel(-1))
}

func TestRegisterDBNilIsNoop(t *testing.T) {
	m := New("svc", "v")

	assert.NotPanics(t, func() { m.RegisterDB(nil, "db") })

	_, present := familyNames(t, m)["sql_requests_in_flight_total"]
	assert.False(t, present, "a nil pool must not register collectors")
}

// stubDriver backs a *sql.DB whose pool exists but is never exercised, which is
// enough to assert the standard go_sql_* families get registered.
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) { return stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(string) (driver.Stmt, error) { return nil, errStub }
func (stubConn) Close() error                        { return nil }
func (stubConn) Begin() (driver.Tx, error)           { return nil, errStub }

var errStub = errors.New("stub driver: not implemented")

func TestRegisterDBAddsPoolCollectors(t *testing.T) {
	sql.Register("metrics_test_stub", stubDriver{})

	db, err := sql.Open("metrics_test_stub", "")
	require.NoError(t, err)
	defer db.Close()

	m := New("svc", "v")
	m.RegisterDB(db, "primary")

	names := familyNames(t, m)
	for _, want := range []string{
		"go_sql_in_use_connections",
		"go_sql_idle_connections",
		"go_sql_wait_count_total",
	} {
		assert.Contains(t, names, want)
	}
}

func TestRegisterRedisNilIsNoop(t *testing.T) {
	m := New("svc", "v")
	m.RegisterRedis(nil)

	for name := range familyNames(t, m) {
		assert.NotContains(t, name, "redis_pool_")
	}
}

func TestRegisterRedisReportsPoolStats(t *testing.T) {
	m := New("svc", "v")
	m.RegisterRedis(func() *redis.PoolStats {
		return &redis.PoolStats{
			Hits:            7,
			Misses:          2,
			TotalConns:      4,
			IdleConns:       3,
			PendingRequests: 1,
			WaitDurationNs:  1_500_000_000,
		}
	})

	names := familyNames(t, m)

	gauge := func(name string) float64 {
		f, ok := names[name]
		require.True(t, ok, "missing %s", name)
		return f.GetMetric()[0].GetGauge().GetValue()
	}

	assert.Equal(t, float64(7), gauge("redis_pool_hits_total"))
	assert.Equal(t, float64(2), gauge("redis_pool_misses_total"))
	assert.Equal(t, float64(4), gauge("redis_pool_connections"))
	assert.Equal(t, float64(3), gauge("redis_pool_idle_connections"))
	assert.Equal(t, float64(1), gauge("redis_pool_pending_requests"))
	assert.InDelta(t, 1.5, gauge("redis_pool_wait_duration_seconds_total"), 1e-9)
}

func TestRegisterRedisToleratesDisabledClient(t *testing.T) {
	m := New("svc", "v")
	m.RegisterRedis(func() *redis.PoolStats { return nil })

	names := familyNames(t, m)
	f, ok := names["redis_pool_hits_total"]
	require.True(t, ok, "collector should stay registered when Redis is off")
	assert.Equal(t, float64(0), f.GetMetric()[0].GetGauge().GetValue())
}

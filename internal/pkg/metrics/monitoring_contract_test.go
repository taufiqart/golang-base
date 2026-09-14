package metrics

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metricRef matches an identifier that is used as a metric name in PromQL, i.e.
// one followed by a label selector, a range selector or a comparison, rather
// than a bare function argument.
var metricRef = regexp.MustCompile(`\b((?:http_server|redis_pool|go_sql|go_memstats|go_gc|go_|process_|app_info|up)[a-zA-Z0-9_]*)`)

// publishedMetricFamilies is the set of metric names the service really exports,
// gathered once from a fully wired registry.
func publishedMetricNames(t *testing.T) map[string]bool {
	t.Helper()

	m := New("contract-test", "v")
	m.ObserveRequest("/ok", "GET", 200, 0.05)
	m.RegisterRedis(func() *redis.PoolStats { return &redis.PoolStats{} })

	// The pool collectors only appear once a pool is registered, so the stub
	// driver from metrics_test.go stands in for a real database here.
	db, err := sql.Open("metrics_test_stub", "")
	require.NoError(t, err)
	defer db.Close()
	m.RegisterDB(db, "contract")

	rec := httptest.NewRecorder()
	promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	require.Equal(t, 200, rec.Code)

	names := map[string]bool{}
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if strings.HasPrefix(line, "# TYPE ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				names[fields[2]] = true
			}
		}
	}

	// Histogram families are published as _bucket/_sum/_count rather than as the
	// bare name, so accept the base form as well.
	for name := range names {
		for _, suffix := range []string{"_bucket", "_sum", "_count"} {
			if strings.HasSuffix(name, suffix) {
				names[strings.TrimSuffix(name, suffix)] = true
			}
		}
	}

	// go_sql_wait_count_total and friends keep the _total suffix, but queries in
	// the dashboards may use the increase() form of the same name.
	return names
}

// counterAndGaugeNames also accepts the "_total" spelling used by PromQL queries
// against a counter family.
func isPublished(names map[string]bool, candidate string) bool {
	if strings.HasSuffix(candidate, "_count") || strings.HasSuffix(candidate, "_sum") || strings.HasSuffix(candidate, "_bucket") {
		candidate = trimSuffixAny(candidate)
	}
	if names[candidate] {
		return true
	}
	if trimmed := strings.TrimSuffix(candidate, "_total"); trimmed != candidate {
		return names[trimmed] || names[trimmed+"_count"]
	}
	return false
}

func trimSuffixAny(name string) string {
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod not found above %s", dir)
		dir = parent
	}
}

func TestDashboardQueriesPublishRealMetrics(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "monitoring", "grafana", "dashboards", "golang-service.json")

	raw, err := os.ReadFile(path)
	require.NoError(t, err, "dashboard file is missing")

	var dashboard struct {
		Panels []struct {
			Title   string `json:"title"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	}
	require.NoError(t, json.Unmarshal(raw, &dashboard))

	names := publishedMetricNames(t)
	require.NotEmpty(t, names, "the registry must expose something to check against")

	checked := 0
	for _, panel := range dashboard.Panels {
		for _, target := range panel.Targets {
			for _, match := range metricRef.FindAllStringSubmatch(target.Expr, -1) {
				candidate := match[1]
				if candidate == "up" {
					continue
				}
				checked++
				assert.True(t, isPublished(names, candidate),
					"panel %q queries %q, which the service never publishes (panel would read 'No data')",
					panel.Title, candidate)
			}
		}
	}

	assert.Greater(t, checked, 10, "the dashboard must actually be checked")
}

func TestAlertRulesReferenceRealMetrics(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "monitoring", "prometheus", "alert.rules.yml"))
	require.NoError(t, err, "alert rules file is missing")

	names := publishedMetricNames(t)

	checked := 0
	for _, match := range metricRef.FindAllStringSubmatch(string(raw), -1) {
		candidate := match[1]
		if candidate == "up" {
			continue
		}
		checked++
		assert.True(t, isPublished(names, candidate),
			"an alert rule references %q, which the service never publishes", candidate)
	}

	assert.Greater(t, checked, 5, "the alert rules must actually be checked")
}

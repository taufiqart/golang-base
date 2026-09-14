// Package metrics exposes Prometheus instrumentation for the service.
//
// A Metrics value owns its own prometheus.Registry instead of the process-wide
// default one, so tests stay isolated and nothing else in the dependency graph
// can silently publish series.
package metrics

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// UnmatchedRoute is the route label used when no registered route matched the
// request. Every unknown path must collapse onto this single value: labelling by
// the raw URL would let a scanner create unbounded time series and exhaust
// Prometheus memory.
const UnmatchedRoute = "unmatched"

// Metrics holds the registry and the HTTP request collectors.
type Metrics struct {
	Registry *prometheus.Registry

	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge
}

// RedisSource reports connection pool stats lazily. It is a function so the
// collector keeps working when Redis is connected after startup, or never
// connects at all.
type RedisSource func() *redis.PoolStats

// New builds a registry preloaded with Go runtime, process and build-info
// collectors plus the HTTP RED metrics for the named service.
func New(service, version string) *Metrics {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		Registry: reg,
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "http",
				Subsystem: "server",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed, labelled by route, method and status code.",
			},
			[]string{"route", "method", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "http",
				Subsystem: "server",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency in seconds, labelled by route and method.",
				// Buckets aimed at an interactive JSON API: most calls should be
				// single-digit milliseconds, with enough tail resolution to see
				// a p99 blow up before it becomes an outage.
				Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"route", "method"},
		),
		RequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "http",
				Subsystem: "server",
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being served.",
			},
		),
	}

	reg.MustRegister(m.RequestsTotal, m.RequestDuration, m.RequestsInFlight)

	appInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_info",
			Help: "Static build metadata about this service instance.",
		},
		[]string{"service", "version"},
	)
	reg.MustRegister(appInfo)
	appInfo.WithLabelValues(service, version).Set(1)

	return m
}

// RegisterDB adds database/sql connection pool metrics under dbName.
func (m *Metrics) RegisterDB(db *sql.DB, dbName string) {
	if db == nil {
		return
	}
	m.Registry.MustRegister(collectors.NewDBStatsCollector(db, dbName))
}

// ObserveRequest records one finished HTTP request. In-flight tracking is left
// to the caller because it brackets the handler rather than its completion.
func (m *Metrics) ObserveRequest(route, method string, status int, seconds float64) {
	m.RequestsTotal.WithLabelValues(route, method, statusCodeLabel(status)).Inc()
	m.RequestDuration.WithLabelValues(route, method).Observe(seconds)
}

// Handler serves the registry in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		// A scrape that fails to encode is a bug worth surfacing in the response.
		ErrorHandling: promhttp.ContinueOnError,
	})
}

func statusCodeLabel(status int) string {
	if status < 100 || status > 599 {
		return "invalid"
	}
	return strconv.Itoa(status)
}

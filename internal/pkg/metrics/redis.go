package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// RegisterRedis adds go-redis connection pool metrics.
//
// Each gauge reads the pool lazily at scrape time, so no background poller runs
// and a Redis client that connects (or drops) later is still reported
// accurately. A nil source, or one that returns nil because Redis is disabled,
// leaves the endpoint serving the remaining metrics.
func (m *Metrics) RegisterRedis(source RedisSource) {
	if source == nil {
		return
	}

	gauge := func(name, help string, value func(*redis.PoolStats) float64) {
		m.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Namespace: "redis", Subsystem: "pool", Name: name, Help: help},
			func() float64 {
				stats := source()
				if stats == nil {
					return 0
				}
				return value(stats)
			},
		))
	}

	counter := func(name, help string, value func(*redis.PoolStats) uint32) {
		gauge(name, help, func(s *redis.PoolStats) float64 { return float64(value(s)) })
	}

	counter("hits_total", "Connections reused from the idle pool.", func(s *redis.PoolStats) uint32 { return s.Hits })
	counter("misses_total", "Requests that had to open a new connection.", func(s *redis.PoolStats) uint32 { return s.Misses })
	counter("timeouts_total", "Requests that timed out waiting for a connection.", func(s *redis.PoolStats) uint32 { return s.Timeouts })
	counter("wait_count_total", "Requests that waited for a connection to free up.", func(s *redis.PoolStats) uint32 { return s.WaitCount })
	counter("unusable_total", "Connections found unusable when checked out.", func(s *redis.PoolStats) uint32 { return s.Unusable })
	counter("stale_total", "Stale connections removed from the pool.", func(s *redis.PoolStats) uint32 { return s.StaleConns })
	counter("pending_requests", "Requests waiting for a connection right now.", func(s *redis.PoolStats) uint32 { return s.PendingRequests })
	counter("connections", "Connections currently held by the pool.", func(s *redis.PoolStats) uint32 { return s.TotalConns })
	counter("idle_connections", "Connections currently idle in the pool.", func(s *redis.PoolStats) uint32 { return s.IdleConns })

	gauge("wait_duration_seconds_total", "Total time spent waiting for a pool connection.", func(s *redis.PoolStats) float64 {
		return float64(s.WaitDurationNs) / 1e9
	})
}

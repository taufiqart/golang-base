# Monitoring

Local observability stack for this service: **Prometheus** (metrics), **Grafana**
(dashboards), **Jaeger** (traces). Everything here is opt-in — deleting this
folder does not affect the application, which runs happily without a collector.

## What the service exposes

| Endpoint     | Purpose                          | Guard                                   |
| ------------ | -------------------------------- | --------------------------------------- |
| `/health`    | Liveness + dependency status     | none                                    |
| `/metrics`   | Prometheus exposition            | `Authorization: Bearer $METRICS_TOKEN`  |

`/metrics` is **not mounted at all** while `METRICS_TOKEN` is empty. The
exposition reveals internal route templates and traffic volume, so there is no
"open by default" path.

Published families:

- `http_server_requests_total{route,method,status}` — request counter
- `http_server_request_duration_seconds{route,method}` — latency histogram
- `http_server_requests_in_flight` — concurrency gauge
- `go_sql_*` — database connection pool
- `redis_pool_*` — Redis connection pool (zero when Redis is disabled)
- `go_*`, `process_*` — Go runtime and process
- `app_info{service,version}` — build identity

`route` is the **registered path template** (`/api/v1/users/:id`), never the
concrete URL, and unmatched requests collapse onto a single `unmatched` value.
That is deliberate: labelling by raw URL lets any scanner mint unbounded time
series. A test enforces it —
`go test ./internal/pkg/metrics/ -run DashboardQueries`.

## Bring it up

```bash
# 1. Choose a token and start the API with it
export METRICS_TOKEN=dev-metrics-token
make run

# 2. Start the stack (from the repo root)
docker compose -f monitoring/docker-compose.yml up -d
```

| UI          | URL                        | Credentials |
| ----------- | -------------------------- | ----------- |
| Grafana     | http://localhost:3000      | admin/admin |
| Prometheus  | http://localhost:9090      | —           |
| Jaeger      | http://localhost:16686     | —           |

Grafana provisions the Prometheus datasource and imports
`Golang Base — Service Overview` automatically. The scrape job targets
`host.docker.internal:3100`, so the same setup works whether the API runs under
`docker-compose.dev.yml` or as a host process.

If the dashboards are empty, the token is the usual culprit:

```bash
curl -s -H "Authorization: Bearer $METRICS_TOKEN" localhost:3100/metrics | head
```

## Tracing

Disabled by default. To turn it on, start Jaeger (already in the compose file)
and point the service at it:

```bash
export TRACING_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_INSECURE=true
make run
```

Behaviour:

- One server span per HTTP request, named `<METHOD> <route template>`.
- Inbound `traceparent` is honoured, so this service joins a caller's trace;
  sampling decisions propagate rather than reset.
- Database queries are spanned through Bun's query hook. **Bound query values
  are never recorded** — enabling them would ship row data into the trace store.
- `trace_id` and `span_id` are added to every log line for that request, which is
  how you jump from a log to a trace and back. `request_id` continues to work
  unchanged when tracing is off.

## Logs

JSON lines, level-routed to separate files under `logs/` (git-ignored):

```
logs/app.log     everything at or above LOG_LEVEL
logs/debug.log   DEBUG only
logs/info.log    INFO  only
logs/warn.log    WARN  only
logs/error.log   ERROR only
```

stdout receives the same stream as `app.log` for container log drivers. Set
`LOG_PRETTY=true` for readable lines during development. Join a request's log
lines with `jq 'select(.request_id=="<id>")' logs/app.log`.

## Alerts

`prometheus/alert.rules.yml` ships a blunt starting set: service down, 5xx rate,
p95 latency, pool saturation, pool waiting, Redis timeouts, goroutine leak,
memory pressure. Load is on; **the thresholds are guesses** — retune them per
service once you know its normal shape. No Alertmanager is wired in, so add one
when alerts need routing.

## Regenerating the dashboard

The dashboard JSON is generated, not hand-edited:

```bash
python3 monitoring/build_dashboard.py
```

Edit `PANELS` in that script and re-run. `go test ./internal/pkg/metrics/` then
fails if a query references a metric the service does not publish, which is what
keeps panels from silently turning into "No data".

## Monitoring di server terpisah

Stack monitoring (Prometheus, Grafana, Jaeger) bisa jalan di mesin yang sama sekali berbeda dari API. Yang perlu dikonfigurasi:

**Server API (A):**

```bash
# di server A — aplikasi backend
export TRUSTED_PROXIES=10.0.0.0/8   # atau IP jujur reverse-proxy nginx/traefik-nya
export OTEL_EXPORTER_OTLP_ENDPOINT=http://<IP_SERVER_B>:4318
export OTEL_EXPORTER_OTLP_INSECURE=true
export TRACING_ENABLED=true
export METRICS_TOKEN=<token-anda>
make run
```

Pastikan firewall server A menerima port `3100/tcp` dari IP server B, dan `4318` tidak perlu terbuka dari sisi A (outbound saja).

**Server monitoring (B):**

```bash
# di server B — prometheus/grafana/jaeger
export METRICS_TOKEN=<token-yang-sama>
export SCRAPE_TARGET=<IP_SERVER_A>:3100   # default: host.docker.internal:3100
export SCRAPE_SCHEME=http                  # https jika server A di belakang TLS
docker compose -f monitoring/docker-compose.yml up -d
```

| Variabel | Default | Fungsi |
| --- | --- | --- |
| `SCRAPE_TARGET` | `host.docker.internal:3100` | host:port yang diprometheus |
| `SCRAPE_SCHEME` | `http` | `https` jika app di belakang TLS/nginx |
| `METRICS_TOKEN` | (required) | harus sama dengan yang di server A |

**Perhatian keamanan:**

- Jangan expose port `3100` ke internet — `/metrics` memperlihatkan route internal dan traffic pattern.
- Kalau server A dan B di jaringan berbeda, gunakan `SCRAPE_SCHEME=https` + TLS + `METRICS_TOKEN` dengan token kuat.
- Port OTLP `4318` hanya perlu terbuka dari server A ke server B, bukan sebaliknya, dan jangan ke publik.

**Verifikasi dari server B:**

```bash
curl -s http://localhost:9090/api/v1/query?query=up | python3 -c \
  "import sys,json; [print(f'{r["metric"]["job"]}: {r["value"][1]}') for r in json.load(sys.stdin)['data']['result']]"
```

harus menampilkan `golang-base: 1`.

## Notes and limits

- Retention is 15 days in-container; this stack is for development, not a


- Retention is 15 days in-container; this stack is for development, not a
  production metrics history.
- Grafana credentials default to `admin/admin`. Change `GRAFANA_USER` and
  `GRAFANA_PASSWORD` before exposing port 3000 anywhere reachable.
- Jaeger's all-in-one image keeps spans in memory and loses them on restart.
- There is no graceful shutdown handler yet, so spans still buffered when the
  process is killed may be lost. Add `signal.Notify` + `app.Shutdown` if losing
  the tail of a trace matters.

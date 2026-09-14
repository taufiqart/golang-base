#!/usr/bin/env python3
"""Generate monitoring/grafana/dashboards/golang-service.json.

Written as a generator rather than checked-in JSON so the layout stays
consistent and cannot drift into invalid JSON. Re-run after editing PANELS.
"""
import json
import os

DS = {"type": "prometheus", "uid": "prometheus"}
WIDTH = 24

PANELS = [
    {
        "title": "Request rate",
        "type": "timeseries",
        "unit": "reqps",
        "w": 8,
        "targets": [
            ("A", 'sum(rate(http_server_requests_total[$__rate_interval]))', "total"),
            (
                "B",
                'sum by (route) (rate(http_server_requests_total[$__rate_interval]))',
                "{{route}}",
            ),
        ],
    },
    {
        "title": "5xx error ratio",
        "type": "stat",
        "unit": "percentunit",
        "w": 4,
        "thresholds": [(0.01, "green"), (0.05, "yellow"), (0.05, "red")],
        "targets": [
            (
                "A",
                'sum(rate(http_server_requests_total{status=~"5.."}[$__rate_interval]))'
                " / "
                "clamp_min(sum(rate(http_server_requests_total[$__rate_interval])), 0.001)",
                "errors",
            )
        ],
    },
    {
        "title": "4xx error ratio",
        "type": "stat",
        "unit": "percentunit",
        "w": 4,
        "thresholds": [(0.05, "green"), (0.20, "yellow"), (0.20, "red")],
        "targets": [
            (
                "A",
                'sum(rate(http_server_requests_total{status=~"4.."}[$__rate_interval]))'
                " / "
                "clamp_min(sum(rate(http_server_requests_total[$__rate_interval])), 0.001)",
                "client errors",
            )
        ],
    },
    {
        "title": "Requests in flight",
        "type": "stat",
        "unit": "short",
        "w": 4,
        "targets": [("A", "sum(http_server_requests_in_flight)", "in flight")],
    },
    {
        "title": "Process uptime",
        "type": "stat",
        "unit": "s",
        "w": 4,
        "targets": [("A", "time() - process_start_time_seconds", "uptime")],
    },
    {
        "title": "Latency by route (p95)",
        "type": "timeseries",
        "unit": "s",
        "w": 12,
        "targets": [
            (
                "A",
                'histogram_quantile(0.95, sum by (le, route) '
                '(rate(http_server_request_duration_seconds_bucket[$__rate_interval])))',
                "{{route}}",
            )
        ],
    },
    {
        "title": "Latency quantiles (all routes)",
        "type": "timeseries",
        "unit": "s",
        "w": 12,
        "targets": [
            (
                "A",
                'histogram_quantile(0.50, sum by (le) '
                '(rate(http_server_request_duration_seconds_bucket[$__rate_interval])))',
                "p50",
            ),
            (
                "B",
                'histogram_quantile(0.95, sum by (le) '
                '(rate(http_server_request_duration_seconds_bucket[$__rate_interval])))',
                "p95",
            ),
            (
                "C",
                'histogram_quantile(0.99, sum by (le) '
                '(rate(http_server_request_duration_seconds_bucket[$__rate_interval])))',
                "p99",
            ),
        ],
    },
    {
        "title": "Database pool",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "sum(go_sql_in_use_connections)", "in use"),
            ("B", "sum(go_sql_idle_connections)", "idle"),
            ("C", "sum(go_sql_max_open_connections)", "max open"),
            ("D", "sum(go_sql_open_connections)", "opened"),
        ],
    },
    {
        "title": "Database wait pressure",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "sum(increase(go_sql_wait_count_total[$__rate_interval]))", "waits"),
            (
                "B",
                "sum(increase(go_sql_wait_duration_seconds_total[$__rate_interval]))",
                "wait seconds",
            ),
        ],
    },
    {
        "title": "Database connection churn",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "sum(go_sql_open_connections)", "opened"),
            ("B", "sum(increase(go_sql_max_lifetime_closed_total[$__rate_interval]))", "closed: lifetime"),
            ("C", "sum(increase(go_sql_max_idle_closed_total[$__rate_interval]))", "closed: idle"),
            ("D", "sum(increase(go_sql_max_idle_time_closed_total[$__rate_interval]))", "closed: max idle time"),
        ],
    },
    {
        "title": "Redis pool reuse",
        "type": "timeseries",
        "unit": "ops",
        "w": 8,
        "targets": [
            ("A", "sum(rate(redis_pool_hits_total[$__rate_interval]))", "hits"),
            ("B", "sum(rate(redis_pool_misses_total[$__rate_interval]))", "misses"),
        ],
    },
    {
        "title": "Redis pool pressure",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "sum(redis_pool_connections)", "total"),
            ("B", "sum(redis_pool_idle_connections)", "idle"),
            ("C", "sum(redis_pool_pending_requests)", "pending"),
            ("D", "sum(increase(redis_pool_timeouts_total[$__rate_interval]))", "timeouts"),
        ],
    },
    {
        "title": "HTTP traffic vs cache reads",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "sum(rate(http_server_requests_total[$__rate_interval]))", "requests/s"),
            ("B", "sum(rate(redis_pool_hits_total[$__rate_interval]))", "redis hits/s"),
        ],
    },
    {
        "title": "Go heap",
        "type": "timeseries",
        "unit": "bytes",
        "w": 8,
        "targets": [
            ("A", "go_memstats_heap_alloc_bytes", "allocated"),
            ("B", "go_memstats_heap_inuse_bytes", "in use"),
            ("C", "go_memstats_heap_sys_bytes", "from OS"),
            ("D", "go_memstats_next_gc_bytes", "next GC target"),
        ],
    },
    {
        "title": "Goroutines & GC",
        "type": "timeseries",
        "unit": "short",
        "w": 8,
        "targets": [
            ("A", "go_goroutines", "goroutines"),
            ("B", "rate(go_gc_duration_seconds_count[$__rate_interval])", "GC/s"),
            ("C", "go_threads", "OS threads"),
        ],
    },
]


def thresholds(spec):
    steps = [{"color": "green", "value": None}]
    for value, color in spec:
        steps.append({"color": color, "value": value})
    return {"mode": "absolute", "steps": steps}


def build_panel(index, spec):
    field_config = {
        "defaults": {
            "unit": spec["unit"],
            "custom": {
                "drawStyle": "line",
                "lineWidth": 2,
                "fillOpacity": 12 if spec["type"] == "timeseries" else 0,
                "showPoints": "never",
                "spanNulls": True,
            },
            "min": 0,
        },
        "overrides": [],
    }
    if spec["type"] == "stat":
        field_config["defaults"]["color"] = {"mode": "thresholds"}
        if "thresholds" in spec:
            field_config["defaults"]["thresholds"] = thresholds(spec["thresholds"])

    return {
        "id": index,
        "type": spec["type"],
        "title": spec["title"],
        "datasource": DS,
        "gridPos": {"h": 8, "w": spec["w"], "x": 0, "y": 0},
        "fieldConfig": field_config,
        "options": {
            "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
            "reduceOptions": {"values": False, "calcs": ["lastNotNull"], "fields": ""},
        },
        "targets": [
            {
                "refId": ref_id,
                "datasource": DS,
                "expr": expr,
                "legendFormat": legend,
                "interval": "",
                "editorMode": "code",
                "range": True,
            }
            for ref_id, expr, legend in spec["targets"]
        ],
    }


def layout(panels):
    """Pack panels left to right, starting a new row when the width is full."""
    x, y, row_height = 0, 0, 8
    for panel in panels:
        width = panel["gridPos"]["w"]
        if x + width > WIDTH:
            x, y = 0, y + row_height
        panel["gridPos"]["x"], panel["gridPos"]["y"] = x, y
        x += width
    return panels


def main():
    built = layout([build_panel(i + 1, spec) for i, spec in enumerate(PANELS)])

    dashboard = {
        "uid": "golang-base-service",
        "title": "Golang Base — Service Overview",
        "description": "RED metrics, database/Redis pool saturation and Go runtime health for golang-base.",
        "tags": ["golang-base", "generated"],
        "timezone": "browser",
        "schemaVersion": 39,
        "version": 1,
        "editable": True,
        "refresh": "15s",
        "graphTooltip": 1,
        "time": {"from": "now-1h", "to": "now"},
        "timepicker": {"refresh_intervals": ["15s", "30s", "1m", "5m"]},
        "fiscalYearStartMonth": 0,
        "liveNow": False,
        "style": "dark",
        "panels": built,
        "annotations": {
            "list": [
                {
                    "builtIn": 1,
                    "datasource": {"type": "grafana", "uid": "-- Grafana --"},
                    "enable": True,
                    "hide": True,
                    "iconColor": "rgba(0, 211, 255, 1)",
                    "name": "Annotations & Alerts",
                    "type": "dashboard",
                }
            ]
        },
        "templating": {"list": []},
        "links": [],
    }

    here = os.path.dirname(os.path.abspath(__file__))
    target = os.path.join(here, "grafana", "dashboards", "golang-service.json")
    with open(target, "w", encoding="utf-8") as fh:
        json.dump(dashboard, fh, indent=2)
        fh.write("\n")

    print(f"wrote {target} ({len(built)} panels)")


if __name__ == "__main__":
    main()

# Monitoring — Prometheus metrics

> Khớp với `core/metrics` + `cmd/app/main.go` + `deploy/monitoring/` — cập nhật khi code đổi.

## Stack monitoring (docker compose)

```
prometheus   → scrape app:9090 (metrics) — UI http://localhost:9091 (+ alert rules rules.yml)
grafana      → dashboard App Overview — http://localhost:3000 (admin/admin dev)
loki         → log aggregation (filesystem storage) — :3100
alloy        → GỘP 2 vai trò: log container (docker.sock → loki) + OTLP receiver (4317/4318 → jaeger)
jaeger       → UI traces — http://localhost:16686
```

> **Alloy** (grafana/alloy — Apache 2.0, kế thừa promtail + là OTel Collector distribution)
> thay cả promtail lẫn otel-collector. PITFALL: loki.source.docker KHÔNG giữ `__meta_*`
> label → phải `discovery.relabel` map `__meta_docker_container_name → container`
> (xem deploy/monitoring/alloy/config.alloy). Log cũ ngoài window Loki → 400 reject cả
> batch — filter chỉ lấy container backend.

## Tracing (OpenTelemetry → Jaeger)

App tạo **span per request** qua middleware gin (`core/otelx`): route name, method,
status, request_id, user_id; 5xx → status error. Attributes chuẩn semconv.

- Mặc định **TẮT** (zero-config): `OTEL_ENABLED=false` → noop tracer (chi phí ~0)
- Bật: `OTEL_ENABLED=true` + `OTEL_ENDPOINT=localhost:4317`
  (compose đã bật sẵn: `OTEL_ENDPOINT=alloy:4317` → Jaeger)
- Xem trace: **http://localhost:16686** → Service: `tiktok-live-platform`
- Propagation w3c `traceparent` — gọi từ client có header sẽ nối vào trace có sẵn
- Shutdown flush batch (graceful) — không mất span cuối

```bash
docker compose up -d            # cả stack
docker compose up -d prometheus grafana loki alloy jaeger  # chỉ monitoring
```

Config nằm trong `deploy/monitoring/<service>/` — Grafana tự provisioning datasource
(Prometheus + Loki) + dashboard "App Overview" (RPS, latency p95, memory, service_up,
maintenance, 5xx). Forward ports: `FORWARD_PROMETHEUS_PORT` (9091), `FORWARD_GRAFANA_PORT`
(3000), `FORWARD_LOKI_PORT` (3100), `FORWARD_OTEL_GRPC_PORT` (4317), `FORWARD_OTEL_HTTP_PORT`
(4318), `FORWARD_JAEGER_UI_PORT` (16686) — override trong root `.env`.

> Lưu ý: trong compose, app metrics bind `0.0.0.0` (backend:9090) để Prometheus scrape
> qua network — chạy ngoài container vẫn `127.0.0.1:9090`.

## Quyết định kiến trúc: PORT RIÊNG (không gắn vào app)

Metrics serve ở **listener riêng** — mặc định `127.0.0.1:9090/metrics`:

| Lý do | Gắn vào app port | Port riêng ✅ |
|---|---|---|
| Scrape qua middleware (rate-limit, maintenance, auth, recovery) | bị chặn/fail khi app gồng | không bị đụng |
| App port thường public (tunnel/nginx) | metrics lộ ra ngoài | bind localhost an toàn |
| Log nhiễu (scrape 15s/lần) | lẫn request log | tách sạch |
| Lifecycle | phụ thuộc app middleware | độc lập, luôn scrape được |
| Chuẩn Go ecosystem | — | giống exporter/pprof |

## Endpoint

```
GET http://127.0.0.1:9090/metrics   # Prometheus text format (go_* + process_* + custom)
```

### Metric custom (core/metrics)

| Metric | Type | Labels | Ý nghĩa |
|---|---|---|---|
| `http_requests_total` | Counter | method, path (route), status | số request — **kể cả bị rate-limit/maintenance chặn** |
| `http_request_duration_seconds` | Histogram | method, path | latency (buckets 5ms→10s) |
| `http_requests_in_flight` | Gauge | method | request đang xử lý (detect hang) |

> `path` = **route** (vd `/admin/users/:id`) không phải id cụ thể — tránh cardinality explosion.
> Metric chưa từng xuất hiện label → không hiện trong scrape (Prometheus behavior — bình thường).

## Cấu hình (optional)

| Key / env | Default | Ý nghĩa |
|---|---|---|
| `metrics.enabled` / `METRICS_ENABLED` | `true` | bật listener metrics |
| `metrics.host` / `METRICS_HOST` | `127.0.0.1` | chỉ local — đổi `0.0.0.0` khi scrape từ xa |
| `metrics.port` / `METRICS_PORT` | `9090` | port riêng |
| `metrics.path` / `METRICS_PATH` | `/metrics` | path |
| `metrics.auth_token` / `METRICS_AUTH_TOKEN` | `""` (không auth) | set → yêu cầu `Authorization: Bearer <token>` |

## Auth optional (khi expose ra ngoài)

```bash
METRICS_AUTH_TOKEN=my-secret ./gvs
curl -H "Authorization: Bearer my-secret" http://host:9090/metrics
```

Prometheus scrape:

```yaml
scrape_configs:
  - job_name: app
    metrics_path: /metrics
    static_configs:
      - targets: ["app-host:9090"]
    # bearer_token: my-secret   # bật khi METRICS_AUTH_TOKEN set
```

Không có token → `401 unauthorized`; token sai → 401; đúng → 200.

## Verify nhanh

```bash
curl -s localhost:9090/metrics | grep -E "^http_requests_total|^go_goroutines"
curl -s localhost:9090/metrics | grep -E "^app_maintenance_mode|^service_up"
```

## Tài liệu metrics đầy đủ

Danh sách mọi key + type + labels + mô tả + alert/grafana gợi ý:
**[`docs/metrics.md`](metrics.md)** — Metrics Reference.

## Graceful shutdown

Metrics server shutdown cùng app (trong `shutdown()` trước `srv.Shutdown`) — port được
giải phóng sạch khi SIGTERM.

# Metrics Reference — danh sách key Prometheus

> Endpoint: `GET http://127.0.0.1:9090/metrics` (port riêng — xem `docs/monitoring.md`).
> Tất cả metric đều là **Prometheus text format**, không prefix (dùng trực tiếp trong
> Prometheus/Grafana). Ký hiệu: 🟢 tự có sẵn · 🔵 custom của app.

## 🔵 Custom (core/metrics)

| Metric | Type | Labels | Mô tả |
|---|---|---|---|
| `http_requests_total` | Counter | `method`, `path`, `status` | Tổng HTTP request. **`path` = route** (`/admin/users/:id`) — tránh cardinality explosion. Đếm cả request bị rate-limit/maintenance chặn (middleware đặt trước). |
| `http_request_duration_seconds` | Histogram | `method`, `path` | Latency request (buckets 5ms → 10s). Dùng cho p50/p95/p99. |
| `http_requests_in_flight` | Gauge | `method` | Request đang xử lý — phát hiện hang/leak (tăng mãi không giảm). |
| `app_maintenance_mode` | Gauge | — | **1 = app đang bảo trì** (mọi API 503 — trừ GET /config cho healthcheck), 0 = bình thường. Cập nhật NGAY khi bật/tắt qua `admin PUT /config {app.maintenance_mode}`. |
| `service_up` | Gauge | `service` | **Health của service phụ thuộc**: 1 = up, 0 = down. Ping định kỳ **30s** (chạy ngay lúc start). |

### `service_up` — các giá trị `service`

| service | Ping bằng gì | Xuất hiện khi |
|---|---|---|
| `postgres` | `pool.Ping` (master) | luôn (DB bắt buộc) |
| `redis` | `rdb.Ping` | redis sẵn sàng (zero-config không có redis → không xuất) |
| `minio` | `Storage.Health` (BucketExists) | MinIO cấu hình OK (không → không xuất) |
| `mail` | `Mailer.Health` (SMTP dial+close) | SMTP cấu hình OK (không → không xuất) |

> Service không được cấu hình → **không xuất** metric (tránh alert nhiễu).
> Ping thất bại cũng log `WARN health check thất bại` vào app log.

## 🟢 Tự có sẵn (client_golang collectors)

### `go_*` — runtime Go
| Metric | Mô tả |
|---|---|
| `go_goroutines` | số goroutine hiện tại |
| `go_gc_duration_seconds` | thời gian pause GC (summary quantile) |
| `go_memstats_alloc_bytes` / `go_memstats_heap_inuse_bytes` | heap đang dùng |
| `go_memstats_sys_bytes` | bộ nhớ hệ thống cấp |
| `go_info` | version Go |
| `go_threads`, `go_sched_gomaxprocs_threads`, `go_gc_gogc_percent`, `go_gc_gomemlimit_bytes` | runtime config/threads |

### `process_*` — tiến trình OS
| Metric | Mô tả |
|---|---|
| `process_resident_memory_bytes` | **RSS — bộ nhớ thật của app** (quan trọng nhất) |
| `process_virtual_memory_bytes` | virtual memory |
| `process_cpu_seconds_total` | CPU user+system tích luỹ |
| `process_open_fds` / `process_max_fds` | fd đang mở / giới hạn (rò fd khi tiến gần max) |
| `process_start_time_seconds` | thời điểm start (uptime) |
| `process_network_receive_bytes_total` / `process_network_transmit_bytes_total` | bytes mạng vào/ra |

### `promhttp_*` — sức khoẻ của chính endpoint /metrics
| Metric | Mô tả |
|---|---|
| `promhttp_metric_handler_requests_total{code}` | số lần scrape theo status (200/500/503) — tăng 500 = handler lỗi |
| `promhttp_metric_handler_requests_in_flight` | scrape đang chạy |

## ⚠️ Alert rules gợi ý (Prometheus)

```yaml
groups:
  - name: app
    rules:
      # App đang bảo trì — cần biết (chủ động bật hay do nhầm?)
      - alert: AppMaintenanceMode
        expr: app_maintenance_mode == 1
        for: 1m
        labels: { severity: warning }
      # Service phụ thuộc chết
      - alert: DependentServiceDown
        expr: service_up == 0
        for: 2m          # tránh alert nhiễu khi ping transient
        labels: { severity: critical }
      # 5xx tăng đột biến (≥ 5% request trong 5 phút)
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
            / sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        labels: { severity: critical }
      # p95 latency > 500ms trong 10 phút
      - alert: SlowLatency
        expr: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) > 0.5
        for: 10m
        labels: { severity: warning }
      # Rò goroutine / request hang
      - alert: InFlightGrowing
        expr: increase(http_requests_in_flight[10m]) > 0
        for: 5m
        labels: { severity: warning }
```

## Grafana gợi ý

| Panel | Query |
|---|---|
| RPS | `sum(rate(http_requests_total[1m])) by (status)` |
| Latency p95 | `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` |
| Memory | `process_resident_memory_bytes` |
| Service health | `service_up` |
| Maintenance badge | `app_maintenance_mode` (threshold: 1 → đỏ) |

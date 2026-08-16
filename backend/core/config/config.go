package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

// Config — toàn bộ cấu hình STATIC (không đổi runtime — DB, JWT, mail...).
// Dynamic config (đổi runtime qua Redis pub/sub) truy cập qua Manager.GetDynamic/SetDynamic.
type Config struct {
	App      AppConfig      `mapstructure:"app" koanf:"app"`
	Server   ServerConfig   `mapstructure:"server" koanf:"server"`
	Database DatabaseConfig `mapstructure:"database" koanf:"database"`
	JWT      JWTConfig      `mapstructure:"jwt" koanf:"jwt"`
	Redis    RedisConfig    `mapstructure:"redis" koanf:"redis"`
	Cache    CacheConfig    `mapstructure:"cache" koanf:"cache"`
	Storage  StorageConfig  `mapstructure:"storage" koanf:"storage"`
	Mail     MailConfig     `mapstructure:"mail" koanf:"mail"`
	Log      LogConfig      `mapstructure:"log" koanf:"log"`
	Metrics  MetricsConfig  `mapstructure:"metrics" koanf:"metrics"`
	OTel     OTelConfig     `mapstructure:"otel" koanf:"otel"`
	OpenAPI  OpenAPIConfig  `mapstructure:"openapi" koanf:"openapi"`
	Live     LiveConfig     `mapstructure:"live" koanf:"live"`
}

// LiveConfig — TikTok live tracker + Sockudo (Pusher-compatible WS server).
type LiveConfig struct {
	// ConnectionMode — "long_poll" (mặc định) hoặc "websocket" (fallback long-poll).
	ConnectionMode string `mapstructure:"connection_mode" koanf:"connection_mode"`
	// PollIntervalMs — interval long-poll (ms, mặc định 3000).
	PollIntervalMs int `mapstructure:"poll_interval_ms" koanf:"poll_interval_ms"`

	// Sockudo — WS server nhận events realtime (Go server là publisher).
	SockudoURL      string `mapstructure:"sockudo_url" koanf:"sockudo_url"`
	SockudoAppID    string `mapstructure:"sockudo_app_id" koanf:"sockudo_app_id"`
	SockudoAppKey   string `mapstructure:"sockudo_app_key" koanf:"sockudo_app_key"`
	SockudoAppSecret string `mapstructure:"sockudo_app_secret" koanf:"sockudo_app_secret"`
}

// OpenAPIConfig — spec OpenAPI tự động (GET /api/v1/openapi.json).
// Mặc định BẬT; tắt ở production nếu không muốn lộ bề mặt API: OPENAPI_ENABLED=false
// (hoặc config.yml openapi.enabled: false) — không cần sửa code.
type OpenAPIConfig struct {
	Enabled bool `mapstructure:"enabled" koanf:"enabled"`
}

// OTelConfig — OpenTelemetry tracing (OTLP gRPC → collector → Jaeger).
// Mặc định TẮT (zero-config); bật khi có collector: OTEL_ENABLED=true + OTEL_ENDPOINT.
type OTelConfig struct {
	Enabled  bool   `mapstructure:"enabled" koanf:"enabled"`
	Endpoint string `mapstructure:"endpoint" koanf:"endpoint"` // vd localhost:4317, otel-collector:4317
}

// MetricsConfig — Prometheus metrics (PORT RIÊNG, mặc định 127.0.0.1:9090).
// Auth optional: AuthToken set → yêu cầu `Authorization: Bearer <token>`.
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled" koanf:"enabled"`       // bật listener metrics
	Host      string `mapstructure:"host" koanf:"host"`             // mặc định 127.0.0.1 (chỉ local)
	Port      string `mapstructure:"port" koanf:"port"`             // mặc định 9090
	Path      string `mapstructure:"path" koanf:"path"`             // mặc định /metrics
	AuthToken string `mapstructure:"auth_token" koanf:"auth_token"` // "" = không auth; set = Bearer token
}

type AppConfig struct {
	Env   string `mapstructure:"env" koanf:"env"`
	Name  string `mapstructure:"name" koanf:"name"`
	Title string `mapstructure:"title" koanf:"title"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" koanf:"host"` // mặc định 127.0.0.1 (chỉ local)
	Port string `mapstructure:"port" koanf:"port"` // mặc định 3300 — override qua --port / SERVER_PORT

	// RateLimitStore — "" = theo cache store (memory|redis); "memory"/"redis" ép cụ thể.
	RateLimitStore string `mapstructure:"rate_limit_store" koanf:"rate_limit_store"`
}

type DatabaseConfig struct {
	URL      string `mapstructure:"url" koanf:"url"`
	Replicas string `mapstructure:"replicas" koanf:"replicas"` // comma-separated, tuỳ chọn (read-only)
}

// ReplicaURLs — tách chuỗi DATABASE_REPLICAS "url1,url2" thành slice (trim, bỏ rỗng).
func (d DatabaseConfig) ReplicaURLs() []string {
	var out []string
	for _, u := range strings.Split(d.Replicas, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

type JWTConfig struct {
	Secret     string        `mapstructure:"secret" koanf:"secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl" koanf:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl" koanf:"refresh_ttl"`
}

// RedisConfig — DSN đầy đủ (redis://user:pass@host:port/db), không tách thông số.
type RedisConfig struct {
	URL string `mapstructure:"url" koanf:"url"`
}

// Client — redis client từ DSN (redis://...). Lỗi khi DSN sai.
func (c RedisConfig) Client() (*redis.Client, error) {
	opts, err := redis.ParseURL(c.URL)
	if err != nil {
		return nil, fmt.Errorf("redis DSN không hợp lệ %q: %w", c.URL, err)
	}
	return redis.NewClient(opts), nil
}

// CacheConfig — cache multi-store (eko/gocache):
// store: memory (Ristretto, mặc định — zero-config) | redis (dùng chung nhiều instance).
type CacheConfig struct {
	Store      string        `mapstructure:"store" koanf:"store"`
	Prefix     string        `mapstructure:"prefix" koanf:"prefix"`
	DefaultTTL time.Duration `mapstructure:"default_ttl" koanf:"default_ttl"`
}

// StorageConfig — multiple disk (kiểu Laravel Filesystem): default disk + danh sách
// disk với driver riêng (local | s3). App dùng storage.Disk(name) — không truy cập driver trực tiếp.
type StorageConfig struct {
	DefaultDisk string                `mapstructure:"default_disk" koanf:"default_disk"`
	Disks       map[string]DiskConfig `mapstructure:"disks" koanf:"disks"`
}

// DiskConfig — cấu hình 1 disk.
type DiskConfig struct {
	Driver    string `mapstructure:"driver" koanf:"driver"` // "local" | "s3"
	Root      string `mapstructure:"root" koanf:"root"`     // local: thư mục gốc
	URL       string `mapstructure:"url" koanf:"url"`       // local: prefix URL công khai (vd /storage)
	Bucket    string `mapstructure:"bucket" koanf:"bucket"` // s3: bucket
	Endpoint  string `mapstructure:"endpoint" koanf:"endpoint"`
	AccessKey string `mapstructure:"access_key" koanf:"access_key"`
	SecretKey string `mapstructure:"secret_key" koanf:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl" koanf:"use_ssl"`
}

type MailConfig struct {
	Host      string `mapstructure:"host" koanf:"host"`
	Port      int    `mapstructure:"port" koanf:"port"`
	Username  string `mapstructure:"username" koanf:"username"`
	Password  string `mapstructure:"password" koanf:"password"`
	From      string `mapstructure:"from" koanf:"from"`
	TLSPolicy string `mapstructure:"tls_policy" koanf:"tls_policy"` // auto (mặc định) | none | starttls
}

// LogConfig — logger (core/logger): level + file daily.
// Level đổi runtime qua dynamic config log.level (không cần restart).
type LogConfig struct {
	Level        string `mapstructure:"level" koanf:"level"`                   // debug|info|warn|error
	FileEnabled  bool   `mapstructure:"file_enabled" koanf:"file_enabled"`     // ghi thêm file theo ngày
	FileDir      string `mapstructure:"file_dir" koanf:"file_dir"`             // vd "logs"
	FileKeepDays int    `mapstructure:"file_keep_days" koanf:"file_keep_days"` // giữ N ngày
}

// defaults — zero-config: chạy được ngay không cần .env; override qua file/env khi cần.
func defaults() map[string]any {
	return map[string]any{
		"app.env":                 "development",
		"app.name":                "tiktok-live-platform",
		"app.title":               "TikTok Live Platform",
		"server.host":             "127.0.0.1",
		"server.port":             "3330",
		"server.rate_limit":       "100",
		"server.rate_limit_store": "",   // "" = theo cache store (memory|redis); "memory"/"redis" ép cụ thể
		"server.auth_rate_limit":  "10", // auth group (login/OTP/forgot...) — chống brute force
		"database.url":            "postgres://app:app_password_dev@localhost:5433/tiktok_live_platform?sslmode=disable",
		"database.replicas":       "",
		// JWT secret: default CỐ ĐỊNH (dev). Deploy bắt buộc `devtool key:generate`
		// (ghi JWT_SECRET vào .env) — giống laravel artisan key:generate.
		"jwt.secret":        "dev-secret-change-me",
		"jwt.access_ttl":    "15m",
		"jwt.refresh_ttl":   "720h",
		"redis.url":         "redis://localhost:6380/0",
		"cache.store":       "memory",
		"cache.prefix":      "",
		"cache.default_ttl": "5m",
		// Storage — multiple disk (Laravel-style). Mặc định 2 disk local: public + private.
		"storage.default_disk":         "public",
		"storage.disks.public.driver":  "local",
		"storage.disks.public.root":    "./storage/public",
		"storage.disks.public.url":     "/storage",
		"storage.disks.private.driver": "local",
		"storage.disks.private.root":   "./storage/private",
		"storage.disks.s3.driver":      "s3",
		"storage.disks.s3.endpoint":    "localhost:9000",
		"storage.disks.s3.access_key":  "minioadmin",
		"storage.disks.s3.secret_key":  "minioadmin_dev",
		"storage.disks.s3.use_ssl":     "false",
		"storage.disks.s3.bucket":      "uploads",
		"mail.host":                    "localhost",
		"mail.port":                    "1025",
		"mail.username":                "",
		"mail.password":                "",
		"mail.from":                    "no-reply@example.com",
		"mail.tls_policy":              "auto", // auto (local→NoTLS, remote→STARTTLS) | none | starttls
		"log.level":                    "info",
		"log.file_enabled":             "true", // mặc định ghi log vào file theo ngày
		"log.file_dir":                 "logs",
		"log.file_keep_days":           "14",
		// Prometheus metrics — port RIÊNG (tách khỏi app: scrape không qua middleware)
		"metrics.enabled":    "true",
		"metrics.host":       "127.0.0.1",
		"metrics.port":       "9090",
		"metrics.path":       "/metrics",
		"metrics.auth_token": "", // "" = không auth (localhost); set = Bearer token
		// OpenTelemetry tracing — mặc định TẮT (bật khi có otel-collector)
		"otel.enabled":  "false",
		"otel.endpoint": "localhost:4317", // compose: otel-collector:4317
		// OpenAPI spec tự động — mặc định BẬT; tắt khi không muốn lộ spec
		"openapi.enabled": "true",
		// Live tracker — Sockudo (Pusher-compatible WS server) + TikTok connection
		"live.connection_mode":  "long_poll", // "long_poll" | "websocket"
		"live.poll_interval_ms": "3000",
		"live.sockudo_url":       "http://localhost:6002",
		"live.sockudo_app_id":    "demo-app",
		"live.sockudo_app_key":   "demo-key",
		"live.sockudo_app_secret": "demo-secret",
		// Nguồn cấu hình chính: file://config.yml (mặc định) | postgres://...
		"config.dsn": DefaultDSN,
	}
}

// envBindings — map env var → config key (explicit, không magic — tránh lỗi
// underscore trong tên field như JWT_ACCESS_TTL vs jwt.access_ttl).
var envBindings = []struct{ env, key string }{
	{"APP_ENV", "app.env"},
	{"APP_NAME", "app.name"},
	{"APP_TITLE", "app.title"},
	{"SERVER_HOST", "server.host"},
	{"SERVER_PORT", "server.port"},
	{"SERVER_RATE_LIMIT", "server.rate_limit"},
	{"SERVER_RATE_LIMIT_STORE", "server.rate_limit_store"},
	{"SERVER_AUTH_RATE_LIMIT", "server.auth_rate_limit"},
	{"DATABASE_URL", "database.url"},
	{"DATABASE_REPLICAS", "database.replicas"},
	{"JWT_SECRET", "jwt.secret"},
	{"JWT_ACCESS_TTL", "jwt.access_ttl"},
	{"JWT_REFRESH_TTL", "jwt.refresh_ttl"},
	{"REDIS_URL", "redis.url"},
	{"CACHE_STORE", "cache.store"},
	{"CACHE_PREFIX", "cache.prefix"},
	{"CACHE_DEFAULT_TTL", "cache.default_ttl"},
	{"STORAGE_DEFAULT_DISK", "storage.default_disk"},
	// MINIO_* giữ tên env cũ → disk s3 (backward compat compose/.env)
	{"MINIO_ENDPOINT", "storage.disks.s3.endpoint"},
	{"MINIO_ACCESS_KEY", "storage.disks.s3.access_key"},
	{"MINIO_SECRET_KEY", "storage.disks.s3.secret_key"},
	{"MINIO_USE_SSL", "storage.disks.s3.use_ssl"},
	{"MINIO_BUCKET", "storage.disks.s3.bucket"},
	{"MAIL_HOST", "mail.host"},
	{"MAIL_PORT", "mail.port"},
	{"MAIL_USERNAME", "mail.username"},
	{"MAIL_PASSWORD", "mail.password"},
	{"MAIL_FROM", "mail.from"},
	{"MAIL_TLS_POLICY", "mail.tls_policy"},
	{"LOG_LEVEL", "log.level"},
	{"LOG_FILE_ENABLED", "log.file_enabled"},
	{"LOG_FILE_DIR", "log.file_dir"},
	{"LOG_FILE_KEEP_DAYS", "log.file_keep_days"},
	{"METRICS_ENABLED", "metrics.enabled"},
	{"METRICS_HOST", "metrics.host"},
	{"METRICS_PORT", "metrics.port"},
	{"METRICS_PATH", "metrics.path"},
	{"METRICS_AUTH_TOKEN", "metrics.auth_token"},
	{"OTEL_ENABLED", "otel.enabled"},
	{"OTEL_ENDPOINT", "otel.endpoint"},
	{"OPENAPI_ENABLED", "openapi.enabled"},
	// Live tracker (TikTok + Sockudo)
	{"CONNECTION_MODE", "live.connection_mode"},
	{"POLL_INTERVAL_MS", "live.poll_interval_ms"},
	{"SOCKUDO_URL", "live.sockudo_url"},
	{"SOCKUDO_APP_ID", "live.sockudo_app_id"},
	{"SOCKUDO_APP_KEY", "live.sockudo_app_key"},
	{"SOCKUDO_APP_SECRET", "live.sockudo_app_secret"},
	{"CONFIG_DSN", "config.dsn"},
}

// Load — đọc cấu hình theo thứ tự (sau này override trước):
//
//  1. defaults (koanf confmap) — nền zero-config
//  2. file .env (nếu có)      — ĐỌC TRƯỚC để biết CONFIG_DSN
//  3. CONFIG_DSN              — nguồn chính: file://config.yml | postgres://... (bảng app_config)
//  4. OS env                  — override cuối (luôn thắng)
//
// Điểm mấu chốt: .env phải đọc TRƯỚC CONFIG_DSN vì CONFIG_DSN thường nằm trong .env
// — có env mới biết lấy config từ file hay từ database.
func Load(envFile string) (*Manager, error) {
	k := koanf.New(".")

	// 1) Defaults (zero-config)
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	// 2) File .env (tuỳ chọn) — parser dotenv giữ key uppercase (APP_ENV),
	//    map qua envBindings sang config key chuẩn (app.env)
	if envFile != "" {
		if _, err := os.Stat(envFile); err == nil {
			tmp := koanf.New(".")
			if err := tmp.Load(file.Provider(envFile), dotenv.Parser()); err != nil {
				return nil, fmt.Errorf("load %s: %w", envFile, err)
			}
			for _, b := range envBindings {
				if tmp.Exists(b.env) {
					_ = k.Set(b.key, tmp.Get(b.env))
				}
			}
		}
	}

	// 3) CONFIG_DSN — nguồn cấu hình chính.
	// Ưu tiên OS env (đỉnh) → .env qua binding (đã map ở bước 2) → default file://config.yml
	dsn := os.Getenv("CONFIG_DSN")
	if dsn == "" {
		dsn = k.String("config.dsn")
	}
	if dsn == "" {
		dsn = DefaultDSN
	}
	if err := loadFromDSN(k, dsn); err != nil {
		return nil, err
	}

	// 4) OS env (explicit binding — override file/defaults)
	for _, b := range envBindings {
		if v, ok := os.LookupEnv(b.env); ok {
			_ = k.Set(b.key, v)
		}
	}

	var cfg Config
	if err := unmarshal(k, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// Ghi nhận nguồn config (main.go dùng để quyết định Redis có bắt buộc không)
	m := &Manager{K: k, Cfg: cfg, instanceID: newInstanceID()}
	m.dsnSource = detectDSNSource(dsn)
	return m, nil
}

// detectDSNSource — nguồn config thực tế: database | file | defaults.
func detectDSNSource(dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return "database"
	case strings.HasPrefix(dsn, "file://"):
		path := strings.TrimPrefix(dsn, "file://")
		if path == "" {
			path = "config.yml"
		}
		if _, err := os.Stat(path); err != nil {
			return "defaults"
		}
		return "file"
	default:
		return "defaults"
	}
}

// unmarshal — koanf → struct, kèm duration hook (time.Duration từ "15m") + weak type.
func unmarshal(k *koanf.Koanf, out any) error {
	return k.UnmarshalWithConf("", out, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
			),
			WeaklyTypedInput: true, // string → int/bool khi cần
			Result:           out,
		},
	})
}

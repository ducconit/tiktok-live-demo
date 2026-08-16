package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/cmd/app/commands"
	"github.com/ducconit/tiktok-live-platform/backend/core/build"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/logger"
	"github.com/ducconit/tiktok-live-platform/backend/core/mailer"
	"github.com/ducconit/tiktok-live-platform/backend/core/metrics"
	"github.com/ducconit/tiktok-live-platform/backend/core/otelx"
	"github.com/ducconit/tiktok-live-platform/backend/core/otp"
	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/ducconit/tiktok-live-platform/backend/core/storage"
	"github.com/ducconit/tiktok-live-platform/backend/internal/server"
	"github.com/spf13/cobra"
)

// newMailer — mailer từ config; lỗi → nil (app vẫn chạy, endpoint gửi mail báo lỗi rõ).
func newMailer(cfg config.MailConfig) *mailer.Mailer {
	m, err := mailer.New(cfg)
	if err != nil {
		slog.Warn("mailer chưa sẵn sàng (OTP sẽ chỉ in log)", "err", err)
		return nil
	}
	return m
}

// newStorage — multiple disk từ config (Laravel-style). Lỗi → nil (upload báo lỗi rõ).
func newStorage(cfg config.StorageConfig) *storage.Manager {
	st, err := storage.NewManager(cfg)
	if err != nil {
		slog.Warn("storage chưa sẵn sàng (upload sẽ lỗi)", "err", err)
		return nil
	}
	slog.Info("storage ready", "disks", st.DiskNames(), "default", st.DefaultDiskName())
	return st
}

// Build info — inject qua ldflags lúc `make build` (xem Makefile + scripts/version.sh).
var (
	version   = "dev"
	buildHash = "unknown"
	buildDate = "unknown"
)

// shutdownTimeout — thời gian chờ graceful shutdown.
const shutdownTimeout = 10 * time.Second

// app — ứng dụng chính (deploy staging/production).
// Mặc định bind 127.0.0.1:3300; override: --port <n> (hoặc SERVER_PORT env).
func main() {
	root := &cobra.Command{
		Use:   "gvs",
		Short: "tiktok-live-platform application server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), cmd.Flags().Lookup("port").Value.String())
		},
	}
	root.Flags().String("port", "", "port lắng nghe (override SERVER_PORT / mặc định 3300)")
	// CLI commands — mỗi lệnh 1 file trong cmd/app/commands/ (chỉ main.go ở đây)
	root.AddCommand(commands.KeygenCmd())
	root.AddCommand(commands.MigrateCmd())
	root.AddCommand(commands.LogsCmd())

	if err := root.Execute(); err != nil {
		slog.Error("app", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, portOverride string) error {
	mgr, err := config.Load(".env")
	if err != nil {
		return err
	}

	// ---- Logger: stdout + file daily (mặc định bật) + level dynamic ----
	lg, err := logger.New(logger.Config{
		Level:        mgr.Cfg.Log.Level,
		FileEnabled:  mgr.Cfg.Log.FileEnabled,
		FileDir:      mgr.Cfg.Log.FileDir,
		FileKeepDays: mgr.Cfg.Log.FileKeepDays,
		AppName:      mgr.Cfg.App.Name,
	})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = lg.Close() }()
	slog.SetDefault(lg.Logger())
	// Level đổi runtime qua dynamic config log.level (admin PUT /config) — đồng bộ mọi instance
	mgr.OnChange("log.level", func(v any) {
		if s, ok := v.(string); ok {
			if lvl, err := logger.ParseLevel(s); err == nil {
				lg.SetLevel(lvl)
				slog.Info("log level changed", "level", s)
			} else {
				slog.Warn("log.level không hợp lệ — giữ nguyên", "value", s, "err", err)
			}
		}
	})
	slog.Info("logger ready", "level", mgr.Cfg.Log.Level, "file_enabled", mgr.Cfg.Log.FileEnabled, "file_dir", mgr.Cfg.Log.FileDir)

	if portOverride != "" {
		mgr.Cfg.Server.Port = portOverride
		_ = mgr.K.Set("server.port", portOverride)
		slog.Info("override port từ flag --port", "port", portOverride)
	}

	// ---- Database (master + replicas — zero replica = single node) ----
	pool, err := database.NewPool(ctx, mgr.Cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	// ---- Redis (Valkey) ----
	// - Config source = database (multi-instance, config trong bảng app_config):
	//   Redis BẮT BUỘC để đồng bộ dynamic config giữa các instance → lỗi thì retry, không degrade.
	// - Nguồn khác (file/defaults): optional — ping fail → nil (zero-config vẫn chạy).
	rdb, err := mgr.Cfg.Redis.Client()
	if err != nil {
		return fmt.Errorf("redis DSN sai: %w", err)
	}
	defer func() { _ = rdb.Close() }()

	redisRequired := mgr.ConfigSource() == "database"
	if redisRequired {
		if err := retry.Do(ctx, retry.Config{Attempts: 15, InitialWait: 2 * time.Second}, "connect valkey (bắt buộc)", func() error {
			return rdb.Ping(ctx).Err()
		}); err != nil {
			return fmt.Errorf("redis/valkey bắt buộc khi CONFIG_DSN là database (đồng bộ config giữa instance): %w", err)
		}
		slog.Info("redis ready (bắt buộc — config source database)", "dsn", mgr.Cfg.Redis.URL)
	} else if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("redis không ping được — dynamic config disabled (zero-config)", "dsn", mgr.Cfg.Redis.URL, "err", err)
		rdb = nil
	}

	// ---- Cache (eko/gocache — multi-store: memory/Ristretto mặc định | redis) ----
	// Store=redis mà redis chưa sẵn sàng → lỗi RÕ (không ngậm): đổi CACHE_STORE=memory
	// hoặc khởi động redis trước.
	if mgr.Cfg.Cache.Store == "redis" && rdb == nil {
		return fmt.Errorf("cache.store=redis nhưng redis không sẵn sàng — khởi động valkey hoặc đổi CACHE_STORE=memory")
	}
	cm, err := cache.New(cache.Config{
		Store:      mgr.Cfg.Cache.Store,
		Prefix:     mgr.Cfg.Cache.Prefix,
		DefaultTTL: mgr.Cfg.Cache.DefaultTTL,
	}, rdb)
	if err != nil {
		return err
	}
	defer cm.Close()
	slog.Info("cache enabled", "store", mgr.Cfg.Cache.Store, "stores", cm.StoreNames())

	// ---- Dynamic config sync (Redis pub/sub — đồng bộ nhiều instance sau LB) ----
	if err := mgr.InitDynamic(ctx, rdb); err != nil {
		return err
	}
	defer mgr.Close()

	// ---- OpenTelemetry tracing (bật khi OTEL_ENABLED=true — xem core/otelx) ----
	// Provider global (otel.Tracer) — middleware gin dùng; shutdown flush batch.
	traceProvider, err := otelx.Init(otelx.Config{
		Enabled:     mgr.Cfg.OTel.Enabled,
		Endpoint:    mgr.Cfg.OTel.Endpoint,
		ServiceName: mgr.Cfg.App.Name,
	})
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}
	if mgr.Cfg.OTel.Enabled {
		slog.Info("otel tracing enabled", "endpoint", mgr.Cfg.OTel.Endpoint)
	}

	// ---- HTTP server ----
	st := newStorage(mgr.Cfg.Storage)
	mail := newMailer(mgr.Cfg.Mail)
	srv := server.New(mgr, pool, rdb, cm, build.Info{Version: version, BuildHash: buildHash, BuildDate: buildDate}, server.Deps{
		OTP:     otp.NewService(rdb),
		Mailer:  mail,
		Storage: st,
	})

	// ---- Health checks → metrics service_up (ping định kỳ 30s) ----
	checks := []metrics.HealthCheck{
		{Name: "postgres", Ping: func(ctx context.Context) error { return pool.Ping(ctx) }},
	}
	if rdb != nil {
		checks = append(checks, metrics.HealthCheck{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }})
	}
	if st != nil {
		// Health s3 disk (nếu có) — giữ tên "minio" (metrics cũ, không đổi key)
		checks = append(checks, metrics.HealthCheck{Name: "minio", Ping: func(ctx context.Context) error {
			return st.Health(ctx, "s3")
		}})
	}
	if mail != nil {
		checks = append(checks, metrics.HealthCheck{Name: "mail", Ping: mail.Health})
	}
	metrics.StartHealthChecks(ctx, checks)

	// ---- Metrics server (PORT RIÊNG — xem core/metrics) ----
	// Mặc định 127.0.0.1:9090/metrics, KHÔNG auth; set METRICS_AUTH_TOKEN để yêu cầu Bearer.
	var metricsSrv *http.Server
	if mgr.Cfg.Metrics.Enabled {
		mux := http.NewServeMux()
		mux.Handle(mgr.Cfg.Metrics.Path, metrics.Handler(mgr.Cfg.Metrics.AuthToken))
		metricsSrv = &http.Server{
			Addr:              mgr.Cfg.Metrics.Host + ":" + mgr.Cfg.Metrics.Port,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			slog.Info("metrics listening", "addr", metricsSrv.Addr, "path", mgr.Cfg.Metrics.Path, "auth", mgr.Cfg.Metrics.AuthToken != "")
			if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("metrics server lỗi", "err", err)
			}
		}()
	}

	// ---- Graceful shutdown ----
	// Lỗi server (vd bind port thất bại) → shutdown sạch để DEFER chạy (đóng DB/redis/file log),
	// KHÔNG os.Exit (os.Exit bỏ qua defer → connection bị rò).
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", mgr.Cfg.Server.Host+":"+mgr.Cfg.Server.Port, "env", mgr.Cfg.App.Env, "instance", mgr.InstanceID())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	shutdown := func() error {
		slog.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		var errs []error
		if metricsSrv != nil {
			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("metrics server: %w", err))
			}
		}
		if err := srv.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, err)
		}
		// Flush trace batch còn lại (OTLP export) trước khi thoát
		if err := otelx.Shutdown(shutdownCtx, traceProvider); err != nil {
			errs = append(errs, fmt.Errorf("otel shutdown: %w", err))
		}
		return errors.Join(errs...)
	}

	select {
	case err := <-serverErr:
		if err == nil {
			return nil
		}
		slog.Error("server lỗi — đóng kết nối sạch", "err", err)
		_ = shutdown()
		return err
	case <-quit:
		return shutdown()
	}
}

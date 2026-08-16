// Package otelx — OpenTelemetry tracing cho app (OTLP gRPC → otel-collector → Jaeger).
//
// BẬT TẮT qua config: otel.enabled (mặc định FALSE — zero-config chạy như cũ;
// bật khi có collector: OTEL_ENABLED=true + OTEL_ENDPOINT=otel-collector:4317).
// Khi tắt → Tracer() trả noop tracer (chi phí ~0, app không đổi hành vi).
package otelx

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Config — cấu hình tracing (từ core/config OTelConfig).
type Config struct {
	Enabled     bool   // bật TracerProvider + export OTLP
	Endpoint    string // OTLP gRPC endpoint (vd localhost:4317, otel-collector:4317)
	ServiceName string // tên service (span service.name)
}

// Init — khởi tạo TracerProvider (OTLP gRPC exporter + batch processor).
// Chỉ export khi Enabled; ngược lại trả noop provider (mọi span bị bỏ qua).
func Init(cfg Config) (trace.TracerProvider, error) {
	if !cfg.Enabled {
		return noop.NewTracerProvider(), nil
	}

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // collector nội bộ (docker network) — không TLS
	)
	if err != nil {
		return nil, fmt.Errorf("otelx: init exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(newResource(cfg.ServiceName)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return provider, nil
}

// Tracer — tracer cho 1 package (name = package path, chuẩn OTel).
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

package otelx

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// newResource — resource chuẩn (service.name → tên app trên Jaeger).
func newResource(serviceName string) *sdkresource.Resource {
	attrs := []attribute.KeyValue{}
	if serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(serviceName))
	}
	return sdkresource.NewWithAttributes(sdkresource.Default().SchemaURL(), attrs...)
}

// Shutdown — flush batch + đóng exporter (gọi trong graceful shutdown).
func Shutdown(ctx context.Context, provider trace.TracerProvider) error {
	if p, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
		return p.Shutdown(ctx)
	}
	return nil
}

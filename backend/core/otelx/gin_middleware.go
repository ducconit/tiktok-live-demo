package otelx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// GinMiddleware — tạo span HTTP per request (tên = route, chuẩn semconv).
// Khi tracing tắt → noop tracer, span bị bỏ qua (chi phí ~0).
// Gắn với global provider (otel.SetTracerProvider ở Init).
func GinMiddleware() gin.HandlerFunc {
	tracer := Tracer("server.http")
	return func(c *gin.Context) {
		// Extract trace context từ header (w3c traceparent) — support gọi từ bên ngoài
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		opts := []trace.SpanStartOption{
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.HTTPRouteKey.String(route),
				semconv.URLPath(c.Request.URL.Path),
				attribute.String("http.request_id", c.GetString("request_id")),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		}
		ctx, span := tracer.Start(ctx, route, opts...)
		defer span.End()

		// request-scoped context chứa span — service/repo có thể tạo child span
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		span.SetAttributes(
			semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()),
			attribute.String("http.user_id", c.GetString("user_id")),
		)
		if c.Writer.Status() >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, strconv.Itoa(c.Writer.Status()))
		}
	}
}

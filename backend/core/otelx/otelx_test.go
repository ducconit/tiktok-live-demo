package otelx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TestInit_Disabled_ReturnsNoop — tắt → noop provider (zero-config chi phí ~0).
func TestInit_Disabled_ReturnsNoop(t *testing.T) {
	p, err := Init(Config{Enabled: false, Endpoint: "localhost:4317", ServiceName: "test"})
	require.NoError(t, err)
	assert.IsType(t, noop.NewTracerProvider(), p)
}

// TestGinMiddleware_CreatesSpan — request qua middleware → span với attributes chuẩn.
func TestGinMiddleware_CreatesSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// In-memory exporter (tracetest) — verify span không cần collector thật
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		_ = provider.Shutdown(context.Background())
	})

	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/healthz", func(c *gin.Context) {
		c.Set("user_id", "u-123")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1, "phải có đúng 1 span")

	s := spans[0]
	assert.Equal(t, "/healthz", s.Name)
	assert.Equal(t, trace.SpanKindServer, s.SpanKind)
	attrs := map[string]string{}
	for _, kv := range s.Attributes {
		attrs[string(kv.Key)] = kv.Value.String()
	}
	assert.Equal(t, "GET", attrs["http.request.method"])
	assert.Equal(t, "/healthz", attrs["http.route"])
	assert.Equal(t, "200", attrs["http.response.status_code"])
	assert.Equal(t, "u-123", attrs["http.user_id"])
}

// TestGinMiddleware_ErrorStatus — 500 → span status error.
func TestGinMiddleware_ErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		_ = provider.Shutdown(context.Background())
	})

	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/boom", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "500", spans[0].Status.Description)
	assert.Equal(t, "Error", spans[0].Status.Code.String())
}

// TestShutdown_NoopProvider — shutdown provider noop không lỗi.
func TestShutdown_NoopProvider(t *testing.T) {
	p, err := Init(Config{Enabled: false, Endpoint: "x", ServiceName: ""})
	require.NoError(t, err)
	assert.NoError(t, Shutdown(context.Background(), p))
}

package response

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/gin-gonic/gin"
)

func benchCtx(b *testing.B) *gin.Context {
	b.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	return c
}

func BenchmarkOK_SmallData(b *testing.B) {
	c := benchCtx(b)
	payload := map[string]any{"id": "abc", "name": "bench"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OK(c, payload)
	}
}

func BenchmarkOKList_Items(b *testing.B) {
	c := benchCtx(b)
	items := []map[string]any{
		{"id": "1", "name": "a"}, {"id": "2", "name": "b"}, {"id": "3", "name": "c"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OKList(c, items)
	}
}

func BenchmarkError_AppErr(b *testing.B) {
	c := benchCtx(b)
	err := apperr.BadRequest("invalid_json", "Request body không hợp lệ")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Error(c, err)
	}
}

func BenchmarkError_Internal(b *testing.B) {
	c := benchCtx(b)
	err := errors.New("db connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Error(c, err)
	}
}

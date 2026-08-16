package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestT_DefaultLang — không set lang → tiếng Việt (zero-config).
func TestT_DefaultLang(t *testing.T) {
	assert.Equal(t, "API key không hợp lệ", T("", "error.invalid_api_key", nil))
	assert.Equal(t, "API key không hợp lệ", T("fr", "error.invalid_api_key", nil)) // fallback về default
}

// TestT_English — lang en → bản tiếng Anh.
func TestT_English(t *testing.T) {
	assert.Equal(t, "Invalid API key", T("en", "error.invalid_api_key", nil))
}

// TestT_MissingKey — key không tồn tại → trả chính ID (dev thấy key thiếu).
func TestT_MissingKey(t *testing.T) {
	assert.Equal(t, "error.unknown_key", T("vi", "error.unknown_key", nil))
}

// TestT_TemplateData — interpolation {{.Scope}} / {{.Slug}}.
func TestT_TemplateData(t *testing.T) {
	assert.Equal(t, "API key không có scope orders.read",
		T("vi", "error.api_key_scope", map[string]any{"Scope": "orders.read"}))
	assert.Equal(t, "Role admin chưa tồn tại (chạy migrate + seed)",
		T("vi", "error.role_slug_not_found", map[string]any{"Slug": "admin"}))
}

// TestMiddleware_AcceptLanguage — parse Accept-Language → set lang đúng.
func TestMiddleware_AcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, Lang(c))
	})

	cases := []struct{ header, want string }{
		{"", "vi"},                        // không header → default
		{"en-US,en;q=0.9", "en"},          // chuẩn
		{"fr-FR,fr;q=0.9,en;q=0.5", "en"}, // fr không có trong bundle → fallback en
		{"vi-VN,vi;q=0.9,en;q=0.8", "vi"}, // vi-VN → vi
		{"en", "en"},                      // chỉ en
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.header != "" {
			req.Header.Set("Accept-Language", tc.header)
		}
		r.ServeHTTP(w, req)
		assert.Equal(t, tc.want, w.Body.String(), "header %q", tc.header)
	}
}

// TestMiddleware_RenderTheoLang — response msg đổi theo Accept-Language
// (e2e nhỏ: middleware + render qua helper).
func TestMiddleware_RenderTheoLang(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, T(Lang(c), "error.bad_credentials", nil))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, "Email hoặc mật khẩu sai", w.Body.String())

	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en")
	r.ServeHTTP(w2, req)
	require.Equal(t, "Invalid email or password", w2.Body.String())
}

package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env
}

func TestOK_EnvelopeShape(t *testing.T) {
	c, w := setup()
	OK(c, map[string]string{"x": "y"})
	assert.Equal(t, http.StatusOK, w.Code)
	env := decode(t, w)
	assert.Equal(t, "0", env.Code)
	assert.Equal(t, "", env.Msg)
	assert.Equal(t, map[string]any{}, env.Meta)
}

func TestOKList_NilBecomesEmptyArray(t *testing.T) {
	c, w := setup()
	OKList(c, nil)
	env := decode(t, w)
	assert.Equal(t, "[]", mustJSON(env.Data), "nil → mảng rỗng (chuẩn danh sách)")

	c2, w2 := setup()
	OKList(c2, []string(nil))
	env2 := decode(t, w2)
	assert.Equal(t, "[]", mustJSON(env2.Data))
}

func TestOKList_NonNilKept(t *testing.T) {
	c, w := setup()
	OKList(c, []string{"a"})
	env := decode(t, w)
	assert.Equal(t, `["a"]`, mustJSON(env.Data))
}

func TestCreated_Status201(t *testing.T) {
	c, w := setup()
	Created(c, "obj")
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "0", decode(t, w).Code)
}

func TestNoContent_NoBody(t *testing.T) {
	c, w := setup()
	NoContent(c)
	// gin chưa flush header (không có Write) — đọc từ c.Writer.Status()
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Empty(t, w.Body.String())
}

func TestError_Kinds(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string // response code luôn = HTTP status string
		wantMsg string
	}{
		{"bad_request", apperr.BadRequest("invalid_json", "error.invalid_body"), 400, "400", "Dữ liệu không hợp lệ"},
		{"unauthorized", apperr.Unauthorized("bad_token", "error.bad_credentials"), 401, "401", "Email hoặc mật khẩu sai"},
		{"forbidden", apperr.Forbidden("no_perm", "Không có quyền"), 403, "403", "Không có quyền"},
		{"not_found", apperr.NotFound("x", "Không tìm thấy"), 404, "404", "Không tìm thấy"},
		{"conflict", apperr.Conflict("x", "Đã tồn tại"), 409, "409", "Đã tồn tại"},
		{"too_many", apperr.TooManyRequests("x", "Quá nhiều"), 429, "429", "Quá nhiều"},
		{"maintenance", apperr.New(apperr.KindServiceUnavailable, "503", "Bảo trì"), 503, "503", "Bảo trì"},
		{"internal_plain", errors.New("db down"), 500, "500", "Ops! Đang có lỗi xảy ra, vui lòng thử lại sau."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := setup()
			Error(c, tc.err)
			assert.Equal(t, tc.status, w.Code)
			env := decode(t, w)
			assert.Equal(t, tc.code, env.Code, "code phải là HTTP status string")
			assert.Equal(t, tc.wantMsg, env.Msg)
			// Middleware log đọc lại được lỗi
			assert.NotNil(t, ErrorFromContext(c), "lỗi phải lưu vào context cho logger")
		})
	}
}

func TestError_ValidationMeta(t *testing.T) {
	c, w := setup()
	Error(c, apperr.Validation(map[string]string{"email": "Email không hợp lệ"}))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	env := decode(t, w)
	assert.Equal(t, "422", env.Code)
	assert.Equal(t, "Dữ liệu không hợp lệ", env.Msg)
	fields, ok := env.Meta.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Email không hợp lệ", fields["email"])
}

func TestError_InternalDoesNotLeakDetail(t *testing.T) {
	c, w := setup()
	Error(c, apperr.WrapInternal(errors.New("secret: password=abc")))
	env := decode(t, w)
	assert.Equal(t, "Ops! Đang có lỗi xảy ra, vui lòng thử lại sau.", env.Msg)
	assert.NotContains(t, env.Msg, "secret")
	assert.NotContains(t, env.Msg, "password")
}

func TestValidationError_Helper(t *testing.T) {
	c, w := setup()
	ValidationError(c, map[string]string{"email": "sai"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestParsePageParams_DefaultsAndClamps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Không query → default
	c, _ := gin.CreateTestContext(nil)
	p := ParsePageParams(c)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PageSize)

	// Clamp max 100
	req := httptest.NewRequest("GET", "/?page=999&page_size=9999", nil)
	c2, _ := gin.CreateTestContext(nil)
	c2.Request = req
	p2 := ParsePageParams(c2)
	assert.Equal(t, 999, p2.Page, "page không chặn trên")
	assert.Equal(t, 100, p2.PageSize, "page_size chặn max 100")

	// Âm → default
	req3 := httptest.NewRequest("GET", "/?page=-3&page_size=-1", nil)
	c3, _ := gin.CreateTestContext(nil)
	c3.Request = req3
	p3 := ParsePageParams(c3)
	assert.Equal(t, 1, p3.Page)
	assert.Equal(t, 20, p3.PageSize)
}

func TestBuildMeta(t *testing.T) {
	m := BuildMeta(PageParams{Page: 2, PageSize: 10}, 42)
	assert.Equal(t, &Meta{Limit: 10, Page: 2, Total: 42}, m)
}

func TestOKWithMeta(t *testing.T) {
	c, w := setup()
	OKWithMeta(c, []string{"a"}, &Meta{Limit: 10, Page: 1, Total: 1})
	env := decode(t, w)
	assert.Equal(t, "0", env.Code)
	assert.Equal(t, map[string]any{"limit": float64(10), "page": float64(1), "total": float64(1)}, env.Meta)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

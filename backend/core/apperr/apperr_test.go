package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_And_Error(t *testing.T) {
	e := New(KindNotFound, "role_not_found", "Không tìm thấy vai trò")
	assert.Equal(t, KindNotFound, e.Kind)
	assert.Equal(t, "role_not_found", e.Code)
	assert.Equal(t, "Không tìm thấy vai trò", e.Message)
	assert.Equal(t, "Không tìm thấy vai trò", e.Error())
}

func TestHelpers_SetCorrectKind(t *testing.T) {
	cases := []struct {
		name string
		got  *AppError
		kind Kind
	}{
		{"BadRequest", BadRequest("x", "m"), KindInvalid},
		{"Unauthorized", Unauthorized("x", "m"), KindUnauthorized},
		{"Forbidden", Forbidden("x", "m"), KindForbidden},
		{"NotFound", NotFound("x", "m"), KindNotFound},
		{"Conflict", Conflict("x", "m"), KindConflict},
		{"TooLarge", TooLarge("x", "m"), KindPayloadTooLarge},
		{"TooManyRequests", TooManyRequests("x", "m"), KindTooManyRequests},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.kind, tc.got.Kind)
			assert.Equal(t, "m", tc.got.Message)
		})
	}
}

func TestWithDetails_And_WithFields(t *testing.T) {
	e := BadRequest("x", "m").WithDetails("detail").WithFields(map[string]string{"email": "sai"})
	assert.Equal(t, "detail", e.Details)
	assert.Equal(t, "sai", e.Fields["email"])
}

func TestToHTTPStatus_AllKinds(t *testing.T) {
	cases := []struct {
		kind Kind
		want int
	}{
		{KindInvalid, http.StatusBadRequest},
		{KindUnauthorized, http.StatusUnauthorized},
		{KindForbidden, http.StatusForbidden},
		{KindNotFound, http.StatusNotFound},
		{KindConflict, http.StatusConflict},
		{KindPayloadTooLarge, http.StatusRequestEntityTooLarge},
		{KindValidation, http.StatusUnprocessableEntity},
		{KindUnprocessable, http.StatusUnprocessableEntity},
		{KindTooManyRequests, http.StatusTooManyRequests},
		{KindServiceUnavailable, http.StatusServiceUnavailable},
		{KindInternal, http.StatusInternalServerError},
		{Kind(999), http.StatusInternalServerError}, // unknown → 500
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ToHTTPStatus(tc.kind), "kind=%d", tc.kind)
	}
}

func TestFrom_DetectsAppError(t *testing.T) {
	// Trực tiếp
	e := NotFound("x", "m")
	got, ok := From(e)
	require.True(t, ok)
	assert.Same(t, e, got)

	// Bọc qua fmt.Errorf + errors.Join — vẫn detect (errors.As)
	wrapped := errors.Join(e, errors.New("other"))
	got2, ok2 := From(wrapped)
	require.True(t, ok2)
	assert.Equal(t, KindNotFound, got2.Kind)

	// Không phải AppError
	_, ok3 := From(errors.New("plain"))
	assert.False(t, ok3)
}

func TestWrapInternal(t *testing.T) {
	e := WrapInternal(errors.New("db down"))
	assert.Equal(t, KindInternal, e.Kind)
	assert.Equal(t, "500", e.Code)
	assert.Equal(t, "db down", e.Message)

	// AppError đã có → giữ nguyên (không bọc lại)
	orig := NotFound("x", "m")
	wrapped := WrapInternal(orig)
	assert.Equal(t, KindNotFound, wrapped.Kind, "AppError không được bọc lại")
	assert.Same(t, orig, wrapped)

	// AppError bị wrap %w vẫn nhận ra
	boxed := fmt.Errorf("nested: %w", orig)
	got := WrapInternal(boxed)
	assert.Equal(t, KindNotFound, got.Kind, "AppError bị %w wrap vẫn giữ kind")
}

func TestValidation_Kind(t *testing.T) {
	e := Validation(map[string]string{"email": "Email không hợp lệ"})
	assert.Equal(t, KindValidation, e.Kind)
	assert.Equal(t, "Dữ liệu không hợp lệ", e.Message)
	assert.Equal(t, "Email không hợp lệ", e.Fields["email"])
}

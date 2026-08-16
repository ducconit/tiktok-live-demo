package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code mặc định — chuẩn response 4 key.
const (
	CodeOK     = "0"                    // thành công
	Code500    = "500"                  // lỗi nội bộ (không lộ chi tiết)
	Msg500     = "error.internal_error" // message ID (i18n — không lộ chi tiết)
	MsgInvalid = "error.invalid_body"   // message ID (validation)
)

// Envelope — shape chuẩn MỌI response: { code, msg, data, meta }.
//
//	code "0"  → thành công (msg bỏ trống)
//	code != 0 → lỗi: code = HTTP status dạng string ("400", "401", ..., "500", "503")
//	validation (422) → meta = { field: "message lỗi" } (đúng tag json của struct)
type Envelope struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
	Meta any    `json:"meta"`
}

// OK — 200, code "0".
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: CodeOK, Data: data, Meta: map[string]any{}})
}

// OKList — 200, đảm bảo data là mảng (rỗng [] nếu nil).
func OKList(c *gin.Context, items any) {
	OK(c, ensureList(items))
}

// Created — 201, code "0".
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: CodeOK, Data: data, Meta: map[string]any{}})
}

// NoContent — 204, không body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// OKWithMeta — 200 kèm meta (phân trang...).
func OKWithMeta(c *gin.Context, data, meta any) {
	c.JSON(http.StatusOK, Envelope{Code: CodeOK, Data: data, Meta: meta})
}

// ensureList — nil → mảng rỗng (chuẩn: danh sách luôn là []).
func ensureList(items any) any {
	switch v := items.(type) {
	case nil:
		return []any{}
	case []any:
		if v == nil {
			return []any{}
		}
	case []string:
		if v == nil {
			return []string{}
		}
	}
	return items
}

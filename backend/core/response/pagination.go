package response

import (
	"strconv"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// ctxKeyError — gin context key lưu lỗi đã trả cho client (để logger ghi level đúng).
const ctxKeyError = "resp_error"

// Error — map apperr sang HTTP status + envelope 4 key.
// Lưu lỗi vào context để loggerMiddleware ghi log error/warn đúng level.
func Error(c *gin.Context, err error) {
	ae, ok := apperr.From(err)
	if !ok {
		ae = apperr.New(apperr.KindInternal, Code500, Msg500)
	}
	c.Set(ctxKeyError, ae)

	// Chuẩn: code luôn = HTTP status dạng string ("400", "401", ..., "500", "503")
	// — đồng nhất, client map dễ. (ae.Code giữ để log chi tiết.)
	status := apperr.ToHTTPStatus(ae.Kind)
	code := strconv.Itoa(status)

	env := Envelope{Code: code, Data: nil, Meta: map[string]any{}}
	switch ae.Kind {
	case apperr.KindInternal:
		// Không lộ chi tiết lỗi nội bộ — msg mặc định an toàn (render theo lang)
		env.Msg = i18n.T(i18n.Lang(c), Msg500, nil)
	case apperr.KindValidation:
		env.Msg = i18n.T(i18n.Lang(c), MsgInvalid, nil)
		env.Meta = ae.Fields // { field: "message lỗi" } — client hiển thị lên form
	default:
		// ae.Message = message ID (i18n) — render theo ngôn ngữ request (+ template data)
		env.Msg = i18n.T(i18n.Lang(c), ae.Message, ae.Data)
	}
	c.JSON(status, env)
}

// ValidationError — trả 422 với field errors (chuẩn: meta = { field: message }).
func ValidationError(c *gin.Context, fields map[string]string) {
	Error(c, apperr.Validation(fields))
}

// ErrorFromContext — lỗi đã trả (nếu có) — dùng bởi middleware log.
func ErrorFromContext(c *gin.Context) *apperr.AppError {
	v, _ := c.Get(ctxKeyError)
	ae, _ := v.(*apperr.AppError)
	return ae
}

// PageParams — page/page_size chuẩn hoá.
type PageParams struct {
	Page     int
	PageSize int
}

// ParsePageParams đọc query `page` và `page_size` (mặc định/chặn bằng hằng số).
func ParsePageParams(c *gin.Context) PageParams {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = defaultPage
	}
	size, _ := strconv.Atoi(c.Query("page_size"))
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return PageParams{Page: page, PageSize: size}
}

// Meta — thông tin phân trang (chuẩn: chỉ limit, page, total).
type Meta struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Total int `json:"total"`
}

// BuildMeta tính meta phân trang từ total.
func BuildMeta(p PageParams, total int) *Meta {
	return &Meta{Limit: p.PageSize, Page: p.Page, Total: total}
}

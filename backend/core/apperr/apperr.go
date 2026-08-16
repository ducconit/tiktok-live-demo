package apperr

import (
	"errors"
	"net/http"
)

// Kind — loại lỗi nghiệp vụ, map sang HTTP status + response code (chuẩn response 4 key).
type Kind int

const (
	KindInvalid            Kind = iota // 400 — request sai (parse, fallback validation)
	KindUnauthorized                   // 401
	KindForbidden                      // 403
	KindNotFound                       // 404
	KindConflict                       // 409
	KindPayloadTooLarge                // 413 — file quá lớn
	KindValidation                     // 422 — có Fields (field → message)
	KindUnprocessable                  // 422 — fallback (không rõ field nào)
	KindTooManyRequests                // 429 — rate limit
	KindServiceUnavailable             // 503 — maintenance
	KindInternal                       // 500
)

// AppError — lỗi nghiệp vụ có code ổn định cho client.
type AppError struct {
	Kind    Kind
	Code    string // response code: "0" thành công; lỗi = HTTP status dạng string
	Message string // message ID (i18n — render theo lang request)
	Details any
	Fields  map[string]string // validation: field → message (đưa vào meta)
	Data    map[string]any    // template data cho message ID ({{.Field}})
}

func (e *AppError) Error() string { return e.Message }

// New tạo AppError (Message = message ID — xem core/i18n/messages/).
func New(kind Kind, code, message string) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message}
}

// WithData gắn template data cho message (vd {{.Slug}}, {{.Scope}}).
func (e *AppError) WithData(data map[string]any) *AppError {
	e.Data = data
	return e
}

// WithDetails gắn chi tiết lỗi (vd danh sách field validation).
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// WithFields gắn field errors (chuẩn 422: meta = { field: message }).
func (e *AppError) WithFields(fields map[string]string) *AppError {
	e.Fields = fields
	return e
}

// ---- Helpers nhanh ----

func BadRequest(code, msg string) *AppError   { return New(KindInvalid, code, msg) }
func Unauthorized(code, msg string) *AppError { return New(KindUnauthorized, code, msg) }
func Forbidden(code, msg string) *AppError    { return New(KindForbidden, code, msg) }
func NotFound(code, msg string) *AppError     { return New(KindNotFound, code, msg) }
func Conflict(code, msg string) *AppError     { return New(KindConflict, code, msg) }
func TooLarge(code, msg string) *AppError     { return New(KindPayloadTooLarge, code, msg) }
func TooManyRequests(code, msg string) *AppError {
	return New(KindTooManyRequests, code, msg)
}
func Unavailable(code, msg string) *AppError { return New(KindServiceUnavailable, code, msg) }
func Internal(code, msg string) *AppError    { return New(KindInternal, code, msg) }

// Validation — lỗi 422 với field errors (meta = { field: message }).
func Validation(fields map[string]string) *AppError {
	return &AppError{Kind: KindValidation, Code: "422", Message: "Dữ liệu không hợp lệ", Fields: fields}
}

// From unwrap errors.As để lấy *AppError từ chuỗi error (vd bị fmt.Errorf %w / errors.Join bọc).
func From(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// WrapInternal bọc lỗi không xác định thành 500. Nếu lỗi ĐÃ là *AppError
// (vd bị wrap %w) → giữ nguyên kind (không hạ cấp thành internal).
func WrapInternal(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return New(KindInternal, "500", err.Error())
}

// ToHTTPStatus map Kind sang HTTP status.
func ToHTTPStatus(k Kind) int {
	switch k {
	case KindInvalid:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindPayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case KindValidation, KindUnprocessable:
		return http.StatusUnprocessableEntity
	case KindTooManyRequests:
		return http.StatusTooManyRequests
	case KindServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

package validation

import (
	"errors"
	"reflect"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/go-playground/validator/v10"
)

// FieldError — một lỗi field (Tag = validator tag chuẩn — render msg qua i18n theo lang).
type FieldError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Param string `json:"param,omitempty"` // tham số tag (vd min=8 → "8")
}

var validate = validator.New()

func init() {
	// Chuẩn: meta dùng tên field theo JSON tag (client gửi gì, trả về đúng tên đó)
	// — fe.Field() sẽ trả json name thay vì tên Go (vd "role_id" thay vì "RoleID").
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})
}

// FieldsMap — chuyển []FieldError sang map field → message đã render theo ngôn ngữ
// (chuẩn response 422: meta = { field: "message" }).
func FieldsMap(fields []FieldError, lang string) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		m[f.Field] = i18n.T(lang, "validation."+f.Tag, map[string]any{"Param": f.Param})
	}
	return m
}

// ValidateStruct validate struct theo tags `validate:"..."`.
// Trả về danh sách FieldError; nil nếu hợp lệ.
func ValidateStruct(s any) []FieldError {
	if err := validate.Struct(s); err != nil {
		var verr validator.ValidationErrors
		if errors.As(err, &verr) {
			fields := make([]FieldError, 0, len(verr))
			for _, fe := range verr {
				fields = append(fields, FieldError{
					Field: fieldName(fe),
					Tag:   fe.Tag(),
					Param: fe.Param(),
				})
			}
			return fields
		}
		return []FieldError{{Field: "_", Tag: "invalid"}}
	}
	return nil
}

func fieldName(fe validator.FieldError) string {
	return strings.ToLower(fe.Field()) // đã map qua json tag (RegisterTagNameFunc)
}

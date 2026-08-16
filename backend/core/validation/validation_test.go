package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type profileInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Gender   string `json:"gender" validate:"oneof=male female other"`
	Avatar   string `json:"avatar" validate:"omitempty,url"`
	RoleID   string `json:"role_id" validate:"omitempty,uuid"`
}

// reqInput — chỉ required (tách khỏi tag khác — validator trả tag nào không xác định
// khi 1 field rỗng có nhiều tag; test từng case độc lập cho deterministic).
type reqInput struct {
	Email    string `validate:"required"`
	Password string `validate:"required"`
	Gender   string `validate:"required"`
}

type uuidInput struct {
	RoleID string `json:"role_id" validate:"omitempty,uuid"`
}

func TestValidateStruct_Valid(t *testing.T) {
	in := profileInput{
		Email:    "a@b.com",
		Password: "password123",
		Gender:   "male",
	}
	assert.Nil(t, ValidateStruct(&in))
}

func TestValidateStruct_Messages(t *testing.T) {
	in := profileInput{Email: "not-an-email", Password: "short", Gender: "unknown"}
	fields := ValidateStruct(&in)
	assert.Len(t, fields, 3)

	got := FieldsMap(fields, "vi")
	assert.Equal(t, "email không hợp lệ", got["email"])
	assert.Equal(t, "tối thiểu 8 ký tự", got["password"])
	assert.Equal(t, "phải là một trong: male female other", got["gender"])
}

func TestValidateStruct_Required(t *testing.T) {
	in := reqInput{}
	fields := ValidateStruct(&in)
	assert.Len(t, fields, 3) // email, password, gender đều required
	got := FieldsMap(fields, "vi")
	assert.Equal(t, "bắt buộc", got["email"])
	assert.Equal(t, "bắt buộc", got["password"])
	assert.Equal(t, "bắt buộc", got["gender"])
}

func TestValidateStruct_OptionalValidated(t *testing.T) {
	// omitempty — rỗng thì bỏ qua
	in := profileInput{Email: "a@b.com", Password: "password123", Gender: "male", Avatar: ""}
	assert.Nil(t, ValidateStruct(&in))

	// có giá trị sai → lỗi url
	in.Avatar = "not-a-url"
	fields := ValidateStruct(&in)
	assert.Len(t, fields, 1)
	assert.Equal(t, "URL không hợp lệ", FieldsMap(fields, "vi")["avatar"])
}

func TestValidateStruct_UUIDTag(t *testing.T) {
	in := uuidInput{RoleID: "abc"}
	fields := ValidateStruct(&in)
	assert.Len(t, fields, 1)
	assert.Equal(t, "UUID không hợp lệ", FieldsMap(fields, "vi")["role_id"])
}

func TestFieldsMap_Empty(t *testing.T) {
	assert.Empty(t, FieldsMap(nil, "vi"))
	assert.Empty(t, FieldsMap([]FieldError{}, "vi"))
}

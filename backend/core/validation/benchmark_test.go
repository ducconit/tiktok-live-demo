package validation

import "testing"

type benchInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	FullName string `json:"full_name" validate:"required,max=100"`
}

func BenchmarkValidateStruct_Valid(b *testing.B) {
	in := benchInput{Email: "a@b.com", Password: "password123", FullName: "Nguyen Van A"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if fields := ValidateStruct(&in); fields != nil {
			b.Fatal("expected valid")
		}
	}
}

func BenchmarkValidateStruct_Invalid(b *testing.B) {
	in := benchInput{Email: "bad", Password: "short", FullName: ""}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateStruct(&in)
	}
}

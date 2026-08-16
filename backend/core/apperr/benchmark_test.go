package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func BenchmarkFrom_Direct(b *testing.B) {
	e := NotFound("x", "m")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := From(e); !ok {
			b.Fatal("miss")
		}
	}
}

func BenchmarkFrom_Wrapped(b *testing.B) {
	e := fmt.Errorf("wrap: %w", NotFound("x", "m"))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := From(e); !ok {
			b.Fatal("miss")
		}
	}
}

func BenchmarkWrapInternal_Plain(b *testing.B) {
	err := errors.New("db down")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapInternal(err)
	}
}

func BenchmarkToHTTPStatus(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToHTTPStatus(KindValidation)
	}
}

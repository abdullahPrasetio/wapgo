package validator

import "testing"

type benchDTO struct {
	Name  string `json:"name"  validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"required,min=1,max=150"`
}

func BenchmarkValidate_Valid(b *testing.B) {
	v := New()
	dto := benchDTO{Name: "Alice", Email: "alice@example.com", Age: 30}
	b.ReportAllocs()
	for b.Loop() {
		_ = v.Validate(dto)
	}
}

func BenchmarkValidate_Invalid(b *testing.B) {
	v := New()
	dto := benchDTO{Name: "", Email: "not-an-email", Age: 0}
	b.ReportAllocs()
	for b.Loop() {
		_ = v.Validate(dto)
	}
}

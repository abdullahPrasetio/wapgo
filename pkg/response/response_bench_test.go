package response

import (
	"encoding/json"
	"testing"
)

func BenchmarkResponseMarshal_Success(b *testing.B) {
	b.ReportAllocs()
	r := Response{Status: true, Message: "user fetched", Data: map[string]any{"id": "abc-123", "name": "Alice", "email": "alice@example.com"}}
	for b.Loop() {
		_, _ = json.Marshal(r)
	}
}

func BenchmarkResponseMarshal_Error(b *testing.B) {
	b.ReportAllocs()
	r := ErrorResponse{Status: false, Code: ErrValidation, Message: "name: is required; email: must be a valid email"}
	for b.Loop() {
		_, _ = json.Marshal(r)
	}
}

func BenchmarkResponseMarshal_Paginated(b *testing.B) {
	b.ReportAllocs()
	r := PaginatedResponse{
		Status:  true,
		Message: "users fetched",
		Data:    []map[string]any{{"id": "1", "name": "Alice"}, {"id": "2", "name": "Bob"}},
		Pagination: PageMeta{Page: 1, PerPage: 10, Total: 100, TotalPages: 10},
	}
	for b.Loop() {
		_, _ = json.Marshal(r)
	}
}

package auth

import "testing"

var benchCfg = &Config{
	Secret:   "super-secret-key-for-benchmark-32b",
	Issuer:   "wapgo",
	Audience: "wapgo-client",
}

func BenchmarkSign(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Sign("user-123", []string{"admin", "user"}, benchCfg)
	}
}

func BenchmarkVerify(b *testing.B) {
	token, _ := Sign("user-123", []string{"admin"}, benchCfg)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Verify(token, benchCfg)
	}
}

func BenchmarkSignVerifyRoundTrip(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		tok, _ := Sign("user-abc", []string{"user"}, benchCfg)
		_, _ = Verify(tok, benchCfg)
	}
}

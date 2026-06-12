package crypto

import (
	"strings"
	"testing"
)

func testKey() []byte { return []byte("12345678901234567890123456789012") } // 32 bytes

func TestNewEncryptor_InvalidKeySize(t *testing.T) {
	_, err := NewEncryptor([]byte("short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc, err := NewEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plains := []string{"hello", "user@example.com", "081234567890", "", "multi\nline\tvalue"}
	for _, p := range plains {
		ct, err := enc.Encrypt(p)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", p, err)
		}
		got, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", ct, err)
		}
		if got != p {
			t.Fatalf("round-trip mismatch: want %q got %q", p, got)
		}
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	enc, _ := NewEncryptor(testKey())
	ct1, _ := enc.Encrypt("same plaintext")
	ct2, _ := enc.Encrypt("same plaintext")
	if ct1 == ct2 {
		t.Fatal("expected different ciphertexts for same plaintext (non-deterministic nonce)")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	enc, _ := NewEncryptor(testKey())
	ct, _ := enc.Encrypt("secret")

	// Flip a byte near the end.
	b := []byte(ct)
	b[len(b)-1] ^= 0xFF
	_, err := enc.Decrypt(string(b))
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	enc, _ := NewEncryptor(testKey())
	_, err := enc.Decrypt("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	enc, _ := NewEncryptor(testKey())
	// Valid base64 of only 2 bytes — shorter than GCM nonce size (12).
	_, err := enc.Decrypt("dGVzdA==") // "test" in base64
	if err == nil {
		t.Fatal("expected error for payload shorter than nonce")
	}
}

func TestEncrypt_OutputIsBase64URL(t *testing.T) {
	enc, _ := NewEncryptor(testKey())
	ct, _ := enc.Encrypt("data")
	// base64url should not contain '+' or '/'
	if strings.ContainsAny(ct, "+/") {
		t.Fatal("expected base64url encoding without + or /")
	}
}

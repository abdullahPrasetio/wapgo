// Package crypto provides AES-256-GCM authenticated encryption for PII fields.
// A 32-byte key is required (set via FIELD_ENCRYPTION_KEY env var, loaded by config).
// Each call to Encrypt produces a unique nonce so ciphertext is non-deterministic.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrKeySize is returned when the key is not exactly 32 bytes (AES-256).
	ErrKeySize = errors.New("encryption key must be exactly 32 bytes")
	// ErrInvalidCiphertext is returned when decryption or authentication fails.
	ErrInvalidCiphertext = errors.New("invalid or corrupted ciphertext")
)

// Encryptor encrypts and decrypts individual string fields using AES-256-GCM.
// Use NewEncryptor to create one; a single Encryptor is safe for concurrent use.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates an Encryptor from a 32-byte key.
// The key must be loaded from ENV or a secrets manager — never hard-code it.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns a base64url-encoded string of
// [nonce || ciphertext+tag]. Each call generates a fresh random nonce.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	// Seal appends ciphertext+tag to nonce.
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes and decrypts a value produced by Encrypt.
// Returns ErrInvalidCiphertext if the payload is malformed or the GCM tag fails.
func (e *Encryptor) Decrypt(encoded string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	ns := e.gcm.NonceSize()
	if len(data) < ns {
		return "", ErrInvalidCiphertext
	}
	nonce, ciphertext := data[:ns], data[ns:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plaintext), nil
}

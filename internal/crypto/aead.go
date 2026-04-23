// Package crypto provides small wrappers around stdlib crypto primitives
// used by the authentication subsystem. All exported types are safe for
// concurrent use by multiple goroutines unless noted.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// AEADKeySize is the required key length in bytes for NewAEAD
// (256-bit keys). Callers typically read 64 hex chars from env and
// DecodeKey them.
const AEADKeySize = 32

// ErrInvalidKey is returned by NewAEAD / DecodeKey when the key is not
// exactly AEADKeySize bytes.
var ErrInvalidKey = errors.New("aead: key must be 32 bytes")

// ErrInvalidNonce is returned by Open when the stored nonce length does
// not match the AEAD's NonceSize. Indicates schema drift or data
// corruption, not an attack.
var ErrInvalidNonce = errors.New("aead: invalid nonce length")

// AEAD wraps an AES-256-GCM cipher keyed by the 32-byte key passed to
// NewAEAD. Seal returns a random 96-bit nonce alongside the ciphertext;
// store both in adjacent columns. Open rejects tampered inputs via GCM's
// built-in authentication tag.
type AEAD struct {
	aead cipher.AEAD
}

// NewAEAD constructs an AES-256-GCM AEAD from a 32-byte key.
func NewAEAD(key []byte) (*AEAD, error) {
	if len(key) != AEADKeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}
	return &AEAD{aead: gcm}, nil
}

// DecodeKey parses a hex-encoded 32-byte key (the form read from the
// TOTP_ENCRYPTION_KEY env var). Returns ErrInvalidKey on wrong length,
// a wrapped error on non-hex input.
func DecodeKey(hexKey string) ([]byte, error) {
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("aead: key must be hex-encoded: %w", err)
	}
	if len(b) != AEADKeySize {
		return nil, ErrInvalidKey
	}
	return b, nil
}

// Seal encrypts plaintext and returns (ciphertext, nonce). The nonce is
// 96 random bits from crypto/rand; GCM requires a fresh nonce per
// encryption under the same key, so callers MUST NOT reuse an existing
// nonce.
func (a *AEAD) Seal(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, a.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("aead: generate nonce: %w", err)
	}
	ciphertext = a.aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Open decrypts (ciphertext, nonce). Returns an error if the nonce
// length is wrong or the ciphertext fails authentication (tampered or
// wrong key). The error is deliberately opaque — callers should not
// surface it to users.
func (a *AEAD) Open(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != a.aead.NonceSize() {
		return nil, ErrInvalidNonce
	}
	plaintext, err := a.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aead: authentication failed")
	}
	return plaintext, nil
}

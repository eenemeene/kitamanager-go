package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

// randomKey returns a fresh 32-byte AES key for tests.
func randomKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, AEADKeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return k
}

func TestAEAD_RoundTrip(t *testing.T) {
	a, err := NewAEAD(randomKey(t))
	if err != nil {
		t.Fatalf("NewAEAD: %v", err)
	}
	plaintext := []byte("ABCD1234EFGH5678IJKL") // base32 TOTP secret-sized

	ct, nonce, err := a.Seal(plaintext)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	got, err := a.Open(ct, nonce)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("Open = %q, want %q", got, plaintext)
	}
}

func TestAEAD_TamperedCiphertextRejected(t *testing.T) {
	a, _ := NewAEAD(randomKey(t))
	ct, nonce, _ := a.Seal([]byte("secret"))

	// Flip one byte in the ciphertext.
	ct[0] ^= 0x01

	if _, err := a.Open(ct, nonce); err == nil {
		t.Fatal("expected auth error on tampered ciphertext, got nil")
	}
}

func TestAEAD_TamperedNonceRejected(t *testing.T) {
	a, _ := NewAEAD(randomKey(t))
	ct, nonce, _ := a.Seal([]byte("secret"))

	// Flip one byte in the nonce.
	nonce[0] ^= 0x01

	if _, err := a.Open(ct, nonce); err == nil {
		t.Fatal("expected auth error on tampered nonce, got nil")
	}
}

func TestAEAD_KeyMismatchRejected(t *testing.T) {
	a1, _ := NewAEAD(randomKey(t))
	a2, _ := NewAEAD(randomKey(t))

	ct, nonce, _ := a1.Seal([]byte("secret"))

	if _, err := a2.Open(ct, nonce); err == nil {
		t.Fatal("expected auth error when decrypting with wrong key, got nil")
	}
}

func TestAEAD_NonceUniqueness(t *testing.T) {
	// Seal the same plaintext many times; nonces must all differ. GCM
	// silently produces catastrophic key recovery under nonce reuse, so
	// this is a must-have regression guard for the Seal() implementation.
	a, _ := NewAEAD(randomKey(t))
	seen := make(map[string]bool, 1000)
	for i := range 1000 {
		_, nonce, err := a.Seal([]byte("x"))
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		key := string(nonce)
		if seen[key] {
			t.Fatalf("nonce collision at iter %d", i)
		}
		seen[key] = true
	}
}

func TestAEAD_WrongLengthNonce(t *testing.T) {
	a, _ := NewAEAD(randomKey(t))
	ct, _, _ := a.Seal([]byte("secret"))

	if _, err := a.Open(ct, []byte{1, 2, 3}); err != ErrInvalidNonce {
		t.Errorf("expected ErrInvalidNonce for short nonce, got %v", err)
	}
}

func TestNewAEAD_WrongKeySize(t *testing.T) {
	if _, err := NewAEAD(make([]byte, 16)); err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for 16-byte key, got %v", err)
	}
	if _, err := NewAEAD(make([]byte, 64)); err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for 64-byte key, got %v", err)
	}
}

func TestDecodeKey_Valid(t *testing.T) {
	raw := randomKey(t)
	decoded, err := DecodeKey(hex.EncodeToString(raw))
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Error("DecodeKey did not round-trip")
	}
}

func TestDecodeKey_BadHex(t *testing.T) {
	if _, err := DecodeKey("not-hex-at-all!!"); err == nil {
		t.Fatal("expected error for non-hex input")
	}
}

func TestDecodeKey_WrongLength(t *testing.T) {
	// 16-byte key hex-encoded is 32 chars; should be rejected.
	short := strings.Repeat("ab", 16)
	if _, err := DecodeKey(short); err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for 16-byte decoded key, got %v", err)
	}
}

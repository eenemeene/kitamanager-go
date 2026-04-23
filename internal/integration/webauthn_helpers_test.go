//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"testing"
)

// extractChallengeFromOptions pulls the `challenge` base64url string
// out of a PublicKeyCredentialCreationOptions or …RequestOptions
// JSON blob. `wrap` optionally nests the lookup inside a top-level
// key (the creation path returns {"creation_options": {...}}).
func extractChallengeFromOptions(t *testing.T, raw []byte, wrap string) []byte {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode options: %v", err)
	}
	if wrap != "" {
		inner, ok := m[wrap]
		if !ok {
			t.Fatalf("options missing wrap key %q: %s", wrap, raw)
		}
		if err := json.Unmarshal(inner, &m); err != nil {
			t.Fatalf("decode inner %q: %v", wrap, err)
		}
	}
	// The go-webauthn library nests creation options one more level
	// under "publicKey" (matches the browser shape). Request options
	// do the same.
	if pk, ok := m["publicKey"]; ok {
		if err := json.Unmarshal(pk, &m); err != nil {
			t.Fatalf("decode publicKey: %v", err)
		}
	}
	challengeRaw, ok := m["challenge"]
	if !ok {
		t.Fatalf("options missing challenge: %s", raw)
	}
	var challengeStr string
	if err := json.Unmarshal(challengeRaw, &challengeStr); err != nil {
		t.Fatalf("challenge is not a string: %s", challengeRaw)
	}
	bytes, err := base64.RawURLEncoding.DecodeString(challengeStr)
	if err != nil {
		// Try the standard alphabet with padding as a fallback.
		bytes, err = base64.StdEncoding.DecodeString(challengeStr)
		if err != nil {
			t.Fatalf("decode challenge %q: %v", challengeStr, err)
		}
	}
	return bytes
}

// encodeUserHandle mirrors internal/webauthn/webauthn.go's 8-byte
// big-endian uint64 encoding so the assertion's userHandle matches
// what the server commits to at registration time.
func encodeUserHandle(userID uint) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(userID))
	return buf
}

// marshalRawMessage turns anything json-encodable into the
// json.RawMessage the DTO fields expect.
func marshalRawMessage(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	return b
}

// uintToStr formats an unsigned int as a URL path segment.
func uintToStr(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

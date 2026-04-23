package webauthn

import (
	"bytes"
	"io"
)

// bytesReader wraps a byte slice in an io.Reader — the go-webauthn
// parse helpers want an io.Reader and the usual one-liner is to
// hand them a *bytes.Reader.
func bytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

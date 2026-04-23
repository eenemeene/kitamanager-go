// Package webauthn wraps github.com/go-webauthn/webauthn so the rest
// of the service layer has a thin, app-specific surface to call
// against. The wrapper makes three things easier:
//
//  1. Version pinning. go-webauthn is still pre-1.0 and has shipped
//     breaking changes on v0.15 → v0.16 → v0.17. Containing the
//     import footprint to this file means a library upgrade is a
//     focused change in one place.
//  2. Test seams. The registration and assertion verification helpers
//     are exported with types that our tests can construct from a
//     synthetic ES256 authenticator without dragging the full WebAuthn
//     library into the test package.
//  3. UserHandle / UserID encoding. WebAuthn's User interface wants
//     a byte slice for the user handle; we derive it deterministically
//     from our numeric user_id so the same user always gets the same
//     handle across registrations.
package webauthn

import (
	"encoding/binary"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Config describes the RP identity. All three fields are required and
// validated at service construction; origin must be an exact string
// match of the scheme+host(+port) of every page that calls
// navigator.credentials.*.
type Config struct {
	RPID        string
	RPName      string
	RPOrigins   []string
	DisplayName string
}

// Service is the thin wrapper the rest of the backend depends on. It
// owns a configured *webauthn.WebAuthn from the go-webauthn library
// plus the helpers to marshal a model user into the library's User
// interface.
type Service struct {
	lib *webauthn.WebAuthn
}

// New constructs the wrapper. Fails early on an invalid config
// (empty RP id, no origins, etc) so startup won't silently accept
// a broken WebAuthn setup and then fail every ceremony at runtime.
func New(cfg Config) (*Service, error) {
	if cfg.RPID == "" {
		return nil, fmt.Errorf("webauthn: RPID is required")
	}
	if cfg.RPName == "" {
		return nil, fmt.Errorf("webauthn: RPName is required")
	}
	if len(cfg.RPOrigins) == 0 {
		return nil, fmt.Errorf("webauthn: at least one RPOrigin is required")
	}
	w, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn: construct library: %w", err)
	}
	return &Service{lib: w}, nil
}

// Lib returns the underlying library handle for the rare call sites
// that need features not wrapped here. Prefer the method helpers on
// Service; reach for Lib() only when the wrapper has no equivalent.
func (s *Service) Lib() *webauthn.WebAuthn { return s.lib }

// User implements the webauthn.User interface for our app model.
// credentials is the per-user list of already-registered WebAuthn
// credentials, used by the library to populate allowCredentials[] on
// authentication ceremonies. Registration flows pass an empty slice.
type User struct {
	id          uint
	email       string
	displayName string
	credentials []webauthn.Credential
}

// NewUser builds a User facade from our internal user row. email is
// required by WebAuthn's User interface even though we don't use it
// for cryptographic purposes — the authenticator may surface it to
// the user when listing stored credentials.
func NewUser(id uint, email, displayName string, credentials []webauthn.Credential) *User {
	return &User{id: id, email: email, displayName: displayName, credentials: credentials}
}

// WebAuthnID returns the user handle the RP commits to during
// registration. We derive it deterministically from our numeric user
// id so the same user gets the same handle even if registration is
// re-run. 8 bytes big-endian uint64 is enough — WebAuthn spec permits
// any byte slice up to 64 bytes.
func (u *User) WebAuthnID() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(u.id))
	return buf
}

// WebAuthnName returns the RFC-5321 account identifier shown to the
// user in authenticator pickers. Email is the right choice here —
// it's stable and already unique in our data model.
func (u *User) WebAuthnName() string { return u.email }

// WebAuthnDisplayName returns the human-facing name. Most
// authenticators prefer this over the name when prompting.
func (u *User) WebAuthnDisplayName() string { return u.displayName }

// WebAuthnCredentials returns the list of credentials already
// registered for this user. Populates allowCredentials[] on
// authentication; empty on registration.
func (u *User) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// DecodeUserIDFromHandle is the inverse of WebAuthnID: takes the
// byte slice reported by an assertion's userHandle and decodes our
// numeric user_id back out. Used on passwordless flows (future PR)
// to resolve the user; for the current second-factor flow we already
// know the user and only cross-check.
func DecodeUserIDFromHandle(handle []byte) (uint, error) {
	if len(handle) != 8 {
		return 0, fmt.Errorf("webauthn: user handle must be 8 bytes, got %d", len(handle))
	}
	return uint(binary.BigEndian.Uint64(handle)), nil
}

// ParseCreationResponse parses the raw JSON body of the registration
// response (navigator.credentials.create → JSON). Kept here so
// call sites don't import protocol directly.
func ParseCreationResponse(raw []byte) (*protocol.ParsedCredentialCreationData, error) {
	return protocol.ParseCredentialCreationResponseBody(bytesReader(raw))
}

// ParseAssertionResponse parses the raw JSON body of an assertion
// response (navigator.credentials.get → JSON).
func ParseAssertionResponse(raw []byte) (*protocol.ParsedCredentialAssertionData, error) {
	return protocol.ParseCredentialRequestResponseBody(bytesReader(raw))
}

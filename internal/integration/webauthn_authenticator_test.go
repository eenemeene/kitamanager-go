//go:build integration

package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// syntheticAuthenticator is an in-memory WebAuthn authenticator
// that produces real attestation and assertion blobs the
// go-webauthn library can verify. Used by integration tests so we
// exercise the full signature-check / challenge / origin / rpIdHash
// verification path end-to-end without a physical security key.
//
// The authenticator uses an ES256 (ECDSA-P256-SHA256) keypair per
// WebAuthn's COSE algorithm -7. Attestation format is "none" —
// browsers with attestation: "none" return a null attStmt too, so
// this matches our app's real-world config.
type syntheticAuthenticator struct {
	privateKey *ecdsa.PrivateKey
	// Chosen per-test; 16 bytes is the AAGUID length. For "none"
	// attestation the browser zeros this, so we do too.
	aaguid [16]byte
	// One authenticator stores one credential in our simple model
	// (the integration tests never enrol more than one per user).
	credentialID []byte
	signCount    uint32
	rpID         string
}

func newSyntheticAuthenticator(t *testing.T, rpID string) *syntheticAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ES256 key: %v", err)
	}
	credID := make([]byte, 32)
	if _, err := rand.Read(credID); err != nil {
		t.Fatalf("gen credential id: %v", err)
	}
	return &syntheticAuthenticator{
		privateKey:   key,
		credentialID: credID,
		rpID:         rpID,
	}
}

// authData builds the 37+ byte RP-ID-hash-plus-flags-plus-counter
// structure per WebAuthn §6.1. When `includeAttestedCredentialData`
// is true (registration), the extended form also carries AAGUID +
// credentialId + COSE public key after the core header.
func (a *syntheticAuthenticator) authData(includeAttestedCredentialData, userVerified bool) []byte {
	rpIDHash := sha256.Sum256([]byte(a.rpID))
	var flags byte
	flags |= 1 // UP (User Present)
	if userVerified {
		flags |= 1 << 2 // UV (User Verified)
	}
	if includeAttestedCredentialData {
		flags |= 1 << 6 // AT (attested credential data present)
	}

	buf := make([]byte, 0, 256)
	buf = append(buf, rpIDHash[:]...)
	buf = append(buf, flags)
	sc := make([]byte, 4)
	binary.BigEndian.PutUint32(sc, a.signCount)
	buf = append(buf, sc...)

	if includeAttestedCredentialData {
		buf = append(buf, a.aaguid[:]...)
		// credential id length (2 bytes big-endian) + credential id
		credIDLen := make([]byte, 2)
		binary.BigEndian.PutUint16(credIDLen, uint16(len(a.credentialID)))
		buf = append(buf, credIDLen...)
		buf = append(buf, a.credentialID...)
		// COSE_Key for the ES256 public key (map):
		//   1 (kty)  = 2 (EC2)
		//   3 (alg)  = -7 (ES256)
		//   -1 (crv) = 1 (P-256)
		//   -2 (x)   = <32 bytes X>
		//   -3 (y)   = <32 bytes Y>
		x := a.privateKey.PublicKey.X.FillBytes(make([]byte, 32))
		y := a.privateKey.PublicKey.Y.FillBytes(make([]byte, 32))
		coseKey := map[int]any{
			1:  2,
			3:  -7,
			-1: 1,
			-2: x,
			-3: y,
		}
		enc, err := cbor.Marshal(coseKey)
		if err != nil {
			panic(fmt.Sprintf("cbor encode cose key: %v", err))
		}
		buf = append(buf, enc...)
	}
	return buf
}

// makeAttestationObject returns the CBOR attestation object for a
// registration ceremony. Uses "none" attestation format — matches
// what the browser sends back when we configured
// attestation: "none" on the creation options.
func (a *syntheticAuthenticator) makeAttestationObject(challenge []byte, origin string) (attestationObjectB64, clientDataJSONB64 string, err error) {
	authData := a.authData(true, true)
	attObj := map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": authData,
	}
	enc, err := cbor.Marshal(attObj)
	if err != nil {
		return "", "", fmt.Errorf("cbor encode attestationObject: %w", err)
	}
	clientData := map[string]any{
		"type":        "webauthn.create",
		"challenge":   base64.RawURLEncoding.EncodeToString(challenge),
		"origin":      origin,
		"crossOrigin": false,
	}
	cdJSON, err := json.Marshal(clientData)
	if err != nil {
		return "", "", fmt.Errorf("marshal clientDataJSON: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(enc),
		base64.RawURLEncoding.EncodeToString(cdJSON), nil
}

// makeAssertionResponse returns the PublicKeyCredentialJSON body
// for an authentication ceremony. Signs authData ‖ SHA256(clientDataJSON)
// with the ES256 private key so the server's signature verify passes.
func (a *syntheticAuthenticator) makeAssertionResponse(challenge []byte, origin string, userHandle []byte) (map[string]any, error) {
	a.signCount++ // fresh authenticator starts at 0; first assertion bumps to 1
	authData := a.authData(false, true)
	clientData := map[string]any{
		"type":        "webauthn.get",
		"challenge":   base64.RawURLEncoding.EncodeToString(challenge),
		"origin":      origin,
		"crossOrigin": false,
	}
	cdJSON, err := json.Marshal(clientData)
	if err != nil {
		return nil, fmt.Errorf("marshal clientDataJSON: %w", err)
	}
	cdHash := sha256.Sum256(cdJSON)
	msg := append(append([]byte{}, authData...), cdHash[:]...)
	digest := sha256.Sum256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, a.privateKey, digest[:])
	if err != nil {
		return nil, fmt.Errorf("ecdsa sign: %w", err)
	}
	sig := encodeECDSASignature(r, s)

	return map[string]any{
		"id":    base64.RawURLEncoding.EncodeToString(a.credentialID),
		"rawId": base64.RawURLEncoding.EncodeToString(a.credentialID),
		"type":  "public-key",
		"response": map[string]any{
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(cdJSON),
			"signature":         base64.RawURLEncoding.EncodeToString(sig),
			"userHandle":        base64.RawURLEncoding.EncodeToString(userHandle),
		},
		"clientExtensionResults": map[string]any{},
	}, nil
}

// encodeECDSASignature formats (r, s) as the ASN.1 DER sequence the
// WebAuthn spec requires. A standalone implementation beats pulling
// another dep for this one function.
func encodeECDSASignature(r, s *big.Int) []byte {
	type ecdsaSig struct {
		R, S *big.Int
	}
	// crypto/x509's asn1 encoding is easier to reach:
	// asn1.Marshal handles the DER SEQUENCE for us.
	return asn1MarshalECDSASig(r, s)
}

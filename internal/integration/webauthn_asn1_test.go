//go:build integration

package integration

import (
	"encoding/asn1"
	"math/big"
)

// asn1MarshalECDSASig produces the DER SEQUENCE { INTEGER r, INTEGER s }
// that WebAuthn assertion signatures use. Kept in its own file so
// the single-purpose helper doesn't clutter the authenticator.
func asn1MarshalECDSASig(r, s *big.Int) []byte {
	type ecdsaSig struct{ R, S *big.Int }
	out, err := asn1.Marshal(ecdsaSig{R: r, S: s})
	if err != nil {
		panic("asn1 marshal ecdsa sig: " + err.Error())
	}
	return out
}

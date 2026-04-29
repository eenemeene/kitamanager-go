package models

import (
	"encoding/json"
	"time"
)

// Factor types. These are the values stored in factors.type and exposed
// in the API. Each new factor type needs (a) a constant here, (b) a
// subtable migration (see 000010 for totp/backup, 000012 for webauthn),
// (c) a verifier implementation in service/factor.go's switch
// statements, (d) the CHECK constraint in the latest migration
// loosened to include it.
const (
	FactorTypeTOTP        = "totp"
	FactorTypeBackupCodes = "backup_codes"
	FactorTypeWebAuthn    = "webauthn"
)

// Factor is the parent row in the factor-generic data model. Every
// authentication factor a user has — TOTP app, recovery codes, later
// WebAuthn passkey — shares this parent row and carries type-specific
// data in a paired subtable keyed by factor_id.
type Factor struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	Type       string     `gorm:"size:32;not null" json:"type"`
	Label      *string    `gorm:"size:100" json:"label,omitempty"`
	EnabledAt  *time.Time `json:"enabled_at,omitempty" format:"date-time"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" format:"date-time"`
	// ActivationFailures is incremented on every wrong code submitted
	// to /activate. When it reaches FactorActivationFailureLimit the
	// service layer deletes the pending row, forcing re-enrolment.
	// Only meaningful while EnabledAt IS NULL.
	ActivationFailures int `gorm:"not null;default:0" json:"-"`
	// RegistrationChallenge holds the WebAuthn server-issued challenge
	// (typically a serialised go-webauthn SessionData blob) between
	// POST /factors and POST /factors/:id/activate. TOTP and
	// backup_codes factors leave this NULL. Cleared on activation or
	// when the row is swept by CleanupAbandonedPending.
	RegistrationChallenge          []byte     `gorm:"type:bytea" json:"-"`
	RegistrationChallengeExpiresAt *time.Time `json:"-" format:"date-time"`
	CreatedAt                      time.Time  `gorm:"not null" json:"created_at" format:"date-time"`
}

// TableName is explicit because GORM's default would otherwise be "factors"
// — which is what we want — but being explicit here future-proofs against
// casing changes in the pluralizer.
func (Factor) TableName() string { return "factors" }

// FactorTOTPSecret holds the AES-GCM-encrypted TOTP secret and the
// last-used time-step counter that defeats replay attacks.
// The plaintext base32 secret is never stored on disk.
type FactorTOTPSecret struct {
	FactorID         uint   `gorm:"primaryKey" json:"-"`
	SecretCiphertext []byte `gorm:"type:bytea;not null;column:secret_ciphertext" json:"-"`
	SecretNonce      []byte `gorm:"type:bytea;not null;column:secret_nonce" json:"-"`
	LastUsedStep     *int64 `json:"-"`
}

func (FactorTOTPSecret) TableName() string { return "factor_totp_secrets" }

// FactorBackupCode is one single-use recovery code. `used_at IS NULL`
// means available; the verify path flips it atomically with a single
// UPDATE so concurrent attempts can't double-spend a code.
type FactorBackupCode struct {
	ID        uint       `gorm:"primaryKey" json:"-"`
	FactorID  uint       `gorm:"not null;index" json:"-"`
	CodeHash  string     `gorm:"type:char(64);not null" json:"-"`
	UsedAt    *time.Time `json:"-" format:"date-time"`
	CreatedAt time.Time  `gorm:"not null" json:"-" format:"date-time"`
}

func (FactorBackupCode) TableName() string { return "factor_backup_codes" }

// FactorWebAuthnCredential is the per-factor subtable row for
// WebAuthn / FIDO2 credentials. One row per factor; the unique index
// on credential_id enforces the "credential must be globally unique"
// rule from WebAuthn L3 section 7.1 step 22 (an authenticator binds
// to exactly one user account).
//
// Public keys are stored plaintext — they are public by definition,
// and encryption would only confuse operators later. The credential
// id and AAGUID are not sensitive enough to encrypt either. Sign
// count, BE/BS flags, and UV-initialised are consulted on every
// assertion and kept up to date by the verify path.
type FactorWebAuthnCredential struct {
	FactorID          uint      `gorm:"primaryKey" json:"-"`
	CredentialID      []byte    `gorm:"type:bytea;not null;uniqueIndex:idx_factor_webauthn_credential_id" json:"-"`
	PublicKey         []byte    `gorm:"type:bytea;not null" json:"-"`
	AAGUID            []byte    `gorm:"type:bytea;column:aaguid" json:"-"`
	SignCount         int64     `gorm:"not null;default:0" json:"-"`
	Transports        string    `gorm:"size:255" json:"-"`
	AttestationFormat string    `gorm:"size:64" json:"-"`
	BackupEligible    bool      `gorm:"not null;default:false" json:"-"`
	BackupState       bool      `gorm:"not null;default:false" json:"-"`
	UVInitialized     bool      `gorm:"not null;default:false" json:"-"`
	CreatedAt         time.Time `gorm:"not null" json:"-" format:"date-time"`
}

func (FactorWebAuthnCredential) TableName() string { return "factor_webauthn_credentials" }

// FactorEnrollmentPayload is the type-specific blob included in the
// response to `POST /users/:userId/factors` — what the client needs to
// complete enrollment. For TOTP it's a base32 secret + otpauth URI.
// For WebAuthn (future) it will be a `PublicKeyCredentialCreationOptions`.
// The field is an `any` so the handler stays generic; each verifier
// produces its own shape.
type FactorEnrollmentPayload any

// TOTPEnrollmentPayload is what `POST /users/:userId/factors {type:"totp"}`
// returns in the `enrollment` field. Shown once; never retrievable again.
type TOTPEnrollmentPayload struct {
	Secret     string `json:"secret" example:"JBSWY3DPEHPK3PXP"`
	OTPAuthURI string `json:"otpauth_uri" example:"otpauth://totp/KitaManager:..."`
}

// WebAuthnEnrollmentPayload is what `POST /users/:userId/factors
// {type:"webauthn"}` returns. CreationOptions is the raw JSON from
// the go-webauthn library — the client hands it straight to
// `navigator.credentials.create({publicKey: creationOptions})`.
type WebAuthnEnrollmentPayload struct {
	CreationOptions json.RawMessage `json:"creation_options" swaggertype:"object"`
}

// FactorResponse is the per-factor response row. Factor-specific extras
// (like backup_codes_remaining) are added by the handler before
// marshalling.
type FactorResponse struct {
	ID         uint       `json:"id" example:"42"`
	Type       string     `json:"type" enums:"totp,backup_codes,webauthn" example:"totp"`
	Label      *string    `json:"label,omitempty" example:"Authenticator app"`
	EnabledAt  *time.Time `json:"enabled_at,omitempty" format:"date-time"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" format:"date-time"`
	CreatedAt  time.Time  `json:"created_at" format:"date-time"`
	// Activated is a derived convenience field: true iff EnabledAt is set.
	// Clients that don't need the timestamp can just read this boolean.
	Activated bool `json:"activated" example:"true"`
	// BackupCodesRemaining is populated only for factors of type
	// "backup_codes"; nil otherwise.
	BackupCodesRemaining *int `json:"backup_codes_remaining,omitempty" example:"7"`
	// Enrollment is populated only on the `POST` response during
	// initial enrollment; subsequent GETs omit it.
	Enrollment FactorEnrollmentPayload `json:"enrollment,omitempty"`
}

// ToResponse flattens a Factor row into the API shape. Type-specific
// extras (enrollment, backup_codes_remaining) are set separately by the
// caller.
func (f *Factor) ToResponse() FactorResponse {
	return FactorResponse{
		ID:         f.ID,
		Type:       f.Type,
		Label:      f.Label,
		EnabledAt:  f.EnabledAt,
		LastUsedAt: f.LastUsedAt,
		CreatedAt:  f.CreatedAt,
		Activated:  f.EnabledAt != nil,
	}
}

// FactorListResponse wraps the list so the endpoint stays forward-
// compatible with future top-level fields (pagination, counts).
type FactorListResponse struct {
	Factors []FactorResponse `json:"factors"`
}

// LoginFactorDescriptor is the minimal factor shape that /login returns
// in its pending_mfa response, and that POST /auth/mfa/verify echoes
// back alongside its error responses. Kept deliberately small — only
// the fields the unauthenticated client needs to present a factor
// chooser or drive a WebAuthn ceremony. No created_at, last_used_at,
// backup_codes_remaining, or AAGUID — leaking post-login metadata to
// an unauthenticated caller is exactly what this type is designed to
// prevent.
type LoginFactorDescriptor struct {
	ID    uint    `json:"id" example:"42"`
	Type  string  `json:"type" enums:"totp,backup_codes,webauthn" example:"totp"`
	Label *string `json:"label,omitempty" example:"iPhone"`
	// CredentialID is the base64url-encoded credential id, populated
	// only for webauthn factors. The browser needs it to narrow
	// allowCredentials[] on navigator.credentials.get(), which lets
	// the authenticator pre-filter before prompting the user. Safe
	// to expose: credential ids are public by WebAuthn design.
	CredentialID *string `json:"credential_id,omitempty" example:"AQIDBAU..."`
}

// FactorEnrollRequest is the request body for `POST /users/:userId/factors`.
// Password re-entry is required (step-up) to install a factor — a stolen
// session cookie alone must not be able to plant an authenticator the
// attacker controls.
type FactorEnrollRequest struct {
	Type     string  `json:"type" binding:"required" example:"totp"`
	Label    *string `json:"label,omitempty" example:"iPhone"`
	Password string  `json:"password" binding:"required" example:"yourcurrentpassword"`
}

// FactorActivateRequest is the body for `POST /users/:userId/factors/:id/activate`.
// Polymorphic across factor types:
//   - TOTP: Code is a 6-digit time-based code; WebAuthnResponse is unset.
//   - WebAuthn: WebAuthnResponse is the browser's PublicKeyCredential
//     JSON from navigator.credentials.create(); Code is unset.
//
// At least one of the two is required; the handler picks which branch
// to run based on the factor's stored type.
type FactorActivateRequest struct {
	Code             string          `json:"code,omitempty" example:"123456"`
	WebAuthnResponse json.RawMessage `json:"webauthn_response,omitempty" swaggertype:"object"`
}

// FactorActivateResponse carries the activation result. On the first
// primary-factor activation, `backup_codes` is non-nil and contains the
// fresh set the user MUST save now (shown once).
type FactorActivateResponse struct {
	Activated   bool                `json:"activated" example:"true"`
	BackupCodes *BackupCodesPayload `json:"backup_codes,omitempty"`
}

// BackupCodesPayload is the one-time presentation of a set of backup
// codes: the factor id so clients can address it later, plus the raw
// codes. After this response, codes are only known to the user.
type BackupCodesPayload struct {
	FactorID uint     `json:"factor_id" example:"43"`
	Codes    []string `json:"codes" example:"hk7m-93px-2fnr"`
}

// FactorRegenerateRequest is the body for
// `POST /users/:userId/factors/:id/regenerate`. Only meaningful on
// backup_codes factors. Password re-entry required.
type FactorRegenerateRequest struct {
	Password string `json:"password" binding:"required" example:"yourcurrentpassword"`
}

// FactorLabelUpdateRequest is the body for `PATCH /users/:userId/factors/:id`.
// Label-only edit; no step-up required.
type FactorLabelUpdateRequest struct {
	Label *string `json:"label" example:"Old phone"`
}

// FactorDeleteRequest is the body for `DELETE /users/:userId/factors/:id`.
// Password is always required (step-up). `code` is a valid TOTP code
// OR a valid backup code; required when deleting the last primary
// factor, optional otherwise (the service enforces).
type FactorDeleteRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code,omitempty"`
}

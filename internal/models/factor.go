package models

import "time"

// Factor types. These are the values stored in factors.type and exposed
// in the API. Adding a new factor type (e.g. "webauthn") requires:
// (a) a new constant here, (b) a new subtable migration, (c) a new
// verifier implementation in service/, (d) the CHECK constraint in
// migration 000010 loosened.
const (
	FactorTypeTOTP        = "totp"
	FactorTypeBackupCodes = "backup_codes"
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
	EnabledAt  *time.Time `json:"enabled_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `gorm:"not null" json:"created_at"`
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
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time  `gorm:"not null" json:"-"`
}

func (FactorBackupCode) TableName() string { return "factor_backup_codes" }

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

// FactorResponse is the per-factor response row. Factor-specific extras
// (like backup_codes_remaining) are added by the handler before
// marshalling.
type FactorResponse struct {
	ID         uint       `json:"id" example:"42"`
	Type       string     `json:"type" example:"totp"`
	Label      *string    `json:"label,omitempty" example:"Authenticator app"`
	EnabledAt  *time.Time `json:"enabled_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
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
// `code` is a TOTP code for totp factors; backup_codes factors aren't
// activated this way (they're auto-created at first primary activation).
type FactorActivateRequest struct {
	Code string `json:"code" binding:"required" example:"123456"`
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

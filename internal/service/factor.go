package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	cryptopkg "github.com/eenemeene/kitamanager-go/internal/crypto"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	webauthnpkg "github.com/eenemeene/kitamanager-go/internal/webauthn"
)

// BackupCodeCount is how many codes are generated per set. 8 is the
// mainstream default; GitHub uses 16 but we lean toward the smaller,
// still-practical number.
const BackupCodeCount = 8

// FactorActivationFailureLimit caps the number of wrong TOTP codes the
// activation endpoint will accept before the pending factor is deleted.
// This closes the "attacker with a session cookie brute-forces the 6-
// digit TOTP window against a pending row" surface. Picked at 5 to
// match the typical login rate-limit window and leave room for legit
// fat-finger typos.
const FactorActivationFailureLimit = 5

// BackupCodeEntropyBytes is the size of each code before base32-ish
// encoding. 8 bytes = 64 bits of entropy per code. With 8 codes the
// total cumulative entropy is still well beyond brute force.
const BackupCodeEntropyBytes = 8

// totpSkewSteps is the TOTP library's ±step tolerance. One step
// (±30s) matches RFC 6238 recommendation for typical clock drift.
const totpSkewSteps = uint(1)

// totpPeriod is the TOTP window length in seconds. 30 is standard.
const totpPeriod = uint(30)

// FactorService is the business layer for factor-generic MFA. It
// orchestrates the parent `factors` table and per-type subtables via
// FactorStorer, wraps the TOTP library from pquerna/otp and the
// WebAuthn library wrapper from internal/webauthn, and emits audit
// events for every state transition.
type FactorService struct {
	factorStore  store.FactorStorer
	userStore    store.UserStorer
	aead         *cryptopkg.AEAD
	issuer       string
	webAuthn     *webauthnpkg.Service // may be nil if WebAuthn is not configured
	auditService *AuditService
}

// NewFactorService constructs the service. `aead` is the AES-GCM
// wrapper built from TOTP_ENCRYPTION_KEY; `issuer` is the string shown
// in the user's authenticator app; `webAuthn` may be nil if the
// deployment has not configured WEBAUTHN_* env vars, in which case
// any WebAuthn code path returns an apperror.BadRequest with a clear
// "WebAuthn not enabled" message.
func NewFactorService(
	factorStore store.FactorStorer,
	userStore store.UserStorer,
	aead *cryptopkg.AEAD,
	issuer string,
	webAuthn *webauthnpkg.Service,
	auditService *AuditService,
) *FactorService {
	return &FactorService{
		factorStore:  factorStore,
		userStore:    userStore,
		aead:         aead,
		issuer:       issuer,
		webAuthn:     webAuthn,
		auditService: auditService,
	}
}

// ListForUser returns every ACTIVATED factor for the user, flattened
// into response DTOs. Backup-code counts are attached for
// `backup_codes` factor rows.
func (s *FactorService) ListForUser(ctx context.Context, userID uint) ([]models.FactorResponse, error) {
	rows, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "list factors")
	}
	out := make([]models.FactorResponse, 0, len(rows))
	for i := range rows {
		r := rows[i].ToResponse()
		if rows[i].Type == models.FactorTypeBackupCodes {
			n, err := s.factorStore.CountUnusedBackupCodes(ctx, rows[i].ID)
			if err != nil {
				return nil, apperror.InternalWrap(err, "count backup codes")
			}
			r.BackupCodesRemaining = &n
		}
		out = append(out, r)
	}
	return out, nil
}

// GetForUser returns one factor scoped to the caller. Cross-user
// access returns NotFound (via the store's ownership-scoped WHERE).
func (s *FactorService) GetForUser(ctx context.Context, userID, factorID uint) (*models.FactorResponse, error) {
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("factor")
		}
		return nil, apperror.InternalWrap(err, "find factor")
	}
	r := f.ToResponse()
	if f.Type == models.FactorTypeBackupCodes {
		n, err := s.factorStore.CountUnusedBackupCodes(ctx, f.ID)
		if err != nil {
			return nil, apperror.InternalWrap(err, "count backup codes")
		}
		r.BackupCodesRemaining = &n
	}
	return &r, nil
}

// EnrollTOTP starts TOTP enrollment. Requires step-up (password re-
// entry) so a stolen session cookie alone cannot plant an
// authenticator the attacker controls.
//
// If a pending TOTP factor already exists for this user it is
// deleted first — "second enrollment attempt replaces the first"
// avoids orphan-row accumulation.
//
// Returns the factor row plus the enrollment payload (base32 secret +
// otpauth URI) shown to the user once.
//
// No audit event is emitted at this pending-row stage — the audited
// event is `factor_enrolled` on successful ActivateFactor. Abandoned
// pending rows get cleaned up by CleanupAbandonedPendingFactors and
// carry no security signal of their own.
func (s *FactorService) EnrollTOTP(ctx context.Context, userID uint, label *string, password string, accountLabel string) (*models.FactorResponse, error) {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return nil, err
	}

	// Replace any existing pending TOTP for this user.
	if existing, err := s.factorStore.FindPendingByUserAndType(ctx, userID, models.FactorTypeTOTP); err == nil {
		if _, delErr := s.factorStore.DeleteFactor(ctx, existing.ID, userID); delErr != nil {
			slog.Error("delete stale pending TOTP", "user_id", userID, "error", delErr)
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, apperror.InternalWrap(err, "find pending totp")
	}

	// Generate TOTP secret + otpauth URI via the library.
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountLabel,
		Period:      totpPeriod,
	})
	if err != nil {
		return nil, apperror.InternalWrap(err, "generate totp key")
	}

	// Encrypt the base32 secret at rest.
	ct, nonce, err := s.aead.Seal([]byte(key.Secret()))
	if err != nil {
		return nil, apperror.InternalWrap(err, "encrypt totp secret")
	}

	f := &models.Factor{
		UserID:    userID,
		Type:      models.FactorTypeTOTP,
		Label:     label,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.factorStore.CreateFactor(ctx, f); err != nil {
		return nil, apperror.InternalWrap(err, "create factor")
	}
	if err := s.factorStore.CreateTOTPSecret(ctx, &models.FactorTOTPSecret{
		FactorID:         f.ID,
		SecretCiphertext: ct,
		SecretNonce:      nonce,
	}); err != nil {
		return nil, apperror.InternalWrap(err, "create totp secret")
	}

	r := f.ToResponse()
	r.Enrollment = models.TOTPEnrollmentPayload{
		Secret:     key.Secret(),
		OTPAuthURI: key.URL(),
	}
	return &r, nil
}

// ActivateFactor finalises enrollment by verifying a code and
// stamping enabled_at. On the user's first-ever primary factor
// activation, also auto-creates the singleton `backup_codes` factor
// and returns the raw codes for one-time display.
//
// Returns Conflict (apperror.ErrConflict) if the factor was already
// activated by a concurrent request — the compare-and-set UPDATE in
// the store layer makes this a clean two-outcome operation.
//
// `req` is polymorphic by factor type: req.Code for TOTP, the
// req.WebAuthnResponse JSON blob for WebAuthn. The handler passes
// whichever the client sent; this method dispatches on the stored
// factor type.
func (s *FactorService) ActivateFactor(ctx context.Context, userID, factorID uint, req *models.FactorActivateRequest) (*models.FactorActivateResponse, error) {
	if req == nil {
		return nil, apperror.BadRequest("missing request body")
	}
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("factor")
		}
		return nil, apperror.InternalWrap(err, "find factor")
	}
	if f.EnabledAt != nil {
		return nil, apperror.Conflict("factor already activated")
	}

	switch f.Type {
	case models.FactorTypeTOTP:
		if err := s.verifyTOTPForActivation(ctx, f.ID, req.Code); err != nil {
			// Only count "wrong code" (401) as a failure — internal
			// errors (DB/crypto) shouldn't burn a retry budget.
			if errors.Is(err, apperror.ErrUnauthorized) {
				n, incErr := s.factorStore.IncrementActivationFailures(ctx, f.ID, userID)
				if incErr != nil {
					slog.Error("increment activation_failures", "factor_id", f.ID, "error", incErr)
					return nil, err
				}
				if n >= FactorActivationFailureLimit {
					if _, delErr := s.factorStore.DeleteFactor(ctx, f.ID, userID); delErr != nil {
						slog.Error("delete pending factor after activation limit", "factor_id", f.ID, "error", delErr)
					}
					s.auditService.LogFactorActivationLocked(ctx, userID, f.Type)
					return nil, apperror.TooManyRequests("too many wrong codes; re-enroll the factor")
				}
			}
			return nil, err
		}
	case models.FactorTypeWebAuthn:
		// WebAuthn activation runs a completely different ceremony:
		// parse + verify the attestation signature against the
		// challenge we issued on POST /factors. Uses no TOTP/activation-
		// failure machinery — WebAuthn authenticators don't take a
		// human-readable code so there's no brute-force surface here.
		if err := s.activateWebAuthnFactor(ctx, userID, f, req.WebAuthnResponse); err != nil {
			return nil, err
		}
	case models.FactorTypeBackupCodes:
		return nil, apperror.BadRequest("backup_codes factors are auto-activated; do not call activate on them")
	default:
		return nil, apperror.BadRequest(fmt.Sprintf("unsupported factor type: %s", f.Type))
	}

	// Atomic activation — compare-and-set on enabled_at=NULL.
	activated, err := s.factorStore.ActivateFactor(ctx, f.ID, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "activate factor")
	}
	if !activated {
		return nil, apperror.Conflict("factor already activated")
	}

	s.auditService.LogFactorEnrolled(ctx, userID, f.Type)

	// First-primary-factor auto-create: if the user now has exactly
	// one enabled primary factor (the one we just activated) and no
	// backup_codes factor yet, provision backup codes.
	var payload *models.BackupCodesPayload
	if f.Type != models.FactorTypeBackupCodes {
		created, codes, err := s.ensureBackupCodesFactor(ctx, userID)
		if err != nil {
			slog.Error("auto-create backup codes after first primary factor", "user_id", userID, "error", err)
		} else if created {
			payload = codes
		}
	}

	return &models.FactorActivateResponse{
		Activated:   true,
		BackupCodes: payload,
	}, nil
}

// DeleteFactor removes a factor. Password is always required
// (step-up). A code (TOTP or backup) is additionally required when
// the factor is a primary (non-backup) and would be the user's LAST
// primary — otherwise a stolen session could remove the only real
// factor and lock the user into password-only.
//
// When the last primary factor is deleted, any associated
// backup_codes factor is deleted too in the same transaction
// boundary (handled at the store layer via the parent FK cascade we
// emulate here with an explicit delete).
func (s *FactorService) DeleteFactor(ctx context.Context, userID, factorID uint, password, code string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}

	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return apperror.NotFound("factor")
		}
		return apperror.InternalWrap(err, "find factor")
	}

	// Determine if this delete would leave the user with no primary
	// factor — if so, require a code from ANY active factor as
	// additional proof.
	active, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return apperror.InternalWrap(err, "find active factors")
	}
	primariesRemaining := 0
	for _, g := range active {
		if g.ID != f.ID && g.Type != models.FactorTypeBackupCodes {
			primariesRemaining++
		}
	}
	isPrimary := f.Type != models.FactorTypeBackupCodes
	if isPrimary && primariesRemaining == 0 {
		if code == "" {
			return apperror.BadRequest("code is required when removing your last primary factor")
		}
		if err := s.verifyAnyFactorCode(ctx, userID, code); err != nil {
			return err
		}
	} else if code != "" {
		// Even when not strictly required, a provided code must be
		// valid — never accept a bad code silently.
		if err := s.verifyAnyFactorCode(ctx, userID, code); err != nil {
			return err
		}
	}

	// Delete.
	rows, err := s.factorStore.DeleteFactor(ctx, factorID, userID)
	if err != nil {
		return apperror.InternalWrap(err, "delete factor")
	}
	if rows == 0 {
		return apperror.NotFound("factor")
	}

	// If this was the last primary, also sweep backup_codes factor.
	if isPrimary && primariesRemaining == 0 {
		if bf, err := s.factorStore.FindBackupCodesFactor(ctx, userID); err == nil {
			if _, delErr := s.factorStore.DeleteFactor(ctx, bf.ID, userID); delErr != nil {
				slog.Error("sweep backup_codes after last primary delete", "user_id", userID, "error", delErr)
			}
		}
	}

	s.auditService.LogFactorDeleted(ctx, userID, f.Type)
	return nil
}

// UpdateLabel edits a factor's human-readable label. No step-up —
// renaming is not security-sensitive.
func (s *FactorService) UpdateLabel(ctx context.Context, userID, factorID uint, label *string) (*models.FactorResponse, error) {
	if label != nil {
		trimmed := strings.TrimSpace(*label)
		if len(trimmed) > 100 {
			return nil, apperror.BadRequest("label must be 100 characters or fewer")
		}
		if trimmed == "" {
			label = nil
		} else {
			label = &trimmed
		}
	}
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("factor")
		}
		return nil, apperror.InternalWrap(err, "find factor")
	}
	ok, err := s.factorStore.UpdateLabel(ctx, factorID, userID, label)
	if err != nil {
		return nil, apperror.InternalWrap(err, "update label")
	}
	if !ok {
		return nil, apperror.NotFound("factor")
	}
	s.auditService.LogFactorLabelUpdated(ctx, userID, factorID, f.Type)
	return s.GetForUser(ctx, userID, factorID)
}

// RegenerateBackupCodes replaces every existing backup code row for
// the factor (whether used or unused) with a fresh set. Step-up
// required. Only meaningful for backup_codes factors.
func (s *FactorService) RegenerateBackupCodes(ctx context.Context, userID, factorID uint, password string) (*models.BackupCodesPayload, error) {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return nil, err
	}
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("factor")
		}
		return nil, apperror.InternalWrap(err, "find factor")
	}
	if f.Type != models.FactorTypeBackupCodes {
		return nil, apperror.BadRequest("regenerate is only valid for backup_codes factors")
	}

	raw, hashed, err := generateBackupCodes(BackupCodeCount)
	if err != nil {
		return nil, apperror.InternalWrap(err, "generate backup codes")
	}
	rows := make([]models.FactorBackupCode, 0, len(hashed))
	for _, h := range hashed {
		rows = append(rows, models.FactorBackupCode{
			FactorID:  f.ID,
			CodeHash:  h,
			CreatedAt: time.Now().UTC(),
		})
	}
	if err := s.factorStore.ReplaceBackupCodes(ctx, f.ID, rows); err != nil {
		return nil, apperror.InternalWrap(err, "replace backup codes")
	}

	s.auditService.LogBackupCodesRegenerated(ctx, userID)

	return &models.BackupCodesPayload{FactorID: f.ID, Codes: raw}, nil
}

// CleanupAbandonedPendingFactors is called by the periodic GC job in
// cmd/api/main.go. Factors with enabled_at IS NULL that are older
// than the threshold are rows from abandoned enrollment flows.
func (s *FactorService) CleanupAbandonedPendingFactors(ctx context.Context, olderThan time.Duration) error {
	_, err := s.factorStore.CleanupAbandonedPending(ctx, olderThan)
	return err
}

// WebAuthnRegistrationChallengeLifetime caps how long a pending
// WebAuthn enrolment's challenge stays valid. 5 minutes matches the
// WebAuthn §1.2.1 example timeout and the pending_mfa session TTL
// used elsewhere in the two-step login flow.
const WebAuthnRegistrationChallengeLifetime = 5 * time.Minute

// EnrollWebAuthn begins a WebAuthn registration ceremony. Requires
// step-up (password re-entry) so a stolen session can't plant an
// authenticator attacker-controlled. If a pending WebAuthn factor
// already exists for this user + label combination, it is replaced —
// the user may simply have abandoned an earlier attempt.
//
// The returned enrollment payload wraps a PublicKeyCredentialCreationOptionsJSON
// blob that the client hands straight to `navigator.credentials.create()`.
// The server-side SessionData (challenge + expected UV + userHandle)
// is persisted on the factor row's registration_challenge column so
// the activate path can round-trip the verification.
//
// As with EnrollTOTP, no audit event is emitted for the pending row —
// the audited event is `factor_enrolled` on successful ActivateFactor.
func (s *FactorService) EnrollWebAuthn(ctx context.Context, userID uint, label *string, password, accountName, displayName string) (*models.FactorResponse, error) {
	if s.webAuthn == nil {
		return nil, apperror.BadRequest("WebAuthn is not enabled on this deployment")
	}
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return nil, err
	}

	// Replace any existing pending WebAuthn factor so the user can
	// restart a botched ceremony without leaking pending rows.
	if existing, err := s.factorStore.FindPendingByUserAndType(ctx, userID, models.FactorTypeWebAuthn); err == nil {
		if _, delErr := s.factorStore.DeleteFactor(ctx, existing.ID, userID); delErr != nil {
			slog.Error("delete stale pending webauthn", "user_id", userID, "error", delErr)
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, apperror.InternalWrap(err, "find pending webauthn")
	}

	// Collect the user's already-registered credentials so the
	// library can put them into excludeCredentials on the new
	// ceremony — prevents the same hardware key being registered
	// twice as two separate factors for the same user.
	existingCreds, err := s.gatherUserWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	wuser := webauthnpkg.NewUser(userID, accountName, displayName, existingCreds)

	// Create the pending factor row first so we have an id to key
	// the challenge against. No subtable row yet — that comes on
	// activate after attestation verifies.
	f := &models.Factor{
		UserID:    userID,
		Type:      models.FactorTypeWebAuthn,
		Label:     label,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.factorStore.CreateFactor(ctx, f); err != nil {
		return nil, apperror.InternalWrap(err, "create factor")
	}

	// attestation=none + preferred UV + non-discoverable credentials:
	// the second-factor defaults explained in PR 152's design doc.
	creation, sessionData, err := s.webAuthn.Lib().BeginRegistration(wuser)
	if err != nil {
		return nil, apperror.InternalWrap(err, "begin webauthn registration")
	}
	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return nil, apperror.InternalWrap(err, "marshal session data")
	}
	expiresAt := time.Now().UTC().Add(WebAuthnRegistrationChallengeLifetime)
	if err := s.factorStore.SetRegistrationChallenge(ctx, f.ID, sessionJSON, expiresAt); err != nil {
		return nil, apperror.InternalWrap(err, "persist challenge")
	}

	optionsJSON, err := json.Marshal(creation)
	if err != nil {
		return nil, apperror.InternalWrap(err, "marshal creation options")
	}

	r := f.ToResponse()
	r.Enrollment = models.WebAuthnEnrollmentPayload{CreationOptions: optionsJSON}
	return &r, nil
}

// activateWebAuthnFactor is the step-two verifier — called from
// ActivateFactor's type switch. Parses the attestation response,
// round-trips the stored SessionData through the go-webauthn
// library to verify the signature + challenge + origin, and on
// success writes the FactorWebAuthnCredential subtable row and
// clears the challenge. Duplicate credential IDs are rejected
// pre-insert with a Conflict so the user gets a clean error.
func (s *FactorService) activateWebAuthnFactor(ctx context.Context, userID uint, f *models.Factor, responseBody []byte) error {
	if s.webAuthn == nil {
		return apperror.BadRequest("WebAuthn is not enabled on this deployment")
	}
	if len(responseBody) == 0 {
		return apperror.BadRequest("webauthn_response is required")
	}
	if f.RegistrationChallenge == nil {
		return apperror.BadRequest("no registration challenge on factor")
	}
	if f.RegistrationChallengeExpiresAt != nil && f.RegistrationChallengeExpiresAt.Before(time.Now().UTC()) {
		// Delete the stale pending row so a retry produces a fresh
		// challenge rather than re-hitting the expired one.
		_, _ = s.factorStore.DeleteFactor(ctx, f.ID, userID)
		return apperror.Unauthorized("registration challenge expired")
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(f.RegistrationChallenge, &sessionData); err != nil {
		return apperror.InternalWrap(err, "decode session data")
	}

	parsed, err := webauthnpkg.ParseCreationResponse(responseBody)
	if err != nil {
		return apperror.BadRequest(fmt.Sprintf("invalid webauthn response: %v", err))
	}

	existingCreds, err := s.gatherUserWebAuthnCredentials(ctx, userID)
	if err != nil {
		return err
	}
	u, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}
	wuser := webauthnpkg.NewUser(userID, u.Email, u.Name, existingCreds)

	cred, err := s.webAuthn.Lib().CreateCredential(wuser, sessionData, parsed)
	if err != nil {
		return apperror.Unauthorized(fmt.Sprintf("attestation verification failed: %v", err))
	}

	// Uniqueness pre-check — the DB index will also enforce, but
	// we want a clean 409 rather than a 500 on the race.
	if existing, err := s.factorStore.FindWebAuthnCredentialByID(ctx, cred.ID); err == nil && existing.FactorID != f.ID {
		return apperror.Conflict("this security key is already registered")
	}

	row := &models.FactorWebAuthnCredential{
		FactorID:          f.ID,
		CredentialID:      cred.ID,
		PublicKey:         cred.PublicKey,
		AAGUID:            cred.Authenticator.AAGUID,
		SignCount:         int64(cred.Authenticator.SignCount),
		Transports:        joinTransports(cred.Transport),
		AttestationFormat: cred.AttestationType,
		BackupEligible:    cred.Flags.BackupEligible,
		BackupState:       cred.Flags.BackupState,
		UVInitialized:     cred.Flags.UserVerified,
		CreatedAt:         time.Now().UTC(),
	}
	if err := s.factorStore.CreateWebAuthnCredential(ctx, row); err != nil {
		// Catch the unique-index collision if the pre-check raced.
		return apperror.Conflict("this security key is already registered")
	}

	// Clear the challenge so the row can't be re-activated.
	if err := s.factorStore.ClearRegistrationChallenge(ctx, f.ID); err != nil {
		slog.Error("clear registration challenge", "factor_id", f.ID, "error", err)
	}
	return nil
}

// BuildWebAuthnRequestOptions builds the
// PublicKeyCredentialRequestOptions for a login assertion ceremony.
// Called by the AuthService when the pending_mfa flow needs a
// WebAuthn challenge — the full SessionData blob is returned so the
// AuthService can persist it on the sessions.challenge_nonce column.
func (s *FactorService) BuildWebAuthnRequestOptions(ctx context.Context, userID, factorID uint) (optionsJSON, sessionJSON []byte, _ error) {
	if s.webAuthn == nil {
		return nil, nil, apperror.BadRequest("WebAuthn is not enabled on this deployment")
	}
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, apperror.NotFound("factor")
		}
		return nil, nil, apperror.InternalWrap(err, "find factor")
	}
	if f.Type != models.FactorTypeWebAuthn || f.EnabledAt == nil {
		return nil, nil, apperror.BadRequest("factor is not an active webauthn credential")
	}
	creds, err := s.gatherUserWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	// Scope allowCredentials[] to just the one the user picked so
	// the authenticator prompt narrows to the intended key.
	u, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, classifyStoreError(err, "user")
	}
	wuser := webauthnpkg.NewUser(userID, u.Email, u.Name, filterCredentials(creds, factorIDToCredentialID(ctx, s, factorID)))

	opts, session, err := s.webAuthn.Lib().BeginLogin(wuser)
	if err != nil {
		return nil, nil, apperror.InternalWrap(err, "begin webauthn login")
	}
	optionsJSON, err = json.Marshal(opts)
	if err != nil {
		return nil, nil, apperror.InternalWrap(err, "marshal options")
	}
	sessionJSON, err = json.Marshal(session)
	if err != nil {
		return nil, nil, apperror.InternalWrap(err, "marshal session")
	}
	return optionsJSON, sessionJSON, nil
}

// VerifyWebAuthnAssertion is the login-time counterpart to
// activateWebAuthnFactor. The AuthService hands in the SessionData
// it fetched from the pending_mfa row + the raw response body; we
// verify and on success bump sign_count + return. The pending row
// itself is deleted by the AuthService in the same transaction as
// the verify (replay protection).
func (s *FactorService) VerifyWebAuthnAssertion(ctx context.Context, userID, factorID uint, sessionJSON, responseBody []byte) (bool, error) {
	if s.webAuthn == nil {
		return false, apperror.BadRequest("WebAuthn is not enabled on this deployment")
	}
	if len(responseBody) == 0 || len(sessionJSON) == 0 {
		return false, apperror.Unauthorized("invalid webauthn response")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return false, apperror.InternalWrap(err, "decode session data")
	}
	parsed, err := webauthnpkg.ParseAssertionResponse(responseBody)
	if err != nil {
		return false, apperror.Unauthorized("invalid webauthn response")
	}

	creds, err := s.gatherUserWebAuthnCredentials(ctx, userID)
	if err != nil {
		return false, err
	}
	u, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return false, classifyStoreError(err, "user")
	}
	wuser := webauthnpkg.NewUser(userID, u.Email, u.Name, creds)

	cred, err := s.webAuthn.Lib().ValidateLogin(wuser, sessionData, parsed)
	if err != nil {
		return false, apperror.Unauthorized(fmt.Sprintf("assertion verification failed: %v", err))
	}

	// Cross-check the credential actually belongs to the factor the
	// user claimed — stops a valid assertion from one of the user's
	// other credentials from completing login under the wrong factor
	// row (which would bump the wrong sign_count).
	dbCred, err := s.factorStore.FindWebAuthnCredential(ctx, factorID)
	if err != nil {
		return false, apperror.NotFound("factor")
	}
	if string(dbCred.CredentialID) != string(cred.ID) {
		return false, apperror.Unauthorized("credential mismatch")
	}

	// Update sign_count. Per WebAuthn L3 §7.2 step 22: non-increasing
	// counter is a soft clone signal, not a hard failure — many
	// synced-passkey authenticators always return zero. We accept
	// but audit-log the regression.
	if cred.Authenticator.SignCount > 0 && int64(cred.Authenticator.SignCount) <= dbCred.SignCount {
		slog.Warn("webauthn counter regression — possible clone",
			"user_id", userID, "factor_id", factorID,
			"stored", dbCred.SignCount, "new", cred.Authenticator.SignCount)
	}
	if err := s.factorStore.UpdateWebAuthnSignCount(ctx, factorID, int64(cred.Authenticator.SignCount), cred.Flags.BackupState); err != nil {
		slog.Error("update sign_count", "factor_id", factorID, "error", err)
		// Don't fail the login for a sign_count update — the
		// assertion was valid. Clone detection is best-effort.
	}
	_ = s.factorStore.TouchLastUsed(ctx, factorID)
	return true, nil
}

// gatherUserWebAuthnCredentials collects the library-shaped
// credentials for a user so allow/exclude-credentials lists can be
// populated on the two ceremonies. Dormant/pending credentials (no
// activated factor) are filtered out.
func (s *FactorService) gatherUserWebAuthnCredentials(ctx context.Context, userID uint) ([]webauthn.Credential, error) {
	factors, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "list factors")
	}
	out := make([]webauthn.Credential, 0, len(factors))
	for i := range factors {
		if factors[i].Type != models.FactorTypeWebAuthn {
			continue
		}
		cred, err := s.factorStore.FindWebAuthnCredential(ctx, factors[i].ID)
		if err != nil {
			continue
		}
		out = append(out, webauthn.Credential{
			ID:        cred.CredentialID,
			PublicKey: cred.PublicKey,
			Authenticator: webauthn.Authenticator{
				AAGUID: cred.AAGUID,
				// sign_count is capped at uint32 max in the spec; our
				// BIGINT column allows larger values but we never
				// write beyond uint32 (we copy from the authenticator,
				// which is uint32 by definition). Clamp defensively.
				SignCount: safeUint32(cred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: cred.BackupEligible,
				BackupState:    cred.BackupState,
				UserVerified:   cred.UVInitialized,
			},
			Transport: splitTransports(cred.Transports),
		})
	}
	return out, nil
}

// filterCredentials narrows a credential list to the single
// credential whose id matches the given bytes. Used on the login
// allow-list so only the factor the user picked is offered.
func filterCredentials(all []webauthn.Credential, target []byte) []webauthn.Credential {
	if len(target) == 0 {
		return all
	}
	for _, c := range all {
		if string(c.ID) == string(target) {
			return []webauthn.Credential{c}
		}
	}
	return nil
}

// factorIDToCredentialID reads the credential_id bytes for a factor
// id. Returns nil on any error — the caller's filter falls back to
// "offer every credential" in that case.
func factorIDToCredentialID(ctx context.Context, s *FactorService, factorID uint) []byte {
	cred, err := s.factorStore.FindWebAuthnCredential(ctx, factorID)
	if err != nil {
		return nil
	}
	return cred.CredentialID
}

// safeUint32 clamps a non-negative int64 into a uint32 without
// overflow. WebAuthn sign_count is a uint32 on the wire; we store
// the value in BIGINT for headroom, but our reads always round-trip
// through this helper so a malicious or buggy authenticator can't
// underflow or overflow the type.
func safeUint32(v int64) uint32 {
	if v < 0 {
		return 0
	}
	if v > int64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(v)
}

// joinTransports serialises the webauthn protocol transport enum
// slice into the comma-joined string we store in the DB. Empty or
// unknown values round-trip cleanly.
func joinTransports(transports []protocol.AuthenticatorTransport) string {
	if len(transports) == 0 {
		return ""
	}
	parts := make([]string, 0, len(transports))
	for _, t := range transports {
		parts = append(parts, string(t))
	}
	return strings.Join(parts, ",")
}

// splitTransports is the inverse of joinTransports.
func splitTransports(s string) []protocol.AuthenticatorTransport {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]protocol.AuthenticatorTransport, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, protocol.AuthenticatorTransport(p))
		}
	}
	return out
}

// LookupActiveFactor returns a factor by id iff it belongs to the
// user AND has been activated. Used by the AuthService to decide
// which verifier (TOTP vs WebAuthn) to dispatch to on /auth/mfa/verify.
// Returns NotFound for missing / cross-user / pending factors to
// avoid leaking existence.
func (s *FactorService) LookupActiveFactor(ctx context.Context, userID, factorID uint) (*models.Factor, error) {
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("factor")
		}
		return nil, apperror.InternalWrap(err, "find factor")
	}
	if f.EnabledAt == nil {
		return nil, apperror.NotFound("factor")
	}
	return f, nil
}

// DescriptorsForLogin returns the list of factor choices to present to
// the user in the /login MFA-required response. Only activated
// factors; ordering matches the Settings list (TOTP first, most-used
// first, backup_codes last) so the UI's default selection is the
// factor the user most recently used.
//
// Returns an empty slice (not nil) if the user has no active factors —
// callers use len() == 0 as the signal to skip the MFA step.
func (s *FactorService) DescriptorsForLogin(ctx context.Context, userID uint) ([]models.LoginFactorDescriptor, error) {
	rows, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "list factors for login")
	}
	out := make([]models.LoginFactorDescriptor, 0, len(rows))
	for i := range rows {
		out = append(out, models.LoginFactorDescriptor{
			ID:    rows[i].ID,
			Type:  rows[i].Type,
			Label: rows[i].Label,
		})
	}
	return out, nil
}

// HasActivePrimaryFactor answers "does the user need to go through
// two-step login?". A user with only a backup_codes factor (impossible
// today because backup_codes auto-creation requires a primary, but
// keep the guard in case data drift) is NOT considered MFA-enrolled
// for the purpose of the password step.
func (s *FactorService) HasActivePrimaryFactor(ctx context.Context, userID uint) (bool, error) {
	rows, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return false, apperror.InternalWrap(err, "check active factors")
	}
	for i := range rows {
		if rows[i].Type != models.FactorTypeBackupCodes {
			return true, nil
		}
	}
	return false, nil
}

// VerifyCodeForLogin checks `code` against the given factor's type-
// specific verifier. Unlike the step-up verify used by Delete/etc, no
// password is required — the pending_mfa row is the password-
// verification proof. Factor ownership is enforced against the caller
// (the pending row's user_id); cross-user factor IDs return NotFound.
//
// Returns (factorType, nil) on success so the caller can audit which
// factor type was used. On a wrong-but-well-formed code returns
// apperror.Unauthorized; on a missing/cross-user/inactive factor
// returns apperror.NotFound uniformly to avoid leaking factor
// existence.
func (s *FactorService) VerifyCodeForLogin(ctx context.Context, userID, factorID uint, code string) (string, error) {
	f, err := s.factorStore.FindByIDAndUser(ctx, factorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", apperror.NotFound("factor")
		}
		return "", apperror.InternalWrap(err, "find factor")
	}
	if f.EnabledAt == nil {
		// Pending factor — cannot be used to complete a login. Treat
		// as not-found so a stale factor id (pending from an abandoned
		// enrolment) doesn't become an oracle.
		return "", apperror.NotFound("factor")
	}
	switch f.Type {
	case models.FactorTypeTOTP:
		if ok := s.tryTOTPCode(ctx, f.ID, code); !ok {
			return f.Type, apperror.Unauthorized("invalid code")
		}
		return f.Type, nil
	case models.FactorTypeBackupCodes:
		if ok := s.tryBackupCode(ctx, f.ID, code); !ok {
			return f.Type, apperror.Unauthorized("invalid code")
		}
		return f.Type, nil
	case models.FactorTypeWebAuthn:
		// WebAuthn uses VerifyWebAuthnAssertion, not this code-string
		// path. Returning BadRequest makes it clear to the caller
		// that they're on the wrong endpoint; the auth service
		// dispatches to the right verifier before landing here for
		// TOTP/backup.
		return f.Type, apperror.BadRequest("webauthn factors must be verified via the webauthn assertion endpoint")
	default:
		return f.Type, apperror.BadRequest(fmt.Sprintf("unsupported factor type: %s", f.Type))
	}
}

// ----- internal helpers -----

func (s *FactorService) verifyPassword(ctx context.Context, userID uint, password string) error {
	if password == "" {
		return apperror.BadRequest("password is required")
	}
	u, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return apperror.Unauthorized("invalid credentials")
	}
	return nil
}

// verifyTOTPForActivation verifies a code against the encrypted-at-rest
// secret, but does NOT bump last_used_step. That's because activation
// happens before the factor is enabled; enabling it sets enabled_at to
// now() and future verifications will bump the step counter.
func (s *FactorService) verifyTOTPForActivation(ctx context.Context, factorID uint, code string) error {
	secret, err := s.decryptTOTPSecret(ctx, factorID)
	if err != nil {
		return err
	}
	ok, _ := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period: totpPeriod,
		Skew:   totpSkewSteps,
		Digits: otp.DigitsSix,
	})
	if !ok {
		return apperror.Unauthorized("invalid code")
	}
	return nil
}

// verifyAnyFactorCode accepts a code that matches ANY of the user's
// active factors (TOTP or backup). Used by the "delete requires a
// valid code from any active factor" rule.
func (s *FactorService) verifyAnyFactorCode(ctx context.Context, userID uint, code string) error {
	active, err := s.factorStore.FindActiveByUserID(ctx, userID)
	if err != nil {
		return apperror.InternalWrap(err, "find active factors")
	}
	for _, f := range active {
		switch f.Type {
		case models.FactorTypeTOTP:
			if ok := s.tryTOTPCode(ctx, f.ID, code); ok {
				return nil
			}
		case models.FactorTypeBackupCodes:
			if ok := s.tryBackupCode(ctx, f.ID, code); ok {
				return nil
			}
		}
	}
	return apperror.Unauthorized("invalid code")
}

// tryTOTPCode verifies a TOTP code and atomically bumps last_used_step
// on success. Returns true iff the code was accepted AND the
// compare-and-set step-bump succeeded (catches concurrent replays).
func (s *FactorService) tryTOTPCode(ctx context.Context, factorID uint, code string) bool {
	secret, err := s.decryptTOTPSecret(ctx, factorID)
	if err != nil {
		return false
	}
	now := time.Now().UTC()
	// Walk the skew window and find which step, if any, matches.
	for d := -int64(totpSkewSteps); d <= int64(totpSkewSteps); d++ {
		t := now.Add(time.Duration(d) * time.Duration(totpPeriod) * time.Second)
		candidate, err := totp.GenerateCodeCustom(secret, t, totp.ValidateOpts{
			Period: totpPeriod,
			Skew:   0,
			Digits: otp.DigitsSix,
		})
		if err != nil {
			continue
		}
		if candidate != code {
			continue
		}
		step := t.Unix() / int64(totpPeriod)
		ok, err := s.factorStore.AcceptTOTPStep(ctx, factorID, step)
		if err != nil || !ok {
			return false
		}
		_ = s.factorStore.TouchLastUsed(ctx, factorID)
		return true
	}
	return false
}

// tryBackupCode attempts to atomically consume a backup code. Returns
// true iff the hash matched an unused row and the UPDATE won the race.
func (s *FactorService) tryBackupCode(ctx context.Context, factorID uint, rawCode string) bool {
	h := hashBackupCode(normalizeBackupCode(rawCode))
	ok, err := s.factorStore.ConsumeBackupCode(ctx, factorID, h)
	if err != nil || !ok {
		return false
	}
	_ = s.factorStore.TouchLastUsed(ctx, factorID)
	return true
}

func (s *FactorService) decryptTOTPSecret(ctx context.Context, factorID uint) (string, error) {
	row, err := s.factorStore.FindTOTPSecret(ctx, factorID)
	if err != nil {
		return "", classifyStoreError(err, "totp secret")
	}
	plain, err := s.aead.Open(row.SecretCiphertext, row.SecretNonce)
	if err != nil {
		return "", apperror.InternalWrap(err, "decrypt totp secret")
	}
	return string(plain), nil
}

// ensureBackupCodesFactor auto-creates the singleton backup_codes
// factor for the user if one does not exist. Returns (created,
// payload, err) where `created=true` indicates the payload must be
// presented to the user exactly once.
func (s *FactorService) ensureBackupCodesFactor(ctx context.Context, userID uint) (bool, *models.BackupCodesPayload, error) {
	if _, err := s.factorStore.FindBackupCodesFactor(ctx, userID); err == nil {
		return false, nil, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return false, nil, err
	}

	raw, hashed, err := generateBackupCodes(BackupCodeCount)
	if err != nil {
		return false, nil, err
	}

	now := time.Now().UTC()
	f := &models.Factor{
		UserID:    userID,
		Type:      models.FactorTypeBackupCodes,
		EnabledAt: &now,
		CreatedAt: now,
	}
	if err := s.factorStore.CreateFactor(ctx, f); err != nil {
		return false, nil, err
	}
	rows := make([]models.FactorBackupCode, 0, len(hashed))
	for _, h := range hashed {
		rows = append(rows, models.FactorBackupCode{
			FactorID:  f.ID,
			CodeHash:  h,
			CreatedAt: now,
		})
	}
	if err := s.factorStore.InsertBackupCodes(ctx, rows); err != nil {
		return false, nil, err
	}
	return true, &models.BackupCodesPayload{FactorID: f.ID, Codes: raw}, nil
}

// generateBackupCodes returns (raw, hashed) slices. Raw codes are
// shown to the user once; hashed go to the DB. 10 chars, Crockford
// base32 alphabet, 8 bytes of entropy (64 bits) per code.
func generateBackupCodes(n int) ([]string, []string, error) {
	raw := make([]string, 0, n)
	hashed := make([]string, 0, n)
	// Crockford base32: 0-9 plus a-z excluding i, l, o, u. Exactly 32
	// symbols (the alphabet MUST be 32 chars or base32.NewEncoding
	// panics). Ambiguous characters are excluded so users rarely
	// misread a printed code.
	enc := base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)
	for range n {
		b := make([]byte, BackupCodeEntropyBytes)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := enc.EncodeToString(b) // ~13 chars for 8 input bytes
		// Pretty-print: insert a hyphen at the midpoint for readability.
		if mid := len(code) / 2; mid > 0 {
			code = code[:mid] + "-" + code[mid:]
		}
		raw = append(raw, code)
		hashed = append(hashed, hashBackupCode(normalizeBackupCode(code)))
	}
	return raw, hashed, nil
}

// normalizeBackupCode accepts user input and strips anything that
// isn't part of the code alphabet. Users often type hyphens or
// spaces; we accept either.
func normalizeBackupCode(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if r == '-' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hashBackupCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

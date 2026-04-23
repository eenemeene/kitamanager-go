package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
func (s *FactorService) ActivateFactor(ctx context.Context, userID, factorID uint, code string) (*models.FactorActivateResponse, error) {
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
		if err := s.verifyTOTPForActivation(ctx, f.ID, code); err != nil {
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
					s.auditService.LogFactorActivationLocked(userID, f.Type)
					return nil, apperror.TooManyRequests("too many wrong codes; re-enroll the factor")
				}
			}
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

	s.auditService.LogFactorEnrolled(userID, f.Type)

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

	s.auditService.LogFactorDeleted(userID, f.Type)
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
	ok, err := s.factorStore.UpdateLabel(ctx, factorID, userID, label)
	if err != nil {
		return nil, apperror.InternalWrap(err, "update label")
	}
	if !ok {
		return nil, apperror.NotFound("factor")
	}
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

	s.auditService.LogBackupCodesRegenerated(userID)

	return &models.BackupCodesPayload{FactorID: f.ID, Codes: raw}, nil
}

// CleanupAbandonedPendingFactors is called by the periodic GC job in
// cmd/api/main.go. Factors with enabled_at IS NULL that are older
// than the threshold are rows from abandoned enrollment flows.
func (s *FactorService) CleanupAbandonedPendingFactors(ctx context.Context, olderThan time.Duration) error {
	_, err := s.factorStore.CleanupAbandonedPending(ctx, olderThan)
	return err
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

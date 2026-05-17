package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// FactorStore implements FactorStorer over GORM. Every method that
// addresses a single factor by id takes the owning user_id alongside,
// so the WHERE clause scopes the lookup and a cross-user attempt
// returns 0 rows rather than leaking existence.
type FactorStore struct {
	db *gorm.DB
}

// NewFactorStore creates a new FactorStore.
func NewFactorStore(db *gorm.DB) *FactorStore {
	return &FactorStore{db: db}
}

// FindByUserID returns every factor belonging to the user, both
// activated and pending. The handler filters out pending rows from the
// `GET /factors` response; the service sometimes needs the full set.
func (s *FactorStore) FindByUserID(ctx context.Context, userID uint) ([]models.Factor, error) {
	var out []models.Factor
	err := DBFromContext(ctx, s.db).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

// FindActiveByUserID returns only factors with enabled_at IS NOT NULL.
// This is what `GET /factors` returns and what the service checks when
// asking "does this user have any primary factor?"
//
// Sort order reflects how the Settings page wants to display them:
// primary factors first (backup_codes sinks to the bottom), then most-
// recently-used, then creation order as a tiebreaker. Clients get a
// stable, predictable list without needing to re-sort.
func (s *FactorStore) FindActiveByUserID(ctx context.Context, userID uint) ([]models.Factor, error) {
	var out []models.Factor
	err := DBFromContext(ctx, s.db).
		Where("user_id = ? AND enabled_at IS NOT NULL", userID).
		Order("(type = 'backup_codes')::int ASC, last_used_at DESC NULLS LAST, created_at DESC").
		Find(&out).Error
	return out, err
}

// FindByIDAndUser returns the factor iff it belongs to the caller.
// ErrNotFound is returned both when the id is unknown and when it
// belongs to another user — the service layer relies on this to
// produce 404s without leaking factor existence.
func (s *FactorStore) FindByIDAndUser(ctx context.Context, id, userID uint) (*models.Factor, error) {
	var f models.Factor
	err := DBFromContext(ctx, s.db).
		Where("id = ? AND user_id = ?", id, userID).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

// FindPendingByUserAndType returns any factor of the given type that
// is still in the enrollment phase (enabled_at IS NULL) for this user.
// Used to implement "re-enrolling the same type replaces the pending
// row" semantics.
func (s *FactorStore) FindPendingByUserAndType(ctx context.Context, userID uint, factorType string) (*models.Factor, error) {
	var f models.Factor
	err := DBFromContext(ctx, s.db).
		Where("user_id = ? AND type = ? AND enabled_at IS NULL", userID, factorType).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

// FindBackupCodesFactor returns the user's singleton backup_codes
// factor, if one exists. Used by the auto-create flow (must not
// duplicate) and by the backup-code verify path.
func (s *FactorStore) FindBackupCodesFactor(ctx context.Context, userID uint) (*models.Factor, error) {
	var f models.Factor
	err := DBFromContext(ctx, s.db).
		Where("user_id = ? AND type = ?", userID, models.FactorTypeBackupCodes).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

// CreateFactor inserts a new parent factor row. Does not touch
// subtables — caller is responsible for inserting the type-specific
// row in the same transaction.
func (s *FactorStore) CreateFactor(ctx context.Context, f *models.Factor) error {
	return DBFromContext(ctx, s.db).Create(f).Error
}

// ActivateFactor transitions a factor from pending to active
// atomically. Returns (true, nil) if the row was moved from
// enabled_at=NULL → enabled_at=now(), (false, nil) if the row was
// already activated (concurrent activation wins once), or (_, err) on
// DB errors.
//
// This is the compare-and-set that closes the "two concurrent activate
// calls" race.
func (s *FactorStore) ActivateFactor(ctx context.Context, id, userID uint) (bool, error) {
	now := time.Now().UTC()
	res := DBFromContext(ctx, s.db).Model(&models.Factor{}).
		Where("id = ? AND user_id = ? AND enabled_at IS NULL", id, userID).
		Updates(map[string]any{"enabled_at": now, "last_used_at": now})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// IncrementActivationFailures atomically bumps the activation_failures
// counter on a pending factor and returns the post-increment value.
// The row is scoped to user_id AND enabled_at IS NULL so this is a no-
// op on activated factors (defence in depth — the service layer already
// short-circuits on those).
//
// Returns 0 with ErrNotFound if no row matched (wrong id, wrong user,
// or already activated) so the service can translate cleanly to 404.
func (s *FactorStore) IncrementActivationFailures(ctx context.Context, id, userID uint) (int, error) {
	db := DBFromContext(ctx, s.db)
	res := db.Model(&models.Factor{}).
		Where("id = ? AND user_id = ? AND enabled_at IS NULL", id, userID).
		UpdateColumn("activation_failures", gorm.Expr("activation_failures + 1"))
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrNotFound
	}
	var f models.Factor
	if err := db.Select("activation_failures").
		Where("id = ? AND user_id = ?", id, userID).
		First(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return f.ActivationFailures, nil
}

// DeleteFactor removes a factor iff it belongs to the user. Returns
// the number of rows affected: 0 means "not found OR belongs to
// another user" — the caller surfaces this as 404.
func (s *FactorStore) DeleteFactor(ctx context.Context, id, userID uint) (int64, error) {
	res := DBFromContext(ctx, s.db).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Factor{})
	return res.RowsAffected, res.Error
}

// UpdateLabel sets the label; nil clears it.
func (s *FactorStore) UpdateLabel(ctx context.Context, id, userID uint, label *string) (bool, error) {
	res := DBFromContext(ctx, s.db).Model(&models.Factor{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("label", label)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// TouchLastUsed stamps last_used_at on a factor. Called after a
// successful verify of a code belonging to that factor.
func (s *FactorStore) TouchLastUsed(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Model(&models.Factor{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now().UTC()).Error
}

// CleanupAbandonedPending removes factor rows that never completed
// enrollment. Called from the periodic cleanup job in cmd/api/main.go.
// The cutoff matches the "you have one hour to finish enrolling" UX
// contract.
func (s *FactorStore) CleanupAbandonedPending(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	res := DBFromContext(ctx, s.db).
		Where("enabled_at IS NULL AND created_at < ?", cutoff).
		Delete(&models.Factor{})
	return res.RowsAffected, res.Error
}

// --- TOTP subtable methods ---

// CreateTOTPSecret inserts the encrypted secret for a factor.
func (s *FactorStore) CreateTOTPSecret(ctx context.Context, secret *models.FactorTOTPSecret) error {
	return DBFromContext(ctx, s.db).Create(secret).Error
}

// FindTOTPSecret returns the encrypted secret row for a factor id.
// ErrNotFound if the factor is not TOTP-typed or not found.
func (s *FactorStore) FindTOTPSecret(ctx context.Context, factorID uint) (*models.FactorTOTPSecret, error) {
	var secret models.FactorTOTPSecret
	err := DBFromContext(ctx, s.db).Where("factor_id = ?", factorID).First(&secret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &secret, nil
}

// AcceptTOTPStep implements replay prevention. It atomically bumps
// last_used_step if and only if the passed step is strictly greater
// than the stored value (or no step has ever been stored). Returns
// (true, nil) on accept, (false, nil) on replay, (_, err) on DB error.
func (s *FactorStore) AcceptTOTPStep(ctx context.Context, factorID uint, step int64) (bool, error) {
	res := DBFromContext(ctx, s.db).Model(&models.FactorTOTPSecret{}).
		Where("factor_id = ? AND (last_used_step IS NULL OR last_used_step < ?)", factorID, step).
		Update("last_used_step", step)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// --- Backup codes subtable methods ---

// InsertBackupCodes stores a batch of hashed backup codes.
func (s *FactorStore) InsertBackupCodes(ctx context.Context, codes []models.FactorBackupCode) error {
	if len(codes) == 0 {
		return nil
	}
	return DBFromContext(ctx, s.db).Create(&codes).Error
}

// ListBackupCodes returns every backup code row for a factor,
// including both used and unused. Counting unused is a caller concern.
func (s *FactorStore) ListBackupCodes(ctx context.Context, factorID uint) ([]models.FactorBackupCode, error) {
	var out []models.FactorBackupCode
	err := DBFromContext(ctx, s.db).
		Where("factor_id = ?", factorID).
		Find(&out).Error
	return out, err
}

// CountUnusedBackupCodes is a convenience for surfacing
// backup_codes_remaining on the factors-list response.
func (s *FactorStore) CountUnusedBackupCodes(ctx context.Context, factorID uint) (int, error) {
	var n int64
	err := DBFromContext(ctx, s.db).Model(&models.FactorBackupCode{}).
		Where("factor_id = ? AND used_at IS NULL", factorID).
		Count(&n).Error
	return int(n), err
}

// ListUnusedBackupCodes returns every still-redeemable code row for a
// factor (used_at IS NULL). The pre-bcrypt design did a single
// WHERE code_hash = ? UPDATE — impossible now that each row carries
// its own salt — so the verify path lists candidates and uses
// bcrypt.CompareHashAndPassword to find the match, then claims the
// matching row via MarkBackupCodeUsed.
func (s *FactorStore) ListUnusedBackupCodes(ctx context.Context, factorID uint) ([]models.FactorBackupCode, error) {
	var out []models.FactorBackupCode
	err := DBFromContext(ctx, s.db).
		Where("factor_id = ? AND used_at IS NULL", factorID).
		Find(&out).Error
	return out, err
}

// MarkBackupCodeUsed atomically claims a row by primary key iff the
// row is still unused. Returns (true, nil) on the winning claim,
// (false, nil) if the row was already consumed by a concurrent verify
// attempt — this WHERE used_at IS NULL guard is what preserves the
// single-use guarantee under concurrent races.
func (s *FactorStore) MarkBackupCodeUsed(ctx context.Context, id uint) (bool, error) {
	res := DBFromContext(ctx, s.db).Model(&models.FactorBackupCode{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", time.Now().UTC())
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// --- WebAuthn subtable methods ---

// CreateWebAuthnCredential inserts the per-factor WebAuthn credential
// row. The unique index on credential_id rejects duplicates at the
// DB layer — two users cannot bind the same hardware key — and the
// caller should translate that violation into apperror.Conflict so
// the UI can explain "this security key is already registered".
func (s *FactorStore) CreateWebAuthnCredential(ctx context.Context, cred *models.FactorWebAuthnCredential) error {
	return DBFromContext(ctx, s.db).Create(cred).Error
}

// FindWebAuthnCredential returns the credential row for a factor.
// ErrNotFound if the factor is not WebAuthn-typed or the subtable
// row is missing (which shouldn't happen for an activated factor but
// can for one mid-registration).
func (s *FactorStore) FindWebAuthnCredential(ctx context.Context, factorID uint) (*models.FactorWebAuthnCredential, error) {
	var cred models.FactorWebAuthnCredential
	err := DBFromContext(ctx, s.db).Where("factor_id = ?", factorID).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cred, nil
}

// FindWebAuthnCredentialByID looks up a credential by its credential_id
// bytes (the globally unique identifier WebAuthn authenticators
// return). Used at registration time to reject duplicates before
// inserting, and (future PR) for passwordless flows where the user
// is identified by the credential alone.
func (s *FactorStore) FindWebAuthnCredentialByID(ctx context.Context, credentialID []byte) (*models.FactorWebAuthnCredential, error) {
	var cred models.FactorWebAuthnCredential
	err := DBFromContext(ctx, s.db).Where("credential_id = ?", credentialID).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cred, nil
}

// UpdateWebAuthnSignCount bumps sign_count and backup_state on a
// credential after a successful assertion. The WebAuthn L3 spec
// allows authenticators that don't keep a counter (synced passkeys,
// some platform authenticators) to always return zero — the service
// layer decides whether a non-increase is acceptable; this store
// method just writes whatever the service hands down.
//
// backup_state mirrors the current BS flag off the assertion; it can
// toggle on and off over the credential's lifetime as the user opts
// in/out of cloud sync.
func (s *FactorStore) UpdateWebAuthnSignCount(ctx context.Context, factorID uint, newCount int64, backupState bool) error {
	return DBFromContext(ctx, s.db).Model(&models.FactorWebAuthnCredential{}).
		Where("factor_id = ?", factorID).
		Updates(map[string]any{
			"sign_count":   newCount,
			"backup_state": backupState,
		}).Error
}

// SetRegistrationChallenge persists the server-issued challenge blob
// on the pending factor row with an absolute expiry. The blob is
// typically a serialised go-webauthn SessionData (contains the raw
// challenge bytes + the allowed-credentials list + required UV flag)
// so the activate path can round-trip it back into the library's
// verifier.
func (s *FactorStore) SetRegistrationChallenge(ctx context.Context, factorID uint, challenge []byte, expiresAt time.Time) error {
	return DBFromContext(ctx, s.db).Model(&models.Factor{}).
		Where("id = ?", factorID).
		Updates(map[string]any{
			"registration_challenge":            challenge,
			"registration_challenge_expires_at": expiresAt,
		}).Error
}

// ClearRegistrationChallenge wipes the challenge columns. Called
// after successful activation; also called by the cleanup job when a
// row is swept as abandoned. Safe to call on rows that never had a
// challenge (both columns go from NULL to NULL). GORM silently
// swallows nil/gorm.Expr("NULL") on Updates in ways that depend on
// struct tags and types; raw SQL is unambiguous here.
func (s *FactorStore) ClearRegistrationChallenge(ctx context.Context, factorID uint) error {
	return DBFromContext(ctx, s.db).Exec(
		"UPDATE factors SET registration_challenge = NULL, registration_challenge_expires_at = NULL WHERE id = ?",
		factorID,
	).Error
}

// ReplaceBackupCodes deletes every existing code for the factor and
// inserts the new batch, all in one transaction with a row lock on the
// parent factor so two concurrent regenerations serialise.
func (s *FactorStore) ReplaceBackupCodes(ctx context.Context, factorID uint, fresh []models.FactorBackupCode) error {
	return DBFromContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		// Lock the parent factor row; any other transaction attempting
		// to regenerate for the same factor will wait here.
		var parent models.Factor
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", factorID).
			First(&parent).Error; err != nil {
			return err
		}
		if err := tx.Where("factor_id = ?", factorID).Delete(&models.FactorBackupCode{}).Error; err != nil {
			return err
		}
		if len(fresh) > 0 {
			if err := tx.Create(&fresh).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

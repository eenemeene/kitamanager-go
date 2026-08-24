package store

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// auditOrder is the ordering every audit read uses.
//
// The `id DESC` tiebreaker is not decoration. `timestamp` is not unique and
// collisions are routine rather than theoretical: LogResourceUpdateAcrossOrgs
// emits one row per org in a tight loop, and the YAML importers emit one row
// per imported record, so consecutive rows are written microseconds apart and
// PostgreSQL stores TIMESTAMPTZ at microsecond resolution. Ordering by
// timestamp alone leaves the order *within* a tie group undefined, and
// LIMIT/OFFSET paging over an undefined order silently duplicates some rows
// and skips others whenever a page boundary falls inside a tie group.
//
// id is a BIGSERIAL, so it is both unique and consistent with insert order,
// which makes it the correct second key rather than merely a deterministic one.
const auditOrder = "timestamp DESC, id DESC"

// AuditStore handles audit log database operations
type AuditStore struct {
	db *gorm.DB
}

// NewAuditStore creates a new AuditStore
func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Create creates a new audit log entry
func (s *AuditStore) Create(ctx context.Context, log *models.AuditLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now().UTC()
	}
	return DBFromContext(ctx, s.db).Create(log).Error
}

// FindByUser returns audit logs for a specific user
func (s *AuditStore) FindByUser(ctx context.Context, userID uint, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Where("user_id = ?", userID).
		Order(auditOrder).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByAction returns audit logs for a specific action type
func (s *AuditStore) FindByAction(ctx context.Context, action models.AuditAction, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).Where("action = ?", action).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Where("action = ?", action).
		Order(auditOrder).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByDateRange returns audit logs within a date range
func (s *AuditStore) FindByDateRange(ctx context.Context, from, to time.Time, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).Where("timestamp >= ? AND timestamp <= ?", from, to)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Where("timestamp >= ? AND timestamp <= ?", from, to).
		Order(auditOrder).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindAll returns all audit logs with pagination
func (s *AuditStore) FindAll(ctx context.Context, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Order(auditOrder).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindFailedLogins returns failed login attempts, optionally filtered by
// email. The email match is case-insensitive so historical audit rows
// written before email normalization (migration 000009) still participate
// in the lockout counter.
func (s *AuditStore) FindFailedLogins(ctx context.Context, email string, since time.Time, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog

	query := DBFromContext(ctx, s.db).Where("action = ? AND timestamp >= ?", models.AuditActionLoginFailed, since)
	if email != "" {
		query = query.Where("lower(user_email) = lower(?)", email)
	}

	if err := query.Order(auditOrder).Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// CountFailedLoginsSince counts failed login attempts for an email since a
// given time. Case-insensitive — same rationale as FindFailedLogins.
func (s *AuditStore) CountFailedLoginsSince(ctx context.Context, email string, since time.Time) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).
		Where("action = ? AND lower(user_email) = lower(?) AND timestamp >= ?",
			models.AuditActionLoginFailed, email, since).
		Count(&count).Error
	return count, err
}

// CountFailedPasswordChangesSince counts password-change failures for a user
// since a given time. Used by the /me/password lockout to stop an attacker
// with a stolen access token from brute-forcing the current password at full
// API-mutation-rate-limit speed.
func (s *AuditStore) CountFailedPasswordChangesSince(ctx context.Context, userID uint, since time.Time) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).
		Where("action = ? AND user_id = ? AND timestamp >= ?",
			models.AuditActionPasswordChangeFailed, userID, since).
		Count(&count).Error
	return count, err
}

// CountFailedPasswordResetsSince counts /users/:userId/password actor_password
// failures attributed to `actorID` since a given time. Drives the per-actor
// lockout on the admin reset endpoint: an attacker holding a stolen admin
// session must not be able to iterate actor_password candidates at full
// API-mutation-rate-limit speed.
func (s *AuditStore) CountFailedPasswordResetsSince(ctx context.Context, actorID uint, since time.Time) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).
		Where("action = ? AND user_id = ? AND timestamp >= ?",
			models.AuditActionPasswordResetFailed, actorID, since).
		Count(&count).Error
	return count, err
}

// CountFailedMFAChallengesSince counts mfa_challenge_failed events for
// a user since the given time. Backs the per-user lockout that kicks
// in across distinct pending_mfa rows — the per-row counter alone
// could be bypassed by an attacker cycling through many pending rows.
func (s *AuditStore) CountFailedMFAChallengesSince(ctx context.Context, userID uint, since time.Time) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).
		Where("action = ? AND user_id = ? AND timestamp >= ?",
			models.AuditActionMFAChallengeFailed, userID, since).
		Count(&count).Error
	return count, err
}

// FindByID returns a single audit log entry by ID
func (s *AuditStore) FindByID(ctx context.Context, id uint) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := DBFromContext(ctx, s.db).First(&log, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &log, nil
}

// FindAllFiltered returns audit logs with optional filters and pagination.
func (s *AuditStore) FindAllFiltered(ctx context.Context, action string, userID *uint, from *time.Time, to *time.Time, limit, offset int) ([]models.AuditLog, int64, error) {
	return s.findFiltered(ctx, auditFilter{
		action: action,
		userID: userID,
		from:   from,
		to:     to,
	}, limit, offset)
}

// FindByOrganization returns audit logs scoped to a single organization with
// optional filters and pagination. Rows with a NULL organization_id (identity
// events like login and password changes) are deliberately excluded — those
// are only reachable via the superadmin-only global endpoint.
func (s *AuditStore) FindByOrganization(ctx context.Context, orgID uint, action string, userID *uint, from, to *time.Time, limit, offset int) ([]models.AuditLog, int64, error) {
	return s.findFiltered(ctx, auditFilter{
		orgID:  &orgID,
		action: action,
		userID: userID,
		from:   from,
		to:     to,
	}, limit, offset)
}

// auditFilter groups the optional WHERE clauses shared by the filtered
// list queries so the two public variants stay in lockstep.
type auditFilter struct {
	orgID  *uint
	action string
	userID *uint
	from   *time.Time
	to     *time.Time
}

func (f auditFilter) apply(q *gorm.DB) *gorm.DB {
	if f.orgID != nil {
		q = q.Where("organization_id = ?", *f.orgID)
	}
	if f.action != "" {
		// Case-insensitive substring match so "ild" finds child_create /
		// child_update / child_delete in one go. Escape the LIKE
		// metacharacters in user input so a filter containing "%" or "_"
		// does not suddenly widen the match.
		pattern := "%" + escapeLike(f.action) + "%"
		q = q.Where("action ILIKE ? ESCAPE '\\'", pattern)
	}
	if f.userID != nil {
		q = q.Where("user_id = ?", *f.userID)
	}
	if f.from != nil {
		q = q.Where("timestamp >= ?", *f.from)
	}
	if f.to != nil {
		q = q.Where("timestamp <= ?", *f.to)
	}
	return q
}

// escapeLike escapes the LIKE metacharacters %, _, and \ so the caller can
// splice untrusted input into a LIKE pattern without it widening or
// mis-matching. Pair with `ESCAPE '\\'` on the query.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func (s *AuditStore) findFiltered(ctx context.Context, f auditFilter, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := f.apply(DBFromContext(ctx, s.db).Model(&models.AuditLog{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := f.apply(DBFromContext(ctx, s.db).Model(&models.AuditLog{})).
		Order(auditOrder).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// Cleanup removes audit logs older than the specified duration
func (s *AuditStore) Cleanup(ctx context.Context, olderThan time.Time) (int64, error) {
	result := DBFromContext(ctx, s.db).Where("timestamp < ?", olderThan).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

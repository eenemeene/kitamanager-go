package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

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
		Order("timestamp DESC").
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
		Order("timestamp DESC").
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
		Order("timestamp DESC").
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

	if err := DBFromContext(ctx, s.db).Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindFailedLogins returns failed login attempts, optionally filtered by email
func (s *AuditStore) FindFailedLogins(ctx context.Context, email string, since time.Time, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog

	query := DBFromContext(ctx, s.db).Where("action = ? AND timestamp >= ?", models.AuditActionLoginFailed, since)
	if email != "" {
		query = query.Where("user_email = ?", email)
	}

	if err := query.Order("timestamp DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// CountFailedLoginsSince counts failed login attempts for an email since a given time
func (s *AuditStore) CountFailedLoginsSince(ctx context.Context, email string, since time.Time) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.AuditLog{}).
		Where("action = ? AND user_email = ? AND timestamp >= ?",
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
	var logs []models.AuditLog
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.AuditLog{})
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if from != nil {
		query = query.Where("timestamp >= ?", *from)
	}
	if to != nil {
		query = query.Where("timestamp <= ?", *to)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Rebuild query for data fetch (GORM Count consumes the query)
	dataQuery := DBFromContext(ctx, s.db).Model(&models.AuditLog{})
	if action != "" {
		dataQuery = dataQuery.Where("action = ?", action)
	}
	if userID != nil {
		dataQuery = dataQuery.Where("user_id = ?", *userID)
	}
	if from != nil {
		dataQuery = dataQuery.Where("timestamp >= ?", *from)
	}
	if to != nil {
		dataQuery = dataQuery.Where("timestamp <= ?", *to)
	}

	if err := dataQuery.Order("timestamp DESC").
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

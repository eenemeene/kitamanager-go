package store

import (
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
func (s *AuditStore) Create(log *models.AuditLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	return s.db.Create(log).Error
}

// FindByUser returns audit logs for a specific user
func (s *AuditStore) FindByUser(userID uint, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := s.db.Model(&models.AuditLog{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("user_id = ?", userID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByAction returns audit logs for a specific action type
func (s *AuditStore) FindByAction(action models.AuditAction, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := s.db.Model(&models.AuditLog{}).Where("action = ?", action).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("action = ?", action).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByDateRange returns audit logs within a date range
func (s *AuditStore) FindByDateRange(from, to time.Time, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{}).Where("timestamp >= ? AND timestamp <= ?", from, to)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("timestamp >= ? AND timestamp <= ?", from, to).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindAll returns all audit logs with pagination
func (s *AuditStore) FindAll(limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	if err := s.db.Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindFailedLogins returns failed login attempts, optionally filtered by email
func (s *AuditStore) FindFailedLogins(email string, since time.Time, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog

	query := s.db.Where("action = ? AND timestamp >= ?", models.AuditActionLoginFailed, since)
	if email != "" {
		query = query.Where("user_email = ?", email)
	}

	if err := query.Order("timestamp DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// CountFailedLoginsSince counts failed login attempts for an email since a given time
func (s *AuditStore) CountFailedLoginsSince(email string, since time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&models.AuditLog{}).
		Where("action = ? AND user_email = ? AND timestamp >= ?",
			models.AuditActionLoginFailed, email, since).
		Count(&count).Error
	return count, err
}

// Cleanup removes audit logs older than the specified duration
func (s *AuditStore) Cleanup(olderThan time.Time) (int64, error) {
	result := s.db.Where("timestamp < ?", olderThan).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

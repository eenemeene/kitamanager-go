package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ChildAttendanceStore implements ChildAttendanceStorer using GORM.
type ChildAttendanceStore struct {
	db *gorm.DB
}

// NewChildAttendanceStore creates a new ChildAttendanceStore.
func NewChildAttendanceStore(db *gorm.DB) *ChildAttendanceStore {
	return &ChildAttendanceStore{db: db}
}

func (s *ChildAttendanceStore) FindByID(ctx context.Context, id uint) (*models.ChildAttendance, error) {
	var attendance models.ChildAttendance
	if err := DBFromContext(ctx, s.db).Preload("Child").First(&attendance, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &attendance, nil
}

func (s *ChildAttendanceStore) FindByOrganizationAndDate(ctx context.Context, orgID uint, date time.Time, limit, offset int) ([]models.ChildAttendance, int64, error) {
	var records []models.ChildAttendance
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.ChildAttendance{}).
		Where("organization_id = ? AND date = ?", orgID, date)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Preload("Child").
		Where("organization_id = ? AND date = ?", orgID, date).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *ChildAttendanceStore) FindByChildAndDate(ctx context.Context, childID uint, date time.Time) (*models.ChildAttendance, error) {
	var attendance models.ChildAttendance
	if err := DBFromContext(ctx, s.db).Preload("Child").
		Where("child_id = ? AND date = ?", childID, date).
		First(&attendance).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &attendance, nil
}

func (s *ChildAttendanceStore) FindByChildAndDateRange(ctx context.Context, childID uint, from, to time.Time, limit, offset int) ([]models.ChildAttendance, int64, error) {
	var records []models.ChildAttendance
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.ChildAttendance{}).
		Where("child_id = ? AND date >= ? AND date <= ?", childID, from, to)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Preload("Child").
		Where("child_id = ? AND date >= ? AND date <= ?", childID, from, to).
		Order("date DESC").
		Limit(limit).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *ChildAttendanceStore) Create(ctx context.Context, attendance *models.ChildAttendance) error {
	return DBFromContext(ctx, s.db).Create(attendance).Error
}

func (s *ChildAttendanceStore) Update(ctx context.Context, attendance *models.ChildAttendance) error {
	return DBFromContext(ctx, s.db).Save(attendance).Error
}

func (s *ChildAttendanceStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.ChildAttendance{}, id).Error
}

// GetDailySummary returns attendance summary for a given date and organization.
func (s *ChildAttendanceStore) GetDailySummary(ctx context.Context, orgID uint, date time.Time) (*models.ChildAttendanceDailySummaryResponse, error) {
	var records []models.ChildAttendance
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND date = ?", orgID, date).
		Find(&records).Error; err != nil {
		return nil, err
	}

	summary := &models.ChildAttendanceDailySummaryResponse{
		Date:          date.Format(models.DateFormat),
		TotalChildren: len(records),
	}

	for _, r := range records {
		switch r.Status {
		case models.ChildAttendanceStatusPresent:
			summary.Present++
		case models.ChildAttendanceStatusAbsent:
			summary.Absent++
		case models.ChildAttendanceStatusSick:
			summary.Sick++
		case models.ChildAttendanceStatusVacation:
			summary.Vacation++
		}
	}

	return summary, nil
}

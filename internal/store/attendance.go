package store

import (
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// AttendanceStore implements AttendanceStorer using GORM.
type AttendanceStore struct {
	db *gorm.DB
}

// NewAttendanceStore creates a new AttendanceStore.
func NewAttendanceStore(db *gorm.DB) *AttendanceStore {
	return &AttendanceStore{db: db}
}

func (s *AttendanceStore) FindByID(id uint) (*models.Attendance, error) {
	var attendance models.Attendance
	if err := s.db.Preload("Child").First(&attendance, id).Error; err != nil {
		return nil, err
	}
	return &attendance, nil
}

func (s *AttendanceStore) FindByOrganizationAndDate(orgID uint, date time.Time, limit, offset int) ([]models.Attendance, int64, error) {
	var records []models.Attendance
	var total int64

	query := s.db.Model(&models.Attendance{}).
		Where("organization_id = ? AND date = ?", orgID, date)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Preload("Child").
		Where("organization_id = ? AND date = ?", orgID, date).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *AttendanceStore) FindByChildAndDate(childID uint, date time.Time) (*models.Attendance, error) {
	var attendance models.Attendance
	if err := s.db.Preload("Child").
		Where("child_id = ? AND date = ?", childID, date).
		First(&attendance).Error; err != nil {
		return nil, err
	}
	return &attendance, nil
}

func (s *AttendanceStore) FindByChildAndDateRange(childID uint, from, to time.Time, limit, offset int) ([]models.Attendance, int64, error) {
	var records []models.Attendance
	var total int64

	query := s.db.Model(&models.Attendance{}).
		Where("child_id = ? AND date >= ? AND date <= ?", childID, from, to)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Preload("Child").
		Where("child_id = ? AND date >= ? AND date <= ?", childID, from, to).
		Order("date DESC").
		Limit(limit).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *AttendanceStore) Create(attendance *models.Attendance) error {
	return s.db.Create(attendance).Error
}

func (s *AttendanceStore) Update(attendance *models.Attendance) error {
	return s.db.Save(attendance).Error
}

func (s *AttendanceStore) Delete(id uint) error {
	return s.db.Delete(&models.Attendance{}, id).Error
}

// GetDailySummary returns attendance summary for a given date and organization.
func (s *AttendanceStore) GetDailySummary(orgID uint, date time.Time) (*models.DailyAttendanceSummaryResponse, error) {
	var records []models.Attendance
	if err := s.db.Where("organization_id = ? AND date = ?", orgID, date).
		Find(&records).Error; err != nil {
		return nil, err
	}

	summary := &models.DailyAttendanceSummaryResponse{
		Date:          date.Format("2006-01-02"),
		TotalChildren: len(records),
	}

	for _, r := range records {
		switch r.Status {
		case models.AttendanceStatusPresent:
			summary.Present++
		case models.AttendanceStatusAbsent:
			summary.Absent++
		case models.AttendanceStatusSick:
			summary.Sick++
		case models.AttendanceStatusVacation:
			summary.Vacation++
		}
	}

	return summary, nil
}

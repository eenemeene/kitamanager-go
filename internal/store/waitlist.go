package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// WaitlistStore implements WaitlistStorer using GORM.
type WaitlistStore struct {
	db *gorm.DB
}

// NewWaitlistStore creates a new WaitlistStore.
func NewWaitlistStore(db *gorm.DB) *WaitlistStore {
	return &WaitlistStore{db: db}
}

func (s *WaitlistStore) FindByID(id uint) (*models.WaitlistEntry, error) {
	var entry models.WaitlistEntry
	if err := s.db.First(&entry, id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *WaitlistStore) FindByOrganization(orgID uint, limit, offset int) ([]models.WaitlistEntry, int64, error) {
	var entries []models.WaitlistEntry
	var total int64

	query := s.db.Model(&models.WaitlistEntry{}).Where("organization_id = ?", orgID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("organization_id = ?", orgID).
		Order("priority DESC, created_at ASC").
		Limit(limit).Offset(offset).
		Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (s *WaitlistStore) FindByOrganizationAndStatus(orgID uint, status string, limit, offset int) ([]models.WaitlistEntry, int64, error) {
	var entries []models.WaitlistEntry
	var total int64

	query := s.db.Model(&models.WaitlistEntry{}).
		Where("organization_id = ? AND status = ?", orgID, status)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("organization_id = ? AND status = ?", orgID, status).
		Order("priority DESC, created_at ASC").
		Limit(limit).Offset(offset).
		Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (s *WaitlistStore) Create(entry *models.WaitlistEntry) error {
	return s.db.Create(entry).Error
}

func (s *WaitlistStore) Update(entry *models.WaitlistEntry) error {
	return s.db.Save(entry).Error
}

func (s *WaitlistStore) Delete(id uint) error {
	return s.db.Delete(&models.WaitlistEntry{}, id).Error
}

func (s *WaitlistStore) CountByOrganizationAndStatus(orgID uint, status string) (int64, error) {
	var count int64
	if err := s.db.Model(&models.WaitlistEntry{}).
		Where("organization_id = ? AND status = ?", orgID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

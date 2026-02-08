package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type ChildNoteStore struct {
	db *gorm.DB
}

func NewChildNoteStore(db *gorm.DB) *ChildNoteStore {
	return &ChildNoteStore{db: db}
}

func (s *ChildNoteStore) FindByID(id uint) (*models.ChildNote, error) {
	var note models.ChildNote
	if err := s.db.First(&note, id).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *ChildNoteStore) FindByChild(childID uint, limit, offset int) ([]models.ChildNote, int64, error) {
	var notes []models.ChildNote
	var total int64

	query := s.db.Model(&models.ChildNote{}).Where("child_id = ?", childID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("child_id = ?", childID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

func (s *ChildNoteStore) FindByChildAndCategory(childID uint, category string, limit, offset int) ([]models.ChildNote, int64, error) {
	var notes []models.ChildNote
	var total int64

	query := s.db.Model(&models.ChildNote{}).
		Where("child_id = ? AND category = ?", childID, category)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("child_id = ? AND category = ?", childID, category).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

func (s *ChildNoteStore) Create(note *models.ChildNote) error {
	return s.db.Create(note).Error
}

func (s *ChildNoteStore) Update(note *models.ChildNote) error {
	return s.db.Save(note).Error
}

func (s *ChildNoteStore) Delete(id uint) error {
	return s.db.Delete(&models.ChildNote{}, id).Error
}

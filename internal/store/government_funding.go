package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type GovernmentFundingStore struct {
	db *gorm.DB
}

func NewGovernmentFundingStore(db *gorm.DB) *GovernmentFundingStore {
	return &GovernmentFundingStore{db: db}
}

// GovernmentFunding CRUD

func (s *GovernmentFundingStore) FindAll(limit, offset int) ([]models.GovernmentFunding, int64, error) {
	var fundings []models.GovernmentFunding
	var total int64

	if err := s.db.Model(&models.GovernmentFunding{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Limit(limit).Offset(offset).Find(&fundings).Error; err != nil {
		return nil, 0, err
	}

	return fundings, total, nil
}

func (s *GovernmentFundingStore) FindByID(id uint) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.First(&funding, id).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByName(name string) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.Where("name = ?", name).First(&funding).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByIDWithDetails(id uint) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			return db.Order("from_date DESC")
		}).
		Preload("Periods.Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("min_age ASC")
		}).
		Preload("Periods.Entries.Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		First(&funding, id).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) Create(funding *models.GovernmentFunding) error {
	return s.db.Create(funding).Error
}

func (s *GovernmentFundingStore) Update(funding *models.GovernmentFunding) error {
	return s.db.Save(funding).Error
}

func (s *GovernmentFundingStore) Delete(id uint) error {
	return s.db.Delete(&models.GovernmentFunding{}, id).Error
}

// GovernmentFundingPeriod CRUD

func (s *GovernmentFundingStore) FindPeriodByID(id uint) (*models.GovernmentFundingPeriod, error) {
	var period models.GovernmentFundingPeriod
	if err := s.db.
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("min_age ASC")
		}).
		Preload("Entries.Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		First(&period, id).Error; err != nil {
		return nil, err
	}
	return &period, nil
}

func (s *GovernmentFundingStore) FindPeriodsByGovernmentFundingID(governmentFundingID uint) ([]models.GovernmentFundingPeriod, error) {
	var periods []models.GovernmentFundingPeriod
	if err := s.db.Where("government_funding_id = ?", governmentFundingID).Order("from_date DESC").Find(&periods).Error; err != nil {
		return nil, err
	}
	return periods, nil
}

func (s *GovernmentFundingStore) CreatePeriod(period *models.GovernmentFundingPeriod) error {
	return s.db.Create(period).Error
}

func (s *GovernmentFundingStore) UpdatePeriod(period *models.GovernmentFundingPeriod) error {
	return s.db.Save(period).Error
}

func (s *GovernmentFundingStore) DeletePeriod(id uint) error {
	return s.db.Delete(&models.GovernmentFundingPeriod{}, id).Error
}

// GovernmentFundingEntry CRUD

func (s *GovernmentFundingStore) FindEntryByID(id uint) (*models.GovernmentFundingEntry, error) {
	var entry models.GovernmentFundingEntry
	if err := s.db.
		Preload("Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		First(&entry, id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *GovernmentFundingStore) CreateEntry(entry *models.GovernmentFundingEntry) error {
	return s.db.Create(entry).Error
}

func (s *GovernmentFundingStore) UpdateEntry(entry *models.GovernmentFundingEntry) error {
	return s.db.Save(entry).Error
}

func (s *GovernmentFundingStore) DeleteEntry(id uint) error {
	return s.db.Delete(&models.GovernmentFundingEntry{}, id).Error
}

// GovernmentFundingProperty CRUD

func (s *GovernmentFundingStore) FindPropertyByID(id uint) (*models.GovernmentFundingProperty, error) {
	var property models.GovernmentFundingProperty
	if err := s.db.First(&property, id).Error; err != nil {
		return nil, err
	}
	return &property, nil
}

func (s *GovernmentFundingStore) CreateProperty(property *models.GovernmentFundingProperty) error {
	return s.db.Create(property).Error
}

func (s *GovernmentFundingStore) UpdateProperty(property *models.GovernmentFundingProperty) error {
	return s.db.Save(property).Error
}

func (s *GovernmentFundingStore) DeleteProperty(id uint) error {
	return s.db.Delete(&models.GovernmentFundingProperty{}, id).Error
}

// Organization government funding assignment

func (s *GovernmentFundingStore) AssignGovernmentFundingToOrg(orgID, governmentFundingID uint) error {
	return s.db.Model(&models.Organization{}).Where("id = ?", orgID).Update("government_funding_id", governmentFundingID).Error
}

func (s *GovernmentFundingStore) RemoveGovernmentFundingFromOrg(orgID uint) error {
	return s.db.Model(&models.Organization{}).Where("id = ?", orgID).Update("government_funding_id", nil).Error
}

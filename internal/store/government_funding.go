package store

import (
	"time"

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

func (s *GovernmentFundingStore) FindByState(state string) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.Where("state = ?", state).First(&funding).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByStateWithDetails(state string, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			q := db.Order("from_date DESC")
			if activeOn != nil {
				q = q.Scopes(PeriodActiveOn("from_date", "to_date", *activeOn))
			}
			if periodsLimit > 0 {
				q = q.Limit(periodsLimit)
			}
			return q
		}).
		Preload("Periods.Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("key ASC, value ASC, min_age ASC NULLS LAST")
		}).
		Where("state = ?", state).
		First(&funding).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByIDWithDetails(id uint, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := s.db.
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			q := db.Order("from_date DESC")
			if activeOn != nil {
				q = q.Scopes(PeriodActiveOn("from_date", "to_date", *activeOn))
			}
			if periodsLimit > 0 {
				q = q.Limit(periodsLimit)
			}
			return q
		}).
		Preload("Periods.Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("key ASC, value ASC, min_age ASC NULLS LAST")
		}).
		First(&funding, id).Error; err != nil {
		return nil, err
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) CountPeriods(fundingID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&models.GovernmentFundingPeriod{}).Where("government_funding_id = ?", fundingID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
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
		Preload("Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("key ASC, value ASC, min_age ASC NULLS LAST")
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

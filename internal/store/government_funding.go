package store

import (
	"context"
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

func (s *GovernmentFundingStore) FindAll(ctx context.Context, limit, offset int) ([]models.GovernmentFunding, int64, error) {
	var fundings []models.GovernmentFunding
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.GovernmentFunding{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Limit(limit).Offset(offset).Find(&fundings).Error; err != nil {
		return nil, 0, err
	}

	return fundings, total, nil
}

func (s *GovernmentFundingStore) FindByID(ctx context.Context, id uint) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := DBFromContext(ctx, s.db).First(&funding, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByName(ctx context.Context, name string) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := DBFromContext(ctx, s.db).Where("name = ?", name).First(&funding).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByState(ctx context.Context, state string) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := DBFromContext(ctx, s.db).Where("state = ?", state).First(&funding).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByStateWithDetails(ctx context.Context, state string, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := DBFromContext(ctx, s.db).
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
		return nil, WrapNotFound(err)
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) FindByIDWithDetails(ctx context.Context, id uint, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error) {
	var funding models.GovernmentFunding
	if err := DBFromContext(ctx, s.db).
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
		return nil, WrapNotFound(err)
	}
	return &funding, nil
}

func (s *GovernmentFundingStore) CountPeriods(ctx context.Context, fundingID uint) (int64, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.GovernmentFundingPeriod{}).Where("government_funding_id = ?", fundingID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *GovernmentFundingStore) Create(ctx context.Context, funding *models.GovernmentFunding) error {
	return DBFromContext(ctx, s.db).Create(funding).Error
}

func (s *GovernmentFundingStore) Update(ctx context.Context, funding *models.GovernmentFunding) error {
	return DBFromContext(ctx, s.db).Save(funding).Error
}

func (s *GovernmentFundingStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.GovernmentFunding{}, id).Error
}

// GovernmentFundingPeriod CRUD

func (s *GovernmentFundingStore) FindPeriodByID(ctx context.Context, id uint) (*models.GovernmentFundingPeriod, error) {
	var period models.GovernmentFundingPeriod
	if err := DBFromContext(ctx, s.db).
		Preload("Properties", func(db *gorm.DB) *gorm.DB {
			return db.Order("key ASC, value ASC, min_age ASC NULLS LAST")
		}).
		First(&period, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &period, nil
}

func (s *GovernmentFundingStore) FindPeriodsByGovernmentFundingID(ctx context.Context, governmentFundingID uint) ([]models.GovernmentFundingPeriod, error) {
	var periods []models.GovernmentFundingPeriod
	if err := DBFromContext(ctx, s.db).Where("government_funding_id = ?", governmentFundingID).Order("from_date DESC").Find(&periods).Error; err != nil {
		return nil, err
	}
	return periods, nil
}

func (s *GovernmentFundingStore) CreatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error {
	return DBFromContext(ctx, s.db).Create(period).Error
}

func (s *GovernmentFundingStore) UpdatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error {
	return DBFromContext(ctx, s.db).Save(period).Error
}

func (s *GovernmentFundingStore) DeletePeriod(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.GovernmentFundingPeriod{}, id).Error
}

// GovernmentFundingProperty CRUD

func (s *GovernmentFundingStore) FindPropertyByID(ctx context.Context, id uint) (*models.GovernmentFundingProperty, error) {
	var property models.GovernmentFundingProperty
	if err := DBFromContext(ctx, s.db).First(&property, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &property, nil
}

func (s *GovernmentFundingStore) CreateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error {
	return DBFromContext(ctx, s.db).Create(property).Error
}

func (s *GovernmentFundingStore) UpdateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error {
	return DBFromContext(ctx, s.db).Save(property).Error
}

func (s *GovernmentFundingStore) DeleteProperty(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.GovernmentFundingProperty{}, id).Error
}

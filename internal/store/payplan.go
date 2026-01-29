package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// PayPlanStore handles database operations for pay plans.
type PayPlanStore struct {
	db *gorm.DB
}

// NewPayPlanStore creates a new PayPlanStore.
func NewPayPlanStore(db *gorm.DB) *PayPlanStore {
	return &PayPlanStore{db: db}
}

// Create creates a new pay plan.
func (s *PayPlanStore) Create(ctx context.Context, payplan *models.PayPlan) error {
	return s.db.WithContext(ctx).Create(payplan).Error
}

// GetByID retrieves a pay plan by ID.
func (s *PayPlanStore) GetByID(ctx context.Context, id uint) (*models.PayPlan, error) {
	var payplan models.PayPlan
	err := s.db.WithContext(ctx).First(&payplan, id).Error
	if err != nil {
		return nil, err
	}
	return &payplan, nil
}

// GetByIDWithPeriods retrieves a pay plan with all periods and entries.
func (s *PayPlanStore) GetByIDWithPeriods(ctx context.Context, id uint) (*models.PayPlan, error) {
	var payplan models.PayPlan
	err := s.db.WithContext(ctx).
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_periods.\"from\" DESC")
		}).
		Preload("Periods.Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_entries.grade ASC, pay_plan_entries.step ASC")
		}).
		First(&payplan, id).Error
	if err != nil {
		return nil, err
	}
	return &payplan, nil
}

// GetByOrganization retrieves all pay plans for an organization.
func (s *PayPlanStore) GetByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.PayPlan, int64, error) {
	var payplans []models.PayPlan
	var total int64

	query := s.db.WithContext(ctx).Model(&models.PayPlan{}).Where("organization_id = ?", orgID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&payplans).Error
	if err != nil {
		return nil, 0, err
	}

	return payplans, total, nil
}

// Update updates a pay plan.
func (s *PayPlanStore) Update(ctx context.Context, payplan *models.PayPlan) error {
	return s.db.WithContext(ctx).Save(payplan).Error
}

// Delete deletes a pay plan and all related periods and entries.
func (s *PayPlanStore) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete entries first
		if err := tx.Exec(`
			DELETE FROM pay_plan_entries
			WHERE period_id IN (SELECT id FROM pay_plan_periods WHERE pay_plan_id = ?)
		`, id).Error; err != nil {
			return err
		}

		// Delete periods
		if err := tx.Where("pay_plan_id = ?", id).Delete(&models.PayPlanPeriod{}).Error; err != nil {
			return err
		}

		// Delete pay plan
		return tx.Delete(&models.PayPlan{}, id).Error
	})
}

// Period operations

// CreatePeriod creates a new period for a pay plan.
func (s *PayPlanStore) CreatePeriod(ctx context.Context, period *models.PayPlanPeriod) error {
	return s.db.WithContext(ctx).Create(period).Error
}

// GetPeriodByID retrieves a period by ID.
func (s *PayPlanStore) GetPeriodByID(ctx context.Context, id uint) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := s.db.WithContext(ctx).First(&period, id).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// GetPeriodByIDWithEntries retrieves a period with all entries.
func (s *PayPlanStore) GetPeriodByIDWithEntries(ctx context.Context, id uint) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := s.db.WithContext(ctx).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_entries.grade ASC, pay_plan_entries.step ASC")
		}).
		First(&period, id).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// GetPeriodsByPayPlan retrieves all periods for a pay plan.
func (s *PayPlanStore) GetPeriodsByPayPlan(ctx context.Context, payplanID uint) ([]models.PayPlanPeriod, error) {
	var periods []models.PayPlanPeriod
	err := s.db.WithContext(ctx).
		Where("pay_plan_id = ?", payplanID).
		Order("\"from\" DESC").
		Find(&periods).Error
	if err != nil {
		return nil, err
	}
	return periods, nil
}

// GetActivePeriod retrieves the active period for a pay plan at a given date.
func (s *PayPlanStore) GetActivePeriod(ctx context.Context, payplanID uint, date time.Time) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := s.db.WithContext(ctx).
		Preload("Entries").
		Where("pay_plan_id = ? AND \"from\" <= ? AND (\"to\" IS NULL OR \"to\" >= ?)", payplanID, date, date).
		Order("\"from\" DESC").
		First(&period).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// UpdatePeriod updates a period.
func (s *PayPlanStore) UpdatePeriod(ctx context.Context, period *models.PayPlanPeriod) error {
	return s.db.WithContext(ctx).Save(period).Error
}

// DeletePeriod deletes a period and all related entries.
func (s *PayPlanStore) DeletePeriod(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete entries first
		if err := tx.Where("period_id = ?", id).Delete(&models.PayPlanEntry{}).Error; err != nil {
			return err
		}
		// Delete period
		return tx.Delete(&models.PayPlanPeriod{}, id).Error
	})
}

// Entry operations

// CreateEntry creates a new entry for a period.
func (s *PayPlanStore) CreateEntry(ctx context.Context, entry *models.PayPlanEntry) error {
	return s.db.WithContext(ctx).Create(entry).Error
}

// GetEntryByID retrieves an entry by ID.
func (s *PayPlanStore) GetEntryByID(ctx context.Context, id uint) (*models.PayPlanEntry, error) {
	var entry models.PayPlanEntry
	err := s.db.WithContext(ctx).First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetEntriesByPeriod retrieves all entries for a period.
func (s *PayPlanStore) GetEntriesByPeriod(ctx context.Context, periodID uint) ([]models.PayPlanEntry, error) {
	var entries []models.PayPlanEntry
	err := s.db.WithContext(ctx).
		Where("period_id = ?", periodID).
		Order("grade ASC, step ASC").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GetEntry retrieves a specific entry by grade and step.
func (s *PayPlanStore) GetEntry(ctx context.Context, periodID uint, grade string, step int) (*models.PayPlanEntry, error) {
	var entry models.PayPlanEntry
	err := s.db.WithContext(ctx).
		Where("period_id = ? AND grade = ? AND step = ?", periodID, grade, step).
		First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// UpdateEntry updates an entry.
func (s *PayPlanStore) UpdateEntry(ctx context.Context, entry *models.PayPlanEntry) error {
	return s.db.WithContext(ctx).Save(entry).Error
}

// DeleteEntry deletes an entry.
func (s *PayPlanStore) DeleteEntry(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PayPlanEntry{}, id).Error
}

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
	return DBFromContext(ctx, s.db).Create(payplan).Error
}

// GetByID retrieves a pay plan by ID.
func (s *PayPlanStore) FindByID(ctx context.Context, id uint) (*models.PayPlan, error) {
	var payplan models.PayPlan
	err := DBFromContext(ctx, s.db).First(&payplan, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &payplan, nil
}

// GetByIDWithPeriods retrieves a pay plan with all periods and entries.
// If activeOn is non-nil, only periods active on that date are returned.
func (s *PayPlanStore) FindByIDWithPeriods(ctx context.Context, id uint, activeOn *time.Time) (*models.PayPlan, error) {
	var payplan models.PayPlan
	err := DBFromContext(ctx, s.db).
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			q := db.Order("pay_plan_periods.from_date DESC")
			if activeOn != nil {
				q = q.Scopes(PeriodActiveOn("from_date", "to_date", *activeOn))
			}
			return q
		}).
		Preload("Periods.Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_entries.grade ASC, pay_plan_entries.step ASC")
		}).
		First(&payplan, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &payplan, nil
}

// FindByIDsWithPeriods retrieves multiple pay plans by IDs with all periods and entries.
// Returns a map keyed by pay plan ID. IDs that don't exist are silently omitted.
func (s *PayPlanStore) FindByIDsWithPeriods(ctx context.Context, ids []uint) (map[uint]*models.PayPlan, error) {
	result := make(map[uint]*models.PayPlan, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var payplans []models.PayPlan
	err := DBFromContext(ctx, s.db).
		Preload("Periods", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_periods.from_date DESC")
		}).
		Preload("Periods.Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_entries.grade ASC, pay_plan_entries.step ASC")
		}).
		Where("id IN ?", ids).
		Find(&payplans).Error
	if err != nil {
		return nil, err
	}

	for i := range payplans {
		result[payplans[i].ID] = &payplans[i]
	}
	return result, nil
}

// GetByOrganization retrieves all pay plans for an organization.
func (s *PayPlanStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.PayPlan, int64, error) {
	var payplans []models.PayPlan
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.PayPlan{}).Where("organization_id = ?", orgID)

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

// FindByNameAndOrg retrieves a pay plan by name within an organization.
func (s *PayPlanStore) FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.PayPlan, error) {
	var payplan models.PayPlan
	err := DBFromContext(ctx, s.db).
		Where("name = ? AND organization_id = ?", name, orgID).
		First(&payplan).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &payplan, nil
}

// Update updates a pay plan.
func (s *PayPlanStore) Update(ctx context.Context, payplan *models.PayPlan) error {
	return DBFromContext(ctx, s.db).Save(payplan).Error
}

// Delete deletes a pay plan and all related periods and entries.
func (s *PayPlanStore) Delete(ctx context.Context, id uint) error {
	if err := s.DeletePeriodsAndEntries(ctx, id); err != nil {
		return err
	}
	return DBFromContext(ctx, s.db).Delete(&models.PayPlan{}, id).Error
}

// DeletePeriodsAndEntries deletes all periods and entries for a pay plan, but keeps the pay plan itself.
func (s *PayPlanStore) DeletePeriodsAndEntries(ctx context.Context, payplanID uint) error {
	db := DBFromContext(ctx, s.db)

	// Delete entries first (via subquery on periods)
	if err := db.Where("period_id IN (?)",
		db.Model(&models.PayPlanPeriod{}).Select("id").Where("pay_plan_id = ?", payplanID),
	).Delete(&models.PayPlanEntry{}).Error; err != nil {
		return err
	}

	// Delete periods
	return db.Where("pay_plan_id = ?", payplanID).Delete(&models.PayPlanPeriod{}).Error
}

// Period operations

// CreatePeriod creates a new period for a pay plan.
func (s *PayPlanStore) CreatePeriod(ctx context.Context, period *models.PayPlanPeriod) error {
	return DBFromContext(ctx, s.db).Create(period).Error
}

// GetPeriodByID retrieves a period by ID.
func (s *PayPlanStore) FindPeriodByID(ctx context.Context, id uint) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := DBFromContext(ctx, s.db).First(&period, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &period, nil
}

// GetPeriodByIDWithEntries retrieves a period with all entries.
func (s *PayPlanStore) FindPeriodByIDWithEntries(ctx context.Context, id uint) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := DBFromContext(ctx, s.db).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("pay_plan_entries.grade ASC, pay_plan_entries.step ASC")
		}).
		First(&period, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &period, nil
}

// GetPeriodsByPayPlan retrieves all periods for a pay plan.
func (s *PayPlanStore) FindPeriodsByPayPlan(ctx context.Context, payplanID uint) ([]models.PayPlanPeriod, error) {
	var periods []models.PayPlanPeriod
	err := DBFromContext(ctx, s.db).
		Where("pay_plan_id = ?", payplanID).
		Order("from_date DESC").
		Find(&periods).Error
	if err != nil {
		return nil, err
	}
	return periods, nil
}

// GetActivePeriod retrieves the active period for a pay plan at a given date.
func (s *PayPlanStore) FindActivePeriod(ctx context.Context, payplanID uint, date time.Time) (*models.PayPlanPeriod, error) {
	var period models.PayPlanPeriod
	err := DBFromContext(ctx, s.db).
		Preload("Entries").
		Where("pay_plan_id = ?", payplanID).
		Scopes(PeriodActiveOn("from_date", "to_date", date)).
		Order("from_date DESC").
		First(&period).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &period, nil
}

// UpdatePeriod updates a period.
func (s *PayPlanStore) UpdatePeriod(ctx context.Context, period *models.PayPlanPeriod) error {
	return DBFromContext(ctx, s.db).Save(period).Error
}

// DeletePeriod deletes a period and all related entries.
func (s *PayPlanStore) DeletePeriod(ctx context.Context, id uint) error {
	db := DBFromContext(ctx, s.db)
	// Delete entries first
	if err := db.Where("period_id = ?", id).Delete(&models.PayPlanEntry{}).Error; err != nil {
		return err
	}
	// Delete period
	return db.Delete(&models.PayPlanPeriod{}, id).Error
}

// Entry operations

// CreateEntry creates a new entry for a period.
func (s *PayPlanStore) CreateEntry(ctx context.Context, entry *models.PayPlanEntry) error {
	return DBFromContext(ctx, s.db).Create(entry).Error
}

// GetEntryByID retrieves an entry by ID.
func (s *PayPlanStore) FindEntryByID(ctx context.Context, id uint) (*models.PayPlanEntry, error) {
	var entry models.PayPlanEntry
	err := DBFromContext(ctx, s.db).First(&entry, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &entry, nil
}

// GetEntriesByPeriod retrieves all entries for a period.
func (s *PayPlanStore) FindEntriesByPeriod(ctx context.Context, periodID uint) ([]models.PayPlanEntry, error) {
	var entries []models.PayPlanEntry
	err := DBFromContext(ctx, s.db).
		Where("period_id = ?", periodID).
		Order("grade ASC, step ASC").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GetEntry retrieves a specific entry by grade and step.
func (s *PayPlanStore) FindEntry(ctx context.Context, periodID uint, grade string, step int) (*models.PayPlanEntry, error) {
	var entry models.PayPlanEntry
	err := DBFromContext(ctx, s.db).
		Where("period_id = ? AND grade = ? AND step = ?", periodID, grade, step).
		First(&entry).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &entry, nil
}

// UpdateEntry updates an entry.
func (s *PayPlanStore) UpdateEntry(ctx context.Context, entry *models.PayPlanEntry) error {
	return DBFromContext(ctx, s.db).Save(entry).Error
}

// DeleteEntry deletes an entry.
func (s *PayPlanStore) DeleteEntry(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.PayPlanEntry{}, id).Error
}

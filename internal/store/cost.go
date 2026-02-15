package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// CostStore handles database operations for costs and cost entries.
type CostStore struct {
	db         *gorm.DB
	periodStore *PeriodStore[models.CostEntry]
}

// NewCostStore creates a new CostStore.
func NewCostStore(db *gorm.DB) *CostStore {
	return &CostStore{
		db:         db,
		periodStore: NewPeriodStore[models.CostEntry](db, "cost_id"),
	}
}

// Entries returns the period store for cost entries (overlap validation etc.).
func (s *CostStore) Entries() PeriodStorer[models.CostEntry] {
	return s.periodStore
}

// Cost CRUD

// Create creates a new cost.
func (s *CostStore) Create(ctx context.Context, cost *models.Cost) error {
	return DBFromContext(ctx, s.db).Create(cost).Error
}

// FindByID retrieves a cost by ID.
func (s *CostStore) FindByID(ctx context.Context, id uint) (*models.Cost, error) {
	var cost models.Cost
	err := DBFromContext(ctx, s.db).First(&cost, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &cost, nil
}

// FindByIDWithEntries retrieves a cost with all entries.
func (s *CostStore) FindByIDWithEntries(ctx context.Context, id uint) (*models.Cost, error) {
	var cost models.Cost
	err := DBFromContext(ctx, s.db).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("cost_entries.from_date DESC")
		}).
		First(&cost, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &cost, nil
}

// FindByOrganization retrieves all costs for an organization.
func (s *CostStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Cost, int64, error) {
	var costs []models.Cost
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.Cost{}).Where("organization_id = ?", orgID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&costs).Error
	if err != nil {
		return nil, 0, err
	}

	return costs, total, nil
}

// Update updates a cost.
func (s *CostStore) Update(ctx context.Context, cost *models.Cost) error {
	return DBFromContext(ctx, s.db).Save(cost).Error
}

// Delete deletes a cost and all related entries.
func (s *CostStore) Delete(ctx context.Context, id uint) error {
	db := DBFromContext(ctx, s.db)

	// Delete entries first
	if err := db.Where("cost_id = ?", id).Delete(&models.CostEntry{}).Error; err != nil {
		return err
	}

	// Delete cost
	return db.Delete(&models.Cost{}, id).Error
}

// CostEntry CRUD

// CreateEntry creates a new cost entry.
func (s *CostStore) CreateEntry(ctx context.Context, entry *models.CostEntry) error {
	return DBFromContext(ctx, s.db).Create(entry).Error
}

// FindEntryByID retrieves a cost entry by ID.
func (s *CostStore) FindEntryByID(ctx context.Context, id uint) (*models.CostEntry, error) {
	var entry models.CostEntry
	err := DBFromContext(ctx, s.db).First(&entry, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &entry, nil
}

// FindEntriesByCostPaginated retrieves paginated entries for a cost ordered by from_date desc.
func (s *CostStore) FindEntriesByCostPaginated(ctx context.Context, costID uint, limit, offset int) ([]models.CostEntry, int64, error) {
	var entries []models.CostEntry
	var total int64

	db := DBFromContext(ctx, s.db)
	if err := db.Model(&models.CostEntry{}).Where("cost_id = ?", costID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Where("cost_id = ?", costID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	return entries, total, err
}

// UpdateEntry updates a cost entry.
func (s *CostStore) UpdateEntry(ctx context.Context, entry *models.CostEntry) error {
	return DBFromContext(ctx, s.db).Save(entry).Error
}

// DeleteEntry deletes a cost entry.
func (s *CostStore) DeleteEntry(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.CostEntry{}, id).Error
}

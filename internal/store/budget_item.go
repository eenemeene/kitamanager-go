package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// BudgetItemStore handles database operations for budget items and their entries.
type BudgetItemStore struct {
	db          *gorm.DB
	periodStore *PeriodStore[models.BudgetItemEntry]
}

// NewBudgetItemStore creates a new BudgetItemStore.
func NewBudgetItemStore(db *gorm.DB) *BudgetItemStore {
	return &BudgetItemStore{
		db:          db,
		periodStore: NewPeriodStore[models.BudgetItemEntry](db, "budget_item_id"),
	}
}

// Entries returns the period store for budget item entries (overlap validation etc.).
func (s *BudgetItemStore) Entries() PeriodStorer[models.BudgetItemEntry] {
	return s.periodStore
}

// BudgetItem CRUD

// Create creates a new budget item.
func (s *BudgetItemStore) Create(ctx context.Context, item *models.BudgetItem) error {
	return DBFromContext(ctx, s.db).Create(item).Error
}

// FindByID retrieves a budget item by ID.
func (s *BudgetItemStore) FindByID(ctx context.Context, id uint) (*models.BudgetItem, error) {
	var item models.BudgetItem
	err := DBFromContext(ctx, s.db).First(&item, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &item, nil
}

// FindByIDWithEntries retrieves a budget item with all entries.
func (s *BudgetItemStore) FindByIDWithEntries(ctx context.Context, id uint) (*models.BudgetItem, error) {
	var item models.BudgetItem
	err := DBFromContext(ctx, s.db).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("budget_item_entries.from_date DESC")
		}).
		First(&item, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &item, nil
}

// FindByOrganization retrieves all budget items for an organization.
func (s *BudgetItemStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.BudgetItem, int64, error) {
	var items []models.BudgetItem
	var total int64

	db := DBFromContext(ctx, s.db)

	if err := db.Model(&models.BudgetItem{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.
		Where("organization_id = ?", orgID).
		Preload("Entries").
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// FindByOrganizationWithEntries retrieves all budget items for an organization with entries preloaded.
func (s *BudgetItemStore) FindByOrganizationWithEntries(ctx context.Context, orgID uint) ([]models.BudgetItem, error) {
	var items []models.BudgetItem
	err := DBFromContext(ctx, s.db).
		Where("organization_id = ?", orgID).
		Preload("Entries").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// Update updates a budget item.
func (s *BudgetItemStore) Update(ctx context.Context, item *models.BudgetItem) error {
	return DBFromContext(ctx, s.db).Save(item).Error
}

// Delete deletes a budget item and all related entries.
func (s *BudgetItemStore) Delete(ctx context.Context, id uint) error {
	db := DBFromContext(ctx, s.db)

	// Delete entries first
	if err := db.Where("budget_item_id = ?", id).Delete(&models.BudgetItemEntry{}).Error; err != nil {
		return err
	}

	// Delete budget item
	return db.Delete(&models.BudgetItem{}, id).Error
}

// BudgetItemEntry CRUD

// CreateEntry creates a new budget item entry.
func (s *BudgetItemStore) CreateEntry(ctx context.Context, entry *models.BudgetItemEntry) error {
	return DBFromContext(ctx, s.db).Create(entry).Error
}

// FindEntryByID retrieves a budget item entry by ID.
func (s *BudgetItemStore) FindEntryByID(ctx context.Context, id uint) (*models.BudgetItemEntry, error) {
	var entry models.BudgetItemEntry
	err := DBFromContext(ctx, s.db).First(&entry, id).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &entry, nil
}

// FindEntriesByBudgetItemPaginated retrieves paginated entries for a budget item ordered by from_date desc.
func (s *BudgetItemStore) FindEntriesByBudgetItemPaginated(ctx context.Context, budgetItemID uint, limit, offset int) ([]models.BudgetItemEntry, int64, error) {
	var entries []models.BudgetItemEntry
	var total int64

	db := DBFromContext(ctx, s.db)
	if err := db.Model(&models.BudgetItemEntry{}).Where("budget_item_id = ?", budgetItemID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Where("budget_item_id = ?", budgetItemID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	return entries, total, err
}

// UpdateEntry updates a budget item entry.
func (s *BudgetItemStore) UpdateEntry(ctx context.Context, entry *models.BudgetItemEntry) error {
	return DBFromContext(ctx, s.db).Save(entry).Error
}

// DeleteEntry deletes a budget item entry.
func (s *BudgetItemStore) DeleteEntry(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.BudgetItemEntry{}, id).Error
}

package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type OrganizationStore struct {
	db *gorm.DB
}

func NewOrganizationStore(db *gorm.DB) *OrganizationStore {
	return &OrganizationStore{db: db}
}

func (s *OrganizationStore) FindAll(ctx context.Context, search string, limit, offset int) ([]models.Organization, int64, error) {
	var organizations []models.Organization
	var total int64

	countQuery := DBFromContext(ctx, s.db).Model(&models.Organization{})
	if search != "" {
		countQuery = countQuery.Scopes(NameSearch("organizations", "name", search))
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := DBFromContext(ctx, s.db).Model(&models.Organization{})
	if search != "" {
		dataQuery = dataQuery.Scopes(NameSearch("organizations", "name", search))
	}
	if err := dataQuery.Limit(limit).Offset(offset).Find(&organizations).Error; err != nil {
		return nil, 0, err
	}

	return organizations, total, nil
}

func (s *OrganizationStore) FindByID(ctx context.Context, id uint) (*models.Organization, error) {
	var organization models.Organization
	if err := DBFromContext(ctx, s.db).First(&organization, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &organization, nil
}

func (s *OrganizationStore) Create(ctx context.Context, organization *models.Organization) error {
	return DBFromContext(ctx, s.db).Create(organization).Error
}

func (s *OrganizationStore) CreateWithDefaultSection(ctx context.Context, org *models.Organization, defaultSection *models.Section) error {
	return DBFromContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		defaultSection.OrganizationID = org.ID
		return tx.Create(defaultSection).Error
	})
}

func (s *OrganizationStore) Update(ctx context.Context, organization *models.Organization) error {
	return DBFromContext(ctx, s.db).Save(organization).Error
}

// Delete soft-deletes the organization — GORM turns this into
// `UPDATE organizations SET deleted_at = now() WHERE id = ?` thanks
// to the DeletedAt field on the model. Live queries stop seeing
// the row; admin trash-view / HardDelete still can via Unscoped().
func (s *OrganizationStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.Organization{}, id).Error
}

// HardDelete issues the physical DELETE, bypassing the tombstone.
// Cascades through pay_plans, government_funding_bill_periods,
// employees, children, sections, etc. via the FKs defined in
// migration 000001 + 000014.
func (s *OrganizationStore) HardDelete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Unscoped().Delete(&models.Organization{}, id).Error
}

// FindByIDUnscoped fetches whether tombstoned or live.
func (s *OrganizationStore) FindByIDUnscoped(ctx context.Context, id uint, out *models.Organization) error {
	return DBFromContext(ctx, s.db).Unscoped().First(out, id).Error
}

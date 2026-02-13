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
	if err := DBFromContext(ctx, s.db).Preload("Groups").First(&organization, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &organization, nil
}

func (s *OrganizationStore) Create(ctx context.Context, organization *models.Organization) error {
	return DBFromContext(ctx, s.db).Create(organization).Error
}

func (s *OrganizationStore) CreateWithDefaultGroup(ctx context.Context, org *models.Organization, defaultGroup *models.Group) error {
	db := DBFromContext(ctx, s.db)
	if err := db.Create(org).Error; err != nil {
		return err
	}
	defaultGroup.OrganizationID = org.ID
	return db.Create(defaultGroup).Error
}

func (s *OrganizationStore) Update(ctx context.Context, organization *models.Organization) error {
	return DBFromContext(ctx, s.db).Save(organization).Error
}

func (s *OrganizationStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.Organization{}, id).Error
}

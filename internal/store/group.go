package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type GroupStore struct {
	db *gorm.DB
}

func NewGroupStore(db *gorm.DB) *GroupStore {
	return &GroupStore{db: db}
}

func (s *GroupStore) FindAll(ctx context.Context, limit, offset int) ([]models.Group, int64, error) {
	var groups []models.Group
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.Group{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Preload("Organization").Limit(limit).Offset(offset).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (s *GroupStore) FindByID(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	if err := DBFromContext(ctx, s.db).Preload("Users").Preload("Organization").First(&group, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &group, nil
}

func (s *GroupStore) FindByOrganizationPaginated(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.Group, int64, error) {
	var groups []models.Group
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.Group{}).Where("organization_id = ?", orgID)
	if search != "" {
		query = query.Scopes(NameSearch("groups", "name", search))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := DBFromContext(ctx, s.db).Preload("Organization").Where("organization_id = ?", orgID)
	if search != "" {
		dataQuery = dataQuery.Scopes(NameSearch("groups", "name", search))
	}

	if err := dataQuery.Limit(limit).Offset(offset).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (s *GroupStore) FindDefaultGroup(ctx context.Context, orgID uint) (*models.Group, error) {
	var group models.Group
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND is_default = ?", orgID, true).First(&group).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &group, nil
}

func (s *GroupStore) Create(ctx context.Context, group *models.Group) error {
	return DBFromContext(ctx, s.db).Create(group).Error
}

func (s *GroupStore) Update(ctx context.Context, group *models.Group) error {
	return DBFromContext(ctx, s.db).Save(group).Error
}

func (s *GroupStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.Group{}, id).Error
}

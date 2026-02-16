package store

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) FindAll(ctx context.Context, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.User{})
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := DBFromContext(ctx, s.db).Preload("Groups")
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		dataQuery = dataQuery.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
	}

	if err := dataQuery.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Count distinct users in the organization
	countQuery := DBFromContext(ctx, s.db).Model(&models.User{}).
		Distinct().
		Joins("JOIN user_groups ON user_groups.user_id = users.id").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("groups.organization_id = ?", orgID)
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		countQuery = countQuery.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get users with their groups (filtered to this org's groups)
	dataQuery := s.db.
		Distinct().
		Joins("JOIN user_groups ON user_groups.user_id = users.id").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("groups.organization_id = ?", orgID).
		Preload("Groups", "organization_id = ?", orgID)
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		dataQuery = dataQuery.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
	}
	if err := dataQuery.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := DBFromContext(ctx, s.db).Preload("Groups").Preload("Groups.Organization").First(&user, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &user, nil
}

func (s *UserStore) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := DBFromContext(ctx, s.db).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &user, nil
}

func (s *UserStore) EmailExistsForOtherUser(ctx context.Context, email string, excludeUserID uint) (bool, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.User{}).Where("email = ? AND id != ?", email, excludeUserID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *UserStore) Create(ctx context.Context, user *models.User) error {
	return DBFromContext(ctx, s.db).Create(user).Error
}

func (s *UserStore) Update(ctx context.Context, user *models.User) error {
	return DBFromContext(ctx, s.db).Save(user).Error
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, userID uint) error {
	return DBFromContext(ctx, s.db).Model(&models.User{}).Where("id = ?", userID).Update("last_login", time.Now()).Error
}

func (s *UserStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.User{}, id).Error
}

func (s *UserStore) AddToGroup(ctx context.Context, userID, groupID uint) error {
	user := &models.User{ID: userID}
	group := &models.Group{ID: groupID}
	return DBFromContext(ctx, s.db).Model(user).Association("Groups").Append(group)
}

func (s *UserStore) RemoveFromGroup(ctx context.Context, userID, groupID uint) error {
	user := &models.User{ID: userID}
	group := &models.Group{ID: groupID}
	return DBFromContext(ctx, s.db).Model(user).Association("Groups").Delete(group)
}

func (s *UserStore) RemoveFromAllGroupsInOrg(ctx context.Context, userID, orgID uint) error {
	// Find all groups in the organization that the user belongs to
	var groups []models.Group
	err := DBFromContext(ctx, s.db).Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ? AND groups.organization_id = ?", userID, orgID).
		Find(&groups).Error
	if err != nil {
		return err
	}

	// Remove user from each group
	user := &models.User{ID: userID}
	for _, group := range groups {
		g := group // avoid closure issue
		if err := DBFromContext(ctx, s.db).Model(user).Association("Groups").Delete(&g); err != nil {
			return err
		}
	}
	return nil
}

func (s *UserStore) GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error) {
	var orgs []models.Organization
	err := s.db.Distinct().
		Joins("JOIN groups ON groups.organization_id = organizations.id").
		Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", userID).
		Find(&orgs).Error
	return orgs, err
}

func (s *UserStore) FindByOrganizations(ctx context.Context, orgIDs []uint, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	baseQuery := func(q *gorm.DB) *gorm.DB {
		q = q.Distinct().
			Joins("JOIN user_groups ON user_groups.user_id = users.id").
			Joins("JOIN groups ON groups.id = user_groups.group_id").
			Where("groups.organization_id IN ?", orgIDs)
		if search != "" {
			pattern := "%" + strings.ToLower(search) + "%"
			q = q.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
		}
		return q
	}

	if err := baseQuery(DBFromContext(ctx, s.db).Model(&models.User{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := baseQuery(DBFromContext(ctx, s.db).Preload("Groups")).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) SharesOrganization(ctx context.Context, userID1, userID2 uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.UserGroup{}).
		Select("COUNT(DISTINCT ug1.user_id)").
		Table("user_groups ug1").
		Joins("JOIN groups g1 ON g1.id = ug1.group_id").
		Joins("JOIN groups g2 ON g2.organization_id = g1.organization_id").
		Joins("JOIN user_groups ug2 ON ug2.group_id = g2.id").
		Where("ug1.user_id = ? AND ug2.user_id = ?", userID1, userID2).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

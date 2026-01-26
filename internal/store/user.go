package store

import (
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

func (s *UserStore) FindAll(limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Preload("Groups").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByOrganization(orgID uint, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Count distinct users in the organization
	if err := s.db.Model(&models.User{}).
		Distinct().
		Joins("JOIN user_groups ON user_groups.user_id = users.id").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("groups.organization_id = ?", orgID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get users with their groups (filtered to this org's groups)
	if err := s.db.
		Distinct().
		Joins("JOIN user_groups ON user_groups.user_id = users.id").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("groups.organization_id = ?", orgID).
		Preload("Groups", "organization_id = ?", orgID).
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Groups").Preload("Groups.Organization").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) EmailExistsForOtherUser(email string, excludeUserID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&models.User{}).Where("email = ? AND id != ?", email, excludeUserID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *UserStore) Create(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *UserStore) Update(user *models.User) error {
	return s.db.Save(user).Error
}

func (s *UserStore) UpdateLastLogin(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login", time.Now()).Error
}

func (s *UserStore) Delete(id uint) error {
	return s.db.Delete(&models.User{}, id).Error
}

func (s *UserStore) AddToGroup(userID, groupID uint) error {
	user := &models.User{ID: userID}
	group := &models.Group{ID: groupID}
	return s.db.Model(user).Association("Groups").Append(group)
}

func (s *UserStore) RemoveFromGroup(userID, groupID uint) error {
	user := &models.User{ID: userID}
	group := &models.Group{ID: groupID}
	return s.db.Model(user).Association("Groups").Delete(group)
}

func (s *UserStore) RemoveFromAllGroupsInOrg(userID, orgID uint) error {
	// Find all groups in the organization that the user belongs to
	var groups []models.Group
	err := s.db.Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ? AND groups.organization_id = ?", userID, orgID).
		Find(&groups).Error
	if err != nil {
		return err
	}

	// Remove user from each group
	user := &models.User{ID: userID}
	for _, group := range groups {
		g := group // avoid closure issue
		if err := s.db.Model(user).Association("Groups").Delete(&g); err != nil {
			return err
		}
	}
	return nil
}

func (s *UserStore) GetUserOrganizations(userID uint) ([]models.Organization, error) {
	var orgs []models.Organization
	err := s.db.Distinct().
		Joins("JOIN groups ON groups.organization_id = organizations.id").
		Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", userID).
		Find(&orgs).Error
	return orgs, err
}

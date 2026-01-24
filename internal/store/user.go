package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) FindAll() ([]models.User, error) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserStore) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Organizations").Preload("Groups").First(&user, id).Error; err != nil {
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

func (s *UserStore) Create(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *UserStore) Update(user *models.User) error {
	return s.db.Save(user).Error
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

func (s *UserStore) AddToOrganization(userID, orgID uint) error {
	user := &models.User{ID: userID}
	org := &models.Organization{ID: orgID}
	return s.db.Model(user).Association("Organizations").Append(org)
}

func (s *UserStore) RemoveFromOrganization(userID, orgID uint) error {
	user := &models.User{ID: userID}
	org := &models.Organization{ID: orgID}
	return s.db.Model(user).Association("Organizations").Delete(org)
}

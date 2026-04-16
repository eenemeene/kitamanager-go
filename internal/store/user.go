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

// userSearchScope returns a GORM scope that filters users by name or email.
func userSearchScope(search string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if search == "" {
			return q
		}
		pattern := "%" + strings.ToLower(search) + "%"
		return q.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern)
	}
}

func (s *UserStore) FindAll(ctx context.Context, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	scope := userSearchScope(search)

	if err := DBFromContext(ctx, s.db).Model(&models.User{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Scopes(scope).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	scope := userSearchScope(search)

	orgJoin := func(q *gorm.DB) *gorm.DB {
		return q.Distinct().
			Joins("JOIN user_organizations ON user_organizations.user_id = users.id").
			Where("user_organizations.organization_id = ?", orgID)
	}

	if err := DBFromContext(ctx, s.db).Model(&models.User{}).Scopes(orgJoin, scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Scopes(orgJoin, scope).
		Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := DBFromContext(ctx, s.db).First(&user, id).Error; err != nil {
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
	return DBFromContext(ctx, s.db).Model(&models.User{}).Where("id = ?", userID).Update("last_login", time.Now().UTC()).Error
}

func (s *UserStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.User{}, id).Error
}

func (s *UserStore) GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error) {
	var orgs []models.Organization
	err := DBFromContext(ctx, s.db).Distinct().
		Joins("JOIN user_organizations ON user_organizations.organization_id = organizations.id").
		Where("user_organizations.user_id = ?", userID).
		Find(&orgs).Error
	return orgs, err
}

func (s *UserStore) FindByOrganizations(ctx context.Context, orgIDs []uint, search string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	scope := userSearchScope(search)

	orgsJoin := func(q *gorm.DB) *gorm.DB {
		return q.Distinct().
			Joins("JOIN user_organizations ON user_organizations.user_id = users.id").
			Where("user_organizations.organization_id IN ?", orgIDs)
	}

	if err := DBFromContext(ctx, s.db).Model(&models.User{}).Scopes(orgsJoin, scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Scopes(orgsJoin, scope).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserStore) SharesOrganization(ctx context.Context, userID1, userID2 uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Table("user_organizations uo1").
		Joins("JOIN user_organizations uo2 ON uo2.organization_id = uo1.organization_id").
		Where("uo1.user_id = ? AND uo2.user_id = ?", userID1, userID2).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsAdminInSharedOrg checks whether the requester has admin role in at least one
// organization that the target user belongs to.
func (s *UserStore) IsAdminInSharedOrg(ctx context.Context, requesterID, targetUserID uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Table("user_organizations uo_req").
		Joins("JOIN user_organizations uo_target ON uo_target.organization_id = uo_req.organization_id").
		Where("uo_req.user_id = ? AND uo_target.user_id = ? AND uo_req.role = ?",
			requesterID, targetUserID, "admin").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

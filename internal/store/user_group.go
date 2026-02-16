package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserGroupStore handles database operations for user-group relationships
type UserGroupStore struct {
	db *gorm.DB
}

// NewUserGroupStore creates a new UserGroupStore
func NewUserGroupStore(db *gorm.DB) *UserGroupStore {
	return &UserGroupStore{db: db}
}

// AddUserToGroup adds a user to a group with a specified role
func (s *UserGroupStore) AddUserToGroup(ctx context.Context, userID, groupID uint, role models.Role, createdBy string) (*models.UserGroup, error) {
	ug := &models.UserGroup{
		UserID:    userID,
		GroupID:   groupID,
		Role:      role,
		CreatedBy: createdBy,
	}

	if err := DBFromContext(ctx, s.db).Create(ug).Error; err != nil {
		return nil, err
	}

	return ug, nil
}

// UpdateRole updates a user's role in a group
func (s *UserGroupStore) UpdateRole(ctx context.Context, userID, groupID uint, role models.Role) error {
	result := DBFromContext(ctx, s.db).Model(&models.UserGroup{}).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RemoveUserFromGroup removes a user from a group
func (s *UserGroupStore) RemoveUserFromGroup(ctx context.Context, userID, groupID uint) error {
	result := DBFromContext(ctx, s.db).Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&models.UserGroup{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// FindByUserAndGroup finds a specific user-group relationship
func (s *UserGroupStore) FindByUserAndGroup(ctx context.Context, userID, groupID uint) (*models.UserGroup, error) {
	var ug models.UserGroup
	err := DBFromContext(ctx, s.db).Where("user_id = ? AND group_id = ?", userID, groupID).First(&ug).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &ug, nil
}

// FindByUser returns all group memberships for a user
func (s *UserGroupStore) FindByUser(ctx context.Context, userID uint) ([]models.UserGroup, error) {
	var memberships []models.UserGroup
	err := s.db.
		Preload("Group").
		Preload("Group.Organization").
		Where("user_id = ?", userID).
		Find(&memberships).Error
	return memberships, err
}

// FindByGroup returns all user memberships in a group
func (s *UserGroupStore) FindByGroup(ctx context.Context, groupID uint) ([]models.UserGroup, error) {
	var memberships []models.UserGroup
	err := s.db.
		Preload("User").
		Where("group_id = ?", groupID).
		Find(&memberships).Error
	return memberships, err
}

// FindByUserAndOrg returns all user-group memberships for a user in a specific organization
func (s *UserGroupStore) FindByUserAndOrg(ctx context.Context, userID, orgID uint) ([]models.UserGroup, error) {
	var memberships []models.UserGroup
	err := s.db.
		Preload("Group").
		Preload("Group.Organization").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("user_groups.user_id = ? AND groups.organization_id = ?", userID, orgID).
		Find(&memberships).Error
	return memberships, err
}

// GetEffectiveRoleInOrg returns the highest role a user has in an organization
// Returns empty string if user has no role in the organization
func (s *UserGroupStore) GetEffectiveRoleInOrg(ctx context.Context, userID, orgID uint) (models.Role, error) {
	memberships, err := s.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return "", err
	}

	if len(memberships) == 0 {
		return "", nil
	}

	// Find highest precedence role
	var effectiveRole models.Role
	highestPrecedence := 0

	for _, m := range memberships {
		if m.Role.Precedence() > highestPrecedence {
			highestPrecedence = m.Role.Precedence()
			effectiveRole = m.Role
		}
	}

	return effectiveRole, nil
}

// GetUserOrganizationsWithRoles returns all organizations a user belongs to with their effective roles
func (s *UserGroupStore) GetUserOrganizationsWithRoles(ctx context.Context, userID uint) (map[uint]models.Role, error) {
	memberships, err := s.FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Group by organization and find highest role in each
	orgRoles := make(map[uint]models.Role)
	orgPrecedence := make(map[uint]int)

	for _, m := range memberships {
		if m.Group != nil {
			orgID := m.Group.OrganizationID
			if m.Role.Precedence() > orgPrecedence[orgID] {
				orgPrecedence[orgID] = m.Role.Precedence()
				orgRoles[orgID] = m.Role
			}
		}
	}

	return orgRoles, nil
}

// RemoveUserFromOrg removes a user from all groups in an organization
func (s *UserGroupStore) RemoveUserFromOrg(ctx context.Context, userID, orgID uint) error {
	// Find all groups in the organization that the user belongs to
	var groupIDs []uint
	err := DBFromContext(ctx, s.db).Model(&models.UserGroup{}).
		Select("user_groups.group_id").
		Joins("JOIN groups ON groups.id = user_groups.group_id").
		Where("user_groups.user_id = ? AND groups.organization_id = ?", userID, orgID).
		Pluck("user_groups.group_id", &groupIDs).Error
	if err != nil {
		return err
	}

	if len(groupIDs) == 0 {
		return nil
	}

	// Remove user from all these groups
	return DBFromContext(ctx, s.db).Where("user_id = ? AND group_id IN ?", userID, groupIDs).
		Delete(&models.UserGroup{}).Error
}

// SetSuperAdmin sets or unsets superadmin status for a user
func (s *UserGroupStore) SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error {
	result := DBFromContext(ctx, s.db).Model(&models.User{}).
		Where("id = ?", userID).
		Update("is_superadmin", isSuperAdmin)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsSuperAdmin checks if a user is a superadmin
func (s *UserGroupStore) IsSuperAdmin(ctx context.Context, userID uint) (bool, error) {
	var user models.User
	err := DBFromContext(ctx, s.db).Select("is_superadmin").Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return user.IsSuperAdmin, nil
}

// Exists checks if a user-group relationship exists
func (s *UserGroupStore) Exists(ctx context.Context, userID, groupID uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.UserGroup{}).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Count(&count).Error
	return count > 0, err
}

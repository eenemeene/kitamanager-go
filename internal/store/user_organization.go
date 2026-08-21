package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserOrganizationStore handles database operations for user-organization relationships
type UserOrganizationStore struct {
	db *gorm.DB
}

// NewUserOrganizationStore creates a new UserOrganizationStore
func NewUserOrganizationStore(db *gorm.DB) *UserOrganizationStore {
	return &UserOrganizationStore{db: db}
}

// AddUserToOrg adds a user to an organization with a specified role
func (s *UserOrganizationStore) AddUserToOrg(ctx context.Context, userID, orgID uint, role models.Role, createdBy string) (*models.UserOrganization, error) {
	uo := &models.UserOrganization{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           role,
		CreatedBy:      createdBy,
	}

	if err := DBFromContext(ctx, s.db).Create(uo).Error; err != nil {
		return nil, err
	}

	return uo, nil
}

// UpdateRole updates a user's role in an organization
func (s *UserOrganizationStore) UpdateRole(ctx context.Context, userID, orgID uint, role models.Role) error {
	result := DBFromContext(ctx, s.db).Model(&models.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveUserFromOrg removes a user from an organization
func (s *UserOrganizationStore) RemoveUserFromOrg(ctx context.Context, userID, orgID uint) error {
	result := DBFromContext(ctx, s.db).Where("user_id = ? AND organization_id = ?", userID, orgID).
		Delete(&models.UserOrganization{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// FindByUserAndOrg finds a specific user-organization relationship
func (s *UserOrganizationStore) FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*models.UserOrganization, error) {
	var uo models.UserOrganization
	err := DBFromContext(ctx, s.db).Where("user_id = ? AND organization_id = ?", userID, orgID).First(&uo).Error
	if err != nil {
		return nil, WrapNotFound(err)
	}
	return &uo, nil
}

// FindByUser returns all organization memberships for a user
func (s *UserOrganizationStore) FindByUser(ctx context.Context, userID uint) ([]models.UserOrganization, error) {
	var memberships []models.UserOrganization
	err := DBFromContext(ctx, s.db).
		Preload("Organization").
		Where("user_id = ?", userID).
		Find(&memberships).Error
	return memberships, err
}

// GetRoleInOrg returns the role a user has in an organization
// Returns empty string if user has no role in the organization
func (s *UserOrganizationStore) GetRoleInOrg(ctx context.Context, userID, orgID uint) (models.Role, error) {
	var uo models.UserOrganization
	err := DBFromContext(ctx, s.db).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		First(&uo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return uo.Role, nil
}

// GetUserOrganizationsWithRoles returns all organizations a user belongs to with their roles
func (s *UserOrganizationStore) GetUserOrganizationsWithRoles(ctx context.Context, userID uint) (map[uint]models.Role, error) {
	memberships, err := s.FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	orgRoles := make(map[uint]models.Role)
	for _, m := range memberships {
		orgRoles[m.OrganizationID] = m.Role
	}

	return orgRoles, nil
}

// SetSuperAdmin sets or unsets superadmin status for a user
func (s *UserOrganizationStore) SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error {
	result := DBFromContext(ctx, s.db).Model(&models.User{}).
		Where("id = ?", userID).
		Update("is_superadmin", isSuperAdmin)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// OrganizationIsLive reports whether the organization exists and has not been
// tombstoned. Starting the query from the Organization model means GORM's
// soft-delete scope supplies the `deleted_at IS NULL` predicate.
//
// This lives on the membership store because every question this store answers
// — what role does this user hold here, are they a member — is meaningless once
// the organization is gone, and user_organizations rows outlive the tombstone.
// Keeping it here also leaves NewPermissionService's signature alone, which is
// what stops an org-liveness check from rippling into ~60 middleware test call
// sites that construct the authorization middleware directly.
func (s *UserOrganizationStore) OrganizationIsLive(ctx context.Context, orgID uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.Organization{}).
		Where("id = ?", orgID).
		Count(&count).Error
	return count > 0, err
}

// CountUsableSuperAdminsExcluding returns how many superadmins would still be
// able to sign in if `excludeUserID` were removed.
//
// Three things make a superadmin unusable and all three are filtered here:
// tombstoned (GORM's soft-delete scope on models.User), deactivated
// (`active = false`, which RequireAuth rejects), and the user the caller is
// about to remove.
//
// The exclusion is what makes this answer the right question. The previous
// CountSuperAdmins asked "how many are there?" and callers compared against 1 —
// which over-counts when the peer keeping the total above 1 is a superadmin who
// can no longer log in, and under-counts nothing in return. Asking "if I take
// this user out, is anyone left?" is the invariant the guards actually want, and
// it stays correct whether or not the excluded user was usable to begin with.
func (s *UserOrganizationStore) CountUsableSuperAdminsExcluding(ctx context.Context, excludeUserID uint) (int64, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.User{}).
		Where("is_superadmin = ?", true).
		Where("active = ?", true).
		Where("id <> ?", excludeUserID).
		Count(&count).Error
	return count, err
}

// IsSuperAdmin checks if a user is a superadmin
func (s *UserOrganizationStore) IsSuperAdmin(ctx context.Context, userID uint) (bool, error) {
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

// Exists checks if a user-organization relationship exists
func (s *UserOrganizationStore) Exists(ctx context.Context, userID, orgID uint) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Count(&count).Error
	return count > 0, err
}

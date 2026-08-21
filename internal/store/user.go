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

// FindByEmail is case-insensitive. The functional unique index on
// lower(email) (migration 000009) makes this lookup O(index seek).
func (s *UserStore) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := DBFromContext(ctx, s.db).Where("lower(email) = lower(?)", email).First(&user).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &user, nil
}

// EmailExistsForOtherUser is case-insensitive for the same reason as
// FindByEmail — any caller whose input hasn't been normalized still gets
// the right answer.
func (s *UserStore) EmailExistsForOtherUser(ctx context.Context, email string, excludeUserID uint) (bool, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.User{}).Where("lower(email) = lower(?) AND id != ?", email, excludeUserID).Count(&count).Error; err != nil {
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

// Delete soft-deletes the user via GORM's DeletedAt machinery —
// `UPDATE users SET deleted_at = now() WHERE id = ?`. Subsequent
// GORM queries that start from the User model auto-scope the row
// out. Use HardDelete for the physical DELETE path.
func (s *UserStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.User{}, id).Error
}

// HardDelete issues the physical DELETE, bypassing the soft-delete
// tombstone. Cascades through sessions, factors, user_organizations
// via the FKs defined in 000001 + 000014. Irreversible.
func (s *UserStore) HardDelete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Unscoped().Delete(&models.User{}, id).Error
}

// FindByIDUnscoped fetches the user row whether it is tombstoned
// (soft-deleted) or live. Used by admin trash-view and by HardDelete
// so a tombstoned user can still be purged (the retention job
// targets exactly that state).
func (s *UserStore) FindByIDUnscoped(ctx context.Context, id uint, out *models.User) error {
	return DBFromContext(ctx, s.db).Unscoped().First(out, id).Error
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

// SharesOrganization returns true iff both users have an active
// membership in the same organization. Closes audit finding R-M-1
// (security review 2026-05-01): the previous version raw-joined
// user_organizations to itself without filtering tombstoned users on
// either side, so a tombstoned user could still appear as "still
// sharing" with a live user. Today's callers gate this with a prior
// FindByID (auto-scoped) on at least one side, but a future direct
// caller would have inherited the bug. Fixed defensively.
//
// Both userID1 and userID2 must be non-tombstoned (users.deleted_at
// IS NULL on each end of the JOIN). user_organizations rows
// CASCADE-delete with the user, so a hard-deleted user is naturally
// excluded; soft-deleted users keep their user_organizations rows
// hence the explicit filter.
//
// The shared organization must be live too. A tombstoned organization
// keeps its user_organizations rows for the same reason, so without the
// third filter two users went on "sharing" an organization that had been
// deleted — and that shared membership is what grants visibility over
// each other.
func (s *UserStore) SharesOrganization(ctx context.Context, userID1, userID2 uint) (bool, error) {
	var count int64
	err := ExcludeSoftDeletedOrganizations(
		DBFromContext(ctx, s.db).Table("user_organizations uo1").
			Joins("JOIN user_organizations uo2 ON uo2.organization_id = uo1.organization_id").
			Joins("JOIN users u1 ON u1.id = uo1.user_id AND u1.deleted_at IS NULL").
			Joins("JOIN users u2 ON u2.id = uo2.user_id AND u2.deleted_at IS NULL").
			Joins("JOIN organizations ON organizations.id = uo1.organization_id"),
	).
		Where("uo1.user_id = ? AND uo2.user_id = ?", userID1, userID2).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsAdminInSharedOrg checks whether the requester has admin role in at
// least one LIVE organization that the target user belongs to. Closes
// audit finding R-M-1 (same rationale as SharesOrganization): both users
// must be non-tombstoned, and so must the organization that connects
// them — otherwise deleting an organization leaves its admins holding
// admin power over its former members indefinitely.
func (s *UserStore) IsAdminInSharedOrg(ctx context.Context, requesterID, targetUserID uint) (bool, error) {
	var count int64
	err := ExcludeSoftDeletedOrganizations(
		DBFromContext(ctx, s.db).Table("user_organizations uo_req").
			Joins("JOIN user_organizations uo_target ON uo_target.organization_id = uo_req.organization_id").
			Joins("JOIN users u_req ON u_req.id = uo_req.user_id AND u_req.deleted_at IS NULL").
			Joins("JOIN users u_target ON u_target.id = uo_target.user_id AND u_target.deleted_at IS NULL").
			Joins("JOIN organizations ON organizations.id = uo_req.organization_id"),
	).
		Where("uo_req.user_id = ? AND uo_target.user_id = ? AND uo_req.role = ?",
			requesterID, targetUserID, "admin").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

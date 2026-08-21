package service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// Guards protecting superadmin accounts from org admins, and protecting the
// installation from losing its last usable superadmin.
//
// Both rules already existed in one place each — the superadmin-target check
// only on ResetPassword, the last-superadmin check only on the paths that
// remove the row — so these tests exist to pin the rules to every door rather
// than to the door they were first written on.

// deactivate builds the update request for flipping `active` off, which is the
// operation all the lockout cases turn on.
func deactivate() *models.UserUpdateRequest {
	inactive := false
	return &models.UserUpdateRequest{Active: &inactive}
}

// setUserActive forces the active flag directly, for arranging a peer
// superadmin who exists but cannot sign in.
func setUserActive(t *testing.T, db *gorm.DB, userID uint, active bool) {
	t.Helper()
	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("active", active).Error; err != nil {
		t.Fatalf("failed to set active=%v on user %d: %v", active, userID, err)
	}
}

// orgAdminWith returns an org admin who shares `org` with the given target.
func orgAdminWith(t *testing.T, db *gorm.DB, org *models.Organization, target *models.User) *models.User {
	t.Helper()
	admin := createTestUser(t, db, "Org Admin", "orgadmin@example.com", "password")
	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)
	return admin
}

func TestUserService_Update_OrgAdminCannotDeactivateSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Kita Sonnenschein")
	superAdmin := createTestSuperAdmin(t, db)
	// A second superadmin, so the refusal is attributable to the
	// superadmin-target guard and not to the last-superadmin guard.
	createTestSuperAdmin2(t, db)
	admin := orgAdminWith(t, db, org, superAdmin)

	_, err := svc.Update(ctx, superAdmin.ID, deactivate(), admin.ID)
	if err == nil {
		t.Fatal("expected org admin to be refused when deactivating a superadmin")
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	var after models.User
	if err := db.First(&after, superAdmin.ID).Error; err != nil {
		t.Fatalf("reload superadmin: %v", err)
	}
	if !after.Active {
		t.Error("superadmin must still be active after the refused update")
	}
}

func TestUserService_Delete_OrgAdminCannotDeleteSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Kita Sonnenschein")
	superAdmin := createTestSuperAdmin(t, db)
	createTestSuperAdmin2(t, db)
	admin := orgAdminWith(t, db, org, superAdmin)

	err := svc.Delete(ctx, superAdmin.ID, admin.ID)
	if err == nil {
		t.Fatal("expected org admin to be refused when deleting a superadmin")
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	if _, err := svc.GetByID(ctx, superAdmin.ID, superAdmin.ID); err != nil {
		t.Errorf("superadmin must still exist after the refused delete: %v", err)
	}
}

func TestUserService_Update_OrgAdminCanStillUpdateOrdinaryUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Kita Sonnenschein")
	target := createTestUser(t, db, "Erzieherin", "erzieherin@example.com", "password")
	admin := orgAdminWith(t, db, org, target)

	resp, err := svc.Update(ctx, target.ID, deactivate(), admin.ID)
	if err != nil {
		t.Fatalf("org admin must still be able to deactivate an ordinary member: %v", err)
	}
	if resp.Active {
		t.Error("expected the ordinary user to be deactivated")
	}
}

func TestUserService_Update_SuperAdminCanDeactivateAnotherSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)

	resp, err := svc.Update(ctx, superAdmin1.ID, deactivate(), superAdmin2.ID)
	if err != nil {
		t.Fatalf("a superadmin must be able to deactivate a superadmin peer: %v", err)
	}
	if resp.Active {
		t.Error("expected superAdmin1 to be deactivated")
	}
}

func TestUserService_Update_SuperAdminCanEditOwnProfile(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin := createTestSuperAdmin(t, db)

	resp, err := svc.Update(ctx, superAdmin.ID, &models.UserUpdateRequest{Name: "Renamed"}, superAdmin.ID)
	if err != nil {
		t.Fatalf("a superadmin must be able to edit their own profile: %v", err)
	}
	if resp.Name != "Renamed" {
		t.Errorf("Name = %q, want %q", resp.Name, "Renamed")
	}
}

func TestUserService_Update_CannotDeactivateLastSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin := createTestSuperAdmin(t, db)

	// Self-deactivation: the superadmin-target guard passes (requester ==
	// target), so this reaches the lockout guard, which is the point.
	_, err := svc.Update(ctx, superAdmin.ID, deactivate(), superAdmin.ID)
	if err == nil {
		t.Fatal("expected deactivating the last superadmin to be refused")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	var after models.User
	if err := db.First(&after, superAdmin.ID).Error; err != nil {
		t.Fatalf("reload superadmin: %v", err)
	}
	if !after.Active {
		t.Error("the last superadmin must still be active after the refused update")
	}
}

// The peer that keeps the head-count above one is already deactivated, so it
// cannot sign in. Counting rows rather than usable accounts let this through
// and produced exactly the lockout the guard exists to prevent.
func TestUserService_Update_CannotDeactivateLastUsableSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)
	setUserActive(t, db, superAdmin2.ID, false)

	_, err := svc.Update(ctx, superAdmin1.ID, deactivate(), superAdmin1.ID)
	if err == nil {
		t.Fatal("expected refusal: the only other superadmin is already deactivated")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUserService_Delete_CannotDeleteLastUsableSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)
	setUserActive(t, db, superAdmin2.ID, false)

	// superAdmin2 does the deleting so the self-delete guard is not what
	// produces the error; superAdmin1 is the last one who can still log in.
	err := svc.Delete(ctx, superAdmin1.ID, superAdmin2.ID)
	if err == nil {
		t.Fatal("expected refusal: deleting the last superadmin who can still sign in")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// Mirror case: removing a superadmin who is *already* unusable changes nothing
// about the installation's recoverability, so it must stay permitted.
func TestUserService_Delete_CanDeleteInactiveSuperAdminWhileActiveOneRemains(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)
	setUserActive(t, db, superAdmin2.ID, false)

	if err := svc.Delete(ctx, superAdmin2.ID, superAdmin1.ID); err != nil {
		t.Fatalf("deleting an already-deactivated superadmin must stay allowed: %v", err)
	}
}

// Admin power is derived from a shared organization. When that organization is
// tombstoned the derivation must stop, or deleting an org leaves its admins
// with permanent authority over everyone who was in it.
func TestUserService_Update_AdminPowerDoesNotSurviveOrgTombstone(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Kita Sonnenschein")
	target := createTestUser(t, db, "Erzieherin", "erzieherin@example.com", "password")
	admin := orgAdminWith(t, db, org, target)

	// Sanity: the power exists while the organization is live.
	if _, err := svc.Update(ctx, target.ID, &models.UserUpdateRequest{Name: "Before"}, admin.ID); err != nil {
		t.Fatalf("setup: admin should be able to update the member: %v", err)
	}

	if err := db.Delete(&models.Organization{}, org.ID).Error; err != nil {
		t.Fatalf("failed to tombstone organization: %v", err)
	}

	_, err := svc.Update(ctx, target.ID, &models.UserUpdateRequest{Name: "After"}, admin.ID)
	if err == nil {
		t.Fatal("expected the admin to lose authority once the shared org is tombstoned")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

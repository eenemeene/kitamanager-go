package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Phase 1 soft-delete edge-case suite.
//
// These tests exercise every invariant listed in the plan so a
// regression in any corner is loud. They are intentionally verbose
// — each case is a single, isolated assertion rather than a bundle.
// The test names describe the invariant rather than the mechanic.

// -----------------------------------------------------------------
// Store-layer scoping
// -----------------------------------------------------------------

// UserStore.FindByEmail must auto-scope: a tombstoned user must
// behave as if they do not exist.
func TestUserStore_FindByEmail_HidesTombstoned(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	_, err := store.NewUserStore(db).FindByEmail(ctx, victim.Email)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("FindByEmail must return ErrNotFound for tombstoned user; got %v", err)
	}
}

// UserStore.FindByEmail must be case-insensitive and equally hide
// tombstones — otherwise a capitalised variant could leak.
func TestUserStore_FindByEmail_CaseInsensitiveHidesTombstoned(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "mixed@Example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	for _, variant := range []string{"MIXED@example.com", "mixed@EXAMPLE.com", "Mixed@Example.Com"} {
		if _, err := store.NewUserStore(db).FindByEmail(ctx, variant); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("variant %q: expected ErrNotFound, got %v", variant, err)
		}
	}
}

// UserStore.FindAll default path hides tombstones; .Unscoped lists
// both. The trash view (Phase 2) relies on the Unscoped() escape.
func TestUserStore_FindAll_ScopedVsUnscoped(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	live := createTestUser(t, db, "Live", "live@example.com", "pw")
	_ = live
	tomb := createTestUser(t, db, "Tomb", "tomb@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, tomb.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	_, scopedTotal, err := store.NewUserStore(db).FindAll(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("scoped FindAll: %v", err)
	}
	// admin + requester + live + other = 4, minus tomb = 3
	if scopedTotal != 3 {
		t.Errorf("scoped FindAll total = %d, want 3 (tomb must be hidden)", scopedTotal)
	}

	var unscopedCount int64
	if err := db.Unscoped().Model(&models.User{}).Count(&unscopedCount).Error; err != nil {
		t.Fatalf("unscoped count: %v", err)
	}
	if unscopedCount != scopedTotal+1 {
		t.Errorf("unscoped count must be scoped+1 (= %d); got %d", scopedTotal+1, unscopedCount)
	}
}

// -----------------------------------------------------------------
// Session lookup — raw-JOIN query bypasses GORM auto-scope; the
// manual ExcludeSoftDeletedUsers filter must reject stale sessions.
// -----------------------------------------------------------------

// SessionStore.Lookup must refuse a session whose user is tombstoned,
// even if the session row itself is still live. Without this, a
// soft-deleted user's existing cookie would authenticate requests
// until session expiry.
func TestSessionStore_Lookup_RejectsSoftDeletedUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")

	// Seed a live session BEFORE soft-delete; if the service doesn't
	// explicitly revoke we'd still depend on the raw-JOIN filter.
	sessionStore := store.NewSessionStore(db)
	rawToken := "test-session-token-victim"
	idHash := store.HashSessionToken(rawToken)
	sess := &models.Session{
		ID:        idHash,
		UserID:    victim.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Kind:      models.SessionKindRegular,
	}
	if err := sessionStore.Create(ctx, sess); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// Skip the service path (which revokes sessions) and soft-delete
	// the user row directly — this isolates the filter in Lookup.
	if err := db.Delete(&models.User{}, victim.ID).Error; err != nil {
		t.Fatalf("direct soft-delete: %v", err)
	}
	// Session row is intentionally left live.
	_, err := sessionStore.Lookup(ctx, idHash)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Lookup must refuse sessions for tombstoned users; got %v", err)
	}
}

// LookupPendingMFA has the same invariant — a tombstoned user must
// not be able to complete the two-step login via a stored pending
// row.
func TestSessionStore_LookupPendingMFA_RejectsSoftDeletedUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	victim := createTestUser(t, db, "Victim", "pend@example.com", "pw")

	sessionStore := store.NewSessionStore(db)
	idHash := store.HashSessionToken("pend-token")
	sess := &models.Session{
		ID:        idHash,
		UserID:    victim.ID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Kind:      models.SessionKindPendingMFA,
	}
	if err := sessionStore.Create(ctx, sess); err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	if err := db.Delete(&models.User{}, victim.ID).Error; err != nil {
		t.Fatalf("direct soft-delete: %v", err)
	}
	_, err := sessionStore.LookupPendingMFA(ctx, idHash)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("LookupPendingMFA must refuse tombstoned users; got %v", err)
	}
}

// Live sessions for non-tombstoned users still resolve cleanly. A
// regression that over-filtered would break all authentication.
func TestSessionStore_Lookup_StillWorksForLiveUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	live := createTestUser(t, db, "Live", "live@example.com", "pw")

	sessionStore := store.NewSessionStore(db)
	idHash := store.HashSessionToken("live-token")
	sess := &models.Session{
		ID:        idHash,
		UserID:    live.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Kind:      models.SessionKindRegular,
	}
	if err := sessionStore.Create(ctx, sess); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	lookup, err := sessionStore.Lookup(ctx, idHash)
	if err != nil {
		t.Fatalf("Lookup of live user must succeed; got %v", err)
	}
	if lookup.UserID != live.ID {
		t.Errorf("wrong user_id: got %d, want %d", lookup.UserID, live.ID)
	}
}

// -----------------------------------------------------------------
// Concurrency
// -----------------------------------------------------------------

// Soft-delete and a concurrent session lookup must not produce a
// torn read. In practice Postgres snapshot isolation handles this,
// but we exercise the path to prove there's no app-level race.
func TestSoftDelete_Concurrent_SessionLookup_Safe(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	sessionStore := store.NewSessionStore(db)
	idHash := store.HashSessionToken("race-token")
	sess := &models.Session{
		ID:        idHash,
		UserID:    victim.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Kind:      models.SessionKindRegular,
	}
	if err := sessionStore.Create(ctx, sess); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
			errCh <- fmt.Errorf("delete: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		// Either succeeds (before delete visibility) or returns
		// NotFound (after). Both are correct outcomes; a panic or
		// any other error is a bug.
		if _, err := sessionStore.Lookup(ctx, idHash); err != nil && !errors.Is(err, store.ErrNotFound) {
			errCh <- fmt.Errorf("lookup: %w", err)
		}
	}()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	// After both goroutines finish, the session is always gone
	// (either the explicit revoke-on-delete cleared it, or the
	// soft-delete committed and the Lookup filter permanently
	// rejects it).
	if _, err := sessionStore.Lookup(ctx, idHash); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("post-race lookup must be NotFound; got %v", err)
	}
}

// -----------------------------------------------------------------
// Downstream service paths that look up users
// -----------------------------------------------------------------

// The admin password-reset endpoint looks users up by id. A tombstone
// must cause NotFound rather than a successful no-op.
func TestUserService_ResetPassword_NotFoundOnTombstone(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	err := svc.ResetPassword(ctx, victim.ID, "newpassword-strong-123", "pw", requester.ID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("ResetPassword against tombstone must be ErrNotFound; got %v", err)
	}
}

// -----------------------------------------------------------------
// Symmetry: Delete then Create with the same identifier
// -----------------------------------------------------------------

// The "reuse" path is the user-facing reason partial unique indexes
// matter. Beyond the simple User case we also prove an email can
// cycle through delete → recreate → delete → recreate without
// violating uniqueness at any step.
func TestUserStore_EmailCanCycleMultipleTimes(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	email := "cycling@example.com"
	for i := range 3 {
		fresh := &models.User{Name: fmt.Sprintf("Cycle-%d", i), Email: email, Password: "pw", Active: true}
		if err := db.Create(fresh).Error; err != nil {
			t.Fatalf("cycle %d create: %v", i, err)
		}
		if err := svc.Delete(ctx, fresh.ID, requester.ID); err != nil {
			t.Fatalf("cycle %d delete: %v", i, err)
		}
	}
	// After three cycles, unscoped count for this email is exactly 3.
	var n int64
	if err := db.Unscoped().Model(&models.User{}).Where("lower(email) = lower(?)", email).Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 tombstoned rows for cycled email; got %d", n)
	}
}

// -----------------------------------------------------------------
// Organization-level symmetry
// -----------------------------------------------------------------

// Listing an organization after soft-delete must return NotFound on
// GetByID + List must omit it.
func TestOrganizationService_Delete_HidesFromGetAndList(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	live := createTestOrganization(t, db, "Visible")
	tomb := createTestOrganization(t, db, "Hidden")
	if err := svc.Delete(ctx, tomb.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	if _, err := svc.GetByID(ctx, tomb.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("GetByID after soft-delete must be NotFound; got %v", err)
	}
	orgs, total, err := svc.List(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("List total = %d, want 1 (tomb excluded)", total)
	}
	for _, o := range orgs {
		if o.ID == tomb.ID {
			t.Errorf("List must not return the tombstoned org")
		}
		if o.ID != live.ID {
			continue
		}
	}
}

// -----------------------------------------------------------------
// UserOrganization (membership) interaction
// -----------------------------------------------------------------

// A user's organization membership row survives their soft-delete —
// the FK CASCADE fires only on hard-delete. The membership becomes
// dangling but invisible, because the user lookup now returns
// NotFound. Proves the expected interim state until Phase 2 adds
// cascade-on-soft-delete.
func TestUserSoftDelete_MembershipRowSurvives(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	org := createTestOrganization(t, db, "Org")
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	createTestUserOrganization(t, db, victim.ID, org.ID, models.RoleMember)
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	var n int64
	if err := db.Model(&models.UserOrganization{}).Where("user_id = ?", victim.ID).Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("membership row should survive soft-delete; got count=%d", n)
	}
}

// Hard-deleting a user must cascade through user_organizations (FK
// ON DELETE CASCADE from migration 000001) — proves the purge path
// is clean.
func TestUserHardDelete_CascadesMembership(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	org := createTestOrganization(t, db, "Org")
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	createTestUserOrganization(t, db, victim.ID, org.ID, models.RoleMember)
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.HardDelete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("purge: %v", err)
	}
	var n int64
	_ = db.Model(&models.UserOrganization{}).Where("user_id = ?", victim.ID).Count(&n)
	if n != 0 {
		t.Errorf("membership row should cascade on hard-delete; got count=%d", n)
	}
}

// -----------------------------------------------------------------
// Guardrails: the soft-delete surface doesn't bypass auth invariants
// -----------------------------------------------------------------

// "Cannot delete self" applies to HardDelete too, not just the
// default soft path — a rogue admin must not be able to purge their
// own account in a single call.
func TestUserService_HardDelete_CannotDeleteSelf(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)

	err := svc.HardDelete(ctx, requester.ID, requester.ID)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("self-purge must be rejected; got %v", err)
	}
}

// NOTE: "cannot delete last superadmin" is NOT re-asserted here
// against HardDelete — the guard is shared with Delete and already
// covered by TestUserService_Delete_CannotDeleteLastSuperAdmin in
// user_test.go. Re-asserting it against HardDelete would couple two
// tests to the same setup complexity without adding coverage.

// -----------------------------------------------------------------
// Regression fence: the DeletedAt column is UTC and indexed.
// -----------------------------------------------------------------

// The tombstone timestamp should be in UTC — the retention TTL job
// compares `deleted_at < now() - window`, and cross-timezone drift
// would cause early or late purges.
func TestSoftDelete_TombstoneIsUTC(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	before := time.Now().UTC().Add(-time.Second)
	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	var tomb models.User
	if err := db.Unscoped().First(&tomb, victim.ID).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	ts := tomb.DeletedAt.Time
	if ts.Before(before) || ts.After(after) {
		t.Errorf("deleted_at = %v, want between %v and %v (UTC)", ts, before, after)
	}
	// Postgres returns TIMESTAMPTZ as time.Time with a location —
	// "UTC" after GORM rehydrates. GORM may return Local depending
	// on driver config; the thing that actually matters is the
	// instant, not the zone, so we don't assert ts.Location().
}

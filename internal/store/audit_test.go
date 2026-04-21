package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func uintPtr(v uint) *uint { return &v }

func createTestAuditLog(t *testing.T, s *AuditStore, userID uint, action models.AuditAction) *models.AuditLog {
	t.Helper()
	log := &models.AuditLog{
		UserID:       uintPtr(userID),
		UserEmail:    "test@example.com",
		Action:       action,
		ResourceType: "user",
		Details:      "test details",
		IPAddress:    "127.0.0.1",
		Timestamp:    time.Now(),
	}
	if err := s.Create(context.Background(), log); err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}
	return log
}

func TestAuditStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	log := &models.AuditLog{
		UserID:       uintPtr(1),
		UserEmail:    "admin@example.com",
		Action:       models.AuditActionLogin,
		ResourceType: "auth",
		Details:      "successful login",
		IPAddress:    "192.168.1.1",
	}

	err := store.Create(context.Background(), log)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if log.ID == 0 {
		t.Error("expected log ID to be set")
	}
	if log.Timestamp.IsZero() {
		t.Error("expected timestamp to be auto-set")
	}
}

func TestAuditStore_FindByUser(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 1, models.AuditActionUserCreate)
	createTestAuditLog(t, store, 2, models.AuditActionLogin)

	logs, total, err := store.FindByUser(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindByUser_Pagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	for range 5 {
		createTestAuditLog(t, store, 1, models.AuditActionLogin)
	}

	logs, total, err := store.FindByUser(context.Background(), 1, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs (limit), got %d", len(logs))
	}

	logs2, _, err := store.FindByUser(context.Background(), 1, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs2) != 2 {
		t.Errorf("expected 2 logs (offset), got %d", len(logs2))
	}
}

func TestAuditStore_FindByAction(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 1, models.AuditActionUserCreate)
	createTestAuditLog(t, store, 2, models.AuditActionUserCreate)

	logs, total, err := store.FindByAction(context.Background(), models.AuditActionUserCreate, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindByDateRange(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()

	old := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-48 * time.Hour),
	}
	_ = store.Create(context.Background(), old)

	recent := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-1 * time.Hour),
	}
	_ = store.Create(context.Background(), recent)

	from := now.Add(-24 * time.Hour)
	to := now

	logs, total, err := store.FindByDateRange(context.Background(), from, to, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAuditStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 2, models.AuditActionUserCreate)
	createTestAuditLog(t, store, 3, models.AuditActionUserDelete)

	logs, total, err := store.FindAll(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindFailedLogins(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()

	failed := &models.AuditLog{
		UserEmail: "hacker@example.com", Action: models.AuditActionLoginFailed,
		ResourceType: "auth", Timestamp: now.Add(-5 * time.Minute),
	}
	_ = store.Create(context.Background(), failed)

	success := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "user@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-3 * time.Minute),
	}
	_ = store.Create(context.Background(), success)

	// Find all failed logins
	logs, err := store.FindFailedLogins(context.Background(), "", now.Add(-10*time.Minute), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 failed login, got %d", len(logs))
	}

	// Filter by email
	logs, err = store.FindFailedLogins(context.Background(), "hacker@example.com", now.Add(-10*time.Minute), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 failed login for email, got %d", len(logs))
	}

	// Filter by different email
	logs, err = store.FindFailedLogins(context.Background(), "other@example.com", now.Add(-10*time.Minute), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 failed logins, got %d", len(logs))
	}
}

func TestAuditStore_CountFailedLoginsSince(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()
	email := "attacker@example.com"

	for i := range 3 {
		log := &models.AuditLog{
			UserEmail: email, Action: models.AuditActionLoginFailed,
			ResourceType: "auth", Timestamp: now.Add(-time.Duration(i) * time.Minute),
		}
		_ = store.Create(context.Background(), log)
	}

	count, err := store.CountFailedLoginsSince(context.Background(), email, now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	count, err = store.CountFailedLoginsSince(context.Background(), "other@example.com", now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestAuditStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	log := createTestAuditLog(t, store, 1, models.AuditActionLogin)

	found, err := store.FindByID(context.Background(), log.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != log.ID {
		t.Errorf("expected ID %d, got %d", log.ID, found.ID)
	}
	if found.Action != models.AuditActionLogin {
		t.Errorf("expected action %s, got %s", models.AuditActionLogin, found.Action)
	}
}

func TestAuditStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	_, err := store.FindByID(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAuditStore_FindAllFiltered_NoFilters(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 2, models.AuditActionUserCreate)
	createTestAuditLog(t, store, 3, models.AuditActionEmployeeDelete)

	logs, total, err := store.FindAllFiltered(context.Background(), "", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_ByAction(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 2, models.AuditActionLogin)
	createTestAuditLog(t, store, 3, models.AuditActionUserCreate)

	logs, total, err := store.FindAllFiltered(context.Background(), string(models.AuditActionLogin), nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_ByUserID(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)
	createTestAuditLog(t, store, 1, models.AuditActionUserCreate)
	createTestAuditLog(t, store, 2, models.AuditActionLogin)

	userID := uint(1)
	logs, total, err := store.FindAllFiltered(context.Background(), "", &userID, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_ByDateRange(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()

	old := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-72 * time.Hour),
	}
	_ = store.Create(context.Background(), old)

	recent := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionUserCreate,
		ResourceType: "user", Timestamp: now.Add(-1 * time.Hour),
	}
	_ = store.Create(context.Background(), recent)

	from := now.Add(-24 * time.Hour)
	to := now
	logs, total, err := store.FindAllFiltered(context.Background(), "", nil, &from, &to, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()

	// Match: user 1, login, recent
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "u1@example.com", Action: models.AuditActionLogin,
		Timestamp: now.Add(-1 * time.Hour),
	})
	// No match: user 1, login, old
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "u1@example.com", Action: models.AuditActionLogin,
		Timestamp: now.Add(-72 * time.Hour),
	})
	// No match: user 2, login, recent
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(2), UserEmail: "u2@example.com", Action: models.AuditActionLogin,
		Timestamp: now.Add(-1 * time.Hour),
	})
	// No match: user 1, different action, recent
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "u1@example.com", Action: models.AuditActionUserCreate,
		Timestamp: now.Add(-1 * time.Hour),
	})

	userID := uint(1)
	from := now.Add(-24 * time.Hour)
	to := now
	logs, total, err := store.FindAllFiltered(context.Background(), string(models.AuditActionLogin), &userID, &from, &to, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_Pagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	for range 5 {
		createTestAuditLog(t, store, 1, models.AuditActionLogin)
	}

	logs, total, err := store.FindAllFiltered(context.Background(), "", nil, nil, nil, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs (limit), got %d", len(logs))
	}

	logs2, total2, err := store.FindAllFiltered(context.Background(), "", nil, nil, nil, 2, 4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total2 != 5 {
		t.Errorf("expected total 5, got %d", total2)
	}
	if len(logs2) != 1 {
		t.Errorf("expected 1 log (last page), got %d", len(logs2))
	}
}

func TestAuditStore_FindAllFiltered_OrderedByTimestampDesc(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		Timestamp: now.Add(-3 * time.Hour),
	})
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionUserCreate,
		Timestamp: now.Add(-1 * time.Hour),
	})
	_ = store.Create(context.Background(), &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionEmployeeDelete,
		Timestamp: now.Add(-2 * time.Hour),
	})

	logs, _, err := store.FindAllFiltered(context.Background(), "", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
	// Most recent first
	if logs[0].Action != models.AuditActionUserCreate {
		t.Errorf("expected first log action %s, got %s", models.AuditActionUserCreate, logs[0].Action)
	}
	if logs[2].Action != models.AuditActionLogin {
		t.Errorf("expected last log action %s, got %s", models.AuditActionLogin, logs[2].Action)
	}
}

func TestAuditStore_FindAllFiltered_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	logs, total, err := store.FindAllFiltered(context.Background(), string(models.AuditActionLogin), nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindAllFiltered_NonExistentAction(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	createTestAuditLog(t, store, 1, models.AuditActionLogin)

	logs, total, err := store.FindAllFiltered(context.Background(), "nonexistent_action", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestAuditStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)
	orgA := createTestOrganization(t, db, "Kita A")
	orgB := createTestOrganization(t, db, "Kita B")

	// Three distinct rows: orgA, orgB, identity-level (nil).
	orgAID := orgA.ID
	orgBID := orgB.ID
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: models.AuditActionChildDelete,
		ResourceType: "child", OrganizationID: &orgAID, IPAddress: "127.0.0.1",
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(2), UserEmail: "b@example.com", Action: models.AuditActionEmployeeDelete,
		ResourceType: "employee", OrganizationID: &orgBID, IPAddress: "127.0.0.1",
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(3), UserEmail: "c@example.com", Action: models.AuditActionLogin,
		IPAddress: "127.0.0.1",
	})

	// orgA returns exactly its one row; nil-org login stays hidden.
	logs, total, err := store.FindByOrganization(context.Background(), orgA.ID, "", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("orgA: expected 1 row, got total=%d len=%d", total, len(logs))
	}
	if logs[0].Action != models.AuditActionChildDelete {
		t.Errorf("orgA row action = %v, want %v", logs[0].Action, models.AuditActionChildDelete)
	}

	// orgB only sees its own row.
	logsB, totalB, err := store.FindByOrganization(context.Background(), orgB.ID, "", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalB != 1 || logsB[0].Action != models.AuditActionEmployeeDelete {
		t.Errorf("orgB: expected single employee_delete, got total=%d first=%v", totalB, logsB[0].Action)
	}
}

// TestAuditStore_ActionFilter_Substring verifies the action filter applied
// by findFiltered / FindByOrganization / FindAllFiltered is a case-insensitive
// substring match rather than an exact match. A user filtering on "ild" now
// matches every child_* action — the UX fix asked for by operators who
// don't remember the exact action string.
func TestAuditStore_ActionFilter_Substring(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)
	org := createTestOrganization(t, db, "Kita Substring")
	orgID := org.ID
	ip := "127.0.0.1"

	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: models.AuditActionChildDelete,
		ResourceType: "child", OrganizationID: &orgID, IPAddress: ip,
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: "child_create",
		ResourceType: "child", OrganizationID: &orgID, IPAddress: ip,
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: "child_update",
		ResourceType: "child", OrganizationID: &orgID, IPAddress: ip,
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: models.AuditActionEmployeeDelete,
		ResourceType: "employee", OrganizationID: &orgID, IPAddress: ip,
	})
	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: models.AuditActionLogin,
		IPAddress: ip, // no org — identity-level
	})

	tests := []struct {
		name    string
		filter  string
		wantN   int64
		wantAll func(a models.AuditAction) bool // predicate every row must satisfy
	}{
		{
			name:    "empty filter returns all org rows",
			filter:  "ild",
			wantN:   3, // child_delete, child_create, child_update
			wantAll: func(a models.AuditAction) bool { return strings.Contains(string(a), "child") },
		},
		{
			name:    "exact action still works",
			filter:  "child_delete",
			wantN:   1,
			wantAll: func(a models.AuditAction) bool { return a == models.AuditActionChildDelete },
		},
		{
			name:    "case-insensitive — uppercase filter matches lowercase rows",
			filter:  "CHILD",
			wantN:   3,
			wantAll: func(a models.AuditAction) bool { return strings.Contains(string(a), "child") },
		},
		{
			name:    "shared suffix picks up across resource types",
			filter:  "_delete",
			wantN:   2, // child_delete + employee_delete
			wantAll: func(a models.AuditAction) bool { return strings.HasSuffix(string(a), "_delete") },
		},
		{
			name:    "no match for unrelated substring",
			filter:  "xyzzy",
			wantN:   0,
			wantAll: func(models.AuditAction) bool { return true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, total, err := store.FindByOrganization(context.Background(), org.ID, tt.filter, nil, nil, nil, 50, 0)
			if err != nil {
				t.Fatalf("FindByOrganization: %v", err)
			}
			if total != tt.wantN {
				t.Fatalf("total = %d, want %d (rows: %v)", total, tt.wantN, actions(logs))
			}
			if int64(len(logs)) != tt.wantN {
				t.Fatalf("len(logs) = %d, want %d", len(logs), tt.wantN)
			}
			for _, l := range logs {
				if !tt.wantAll(l.Action) {
					t.Errorf("row %v does not satisfy predicate", l.Action)
				}
			}
		})
	}
}

// TestAuditStore_ActionFilter_LikeMetacharactersEscaped guards against a user
// typing % or _ in the filter and accidentally getting a wider match than
// they meant. Without escaping, "%" would match every row; "child_" would
// match child1, child2, etc. if such rows existed.
func TestAuditStore_ActionFilter_LikeMetacharactersEscaped(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)
	org := createTestOrganization(t, db, "Kita Escape")
	orgID := org.ID
	ip := "127.0.0.1"

	mustCreate(t, store, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "a@example.com", Action: "child_delete",
		ResourceType: "child", OrganizationID: &orgID, IPAddress: ip,
	})

	// "%" should not expand into a wildcard — no row has a literal "%" in
	// its action, so the filter must return zero.
	_, total, err := store.FindByOrganization(context.Background(), org.ID, "%", nil, nil, nil, 50, 0)
	if err != nil {
		t.Fatalf("FindByOrganization with %%: %v", err)
	}
	if total != 0 {
		t.Errorf("filter %% returned %d rows; expected 0 (metachar not escaped)", total)
	}

	// "_" similarly must be treated literally. The stored action "child_delete"
	// does contain a literal underscore, so a filter of "_" is still a valid
	// substring and returns the row. What we're guarding against is that a
	// filter like "x_y" wouldn't match "xAy" via the LIKE wildcard for _.
	_, total, err = store.FindByOrganization(context.Background(), org.ID, "a_b", nil, nil, nil, 50, 0)
	if err != nil {
		t.Fatalf("FindByOrganization with a_b: %v", err)
	}
	if total != 0 {
		t.Errorf("filter a_b returned %d rows; expected 0 (underscore not escaped)", total)
	}
}

func actions(logs []models.AuditLog) []models.AuditAction {
	out := make([]models.AuditAction, len(logs))
	for i, l := range logs {
		out[i] = l.Action
	}
	return out
}

func mustCreate(t *testing.T, s *AuditStore, log *models.AuditLog) {
	t.Helper()
	if err := s.Create(context.Background(), log); err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}
}

func TestAuditStore_Cleanup(t *testing.T) {
	db := setupTestDB(t)
	store := NewAuditStore(db)

	now := time.Now()

	old := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-48 * time.Hour),
	}
	_ = store.Create(context.Background(), old)

	recent := &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "test@example.com", Action: models.AuditActionLogin,
		ResourceType: "auth", Timestamp: now.Add(-1 * time.Hour),
	}
	_ = store.Create(context.Background(), recent)

	deleted, err := store.Cleanup(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	_, total, _ := store.FindAll(context.Background(), 10, 0)
	if total != 1 {
		t.Errorf("expected 1 remaining, got %d", total)
	}
}

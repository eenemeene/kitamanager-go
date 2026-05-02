package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// mockAuditStore implements store.AuditStorer for testing resilience behavior.
type mockAuditStore struct {
	createErr   error
	createCount atomic.Int64
}

func (m *mockAuditStore) Create(_ context.Context, _ *models.AuditLog) error {
	m.createCount.Add(1)
	return m.createErr
}

func (m *mockAuditStore) FindAll(context.Context, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) FindByUser(context.Context, uint, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) FindByAction(context.Context, models.AuditAction, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) FindByDateRange(context.Context, time.Time, time.Time, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) FindFailedLogins(context.Context, string, time.Time, int) ([]models.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditStore) CountFailedLoginsSince(context.Context, string, time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuditStore) CountFailedPasswordChangesSince(context.Context, uint, time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuditStore) CountFailedPasswordResetsSince(context.Context, uint, time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuditStore) CountFailedMFAChallengesSince(context.Context, uint, time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuditStore) FindByID(context.Context, uint) (*models.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditStore) FindByOrganization(context.Context, uint, string, *uint, *time.Time, *time.Time, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) FindAllFiltered(context.Context, string, *uint, *time.Time, *time.Time, int, int) ([]models.AuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockAuditStore) Cleanup(context.Context, time.Time) (int64, error) { return 0, nil }

func TestAuditService_NewAndShutdown(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	// Shutdown should complete cleanly without panic
	svc.Shutdown()
}

func TestAuditService_LogLogin(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	var userID uint = 1
	svc.LogLogin(context.Background(), userID, "user@example.com", "127.0.0.1", "TestAgent/1.0")
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log.Action != models.AuditActionLogin {
		t.Errorf("Action = %v, want %v", log.Action, models.AuditActionLogin)
	}
	if log.UserID == nil || *log.UserID != userID {
		t.Errorf("UserID = %v, want %d", log.UserID, userID)
	}
	if log.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %v, want user@example.com", log.UserEmail)
	}
	if log.IPAddress != "127.0.0.1" {
		t.Errorf("IPAddress = %v, want 127.0.0.1", log.IPAddress)
	}
	if log.UserAgent != "TestAgent/1.0" {
		t.Errorf("UserAgent = %v, want TestAgent/1.0", log.UserAgent)
	}
	if !log.Success {
		t.Error("expected Success = true")
	}
	if log.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestAuditService_LogLoginFailed(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogLoginFailed(context.Background(), "bad@example.com", "10.0.0.1", "BadAgent/1.0", "invalid password")
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	log := logs[0]
	if log.Action != models.AuditActionLoginFailed {
		t.Errorf("Action = %v, want %v", log.Action, models.AuditActionLoginFailed)
	}
	if log.UserEmail != "bad@example.com" {
		t.Errorf("UserEmail = %v, want bad@example.com", log.UserEmail)
	}
	if log.Success {
		t.Error("expected Success = false")
	}

	var details map[string]string
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if details["reason"] != "invalid password" {
		t.Errorf("details[reason] = %v, want 'invalid password'", details["reason"])
	}
}

func TestAuditService_LogSuperAdminChange(t *testing.T) {
	ctx := context.Background()

	t.Run("grant", func(t *testing.T) {
		db := setupTestDB(t)
		auditStore := store.NewAuditStore(db)
		svc := NewAuditService(auditStore)

		svc.LogSuperAdminChange(context.Background(), 1, "actor@example.com", 2, "target@example.com", true, "127.0.0.1")
		svc.Shutdown()

		logs, _, err := store.NewAuditStore(db).FindByAction(ctx, models.AuditActionSuperAdminGrant, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 log, got %d", len(logs))
		}
		if logs[0].Action != models.AuditActionSuperAdminGrant {
			t.Errorf("Action = %v, want %v", logs[0].Action, models.AuditActionSuperAdminGrant)
		}
		if logs[0].ResourceType != "user" {
			t.Errorf("ResourceType = %v, want user", logs[0].ResourceType)
		}
		if logs[0].ResourceID == nil || *logs[0].ResourceID != 2 {
			t.Errorf("ResourceID = %v, want 2", logs[0].ResourceID)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		db := setupTestDB(t)
		auditStore := store.NewAuditStore(db)
		svc := NewAuditService(auditStore)

		svc.LogSuperAdminChange(context.Background(), 1, "actor@example.com", 3, "revoked@example.com", false, "127.0.0.1")
		svc.Shutdown()

		logs, _, err := store.NewAuditStore(db).FindByAction(ctx, models.AuditActionSuperAdminRevoke, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 log, got %d", len(logs))
		}
		if logs[0].Action != models.AuditActionSuperAdminRevoke {
			t.Errorf("Action = %v, want %v", logs[0].Action, models.AuditActionSuperAdminRevoke)
		}
	})
}

func TestAuditService_LogUserAddToOrg(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Kita")
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogUserAddToOrg(context.Background(), 1, "actor@example.com", 5, org.ID, "admin", "127.0.0.1")
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	log := logs[0]
	if log.Action != models.AuditActionUserAddToOrg {
		t.Errorf("Action = %v, want %v", log.Action, models.AuditActionUserAddToOrg)
	}
	if log.ResourceType != "user_organization" {
		t.Errorf("ResourceType = %v, want user_organization", log.ResourceType)
	}
	if log.OrganizationID == nil || *log.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %v, want %d", log.OrganizationID, org.ID)
	}

	var details map[string]any
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if uint(details["organization_id"].(float64)) != org.ID {
		t.Errorf("details[organization_id] = %v, want %d", details["organization_id"], org.ID)
	}
	if details["role"] != "admin" {
		t.Errorf("details[role] = %v, want admin", details["role"])
	}
}

func TestAuditService_LogUserRemoveFromOrg(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Kita")
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogUserRemoveFromOrg(context.Background(), 1, "actor@example.com", 5, org.ID, "127.0.0.1")
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	log := logs[0]
	if log.Action != models.AuditActionUserRemoveFromOrg {
		t.Errorf("Action = %v, want %v", log.Action, models.AuditActionUserRemoveFromOrg)
	}
	if log.ResourceType != "user_organization" {
		t.Errorf("ResourceType = %v, want user_organization", log.ResourceType)
	}
	if log.OrganizationID == nil || *log.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %v, want %d", log.OrganizationID, org.ID)
	}

	var details map[string]any
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if uint(details["organization_id"].(float64)) != org.ID {
		t.Errorf("details[organization_id] = %v, want %d", details["organization_id"], org.ID)
	}
}

func TestAuditService_LogRoleChange(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Kita")
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogRoleChange(context.Background(), 1, "actor@example.com", 5, org.ID, "manager", "admin", "127.0.0.1")
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	log := logs[0]
	if log.Action != models.AuditActionRoleChange {
		t.Errorf("Action = %v, want %v", log.Action, models.AuditActionRoleChange)
	}
	if log.OrganizationID == nil || *log.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %v, want %d", log.OrganizationID, org.ID)
	}

	var details map[string]any
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if details["old_role"] != "manager" {
		t.Errorf("details[old_role] = %v, want manager", details["old_role"])
	}
	if details["new_role"] != "admin" {
		t.Errorf("details[new_role] = %v, want admin", details["new_role"])
	}
}

func TestAuditService_LogResourceDelete(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		wantAction   models.AuditAction
	}{
		{"employee", "employee", models.AuditActionEmployeeDelete},
		{"child", "child", models.AuditActionChildDelete},
		{"organization", "organization", models.AuditActionOrgDelete},
		{"user", "user", models.AuditActionUserDelete},
		{"unknown type", "widget", "widget_delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			auditStore := store.NewAuditStore(db)
			svc := NewAuditService(auditStore)
			ctx := context.Background()

			svc.LogResourceDelete(context.Background(), 1, "actor@example.com", tt.resourceType, 42, "Test Resource", "127.0.0.1", nil)
			svc.Shutdown()

			logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if total != 1 {
				t.Fatalf("expected total 1, got %d", total)
			}

			log := logs[0]
			if log.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", log.Action, tt.wantAction)
			}
			if log.ResourceType != tt.resourceType {
				t.Errorf("ResourceType = %v, want %v", log.ResourceType, tt.resourceType)
			}
			if log.ResourceID == nil || *log.ResourceID != 42 {
				t.Errorf("ResourceID = %v, want 42", log.ResourceID)
			}

			var details map[string]any
			if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
				t.Fatalf("failed to unmarshal details: %v", err)
			}
			if details["resource_name"] != "Test Resource" {
				t.Errorf("details[resource_name] = %v, want Test Resource", details["resource_name"])
			}
		})
	}
}

func TestAuditService_LogResourceCreate(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		wantAction   models.AuditAction
	}{
		{"employee (default)", "employee", "employee_create"},
		{"user", "user", models.AuditActionUserCreate},
		{"organization", "organization", models.AuditActionOrgCreate},
		{"unknown type", "widget", "widget_create"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			auditStore := store.NewAuditStore(db)
			svc := NewAuditService(auditStore)
			ctx := context.Background()

			svc.LogResourceCreate(context.Background(), 1, "actor@example.com", tt.resourceType, 50, "Test Resource", "127.0.0.1", nil)
			svc.Shutdown()

			logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if total != 1 {
				t.Fatalf("expected total 1, got %d", total)
			}

			log := logs[0]
			if log.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", log.Action, tt.wantAction)
			}
			if log.ResourceType != tt.resourceType {
				t.Errorf("ResourceType = %v, want %v", log.ResourceType, tt.resourceType)
			}
			if log.ResourceID == nil || *log.ResourceID != 50 {
				t.Errorf("ResourceID = %v, want 50", log.ResourceID)
			}
		})
	}
}

func TestAuditService_LogResourceUpdate(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogResourceUpdate(context.Background(), 1, "actor@example.com", "child", 30, "Jane Doe", "127.0.0.1", nil)
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindAll(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	log := logs[0]
	if log.Action != "child_update" {
		t.Errorf("Action = %v, want child_update", log.Action)
	}
	if log.ResourceType != "child" {
		t.Errorf("ResourceType = %v, want child", log.ResourceType)
	}
	if log.ResourceID == nil || *log.ResourceID != 30 {
		t.Errorf("ResourceID = %v, want 30", log.ResourceID)
	}
}

func TestAuditService_GetLogs(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	// Add multiple logs
	for i := range 5 {
		svc.LogLogin(context.Background(), uint(i+1), "user@example.com", "127.0.0.1", "Agent")
	}
	svc.Shutdown()

	// Use a read-only service (no channel needed)
	readSvc := &AuditService{store: store.NewAuditStore(db)}

	// Verify total
	logs, total, err := readSvc.GetLogs(ctx, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}

	// Test pagination - limit 2
	logs, total, err = readSvc.GetLogs(ctx, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5 with limit, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs with limit, got %d", len(logs))
	}

	// Test pagination - offset 3
	logs, total, err = readSvc.GetLogs(ctx, 100, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5 with offset, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs with offset 3, got %d", len(logs))
	}
}

func TestAuditService_GetLogsByUser(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	// Log for user 1
	svc.LogLogin(context.Background(), 1, "user1@example.com", "127.0.0.1", "Agent")
	svc.LogLogin(context.Background(), 1, "user1@example.com", "127.0.0.1", "Agent")

	// Log for user 2
	svc.LogLogin(context.Background(), 2, "user2@example.com", "127.0.0.1", "Agent")

	svc.Shutdown()

	readSvc := &AuditService{store: store.NewAuditStore(db)}

	// Filter by user 1
	logs, total, err := readSvc.GetLogsByUser(ctx, 1, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2 for user 1, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs for user 1, got %d", len(logs))
	}

	// Filter by user 2
	logs, total, err = readSvc.GetLogsByUser(ctx, 2, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for user 2, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log for user 2, got %d", len(logs))
	}

	// Non-existent user
	logs, total, err = readSvc.GetLogsByUser(ctx, 999, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0 for non-existent user, got %d", total)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs for non-existent user, got %d", len(logs))
	}
}

func TestAuditService_CountRecentFailedLogins(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	// Add failed login attempts
	svc.LogLoginFailed(context.Background(), "fail@example.com", "127.0.0.1", "Agent", "bad password")
	svc.LogLoginFailed(context.Background(), "fail@example.com", "127.0.0.1", "Agent", "bad password")
	svc.LogLoginFailed(context.Background(), "fail@example.com", "127.0.0.1", "Agent", "bad password")
	// Different email
	svc.LogLoginFailed(context.Background(), "other@example.com", "127.0.0.1", "Agent", "bad password")

	svc.Shutdown()

	readSvc := &AuditService{store: store.NewAuditStore(db)}

	count, err := readSvc.CountRecentFailedLogins(ctx, "fail@example.com", 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// Different email should have 1
	count, err = readSvc.CountRecentFailedLogins(ctx, "other@example.com", 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Non-existent email should be 0
	count, err = readSvc.CountRecentFailedLogins(ctx, "none@example.com", 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestAuditService_GetLogsFiltered(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogLogin(context.Background(), 1, "user1@example.com", "127.0.0.1", "Agent")
	svc.LogLogin(context.Background(), 2, "user2@example.com", "127.0.0.1", "Agent")
	svc.LogLoginFailed(context.Background(), "bad@example.com", "127.0.0.1", "Agent", "wrong password")
	svc.LogResourceCreate(context.Background(), 1, "user1@example.com", "employee", 10, "Jane", "127.0.0.1", nil)
	svc.Shutdown()

	readSvc := &AuditService{store: store.NewAuditStore(db)}

	t.Run("no filters returns all", func(t *testing.T) {
		logs, total, err := readSvc.GetLogsFiltered(ctx, "", nil, nil, nil, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 4 {
			t.Errorf("expected total 4, got %d", total)
		}
		if len(logs) != 4 {
			t.Errorf("expected 4 logs, got %d", len(logs))
		}
	})

	t.Run("filter by action (substring)", func(t *testing.T) {
		// The action filter is a case-insensitive substring match, so the
		// fragment "login" also matches "login_failed". Two LogLogin rows
		// plus one LogLoginFailed row = 3.
		logs, total, err := readSvc.GetLogsFiltered(ctx, string(models.AuditActionLogin), nil, nil, nil, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(logs) != 3 {
			t.Errorf("expected 3 logs, got %d", len(logs))
		}
	})

	t.Run("filter by action — exact action string still works", func(t *testing.T) {
		// "employee_create" is unique enough that the substring behavior
		// does not widen the match. Guards against regressions where a
		// full action string suddenly returns extras.
		logs, total, err := readSvc.GetLogsFiltered(ctx, "employee_create", nil, nil, nil, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(logs) != 1 || logs[0].Action != "employee_create" {
			t.Errorf("got logs=%v, want single employee_create", logs)
		}
	})

	t.Run("filter by user_id", func(t *testing.T) {
		userID := uint(1)
		logs, total, err := readSvc.GetLogsFiltered(ctx, "", &userID, nil, nil, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(logs) != 2 {
			t.Errorf("expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		logs, total, err := readSvc.GetLogsFiltered(ctx, "", nil, nil, nil, 2, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 4 {
			t.Errorf("expected total 4, got %d", total)
		}
		if len(logs) != 2 {
			t.Errorf("expected 2 logs (limit), got %d", len(logs))
		}
	})
}

func TestAuditService_GetLogByID(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogLogin(context.Background(), 1, "user@example.com", "127.0.0.1", "Agent")
	svc.Shutdown()

	readSvc := &AuditService{store: store.NewAuditStore(db)}

	// Get all logs to find the ID
	logs, _, _ := readSvc.GetLogs(ctx, 100, 0)
	if len(logs) == 0 {
		t.Fatal("expected at least one log")
	}

	t.Run("found", func(t *testing.T) {
		log, err := readSvc.GetLogByID(ctx, logs[0].ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if log.ID != logs[0].ID {
			t.Errorf("expected ID %d, got %d", logs[0].ID, log.ID)
		}
		if log.Action != models.AuditActionLogin {
			t.Errorf("expected action %s, got %s", models.AuditActionLogin, log.Action)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := readSvc.GetLogByID(ctx, 99999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAuditService_GetLogsFiltered_NilService(t *testing.T) {
	var svc *AuditService
	ctx := context.Background()

	logs, total, err := svc.GetLogsFiltered(ctx, "", nil, nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logs != nil {
		t.Errorf("expected nil logs, got %v", logs)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestAuditService_GetLogByID_NilService(t *testing.T) {
	var svc *AuditService
	ctx := context.Background()

	_, err := svc.GetLogByID(ctx, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuditService_FallbackOnFullChannel(t *testing.T) {
	mock := &mockAuditStore{}
	svc := &AuditService{
		store: mock,
		logCh: make(chan *models.AuditLog, 1), // tiny buffer
		done:  make(chan struct{}),
	}
	// Do NOT start processLogs — the channel will stay full after 1 entry.

	// First entry fills the channel (async path).
	svc.log(context.Background(), &models.AuditLog{Action: "test1"})
	if svc.FallbackCount() != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", svc.FallbackCount())
	}

	// Second entry should trigger synchronous fallback.
	svc.log(context.Background(), &models.AuditLog{Action: "test2"})
	if svc.FallbackCount() != 1 {
		t.Errorf("expected 1 fallback, got %d", svc.FallbackCount())
	}
	if svc.DroppedCount() != 0 {
		t.Errorf("expected 0 dropped, got %d", svc.DroppedCount())
	}
	// The fallback entry was written via Create.
	if mock.createCount.Load() != 1 {
		t.Errorf("expected 1 store.Create call (fallback), got %d", mock.createCount.Load())
	}

	// Drain the channel so we can close cleanly.
	<-svc.logCh
	// Start worker so Shutdown completes.
	go svc.processLogs()
	svc.Shutdown()
}

func TestAuditService_DroppedOnStoreFailure(t *testing.T) {
	mock := &mockAuditStore{createErr: errors.New("db down")}
	svc := &AuditService{
		store: mock,
		logCh: make(chan *models.AuditLog, 1),
		done:  make(chan struct{}),
	}
	// Do NOT start processLogs.

	// Fill the channel.
	svc.log(context.Background(), &models.AuditLog{Action: "fill"})

	// This should fallback AND fail the store write.
	svc.log(context.Background(), &models.AuditLog{Action: "drop"})

	if svc.FallbackCount() != 1 {
		t.Errorf("expected 1 fallback, got %d", svc.FallbackCount())
	}
	if svc.DroppedCount() != 1 {
		t.Errorf("expected 1 dropped, got %d", svc.DroppedCount())
	}

	// Drain and shutdown cleanly.
	<-svc.logCh
	go svc.processLogs()
	svc.Shutdown()
}

func TestAuditService_ShutdownDrainsChannel(t *testing.T) {
	mock := &mockAuditStore{}
	svc := NewAuditService(mock)

	// Send several entries.
	for range 10 {
		svc.log(context.Background(), &models.AuditLog{Action: models.AuditAction("test")})
	}

	svc.Shutdown()

	// All 10 should have been written via the async worker.
	if mock.createCount.Load() != 10 {
		t.Errorf("expected 10 store.Create calls, got %d", mock.createCount.Load())
	}
	if svc.FallbackCount() != 0 {
		t.Errorf("expected 0 fallbacks, got %d", svc.FallbackCount())
	}
}

// TestAuditService_log_StampsRequestIDFromContext locks in the core
// invariant of the request-id plumbing: any audit row emitted inside
// a request context carries that request's X-Request-ID.
func TestAuditService_log_StampsRequestIDFromContext(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	ctx := middleware.ContextWithRequestIDForTest(context.Background(), "req-abc-123")
	svc.LogLogin(ctx, 1, "u@example.com", "127.0.0.1", "agent")
	svc.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ?", models.AuditActionLogin).Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].RequestID != "req-abc-123" {
		t.Errorf("request_id: want %q, got %q", "req-abc-123", rows[0].RequestID)
	}
}

// Non-HTTP callers pass context.Background(); those rows must keep an
// empty request_id, which translates to NULL in the DB. Locks in the
// "opt-in correlation" semantic — not every audit row has a request.
func TestAuditService_log_EmptyRequestIDForBareContext(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	svc.LogLogin(context.Background(), 1, "u@example.com", "127.0.0.1", "agent")
	svc.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ?", models.AuditActionLogin).Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].RequestID != "" {
		t.Errorf("expected empty request_id for non-HTTP caller, got %q", rows[0].RequestID)
	}
}

// An explicitly-set RequestID on the entry wins over the ctx value.
// This is the escape hatch for tooling that synthesises a correlation
// id (e.g. an import job that wants to tag every one of its rows with
// a run id).
func TestAuditService_log_ExplicitRequestIDWinsOverContext(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	ctx := middleware.ContextWithRequestIDForTest(context.Background(), "from-ctx")
	svc.log(ctx, &models.AuditLog{
		Action:    models.AuditActionLogin,
		RequestID: "explicit-override",
		Success:   true,
	})
	svc.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ?", models.AuditActionLogin).Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 || rows[0].RequestID != "explicit-override" {
		t.Errorf("expected explicit request_id to survive, got %+v", rows)
	}
}

func TestAuditService_NilSafety(t *testing.T) {
	ctx := context.Background()

	t.Run("nil service", func(t *testing.T) {
		var svc *AuditService

		// GetLogs returns empty
		logs, total, err := svc.GetLogs(ctx, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if logs != nil {
			t.Errorf("expected nil logs, got %v", logs)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}

		// GetLogsByUser returns empty
		logs, total, err = svc.GetLogsByUser(ctx, 1, 100, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if logs != nil {
			t.Errorf("expected nil logs, got %v", logs)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}

		// CountRecentFailedLogins returns 0
		count, err := svc.CountRecentFailedLogins(ctx, "test@example.com", 1*time.Hour)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}

		// Shutdown doesn't panic
		svc.Shutdown()
	})

	t.Run("nil channel", func(t *testing.T) {
		svc := &AuditService{}

		// Log methods should not panic with nil channel
		svc.LogLogin(context.Background(), 1, "test@example.com", "127.0.0.1", "Agent")
		svc.LogLoginFailed(context.Background(), "test@example.com", "127.0.0.1", "Agent", "reason")
		svc.LogSuperAdminChange(context.Background(), 1, "actor@example.com", 2, "test@example.com", true, "127.0.0.1")
		svc.LogUserAddToOrg(context.Background(), 1, "actor@example.com", 2, 3, "admin", "127.0.0.1")
		svc.LogUserRemoveFromOrg(context.Background(), 1, "actor@example.com", 2, 3, "127.0.0.1")
		svc.LogRoleChange(context.Background(), 1, "actor@example.com", 2, 3, "old", "new", "127.0.0.1")
		svc.LogResourceDelete(context.Background(), 1, "actor@example.com", "employee", 2, "name", "127.0.0.1", nil)
		svc.LogResourceCreate(context.Background(), 1, "actor@example.com", "employee", 2, "name", "127.0.0.1", nil)
		svc.LogResourceUpdate(context.Background(), 1, "actor@example.com", "employee", 2, "name", "127.0.0.1", nil)

		// Shutdown doesn't panic
		svc.Shutdown()
	})
}

// TestAuditService_ActorEmailSnapshot drives every Log* method that carries an
// actor email and asserts the row's UserEmail field matches what was passed.
// This guards against the regression where handlers were logging only UserID,
// leaving UserEmail empty and rendering the audit UI "User" column as a dash.
func TestAuditService_ActorEmailSnapshot(t *testing.T) {
	const actorID uint = 7
	const actorEmail = "actor@example.com"

	cases := []struct {
		name        string
		needsOrg    bool // true when the Log method sets OrganizationID (FK)
		wantAction  models.AuditAction
		invoke      func(svc *AuditService, orgID uint)
		assertExtra func(t *testing.T, log models.AuditLog)
	}{
		{
			name:       "LogResourceCreate",
			wantAction: "employee_create",
			invoke: func(svc *AuditService, _ uint) {
				svc.LogResourceCreate(context.Background(), actorID, actorEmail, "employee", 1, "Jane", "127.0.0.1", nil)
			},
		},
		{
			name:       "LogResourceUpdate",
			wantAction: "employee_update",
			invoke: func(svc *AuditService, _ uint) {
				svc.LogResourceUpdate(context.Background(), actorID, actorEmail, "employee", 1, "Jane", "127.0.0.1", nil)
			},
		},
		{
			name:       "LogResourceDelete",
			wantAction: models.AuditActionEmployeeDelete,
			invoke: func(svc *AuditService, _ uint) {
				svc.LogResourceDelete(context.Background(), actorID, actorEmail, "employee", 1, "Jane", "127.0.0.1", nil)
			},
		},
		{
			name:       "LogUserAddToOrg",
			needsOrg:   true,
			wantAction: models.AuditActionUserAddToOrg,
			invoke: func(svc *AuditService, orgID uint) {
				svc.LogUserAddToOrg(context.Background(), actorID, actorEmail, 2, orgID, "admin", "127.0.0.1")
			},
		},
		{
			name:       "LogUserRemoveFromOrg",
			needsOrg:   true,
			wantAction: models.AuditActionUserRemoveFromOrg,
			invoke: func(svc *AuditService, orgID uint) {
				svc.LogUserRemoveFromOrg(context.Background(), actorID, actorEmail, 2, orgID, "127.0.0.1")
			},
		},
		{
			name:       "LogRoleChange",
			needsOrg:   true,
			wantAction: models.AuditActionRoleChange,
			invoke: func(svc *AuditService, orgID uint) {
				svc.LogRoleChange(context.Background(), actorID, actorEmail, 2, orgID, "manager", "admin", "127.0.0.1")
			},
		},
		{
			name:       "LogSuperAdminChange_grant",
			wantAction: models.AuditActionSuperAdminGrant,
			invoke: func(svc *AuditService, _ uint) {
				svc.LogSuperAdminChange(context.Background(), actorID, actorEmail, 2, "target@example.com", true, "127.0.0.1")
			},
			assertExtra: func(t *testing.T, log models.AuditLog) {
				t.Helper()
				var details map[string]any
				if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
					t.Fatalf("details: %v", err)
				}
				// Actor email belongs on the row, target email belongs in Details.
				// If these ever get swapped the "User" column will show the target.
				if details["target_user_email"] != "target@example.com" {
					t.Errorf("details[target_user_email]=%v", details["target_user_email"])
				}
			},
		},
		{
			name:       "LogPasswordReset",
			wantAction: models.AuditActionPasswordReset,
			invoke: func(svc *AuditService, _ uint) {
				svc.LogPasswordReset(context.Background(), actorID, actorEmail, 2, "target@example.com", "127.0.0.1")
			},
			assertExtra: func(t *testing.T, log models.AuditLog) {
				t.Helper()
				var details map[string]any
				if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
					t.Fatalf("details: %v", err)
				}
				if details["target_user_email"] != "target@example.com" {
					t.Errorf("details[target_user_email]=%v", details["target_user_email"])
				}
			},
		},
		{
			name:       "LogDataExport",
			needsOrg:   true,
			wantAction: "child_export",
			invoke: func(svc *AuditService, orgID uint) {
				svc.LogDataExport(context.Background(), actorID, actorEmail, "child", orgID, 5, "127.0.0.1")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			var orgID uint
			if tc.needsOrg {
				orgID = createTestOrganization(t, db, "Test Kita").ID
			}
			svc := NewAuditService(store.NewAuditStore(db))
			tc.invoke(svc, orgID)
			svc.Shutdown()

			logs, total, err := store.NewAuditStore(db).FindByAction(context.Background(), tc.wantAction, 10, 0)
			if err != nil {
				t.Fatalf("FindByAction: %v", err)
			}
			if total != 1 {
				t.Fatalf("expected 1 row, got %d", total)
			}
			row := logs[0]
			if row.UserEmail != actorEmail {
				t.Errorf("UserEmail = %q, want %q", row.UserEmail, actorEmail)
			}
			if row.UserID == nil || *row.UserID != actorID {
				t.Errorf("UserID = %v, want %d", row.UserID, actorID)
			}
			if tc.assertExtra != nil {
				tc.assertExtra(t, row)
			}
		})
	}
}

// TestAuditService_ActorEmail_EmptyString ensures the Log* path doesn't reject
// or drop entries when the context has no email (e.g. for a token issued
// before the email claim existed, or unauthenticated code paths). An empty
// UserEmail is acceptable; a missing row is not.
func TestAuditService_ActorEmail_EmptyString(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	svc.LogResourceDelete(context.Background(), 42, "", "child", 99, "Unknown", "127.0.0.1", nil)
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindByAction(context.Background(), models.AuditActionChildDelete, 10, 0)
	if err != nil {
		t.Fatalf("FindByAction: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 row, got %d", total)
	}
	if logs[0].UserEmail != "" {
		t.Errorf("UserEmail = %q, want empty", logs[0].UserEmail)
	}
	if logs[0].UserID == nil || *logs[0].UserID != 42 {
		t.Errorf("UserID = %v, want 42", logs[0].UserID)
	}
}

// TestAuditService_ActorEmail_MaxLength verifies a 255-character email (the
// column's VARCHAR limit) persists intact. The schema would truncate or fail
// beyond this; catching it here keeps the boundary documented in a test.
func TestAuditService_ActorEmail_MaxLength(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	// 64-char local part + @ + 190-char domain = 255 chars total.
	local := strings.Repeat("a", 64)
	domain := strings.Repeat("b", 186) + ".com"
	email := local + "@" + domain
	if len(email) != 255 {
		t.Fatalf("test data wrong length: %d", len(email))
	}

	svc.LogResourceCreate(context.Background(), 1, email, "child", 1, "n", "127.0.0.1", nil)
	svc.Shutdown()

	logs, total, err := store.NewAuditStore(db).FindByAction(context.Background(), "child_create", 10, 0)
	if err != nil {
		t.Fatalf("FindByAction: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 row, got %d", total)
	}
	if logs[0].UserEmail != email {
		t.Errorf("UserEmail length = %d, want 255 (got %q)", len(logs[0].UserEmail), logs[0].UserEmail)
	}
}

// TestAuditService_SuperAdminChange_ActorNotSwappedWithTarget guards against
// a parameter-swap regression in the call sites. If a future refactor flipped
// actor and target emails, LogSuperAdminChange would mislabel who performed
// the change — exactly the class of mistake a compliance audit catches.
func TestAuditService_SuperAdminChange_ActorNotSwappedWithTarget(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAuditService(store.NewAuditStore(db))

	svc.LogSuperAdminChange(context.Background(), 10, "admin@example.com", 20, "promoted@example.com", true, "127.0.0.1")
	svc.Shutdown()

	logs, _, err := store.NewAuditStore(db).FindByAction(context.Background(), models.AuditActionSuperAdminGrant, 10, 0)
	if err != nil {
		t.Fatalf("FindByAction: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 row, got %d", len(logs))
	}
	row := logs[0]
	if row.UserEmail != "admin@example.com" {
		t.Errorf("row.UserEmail = %q, want admin@example.com (the actor)", row.UserEmail)
	}
	if row.UserID == nil || *row.UserID != 10 {
		t.Errorf("row.UserID = %v, want 10 (the actor)", row.UserID)
	}
	if row.ResourceID == nil || *row.ResourceID != 20 {
		t.Errorf("row.ResourceID = %v, want 20 (the target)", row.ResourceID)
	}

	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("details: %v", err)
	}
	if details["target_user_email"] != "promoted@example.com" {
		t.Errorf("details[target_user_email] = %v, want promoted@example.com", details["target_user_email"])
	}
	if details["target_user_email"] == row.UserEmail {
		t.Errorf("target and actor emails are identical — likely a parameter swap")
	}
}

// TestAuditService_LogAuditLogPurged emits the self-marker that the
// hourly retention sweeper writes after deleting old rows. Closes
// audit finding O-M-8 follow-up: an investigator must be able to
// distinguish "rows missing because retention purged them" from
// "rows missing because someone tampered."
func TestAuditService_LogAuditLogPurged(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)

	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour) // 30 days ago
	svc.LogAuditLogPurged(context.Background(), 142, cutoff)
	svc.Shutdown()

	logs, _, err := auditStore.FindByAction(context.Background(), models.AuditActionAuditLogPurged, 100, 0)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit_log_purged row, got %d", len(logs))
	}
	row := logs[0]
	if row.UserID != nil {
		t.Errorf("UserID should be nil (system actor), got %v", row.UserID)
	}
	if row.ResourceType != "audit_log" {
		t.Errorf("ResourceType = %q, want audit_log", row.ResourceType)
	}
	if !row.Success {
		t.Error("Success = false, want true")
	}
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("details: %v", err)
	}
	// JSON-decoded numbers come back as float64.
	if details["deleted_rows"].(float64) != 142 {
		t.Errorf("details.deleted_rows = %v, want 142", details["deleted_rows"])
	}
	if details["older_than"] == "" {
		t.Error("details.older_than missing")
	}
}

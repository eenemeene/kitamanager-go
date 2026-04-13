package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

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
func (m *mockAuditStore) FindByID(context.Context, uint) (*models.AuditLog, error) {
	return nil, nil
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
	svc.LogLogin(userID, "user@example.com", "127.0.0.1", "TestAgent/1.0")
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

	svc.LogLoginFailed("bad@example.com", "10.0.0.1", "BadAgent/1.0", "invalid password")
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

		svc.LogSuperAdminChange(1, 2, "target@example.com", true, "127.0.0.1")
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

		svc.LogSuperAdminChange(1, 3, "revoked@example.com", false, "127.0.0.1")
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
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogUserAddToOrg(1, 5, 10, "admin", "127.0.0.1")
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

	var details map[string]any
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if details["organization_id"].(float64) != 10 {
		t.Errorf("details[organization_id] = %v, want 10", details["organization_id"])
	}
	if details["role"] != "admin" {
		t.Errorf("details[role] = %v, want admin", details["role"])
	}
}

func TestAuditService_LogUserRemoveFromOrg(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogUserRemoveFromOrg(1, 5, 10, "127.0.0.1")
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

	var details map[string]any
	if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}
	if details["organization_id"].(float64) != 10 {
		t.Errorf("details[organization_id] = %v, want 10", details["organization_id"])
	}
}

func TestAuditService_LogRoleChange(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	svc := NewAuditService(auditStore)
	ctx := context.Background()

	svc.LogRoleChange(1, 5, 10, "manager", "admin", "127.0.0.1")
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

			svc.LogResourceDelete(1, tt.resourceType, 42, "Test Resource", "127.0.0.1")
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

			svc.LogResourceCreate(1, tt.resourceType, 50, "Test Resource", "127.0.0.1")
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

	svc.LogResourceUpdate(1, "child", 30, "Jane Doe", "127.0.0.1")
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
		svc.LogLogin(uint(i+1), "user@example.com", "127.0.0.1", "Agent")
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
	svc.LogLogin(1, "user1@example.com", "127.0.0.1", "Agent")
	svc.LogLogin(1, "user1@example.com", "127.0.0.1", "Agent")

	// Log for user 2
	svc.LogLogin(2, "user2@example.com", "127.0.0.1", "Agent")

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
	svc.LogLoginFailed("fail@example.com", "127.0.0.1", "Agent", "bad password")
	svc.LogLoginFailed("fail@example.com", "127.0.0.1", "Agent", "bad password")
	svc.LogLoginFailed("fail@example.com", "127.0.0.1", "Agent", "bad password")
	// Different email
	svc.LogLoginFailed("other@example.com", "127.0.0.1", "Agent", "bad password")

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

	svc.LogLogin(1, "user1@example.com", "127.0.0.1", "Agent")
	svc.LogLogin(2, "user2@example.com", "127.0.0.1", "Agent")
	svc.LogLoginFailed("bad@example.com", "127.0.0.1", "Agent", "wrong password")
	svc.LogResourceCreate(1, "employee", 10, "Jane", "127.0.0.1")
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

	t.Run("filter by action", func(t *testing.T) {
		logs, total, err := readSvc.GetLogsFiltered(ctx, string(models.AuditActionLogin), nil, nil, nil, 100, 0)
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

	svc.LogLogin(1, "user@example.com", "127.0.0.1", "Agent")
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
	svc.log(&models.AuditLog{Action: "test1"})
	if svc.FallbackCount() != 0 {
		t.Fatalf("expected 0 fallbacks, got %d", svc.FallbackCount())
	}

	// Second entry should trigger synchronous fallback.
	svc.log(&models.AuditLog{Action: "test2"})
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
	svc.log(&models.AuditLog{Action: "fill"})

	// This should fallback AND fail the store write.
	svc.log(&models.AuditLog{Action: "drop"})

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
		svc.log(&models.AuditLog{Action: models.AuditAction("test")})
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
		svc.LogLogin(1, "test@example.com", "127.0.0.1", "Agent")
		svc.LogLoginFailed("test@example.com", "127.0.0.1", "Agent", "reason")
		svc.LogSuperAdminChange(1, 2, "test@example.com", true, "127.0.0.1")
		svc.LogUserAddToOrg(1, 2, 3, "admin", "127.0.0.1")
		svc.LogUserRemoveFromOrg(1, 2, 3, "127.0.0.1")
		svc.LogRoleChange(1, 2, 3, "old", "new", "127.0.0.1")
		svc.LogResourceDelete(1, "employee", 2, "name", "127.0.0.1")
		svc.LogResourceCreate(1, "employee", 2, "name", "127.0.0.1")
		svc.LogResourceUpdate(1, "employee", 2, "name", "127.0.0.1")

		// Shutdown doesn't panic
		svc.Shutdown()
	})
}

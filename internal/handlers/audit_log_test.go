package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v3/model"
	fileadapter "github.com/casbin/casbin/v3/persist/file-adapter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func seedAuditLogs(t *testing.T, auditStore *store.AuditStore) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	logs := []models.AuditLog{
		{UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin, IPAddress: "127.0.0.1", Success: true, Timestamp: now.Add(-1 * time.Hour)},
		{UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionEmployeeDelete, ResourceType: "employee", ResourceID: uintPtr(10), IPAddress: "127.0.0.1", Success: true, Timestamp: now.Add(-2 * time.Hour)},
		{UserEmail: "hacker@test.com", Action: models.AuditActionLoginFailed, IPAddress: "10.0.0.1", Success: false, Timestamp: now.Add(-3 * time.Hour)},
		{UserID: uintPtr(2), UserEmail: "user@test.com", Action: models.AuditActionLogin, IPAddress: "127.0.0.1", Success: true, Timestamp: now.Add(-4 * time.Hour)},
		{UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionUserCreate, ResourceType: "user", ResourceID: uintPtr(5), IPAddress: "127.0.0.1", Success: true, Timestamp: now.Add(-5 * time.Hour)},
	}
	for i := range logs {
		if err := auditStore.Create(ctx, &logs[i]); err != nil {
			t.Fatalf("failed to seed audit log: %v", err)
		}
	}
}

func uintPtr(v uint) *uint { return &v }

func TestAuditLogHandler_List(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 5 {
		t.Errorf("expected 5 audit logs, got %d", len(response.Data))
	}
	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
}

func TestAuditLogHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 audit logs, got %d", len(response.Data))
	}
	if response.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Total)
	}
}

func TestAuditLogHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	// Page 1, limit 2
	w := performRequest(r, "GET", "/audit-logs?page=1&limit=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 audit logs, got %d", len(response.Data))
	}
	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}

	// Page 3, limit 2
	w = performRequest(r, "GET", "/audit-logs?page=3&limit=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 audit log on last page, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_FilterByAction(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?action=login", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	// Substring match: "login" now also surfaces "login_failed", so the
	// seed's 2 login + 1 login_failed rows all come back.
	if len(response.Data) != 3 {
		t.Errorf("expected 3 login-matching logs, got %d", len(response.Data))
	}
	for _, log := range response.Data {
		if !strings.Contains(string(log.Action), "login") {
			t.Errorf("action %q does not contain 'login'", log.Action)
		}
	}
}

func TestAuditLogHandler_List_FilterByAction_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?action=nonexistent_action", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 logs, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_FilterByUserID(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?user_id=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 3 {
		t.Errorf("expected 3 logs for user 1, got %d", len(response.Data))
	}
	for _, log := range response.Data {
		if log.UserID == nil || *log.UserID != 1 {
			t.Errorf("expected user_id 1, got %v", log.UserID)
		}
	}
}

func TestAuditLogHandler_List_FilterByUserID_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?user_id=999", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 logs, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_FilterByDateRange(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create logs at known dates
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionUserCreate,
		Timestamp: time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC),
	})
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionEmployeeDelete,
		Timestamp: now,
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?from=2025-06-01&to=2025-06-30", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 log in date range, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_FilterByDateRange_InvalidFrom(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?from=not-a-date", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_FilterByDateRange_InvalidTo(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?to=not-a-date", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_FilterByDateRange_ToBeforeFrom(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?from=2025-07-01&to=2025-06-01", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_FilterByDateRange_ExceedsMaxRange(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?from=2020-01-01&to=2027-01-01", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?user_id=abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_InvalidPagination(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?page=abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()

	// user 1, login, June 2025
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})
	// user 1, login, August 2025 (outside date range)
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 8, 15, 12, 0, 0, 0, time.UTC),
	})
	// user 2, login, June 2025 (wrong user)
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(2), UserEmail: "user@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})
	// user 1, different action, June 2025 (wrong action)
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionUserCreate,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?action=login&user_id=1&from=2025-06-01&to=2025-06-30", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 log with combined filters, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_OrderedByTimestampDesc(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) < 2 {
		t.Fatal("expected at least 2 logs for ordering test")
	}

	// Verify descending timestamp order
	for i := 1; i < len(response.Data); i++ {
		if response.Data[i].Timestamp.After(response.Data[i-1].Timestamp) {
			t.Errorf("log at index %d (%v) is after log at index %d (%v), expected descending order",
				i, response.Data[i].Timestamp, i-1, response.Data[i-1].Timestamp)
		}
	}
}

func TestAuditLogHandler_List_ResponseFields(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()

	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID:       uintPtr(1),
		UserEmail:    "admin@test.com",
		Action:       models.AuditActionEmployeeDelete,
		ResourceType: "employee",
		ResourceID:   uintPtr(42),
		IPAddress:    "192.168.1.100",
		Details:      `{"resource_name":"John Doe"}`,
		Success:      true,
		Timestamp:    time.Now().UTC(),
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	// The global feed is superadmin-only in routes.go, and the handler now
	// reads ctxkeys.IsSuperAdmin to decide between the recorded IP and its
	// network prefix. Reached without the authorization middleware the
	// handler fails closed and anonymizes — correct, but not what this test
	// asserts, which is the set of fields a superadmin receives.
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.IsSuperAdmin, true)
		c.Next()
	})
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 log, got %d", len(response.Data))
	}

	log := response.Data[0]
	if log.UserEmail != "admin@test.com" {
		t.Errorf("expected user_email admin@test.com, got %s", log.UserEmail)
	}
	if log.Action != models.AuditActionEmployeeDelete {
		t.Errorf("expected action employee_delete, got %s", log.Action)
	}
	if log.ResourceType != "employee" {
		t.Errorf("expected resource_type employee, got %s", log.ResourceType)
	}
	if log.ResourceID == nil || *log.ResourceID != 42 {
		t.Errorf("expected resource_id 42, got %v", log.ResourceID)
	}
	if log.IPAddress != "192.168.1.100" {
		t.Errorf("expected ip_address 192.168.1.100, got %s", log.IPAddress)
	}
	if log.Details != `{"resource_name":"John Doe"}` {
		t.Errorf("expected details, got %s", log.Details)
	}
	if !log.Success {
		t.Error("expected success true")
	}
}

func TestAuditLogHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs/:auditLogId", handler.Get)

	// First, get a valid log ID
	logs, _, _ := auditStore.FindAll(context.Background(), 1, 0)
	if len(logs) == 0 {
		t.Fatal("expected at least one log")
	}

	w := performRequest(r, "GET", fmt.Sprintf("/audit-logs/%d", logs[0].ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AuditLogResponse
	parseResponse(t, w, &response)

	if response.ID != logs[0].ID {
		t.Errorf("expected ID %d, got %d", logs[0].ID, response.ID)
	}
}

func TestAuditLogHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs/:auditLogId", handler.Get)

	w := performRequest(r, "GET", "/audit-logs/99999", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAuditLogHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs/:auditLogId", handler.Get)

	w := performRequest(r, "GET", "/audit-logs/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs/:auditLogId", handler.Get)

	w := performRequest(r, "GET", "/audit-logs/0", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAuditLogHandler_Get_NegativeID(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs/:auditLogId", handler.Get)

	w := performRequest(r, "GET", "/audit-logs/-1", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_OnlyFromFilter(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()

	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?from=2025-06-01", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 log from June onwards, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_OnlyToFilter(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()

	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserID: uintPtr(1), UserEmail: "admin@test.com", Action: models.AuditActionLogin,
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?to=2025-03-01", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 log before March, got %d", len(response.Data))
	}
}

func TestAuditLogHandler_List_FailedLoginLogs(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?action=login_failed", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 failed login log, got %d", len(response.Data))
	}
	if response.Data[0].Success {
		t.Error("expected success=false for failed login")
	}
}

func TestAuditLogHandler_List_NilOptionalFieldsInResponse(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	ctx := context.Background()

	// Create log without UserID or ResourceID
	_ = auditStore.Create(ctx, &models.AuditLog{
		UserEmail: "anonymous@test.com",
		Action:    models.AuditActionLoginFailed,
		IPAddress: "10.0.0.1",
		Timestamp: time.Now().UTC(),
	})

	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 log, got %d", len(response.Data))
	}
	if response.Data[0].UserID != nil {
		t.Errorf("expected nil user_id, got %v", response.Data[0].UserID)
	}
	if response.Data[0].ResourceID != nil {
		t.Errorf("expected nil resource_id, got %v", response.Data[0].ResourceID)
	}
}

func TestAuditLogHandler_List_UserIDNegative(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?user_id=-1", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuditLogHandler_List_LimitExceedsMax(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := setupTestRouter()
	r.GET("/audit-logs", handler.List)

	w := performRequest(r, "GET", "/audit-logs?limit=1000", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for limit > 100, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- Authorization / Security Tests ---
//
// These tests wire up the real RequireSuperAdmin() middleware with a Casbin
// enforcer and verify that only superadmins can access the audit-log endpoints.

func getModelPathForHandlers(t *testing.T) string {
	t.Helper()
	paths := []string{
		"../../configs/rbac_model.conf",
		"configs/rbac_model.conf",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}
	t.Fatal("Could not find rbac_model.conf")
	return ""
}

func setupAuthzMiddleware(t *testing.T, db *gorm.DB) *middleware.AuthorizationMiddleware {
	t.Helper()

	modelPath := getModelPathForHandlers(t)

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	if err := os.WriteFile(policyFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create temp policy file: %v", err)
	}

	adapter := fileadapter.NewAdapter(policyFile)
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		t.Fatalf("failed to load model: %v", err)
	}

	enforcer, err := rbac.NewEnforcerWithAdapter(adapter, modelPath)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}
	enforcer.SetModel(m)

	if err := enforcer.SeedDefaultPolicies(); err != nil {
		t.Fatalf("failed to seed policies: %v", err)
	}

	userOrgStore := store.NewUserOrganizationStore(db)
	permissionService := rbac.NewPermissionService(userOrgStore, enforcer)
	return middleware.NewAuthorizationMiddleware(permissionService)
}

func setupAuditLogRouterWithAuthz(t *testing.T, db *gorm.DB, userID *uint) *gin.Engine {
	t.Helper()

	authzMw := setupAuthzMiddleware(t, db)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := gin.New()
	if userID != nil {
		uid := *userID
		r.Use(func(c *gin.Context) {
			c.Set(ctxkeys.UserID, uid)
			c.Next()
		})
	}
	r.GET("/api/v1/audit-logs", authzMw.RequireSuperAdmin(), handler.List)
	r.GET("/api/v1/audit-logs/:auditLogId", authzMw.RequireSuperAdmin(), handler.Get)
	return r
}

func TestAuditLogHandler_Authorization_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	// No userID → RequireSuperAdmin should return 401
	r := setupAuditLogRouterWithAuthz(t, db, nil)

	for _, path := range []string{"/api/v1/audit-logs", "/api/v1/audit-logs/1"} {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected status %d, got %d: %s", path, http.StatusUnauthorized, w.Code, w.Body.String())
		}
	}
}

func TestAuditLogHandler_Authorization_AdminForbidden(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	// Create a regular admin user (not superadmin)
	admin := createTestUser(t, db, "Admin User", "admin-authz@test.com", "password123")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)

	r := setupAuditLogRouterWithAuthz(t, db, &admin.ID)

	for _, path := range []string{"/api/v1/audit-logs", "/api/v1/audit-logs/1"} {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("%s: expected status %d for admin, got %d: %s", path, http.StatusForbidden, w.Code, w.Body.String())
		}
	}
}

func TestAuditLogHandler_Authorization_ManagerForbidden(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	// Create a manager user
	manager := createTestUser(t, db, "Manager User", "manager-authz@test.com", "password123")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, manager.ID, org.ID, models.RoleManager)

	r := setupAuditLogRouterWithAuthz(t, db, &manager.ID)

	for _, path := range []string{"/api/v1/audit-logs", "/api/v1/audit-logs/1"} {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("%s: expected status %d for manager, got %d: %s", path, http.StatusForbidden, w.Code, w.Body.String())
		}
	}
}

// setupOrgAuditLogRouter wires the per-org audit log endpoint with the same
// authorization middleware the real router uses, so these tests exercise the
// full RBAC path: RequirePermission(ResourceAuditLog, ActionRead).
func setupOrgAuditLogRouter(t *testing.T, db *gorm.DB, userID *uint) *gin.Engine {
	t.Helper()

	authzMw := setupAuthzMiddleware(t, db)
	auditService := createAuditService(db)
	handler := NewAuditLogHandler(auditService)

	r := gin.New()
	if userID != nil {
		uid := *userID
		r.Use(func(c *gin.Context) {
			c.Set(ctxkeys.UserID, uid)
			c.Next()
		})
	}
	r.GET("/api/v1/organizations/:orgId/audit-logs",
		authzMw.RequirePermission(rbac.ResourceAuditLog, rbac.ActionRead),
		handler.ListByOrganization)
	return r
}

func TestAuditLogHandler_ListByOrganization_Admin(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)

	// Two orgs. Admin belongs to orgA only.
	orgA := createTestOrganization(t, db, "Kita A")
	orgB := createTestOrganization(t, db, "Kita B")
	admin := createTestUser(t, db, "Admin", "admin-org-audit@test.com", "password123")
	createTestUserOrganization(t, db, admin.ID, orgA.ID, models.RoleAdmin)

	// Three seed rows: one in orgA, one in orgB, one identity-level (nil).
	ctx := context.Background()
	mustSeed := func(log *models.AuditLog) {
		if err := auditStore.Create(ctx, log); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	mustSeed(&models.AuditLog{
		UserID: &admin.ID, UserEmail: admin.Email, Action: models.AuditActionChildDelete,
		ResourceType: "child", OrganizationID: &orgA.ID, IPAddress: "127.0.0.1", Success: true,
		Timestamp: time.Now().UTC().Add(-1 * time.Hour),
	})
	mustSeed(&models.AuditLog{
		UserID: uintPtr(999), UserEmail: "other@test.com", Action: models.AuditActionEmployeeDelete,
		ResourceType: "employee", OrganizationID: &orgB.ID, IPAddress: "127.0.0.1", Success: true,
		Timestamp: time.Now().UTC().Add(-2 * time.Hour),
	})
	mustSeed(&models.AuditLog{
		UserID: &admin.ID, UserEmail: admin.Email, Action: models.AuditActionLogin,
		IPAddress: "127.0.0.1", Success: true,
		Timestamp: time.Now().UTC().Add(-3 * time.Hour),
	})

	r := setupOrgAuditLogRouter(t, db, &admin.ID)

	// Admin of orgA sees exactly one row — their org's child_delete event.
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/organizations/%d/audit-logs", orgA.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin on own org: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PaginatedResponse[models.AuditLogResponse]
	parseResponse(t, w, &resp)
	if resp.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("admin on own org: expected 1 row (not login, not cross-org), got total=%d len=%d", resp.Total, len(resp.Data))
	}
	if resp.Data[0].Action != models.AuditActionChildDelete {
		t.Errorf("unexpected action: %v", resp.Data[0].Action)
	}

	// Same admin cannot read orgB's audit log.
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/organizations/%d/audit-logs", orgB.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("admin on other org: expected 403, got %d", w.Code)
	}
}

func TestAuditLogHandler_ListByOrganization_ManagerForbidden(t *testing.T) {
	db := setupTestDB(t)
	manager := createTestUser(t, db, "Manager", "manager-org-audit@test.com", "password123")
	org := createTestOrganization(t, db, "Kita Manager Denied")
	createTestUserOrganization(t, db, manager.ID, org.ID, models.RoleManager)

	r := setupOrgAuditLogRouter(t, db, &manager.ID)
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/organizations/%d/audit-logs", org.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for manager, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_Authorization_SuperadminAllowed(t *testing.T) {
	db := setupTestDB(t)
	auditStore := store.NewAuditStore(db)
	seedAuditLogs(t, auditStore)

	// Create a superadmin user
	superadmin := createTestSuperAdmin(t, db)

	r := setupAuditLogRouterWithAuthz(t, db, &superadmin.ID)

	// List endpoint should succeed
	req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List: expected status %d for superadmin, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Get endpoint with a valid ID should succeed
	logs, _, _ := auditStore.FindAll(context.Background(), 1, 0)
	if len(logs) == 0 {
		t.Fatal("expected at least one audit log")
	}

	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/audit-logs/%d", logs[0].ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Get: expected status %d for superadmin, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

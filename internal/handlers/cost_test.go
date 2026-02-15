package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"gorm.io/gorm"
)

func createCostService(db *gorm.DB) *service.CostService {
	costStore := store.NewCostStore(db)
	transactor := store.NewTransactor(db)
	return service.NewCostService(costStore, transactor)
}

func TestCostHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/costs", handler.Create)

	body := models.CostCreateRequest{Name: "Rent"}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/costs", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.CostResponse
	parseResponse(t, w, &result)
	if result.Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", result.Name)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected org ID %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestCostHandler_Create_MissingName(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/costs", handler.Create)

	body := models.CostCreateRequest{Name: ""}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/costs", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestCostHandler_List(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create test costs directly in DB
	for _, name := range []string{"Rent", "Insurance", "Utilities"} {
		db.Create(&models.Cost{OrganizationID: org.ID, Name: name})
	}

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/costs", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/costs", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.CostResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 3 {
		t.Errorf("expected 3 costs, got %d", len(response.Data))
	}
	if response.Total != 3 {
		t.Errorf("expected total 3, got %d", response.Total)
	}
	if response.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Page)
	}
}

func TestCostHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost with entries directly in DB
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)
	db.Create(&models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents: 150000,
		Notes:       "Monthly office rent",
	})

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/costs/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/costs/%d", org.ID, cost.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.CostDetailResponse
	parseResponse(t, w, &result)
	if result.Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", result.Name)
	}
	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.Entries))
	}
}

func TestCostHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/costs/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/costs/9999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestCostHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost directly in DB
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	r := setupTestRouter()
	r.PUT("/api/v1/organizations/:orgId/costs/:id", handler.Update)

	body := models.CostUpdateRequest{Name: "Office Rent"}
	w := performRequest(r, "PUT", fmt.Sprintf("/api/v1/organizations/%d/costs/%d", org.ID, cost.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.CostResponse
	parseResponse(t, w, &result)
	if result.Name != "Office Rent" {
		t.Errorf("expected name 'Office Rent', got '%s'", result.Name)
	}
}

func TestCostHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost directly in DB
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/costs/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/costs/%d", org.ID, cost.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestCostHandler_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost directly in DB
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/costs/:id/entries", handler.CreateEntry)

	body := map[string]interface{}{
		"from":         "2024-01-01T00:00:00Z",
		"to":           "2024-12-31T00:00:00Z",
		"amount_cents": 150000,
		"notes":        "Monthly office rent",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries", org.ID, cost.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.CostEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 150000 {
		t.Errorf("expected amount_cents 150000, got %d", result.AmountCents)
	}
	if result.Notes != "Monthly office rent" {
		t.Errorf("expected notes 'Monthly office rent', got '%s'", result.Notes)
	}
	if result.CostID != cost.ID {
		t.Errorf("expected cost_id %d, got %d", cost.ID, result.CostID)
	}
}

func TestCostHandler_CreateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost directly in DB
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	// Create existing entry
	to := parseTime(t, "2024-12-31T00:00:00Z")
	db.Create(&models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z"), To: &to},
		AmountCents: 150000,
	})

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/costs/:id/entries", handler.CreateEntry)

	// Try to create an overlapping entry
	body := map[string]interface{}{
		"from":         "2024-06-01T00:00:00Z",
		"to":           "2025-06-30T00:00:00Z",
		"amount_cents": 160000,
	}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries", org.ID, cost.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestCostHandler_ListEntries(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost with multiple entries
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	to1 := parseTime(t, "2024-06-30T00:00:00Z")
	db.Create(&models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z"), To: &to1},
		AmountCents: 150000,
	})
	db.Create(&models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-07-01T00:00:00Z")},
		AmountCents: 160000,
	})

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/costs/:id/entries", handler.ListEntries)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries", org.ID, cost.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.CostEntryResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 2 {
		t.Errorf("expected 2 entries, got %d", len(response.Data))
	}
	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestCostHandler_GetEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost and entry
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents: 150000,
		Notes:       "Monthly rent",
	}
	db.Create(entry)

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/costs/:id/entries/:entryId", handler.GetEntry)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries/%d", org.ID, cost.ID, entry.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.CostEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 150000 {
		t.Errorf("expected amount_cents 150000, got %d", result.AmountCents)
	}
	if result.Notes != "Monthly rent" {
		t.Errorf("expected notes 'Monthly rent', got '%s'", result.Notes)
	}
}

func TestCostHandler_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost and entry
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents: 150000,
		Notes:       "Monthly rent",
	}
	db.Create(entry)

	r := setupTestRouter()
	r.PUT("/api/v1/organizations/:orgId/costs/:id/entries/:entryId", handler.UpdateEntry)

	body := map[string]interface{}{
		"from":         "2024-01-01T00:00:00Z",
		"to":           "2024-12-31T00:00:00Z",
		"amount_cents": 175000,
		"notes":        "Updated rent",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries/%d", org.ID, cost.ID, entry.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.CostEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 175000 {
		t.Errorf("expected amount_cents 175000, got %d", result.AmountCents)
	}
	if result.Notes != "Updated rent" {
		t.Errorf("expected notes 'Updated rent', got '%s'", result.Notes)
	}
}

func TestCostHandler_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createCostService(db)
	handler := NewCostHandler(svc, createAuditService(db))

	// Create cost and entry
	cost := &models.Cost{OrganizationID: org.ID, Name: "Rent"}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID:      cost.ID,
		Period:      models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents: 150000,
	}
	db.Create(entry)

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/costs/:id/entries/:entryId", handler.DeleteEntry)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/costs/%d/entries/%d", org.ID, cost.ID, entry.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

// parseTime is a helper to parse RFC3339 time strings in tests.
func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", value, err)
	}
	return parsed
}

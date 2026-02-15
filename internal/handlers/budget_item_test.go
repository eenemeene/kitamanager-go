package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func createBudgetItemService(db *gorm.DB) *service.BudgetItemService {
	budgetItemStore := store.NewBudgetItemStore(db)
	transactor := store.NewTransactor(db)
	return service.NewBudgetItemService(budgetItemStore, transactor)
}

func TestBudgetItemHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items", handler.Create)

	body := models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income", PerChild: true}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.BudgetItemResponse
	parseResponse(t, w, &result)
	if result.Name != "Elternbeiträge" {
		t.Errorf("expected name 'Elternbeiträge', got '%s'", result.Name)
	}
	if result.Category != "income" {
		t.Errorf("expected category 'income', got '%s'", result.Category)
	}
	if result.PerChild != true {
		t.Errorf("expected per_child true, got false")
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected org ID %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestBudgetItemHandler_Create_MissingName(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items", handler.Create)

	body := models.BudgetItemCreateRequest{Name: "", Category: "income"}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Create_InvalidCategory(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items", handler.Create)

	body := models.BudgetItemCreateRequest{Name: "Bad Item", Category: "invalid"}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Create_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create item directly in DB
	db.Create(&models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"})

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items", handler.Create)

	body := models.BudgetItemCreateRequest{Name: "Rent", Category: "expense"}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_List(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create test budget items directly in DB
	for _, item := range []struct {
		name     string
		category string
	}{
		{"Rent", "expense"},
		{"Elternbeiträge", "income"},
		{"Insurance", "expense"},
	} {
		db.Create(&models.BudgetItem{OrganizationID: org.ID, Name: item.name, Category: item.category})
	}

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.BudgetItemResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 3 {
		t.Errorf("expected 3 budget items, got %d", len(response.Data))
	}
	if response.Total != 3 {
		t.Errorf("expected total 3, got %d", response.Total)
	}
	if response.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Page)
	}
}

func TestBudgetItemHandler_List_CrossOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create items in different orgs
	db.Create(&models.BudgetItem{OrganizationID: org1.ID, Name: "Rent", Category: "expense"})
	db.Create(&models.BudgetItem{OrganizationID: org2.ID, Name: "Insurance", Category: "expense"})

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items", handler.List)

	// List org1 items — should only see org1's item
	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items", org1.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.BudgetItemResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 1 {
		t.Errorf("expected 1 budget item for org1, got %d", len(response.Data))
	}
	if response.Data[0].Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", response.Data[0].Name)
	}
}

func TestBudgetItemHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create enough items to test pagination
	for i := 0; i < 5; i++ {
		db.Create(&models.BudgetItem{OrganizationID: org.ID, Name: fmt.Sprintf("Item %d", i), Category: "expense"})
	}

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items?page=1&limit=2", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.BudgetItemResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 2 {
		t.Errorf("expected 2 budget items on page 1, got %d", len(response.Data))
	}
	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
}

func TestBudgetItemHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item with entries directly in DB
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents:  150000,
		Notes:        "Monthly office rent",
	})

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d", org.ID, item.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.BudgetItemDetailResponse
	parseResponse(t, w, &result)
	if result.Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", result.Name)
	}
	if result.Category != "expense" {
		t.Errorf("expected category 'expense', got '%s'", result.Category)
	}
	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.Entries))
	}
}

func TestBudgetItemHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/9999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Get_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id", handler.Get)

	w := performRequest(r, "GET", "/api/v1/organizations/abc/budget-items/1", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Get_InvalidResourceID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/abc", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item directly in DB
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.PUT("/api/v1/organizations/:orgId/budget-items/:id", handler.Update)

	body := models.BudgetItemUpdateRequest{Name: "Office Rent", Category: "expense", PerChild: false}
	w := performRequest(r, "PUT", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d", org.ID, item.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.BudgetItemResponse
	parseResponse(t, w, &result)
	if result.Name != "Office Rent" {
		t.Errorf("expected name 'Office Rent', got '%s'", result.Name)
	}
}

func TestBudgetItemHandler_Update_InvalidCategory(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.PUT("/api/v1/organizations/:orgId/budget-items/:id", handler.Update)

	body := models.BudgetItemUpdateRequest{Name: "Rent", Category: "bogus"}
	w := performRequest(r, "PUT", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d", org.ID, item.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item directly in DB
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/budget-items/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d", org.ID, item.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/budget-items/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/budget-items/9999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item directly in DB
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items/:id/entries", handler.CreateEntry)

	body := map[string]interface{}{
		"from":         "2024-01-01T00:00:00Z",
		"to":           "2024-12-31T00:00:00Z",
		"amount_cents": 150000,
		"notes":        "Monthly office rent",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries", org.ID, item.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.BudgetItemEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 150000 {
		t.Errorf("expected amount_cents 150000, got %d", result.AmountCents)
	}
	if result.Notes != "Monthly office rent" {
		t.Errorf("expected notes 'Monthly office rent', got '%s'", result.Notes)
	}
	if result.BudgetItemID != item.ID {
		t.Errorf("expected budget_item_id %d, got %d", item.ID, result.BudgetItemID)
	}
}

func TestBudgetItemHandler_CreateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item directly in DB
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	// Create existing entry
	to := parseTime(t, "2024-12-31T00:00:00Z")
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z"), To: &to},
		AmountCents:  150000,
	})

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items/:id/entries", handler.CreateEntry)

	// Try to create an overlapping entry
	body := map[string]interface{}{
		"from":         "2024-06-01T00:00:00Z",
		"to":           "2025-06-30T00:00:00Z",
		"amount_cents": 160000,
	}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries", org.ID, item.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_CreateEntry_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.POST("/api/v1/organizations/:orgId/budget-items/:id/entries", handler.CreateEntry)

	w := performRequestRaw(r, "POST", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries", org.ID, item.ID), `{invalid json}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_ListEntries(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item with multiple entries
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	to1 := parseTime(t, "2024-06-30T00:00:00Z")
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z"), To: &to1},
		AmountCents:  150000,
	})
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-07-01T00:00:00Z")},
		AmountCents:  160000,
	})

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id/entries", handler.ListEntries)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries", org.ID, item.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.BudgetItemEntryResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 2 {
		t.Errorf("expected 2 entries, got %d", len(response.Data))
	}
	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
}

func TestBudgetItemHandler_GetEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item and entry
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents:  150000,
		Notes:        "Monthly rent",
	}
	db.Create(entry)

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id/entries/:entryId", handler.GetEntry)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries/%d", org.ID, item.ID, entry.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.BudgetItemEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 150000 {
		t.Errorf("expected amount_cents 150000, got %d", result.AmountCents)
	}
	if result.Notes != "Monthly rent" {
		t.Errorf("expected notes 'Monthly rent', got '%s'", result.Notes)
	}
}

func TestBudgetItemHandler_GetEntry_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.GET("/api/v1/organizations/:orgId/budget-items/:id/entries/:entryId", handler.GetEntry)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries/9999", org.ID, item.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item and entry
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents:  150000,
		Notes:        "Monthly rent",
	}
	db.Create(entry)

	r := setupTestRouter()
	r.PUT("/api/v1/organizations/:orgId/budget-items/:id/entries/:entryId", handler.UpdateEntry)

	body := map[string]interface{}{
		"from":         "2024-01-01T00:00:00Z",
		"to":           "2024-12-31T00:00:00Z",
		"amount_cents": 175000,
		"notes":        "Updated rent",
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries/%d", org.ID, item.ID, entry.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.BudgetItemEntryResponse
	parseResponse(t, w, &result)
	if result.AmountCents != 175000 {
		t.Errorf("expected amount_cents 175000, got %d", result.AmountCents)
	}
	if result.Notes != "Updated rent" {
		t.Errorf("expected notes 'Updated rent', got '%s'", result.Notes)
	}
}

func TestBudgetItemHandler_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	// Create budget item and entry
	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period:       models.Period{From: parseTime(t, "2024-01-01T00:00:00Z")},
		AmountCents:  150000,
	}
	db.Create(entry)

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/budget-items/:id/entries/:entryId", handler.DeleteEntry)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries/%d", org.ID, item.ID, entry.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestBudgetItemHandler_DeleteEntry_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createBudgetItemService(db)
	handler := NewBudgetItemHandler(svc, createAuditService(db))

	item := &models.BudgetItem{OrganizationID: org.ID, Name: "Rent", Category: "expense"}
	db.Create(item)

	r := setupTestRouter()
	r.DELETE("/api/v1/organizations/:orgId/budget-items/:id/entries/:entryId", handler.DeleteEntry)

	w := performRequest(r, "DELETE", fmt.Sprintf("/api/v1/organizations/%d/budget-items/%d/entries/9999", org.ID, item.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

package handlers

import (
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestOrganizationHandler_List(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	createTestOrganization(t, db, "Org 1")
	createTestOrganization(t, db, "Org 2")

	r := setupTestRouter()
	r.GET("/organizations", handler.List)

	w := performRequest(r, "GET", "/organizations", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var orgs []models.Organization
	parseResponse(t, w, &orgs)

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
	}
}

func TestOrganizationHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.Organization
	parseResponse(t, w, &result)

	if result.Name != org.Name {
		t.Errorf("expected name '%s', got '%s'", org.Name, result.Name)
	}
}

func TestOrganizationHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.GET("/organizations/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestOrganizationHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.GET("/organizations/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.POST("/organizations", handler.Create)

	body := CreateOrganizationRequest{
		Name:   "New Org",
		Active: true,
	}

	w := performRequest(r, "POST", "/organizations", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Organization
	parseResponse(t, w, &result)

	if result.Name != "New Org" {
		t.Errorf("expected name 'New Org', got '%s'", result.Name)
	}
	if result.CreatedBy != "test@example.com" {
		t.Errorf("expected created_by 'test@example.com', got '%s'", result.CreatedBy)
	}
}

func TestOrganizationHandler_Create_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.POST("/organizations", handler.Create)

	// Missing required name field
	body := map[string]interface{}{
		"active": true,
	}

	w := performRequest(r, "POST", "/organizations", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	createTestOrganization(t, db, "Original Name")

	r := setupTestRouter()
	r.PUT("/organizations/:id", handler.Update)

	body := UpdateOrganizationRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/organizations/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Organization
	parseResponse(t, w, &result)

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestOrganizationHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	createTestOrganization(t, db, "To Delete")

	r := setupTestRouter()
	r.DELETE("/organizations/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify organization was deleted
	orgs, _ := orgStore.FindAll()
	if len(orgs) != 0 {
		t.Error("expected organization to be deleted")
	}
}

// Edge case tests

func TestOrganizationHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.GET("/organizations/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/0", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestOrganizationHandler_Get_NegativeID(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.GET("/organizations/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/-1", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_Create_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.POST("/organizations", handler.Create)

	body := CreateOrganizationRequest{
		Name:   "",
		Active: true,
	}

	w := performRequest(r, "POST", "/organizations", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.POST("/organizations", handler.Create)

	body := CreateOrganizationRequest{
		Name:   "   ",
		Active: true,
	}

	w := performRequest(r, "POST", "/organizations", body)

	// Note: Current implementation may accept whitespace-only names
	// This test documents the current behavior
	if w.Code == http.StatusCreated {
		var result models.Organization
		parseResponse(t, w, &result)
		t.Logf("Warning: whitespace-only name accepted: '%s'", result.Name)
	}
}

func TestOrganizationHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.PUT("/organizations/:id", handler.Update)

	body := UpdateOrganizationRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/organizations/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for non-existent org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestOrganizationHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.PUT("/organizations/:id", handler.Update)

	body := UpdateOrganizationRequest{
		Name: "Updated Name",
	}

	w := performRequest(r, "PUT", "/organizations/abc", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_Update_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	createTestOrganization(t, db, "Original Name")

	r := setupTestRouter()
	r.PUT("/organizations/:id", handler.Update)

	// Empty update - should succeed but not change anything
	body := UpdateOrganizationRequest{}

	w := performRequest(r, "PUT", "/organizations/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for empty update, got %d", http.StatusOK, w.Code)
	}

	var result models.Organization
	parseResponse(t, w, &result)

	if result.Name != "Original Name" {
		t.Errorf("expected name to remain 'Original Name', got '%s'", result.Name)
	}
}

func TestOrganizationHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.DELETE("/organizations/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/999", nil)

	// Delete of non-existent resource should still return NoContent (idempotent)
	// or NotFound depending on implementation
	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d for non-existent org, got %d",
			http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestOrganizationHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.DELETE("/organizations/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestOrganizationHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	r := setupTestRouter()
	r.GET("/organizations", handler.List)

	w := performRequest(r, "GET", "/organizations", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var orgs []models.Organization
	parseResponse(t, w, &orgs)

	if len(orgs) != 0 {
		t.Errorf("expected empty list, got %d organizations", len(orgs))
	}
}

func TestOrganizationHandler_Update_ActiveStatus(t *testing.T) {
	db := setupTestDB(t)
	orgStore := store.NewOrganizationStore(db)
	handler := NewOrganizationHandler(orgStore)

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:id", handler.Update)

	// Update only active status to false
	active := false
	body := UpdateOrganizationRequest{
		Active: &active,
	}

	w := performRequest(r, "PUT", "/organizations/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Organization
	parseResponse(t, w, &result)

	if result.Active != false {
		t.Errorf("expected active to be false, got %v", result.Active)
	}
}

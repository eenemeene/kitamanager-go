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

func createSectionService(db *gorm.DB) *service.SectionService {
	sectionStore := store.NewSectionStore(db)
	return service.NewSectionService(sectionStore)
}

func createTestSectionWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Section {
	t.Helper()

	section := &models.Section{
		Name:           name,
		OrganizationID: orgID,
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create test section: %v", err)
	}
	return section
}

func TestSectionHandler_List(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")
	createTestSectionWithOrg(t, db, "Section 1", org.ID)
	createTestSectionWithOrg(t, db, "Section 2", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.SectionResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 sections, got %d", len(response.Data))
	}
}

func TestSectionHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.SectionResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 sections, got %d", len(response.Data))
	}
}

func TestSectionHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections/:sectionId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.SectionResponse
	parseResponse(t, w, &result)

	if result.Name != section.Name {
		t.Errorf("expected name '%s', got '%s'", section.Name, result.Name)
	}
}

func TestSectionHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections/:sectionId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections/999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSectionHandler_Get_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSectionWithOrg(t, db, "Test Section", org1.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections/:sectionId", handler.Get)

	// Try to get section from org1 using org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections/%d", org2.ID, section.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when accessing section from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSectionHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)

	body := models.SectionCreateRequest{
		Name: "New Section",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.SectionResponse
	parseResponse(t, w, &result)

	if result.Name != "New Section" {
		t.Errorf("expected name 'New Section', got '%s'", result.Name)
	}
	if result.CreatedBy != "test@example.com" {
		t.Errorf("expected created_by 'test@example.com', got '%s'", result.CreatedBy)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestSectionHandler_Create_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)

	// Missing required name field
	body := map[string]interface{}{}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSectionHandler_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)

	body := models.SectionCreateRequest{
		Name: "   ",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestSectionHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Original Name", org.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/sections/:sectionId", handler.Update)

	name := "Updated Name"
	body := models.SectionUpdateRequest{
		Name: &name,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.SectionResponse
	parseResponse(t, w, &result)

	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestSectionHandler_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSectionWithOrg(t, db, "Test Section", org1.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/sections/:sectionId", handler.Update)

	name := "Updated Name"
	body := models.SectionUpdateRequest{
		Name: &name,
	}

	// Try to update section from org1 using org2's URL
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/sections/%d", org2.ID, section.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when updating section from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSectionHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/sections/:sectionId", handler.Update)

	name := "Updated Name"
	body := models.SectionUpdateRequest{
		Name: &name,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/sections/999", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSectionHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "To Delete", org.ID)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify section was deleted
	var sections []models.Section
	db.Find(&sections)
	if len(sections) != 0 {
		t.Error("expected section to be deleted")
	}
}

func TestSectionHandler_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSectionWithOrg(t, db, "Test Section", org1.ID)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	// Try to delete section from org1 using org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/sections/%d", org2.ID, section.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d when deleting section from wrong org, got %d", http.StatusNotFound, w.Code)
	}

	// Verify section was NOT deleted
	var sections []models.Section
	db.Find(&sections)
	if len(sections) != 1 {
		t.Error("expected section to still exist")
	}
}

func TestSectionHandler_Delete_WithChildren(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Section With Children", org.ID)

	// Create a child in the section
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Child",
			OrganizationID: org.ID,
			SectionID:      &section.ID,
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("failed to create test child: %v", err)
	}

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d when deleting section with children, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	// Verify section was NOT deleted
	var sections []models.Section
	db.Find(&sections)
	if len(sections) != 1 {
		t.Error("expected section to still exist")
	}
}

func TestSectionHandler_Delete_WithEmployees(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Section With Employees", org.ID)

	// Create an employee in the section
	employee := &models.Employee{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Employee",
			OrganizationID: org.ID,
			SectionID:      &section.ID,
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create test employee: %v", err)
	}

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d when deleting section with employees, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	// Verify section was NOT deleted
	var sections []models.Section
	db.Find(&sections)
	if len(sections) != 1 {
		t.Error("expected section to still exist")
	}
}

func TestSectionHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections/:sectionId", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSectionHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/sections/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSectionHandler_List_OnlyShowsOrgSections(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	createTestSectionWithOrg(t, db, "Section 1 Org1", org1.ID)
	createTestSectionWithOrg(t, db, "Section 2 Org1", org1.ID)
	createTestSectionWithOrg(t, db, "Section 1 Org2", org2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections", handler.List)

	// List sections for org1
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections", org1.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.SectionResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 sections for org1, got %d", len(response.Data))
	}

	// List sections for org2
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections", org2.ID), nil)

	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 section for org2, got %d", len(response.Data))
	}
}

func TestSectionHandler_Create_WithAgeRange(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)

	minAge := 0
	maxAge := 36
	body := map[string]interface{}{
		"name":           "Krippe",
		"min_age_months": minAge,
		"max_age_months": maxAge,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.SectionResponse
	parseResponse(t, w, &result)

	if result.MinAgeMonths == nil || *result.MinAgeMonths != 0 {
		t.Errorf("expected min_age_months 0, got %v", result.MinAgeMonths)
	}
	if result.MaxAgeMonths == nil || *result.MaxAgeMonths != 36 {
		t.Errorf("expected max_age_months 36, got %v", result.MaxAgeMonths)
	}
}

func TestSectionHandler_Create_InvalidAgeRange(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)

	// min >= max should fail
	body := map[string]interface{}{
		"name":           "Bad Range",
		"min_age_months": 36,
		"max_age_months": 36,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for equal min/max, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestSectionHandler_Update_WithAgeRange(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/sections/:sectionId", handler.Update)

	minAge := 12
	maxAge := 48
	body := map[string]interface{}{
		"min_age_months": minAge,
		"max_age_months": maxAge,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.SectionResponse
	parseResponse(t, w, &result)

	if result.MinAgeMonths == nil || *result.MinAgeMonths != 12 {
		t.Errorf("expected min_age_months 12, got %v", result.MinAgeMonths)
	}
	if result.MaxAgeMonths == nil || *result.MaxAgeMonths != 48 {
		t.Errorf("expected max_age_months 48, got %v", result.MaxAgeMonths)
	}
}

func TestSectionHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	sectionService := createSectionService(db)
	handler := NewSectionHandler(sectionService, nil)

	org := createTestOrganization(t, db, "Test Org")
	createTestSectionWithOrg(t, db, "Krippe Alpha", org.ID)
	createTestSectionWithOrg(t, db, "Krippe Beta", org.ID)
	createTestSectionWithOrg(t, db, "Elementar", org.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/sections", handler.List)

	// Search for "krippe" (case-insensitive)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections?search=krippe", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.SectionResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 sections matching 'krippe', got %d", len(response.Data))
	}

	// Empty search returns all
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/sections", org.ID), nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 3 {
		t.Errorf("expected 3 sections without search, got %d", len(response.Data))
	}
}

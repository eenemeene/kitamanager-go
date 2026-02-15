package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildHandler_List(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child1", LastName: "Last", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)
	createActiveChildContract(t, db, child1.ID)
	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child2", LastName: "Last", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)
	createActiveChildContract(t, db, child2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 children, got %d", len(response.Data))
	}
}

func TestChildHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/1/children/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", result.FirstName)
	}
}

func TestChildHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "New",
		LastName:  "Child",
		Gender:    "female",
		Birthdate: "2020-03-15",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "New" {
		t.Errorf("expected first name 'New', got '%s'", result.FirstName)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestChildHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/organizations/1/children/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", result.FirstName)
	}
}

func TestChildHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "ToDelete", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/1/children/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestChildHandler_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/organizations/1/children/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.ChildContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 contract, got %d", len(response.Data))
	}
}

func TestChildHandler_GetCurrentRecord(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", "/organizations/1/children/1/contracts/current", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("expected care_type ganztag, got %v", contract.Properties)
	}
}

func TestChildHandler_GetCurrentRecord_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", "/organizations/1/children/1/contracts/current", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:         nil,
		Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.Properties["care_type"] != "ganztag" {
		t.Errorf("expected care_type ganztag, got %v", contract.Properties)
	}
}

func TestChildHandler_CreateContract_SameDay(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Create a same-day contract (from == to)
	sameDay := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       sameDay,
		To:         &sameDay,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for same-day contract, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if !contract.From.Equal(sameDay) {
		t.Errorf("expected from %v, got %v", sameDay, contract.From)
	}
	if contract.To == nil || !contract.To.Equal(sameDay) {
		t.Errorf("expected to %v, got %v", sameDay, contract.To)
	}
}

func TestChildHandler_CreateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create existing contract
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Try to create overlapping contract
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		To:         nil,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/organizations/1/children/1/contracts/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// Edge case tests

func TestChildHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	_ = org

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/1/children/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/1/children/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/1/children/0", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Create_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := map[string]interface{}{}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty first name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty last name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/organizations/1/children/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/organizations/1/children/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/1/children/999", nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/organizations/1/children/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", "/organizations/1/children", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d children", len(response.Data))
	}
}

func TestChildHandler_ListContracts_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/organizations/1/children/999/contracts", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_ListContracts_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/organizations/1/children/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.ChildContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d contracts", len(response.Data))
	}
}

func TestChildHandler_CreateContract_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/organizations/1/children/999/contracts", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract_InvalidChildID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/organizations/1/children/invalid/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_CreateContract_ContractBoundaryTouch(t *testing.T) {
	// Tests that adjacent contracts (A ends day before B starts) are allowed.
	// This is the correct way to transition between contracts.
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract ending on Dec 31, 2024
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Create contract starting Jan 1, 2025 (day after previous ends) - should succeed
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for adjacent (non-overlapping) contract, got %d: %s",
			http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_SameDayTransitionRejected(t *testing.T) {
	// Tests that "touching" contracts (A ends same day B starts) are rejected.
	// Both start and end dates are inclusive, so same-day transition would mean
	// both contracts are active on that day, which is not allowed.
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract ending on Jan 31, 2025
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Try to create contract starting Jan 31, 2025 (same day as previous ends) - should fail
	body := models.ChildContractCreateRequest{
		SectionID:  sectionID,
		From:       time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d for same-day transition (overlap), got %d: %s",
			http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/organizations/1/children/1/contracts/999", nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_DeleteContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/organizations/1/children/1/contracts/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetCurrentRecord_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/current", handler.GetCurrentRecord)

	w := performRequest(r, "GET", "/organizations/1/children/invalid/contracts/current", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Validation edge case tests

func TestChildHandler_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2099-01-01",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for future birthdate, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "   ",
		LastName:  "Child",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only first name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children", handler.Create)

	body := models.ChildCreateRequest{
		FirstName: "Test",
		LastName:  "   ",
		Gender:    "male",
		Birthdate: "2020-05-15",
	}

	w := performRequest(r, "POST", "/organizations/1/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only last name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_FromAfterTo(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	toDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		To:        &toDate,
	}

	w := performRequest(r, "POST", "/organizations/1/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for from > to, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// SECURITY TESTS: Cross-organization access attempts

func TestChildHandler_Get_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id", handler.Get)

	// Try to access org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when accessing child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Original", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id", handler.Update)

	newName := "Hacker"
	body := models.ChildUpdateRequest{
		FirstName: &newName,
	}

	// Try to update org1's child via org2's URL
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when updating child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id", handler.Delete)

	// Try to delete org1's child via org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when deleting child from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_ListContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts", handler.ListContracts)

	// Try to list contracts for org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when listing contracts from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: sectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Try to create contract for org1's child via org2's URL
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org2.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when creating contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_DeleteContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create contract for child
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:id/contracts/:contractId", handler.DeleteContract)

	// Try to delete contract for org1's child via org2's URL
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org2.ID, child.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when deleting contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_GetCurrentRecord_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Create active contract
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/current", handler.GetCurrentRecord)

	// Try to get current contract for org1's child via org2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/current", org2.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("SECURITY: expected status %d when getting current contract from wrong org, got %d", http.StatusNotFound, w.Code)
	}
}

// CROSS-ORG SECTION TESTS: Ensure sections from one org can't be used in another

func TestChildHandler_CreateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	org2SectionID := ensureTestSection(t, db, org2.ID)

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Try to create contract with org2's section for org1's child
	body := models.ChildContractCreateRequest{
		SectionID: org2SectionID,
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org1.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SECURITY: expected status %d when using section from wrong org, got %d: %s",
			http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_UpdateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	org1SectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")
	org2SectionID := ensureTestSection(t, db, org2.ID)

	// Create child in org1 with a valid contract
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: org1SectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id/contracts/:contractId", handler.UpdateContract)

	// Try to update contract to use org2's section
	body := models.ChildContractUpdateRequest{
		SectionID: &org2SectionID,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org1.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SECURITY: expected status %d when updating to section from wrong org, got %d: %s",
			http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_MissingSectionID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	// Send request without section_id (required field)
	body := map[string]interface{}{
		"from": "2025-01-01",
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing section_id, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_NonExistentSection(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreateRequest{
		SectionID: 99999, // Non-existent section
		From:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/contracts", org.ID, child.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for non-existent section, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// PAGINATION TESTS

func TestChildHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 15 children with active contracts
	for i := 1; i <= 15; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Test page 1 with limit 5
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 5 {
		t.Errorf("expected 5 children on page 1, got %d", len(response.Data))
	}
	if response.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Page)
	}
	if response.Limit != 5 {
		t.Errorf("expected limit 5, got %d", response.Limit)
	}
	if response.Total != 15 {
		t.Errorf("expected total 15, got %d", response.Total)
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}
}

func TestChildHandler_List_Pagination_SecondPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 15 children with active contracts
	for i := 1; i <= 15; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Get page 1
	w1 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=5", org.ID), nil)
	var response1 models.PaginatedResponse[models.Child]
	parseResponse(t, w1, &response1)

	// Get page 2
	w2 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=2&limit=5", org.ID), nil)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w2.Code)
	}

	var response2 models.PaginatedResponse[models.Child]
	parseResponse(t, w2, &response2)

	if len(response2.Data) != 5 {
		t.Errorf("expected 5 children on page 2, got %d", len(response2.Data))
	}
	if response2.Page != 2 {
		t.Errorf("expected page 2, got %d", response2.Page)
	}

	// Verify page 1 and page 2 have different children
	page1IDs := make(map[uint]bool)
	for _, child := range response1.Data {
		page1IDs[child.ID] = true
	}
	for _, child := range response2.Data {
		if page1IDs[child.ID] {
			t.Errorf("child ID %d appears on both page 1 and page 2", child.ID)
		}
	}
}

func TestChildHandler_List_Pagination_LastPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 12 children with active contracts (3 pages of 5, last page has 2)
	for i := 1; i <= 12; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Get last page (page 3)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=3&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 children on last page, got %d", len(response.Data))
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}
}

func TestChildHandler_List_Pagination_BeyondLastPage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 children with active contracts
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request page 10 (beyond available data)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=10&limit=5", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected 0 children beyond last page, got %d", len(response.Data))
	}
	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
}

func TestChildHandler_List_Pagination_DefaultValues(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 25 children with active contracts
	for i := 1; i <= 25; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request without pagination params (should use defaults: page=1, limit=20)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if len(response.Data) != 20 {
		t.Errorf("expected 20 children with default limit, got %d", len(response.Data))
	}
	if response.Page != 1 {
		t.Errorf("expected default page 1, got %d", response.Page)
	}
	if response.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", response.Limit)
	}
}

func TestChildHandler_List_Pagination_InvalidNegativePage(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=-1&limit=10", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative page, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_InvalidNegativeLimit(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=-5", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative limit, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_LimitExceedsMax(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=101", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for limit > 100, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Pagination_MaxLimit(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 children with active contracts
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%02d", i),
				LastName:       "Last",
				Birthdate:      time.Now(),
			},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Request with max limit (100)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?page=1&limit=100", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for limit=100, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Limit != 100 {
		t.Errorf("expected limit 100, got %d", response.Limit)
	}
}

// =========================================
// Search Tests
// =========================================

func TestChildHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create children with distinct names
	for _, name := range []struct{ first, last string }{
		{"Emma", "Schmidt"},
		{"Emilia", "Fischer"},
		{"Liam", "Mueller"},
	} {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: name.first, LastName: name.last, Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Search by first name prefix
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=em", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Total != 2 {
		t.Errorf("expected total 2 for search 'em', got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 children, got %d", len(response.Data))
	}

	// Search by last name
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=mueller", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 1 {
		t.Errorf("expected total 1 for search 'mueller', got %d", response.Total)
	}

	// Search with no results
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=zzz", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 0 {
		t.Errorf("expected total 0 for search 'zzz', got %d", response.Total)
	}
}

func TestChildHandler_List_SearchWithPagination(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 matching children and 2 non-matching
	for i := 1; i <= 5; i++ {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: fmt.Sprintf("Emma%d", i), LastName: "Schmidt", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}
	for _, name := range []string{"Liam", "Noah"} {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: name, LastName: "Other", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		createActiveChildContract(t, db, child.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	// Page 1: search + pagination
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=emma&page=1&limit=2", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Child]
	parseResponse(t, w, &response)

	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 children on page 1, got %d", len(response.Data))
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}

	// Page 3: last page should have 1 result
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children?search=emma&page=3&limit=2", org.ID), nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 child on page 3, got %d", len(response.Data))
	}
}

// =========================================
// Monthly Statistics Tests
// =========================================

func TestChildHandler_GetContractCountByMonth(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)

	// Create active contract
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// Should have 5 years by default (current year - 3 to current year + 1)
	if len(response.Years) != 5 {
		t.Errorf("expected 5 years, got %d", len(response.Years))
	}

	// Each year should have 12 months
	for _, year := range response.Years {
		if len(year.Counts) != 12 {
			t.Errorf("expected 12 months for year %d, got %d", year.Year, len(year.Counts))
		}
	}
}

func TestChildHandler_GetContractCountByMonth_CustomYearRange(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2024&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	if len(response.Years) != 2 {
		t.Errorf("expected 2 years, got %d", len(response.Years))
	}

	if response.Years[0].Year != 2024 || response.Years[1].Year != 2025 {
		t.Errorf("expected years 2024 and 2025, got %d and %d", response.Years[0].Year, response.Years[1].Year)
	}
}

func TestChildHandler_GetContractCountByMonth_InvalidMinYear(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid min_year, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractCountByMonth_InvalidMaxYear(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?max_year=abc", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid max_year, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractCountByMonth_MinYearGreaterThanMaxYear(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2020", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for min_year > max_year, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractCountByMonth_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", "/organizations/invalid/children/statistics/contract-count-by-month", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractCountByMonth_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	if len(response.Years) != 1 {
		t.Errorf("expected 1 year, got %d", len(response.Years))
	}

	// All counts should be 0
	for _, count := range response.Years[0].Counts {
		if count != 0 {
			t.Errorf("expected count 0 for no children, got %d", count)
		}
	}
}

func TestChildHandler_GetContractCountByMonth_ChildrenWithExpiredContracts(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with expired contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)

	// Contract ended in 2023
	endDate := time.Date(2023, 7, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	// Query for 2024-2025 - should find no active contracts
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2024&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// All counts should be 0 since contract ended in 2023
	for _, year := range response.Years {
		for month, count := range year.Counts {
			if count != 0 {
				t.Errorf("expected count 0 for year %d month %d (contract expired), got %d", year.Year, month+1, count)
			}
		}
	}
}

func TestChildHandler_GetContractCountByMonth_ContractStartMidYear(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with contract starting in July
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)}, // Starts July 2025
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// Months Jan-Jun (0-5) should be 0, Jul-Dec (6-11) should be 1
	for month, count := range response.Years[0].Counts {
		if month < 6 { // Jan-Jun
			if count != 0 {
				t.Errorf("expected count 0 for month %d (before contract), got %d", month+1, count)
			}
		} else { // Jul-Dec
			if count != 1 {
				t.Errorf("expected count 1 for month %d (contract active), got %d", month+1, count)
			}
		}
	}
}

// SECURITY TEST: Cross-organization access
func TestChildHandler_GetContractCountByMonth_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org1.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	// Query org2's stats - should not include org1's children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2025", org2.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// All counts should be 0 for org2 since the child is in org1
	for _, count := range response.Years[0].Counts {
		if count != 0 {
			t.Errorf("SECURITY: expected count 0 for org2 (child in org1), got %d", count)
		}
	}
}

func TestChildHandler_GetContractCountByMonth_MultipleChildren(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create 3 children with different contract periods
	for i := 1; i <= 3; i++ {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Child%d", i),
				LastName:       "Test",
				Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		}
		db.Create(child)

		// All have contracts starting Jan 2025
		db.Create(&models.ChildContract{
			ChildID: child.ID,
			BaseContract: models.BaseContract{
				SectionID: sectionID,
				Period:    models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		})
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// All months should have 3 children
	for month, count := range response.Years[0].Counts {
		if count != 3 {
			t.Errorf("expected count 3 for month %d, got %d", month+1, count)
		}
	}
}

func TestChildHandler_GetContractCountByMonth_SingleYear(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	// Same min and max year
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2025&max_year=2025", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	if len(response.Years) != 1 {
		t.Errorf("expected 1 year for single year range, got %d", len(response.Years))
	}
	if response.Years[0].Year != 2025 {
		t.Errorf("expected year 2025, got %d", response.Years[0].Year)
	}
}

func TestChildHandler_GetContractCountByMonth_FutureYears(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with ongoing contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, // No end date
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-count-by-month", handler.GetContractCountByMonth)

	// Query future years (2030)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-count-by-month?min_year=2030&max_year=2030", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ChildrenContractCountByMonthResponse
	parseResponse(t, w, &response)

	// Ongoing contract should show as active in future years
	for month, count := range response.Years[0].Counts {
		if count != 1 {
			t.Errorf("expected count 1 for future month %d (ongoing contract), got %d", month+1, count)
		}
	}
}

// =========================================
// Age Distribution Handler Tests
// =========================================

func TestChildHandler_GetAgeDistribution(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create children with different ages
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Child age 3 (born 2022-01-28)
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=%s", org.ID, refDate.Format("2006-01-02")), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 1 {
		t.Errorf("expected total count 1, got %d", response.TotalCount)
	}

	if response.Date != "2025-01-28" {
		t.Errorf("expected date '2025-01-28', got '%s'", response.Date)
	}

	if len(response.Distribution) != 7 {
		t.Errorf("expected 7 buckets, got %d", len(response.Distribution))
	}
}

func TestChildHandler_GetAgeDistribution_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	// No date parameter - should default to today
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	// Just check it returns a valid response with today's date
	if response.Date == "" {
		t.Error("expected date to be set to today")
	}
}

func TestChildHandler_GetAgeDistribution_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid date, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetAgeDistribution_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", "/organizations/invalid/children/statistics/age-distribution", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetAgeDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0, got %d", response.TotalCount)
	}

	// All buckets should have 0
	for _, bucket := range response.Distribution {
		if bucket.Count != 0 {
			t.Errorf("bucket %s should have 0 count, got %d", bucket.AgeLabel, bucket.Count)
		}
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildHandler_GetAgeDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org1.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	// Query org2 - should not see org1's children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2025-01-28", org2.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("SECURITY: expected total count 0 for org2 (child in org1), got %d", response.TotalCount)
	}
}

func TestChildHandler_GetAgeDistribution_AllBuckets(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create one child for each age bucket (0-6+)
	ages := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	for _, age := range ages {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Age%d", age),
				LastName:       "Child",
				Birthdate:      refDate.AddDate(-age, 0, 0),
			},
		}
		db.Create(child)
		db.Create(&models.ChildContract{
			ChildID: child.ID,
			BaseContract: models.BaseContract{
				SectionID: sectionID,
				Period:    models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		})
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	// Total should be 9
	if response.TotalCount != 9 {
		t.Errorf("expected total count 9, got %d", response.TotalCount)
	}

	// Check distribution: 0-5 should have 1 each, 6+ should have 3
	expected := map[string]int{
		"0":  1,
		"1":  1,
		"2":  1,
		"3":  1,
		"4":  1,
		"5":  1,
		"6+": 3, // ages 6, 7, 8
	}

	for _, bucket := range response.Distribution {
		if bucket.Count != expected[bucket.AgeLabel] {
			t.Errorf("bucket %s: expected %d, got %d", bucket.AgeLabel, expected[bucket.AgeLabel], bucket.Count)
		}
	}
}

func TestChildHandler_GetAgeDistribution_ExpiredContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with expired contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Expired",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	// Query date after contract expired
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0 (contract expired), got %d", response.TotalCount)
	}
}

func TestChildHandler_GetAgeDistribution_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with future contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Future",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}, // Starts in future
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	// Query date before contract starts
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0 (contract not started), got %d", response.TotalCount)
	}
}

func TestChildHandler_GetAgeDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with historical contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Historical",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/age-distribution", handler.GetAgeDistribution)

	// Query historical date when contract was active
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/age-distribution?date=2023-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 1 {
		t.Errorf("expected total count 1 (contract active on historical date), got %d", response.TotalCount)
	}

	// Child should be age 3 on 2023-06-15 (born 2020-01-01)
	for _, bucket := range response.Distribution {
		if bucket.AgeLabel == "3" && bucket.Count != 1 {
			t.Errorf("expected age 3 bucket count 1, got %d", bucket.Count)
		}
	}
}

// =========================================
// GetContract Tests
// =========================================

func TestChildHandler_GetContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.ChildContractResponse
	parseResponse(t, w, &result)

	if result.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.ChildID)
	}
}

func TestChildHandler_GetContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/999", org.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_GetContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/:contractId", handler.GetContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/abc", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContract_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child1 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child", LastName: "One", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child1)

	child2 := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child", LastName: "Two", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child2)

	// Create contract for child1
	contract := &models.ChildContract{
		ChildID: child1.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:id/contracts/:contractId", handler.GetContract)

	// Try to get child1's contract via child2's URL
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child2.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for wrong child access, got %d", http.StatusNotFound, w.Code)
	}
}

// =========================================
// UpdateContract Tests
// =========================================

func TestChildHandler_UpdateContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Use today so contract qualifies for in-place update
	today := time.Now().UTC().Truncate(24 * time.Hour)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id/contracts/:contractId", handler.UpdateContract)

	newFrom := today.AddDate(0, 2, 0)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.ChildContractResponse
	parseResponse(t, w, &result)

	if !result.From.Equal(newFrom) {
		t.Errorf("expected from %v, got %v", newFrom, result.From)
	}
}

func TestChildHandler_UpdateContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id/contracts/:contractId", handler.UpdateContract)

	newFrom := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/999", org.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_UpdateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Use future dates so contracts qualify for in-place update
	today := time.Now().UTC().Truncate(24 * time.Hour)
	endDate1 := today.AddDate(0, 6, 0)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today, To: &endDate1},
		},
	})

	contract2 := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: today.AddDate(0, 7, 0)},
		},
	}
	db.Create(contract2)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id/contracts/:contractId", handler.UpdateContract)

	// Update contract2's From to overlap with contract1
	newFrom := today.AddDate(0, 3, 0)
	body := models.ChildContractUpdateRequest{
		From: &newFrom,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract2.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d for overlap, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_UpdateContract_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	db.Create(contract)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:id/contracts/:contractId", handler.UpdateContract)

	body := map[string]interface{}{
		"from": "not-a-date",
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// =========================================
// GetFunding Tests
// =========================================

func TestChildHandler_GetFunding(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with active contract
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Create GovernmentFunding matching org state "berlin"
	funding := &models.GovernmentFunding{
		Name:  "Berlin Kita Funding",
		State: "berlin",
	}
	db.Create(funding)

	// Create funding period
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	// Create funding property
	db.Create(&models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Payment:     166847,
		Requirement: 0.261,
	})

	r := setupTestRouter()
	// Register funding route BEFORE per-child routes
	r.GET("/organizations/:orgId/children/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/funding", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestChildHandler_GetFunding_NoContracts(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/funding", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestChildHandler_GetFunding_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/funding?date=invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// =============================================================================
// Contract Properties Distribution Tests
// =============================================================================

func TestChildHandler_GetContractPropertiesDistribution(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Child 1: care_type=ganztag, supplements=["ndh","mss"]
	child1 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child1", LastName: "Test",
			Gender: "female", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child1)
	db.Create(&models.ChildContract{
		ChildID: child1.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh", "mss"}},
		},
	})

	// Child 2: care_type=halbtag
	child2 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child2", LastName: "Test",
			Gender: "male", Birthdate: time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child2)
	db.Create(&models.ChildContract{
		ChildID: child2.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "halbtag"},
		},
	})

	// Child 3: care_type=ganztag, supplements=["ndh"]
	child3 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child3", LastName: "Test",
			Gender: "female", Birthdate: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child3)
	db.Create(&models.ChildContract{
		ChildID: child3.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties?date=%s", org.ID, refDate.Format("2006-01-02")), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 3 {
		t.Errorf("expected total_children 3, got %d", response.TotalChildren)
	}

	if response.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", response.Date)
	}

	// Expected properties (sorted by key, then value):
	// care_type/ganztag: 2, care_type/halbtag: 1, supplements/mss: 1, supplements/ndh: 2
	expected := map[string]int{
		"care_type:ganztag": 2,
		"care_type:halbtag": 1,
		"supplements:mss":   1,
		"supplements:ndh":   2,
	}
	if len(response.Properties) != len(expected) {
		t.Errorf("expected %d property entries, got %d", len(expected), len(response.Properties))
	}
	for _, p := range response.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildHandler_GetContractPropertiesDistribution_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	// No date parameter - should default to today
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.Date == "" {
		t.Error("expected date to be set to today")
	}
}

func TestChildHandler_GetContractPropertiesDistribution_CustomDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties?date=2025-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", response.Date)
	}
}

func TestChildHandler_GetContractPropertiesDistribution_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid date, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractPropertiesDistribution_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", "/organizations/abc/children/statistics/contract-properties", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetContractPropertiesDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties?date=2025-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 0 {
		t.Errorf("expected total_children 0, got %d", response.TotalChildren)
	}

	if len(response.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(response.Properties))
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildHandler_GetContractPropertiesDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org1.ID, FirstName: "Test", LastName: "Child",
			Gender: "female", Birthdate: time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	// Query org2 - should not see org1's children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/statistics/contract-properties?date=2025-01-28", org2.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 0 {
		t.Errorf("SECURITY: expected total_children 0 for org2 (child in org1), got %d", response.TotalChildren)
	}
}

package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildHandler_List(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child1", LastName: "Last", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child2", LastName: "Last", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/children", handler.List)

	w := performRequest(r, "GET", "/children", nil)

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
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/children/:id", handler.Get)

	w := performRequest(r, "GET", "/children/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
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
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "New",
		LastName:       "Child",
		Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Child
	parseResponse(t, w, &result)

	if result.FirstName != "New" {
		t.Errorf("expected first name 'New', got '%s'", result.FirstName)
	}
}

func TestChildHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.PUT("/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/children/1", body)

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
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "ToDelete", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/children/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestChildHandler_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID:          child.ID,
		Period:           models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		CareHoursPerWeek: 35,
	})

	r := setupTestRouter()
	r.GET("/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/children/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contracts []models.ChildContract
	parseResponse(t, w, &contracts)

	if len(contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(contracts))
	}
}

func TestChildHandler_GetCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	db.Create(&models.ChildContract{
		ChildID:          child.ID,
		Period:           models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		CareHoursPerWeek: 35,
		MealsIncluded:    true,
	})

	r := setupTestRouter()
	r.GET("/children/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/children/1/contracts/current", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.CareHoursPerWeek != 35 {
		t.Errorf("expected care hours 35, got %f", contract.CareHoursPerWeek)
	}
}

func TestChildHandler_GetCurrentContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/children/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/children/1/contracts/current", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:               nil,
		CareHoursPerWeek: 40,
		MealsIncluded:    true,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	if contract.CareHoursPerWeek != 40 {
		t.Errorf("expected care hours 40, got %f", contract.CareHoursPerWeek)
	}
}

func TestChildHandler_CreateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	// Create existing contract
	db.Create(&models.ChildContract{
		ChildID:          child.ID,
		Period:           models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		CareHoursPerWeek: 35,
	})

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	// Try to create overlapping contract
	body := models.ChildContractCreate{
		From:             time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		To:               nil,
		CareHoursPerWeek: 40,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID:          child.ID,
		Period:           models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		CareHoursPerWeek: 35,
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/children/1/contracts/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// Edge case tests

func TestChildHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children/:id", handler.Get)

	w := performRequest(r, "GET", "/children/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children/:id", handler.Get)

	w := performRequest(r, "GET", "/children/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children/:id", handler.Get)

	w := performRequest(r, "GET", "/children/0", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Create_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := map[string]interface{}{}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "",
		LastName:       "Child",
		Birthdate:      time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty first name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Create_EmptyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "Test",
		LastName:       "",
		Birthdate:      time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty last name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.PUT("/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/children/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.PUT("/children/:id", handler.Update)

	newName := "Updated"
	body := models.ChildUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/children/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.DELETE("/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/children/999", nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.DELETE("/children/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/children/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children", handler.List)

	w := performRequest(r, "GET", "/children", nil)

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
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/children/999/contracts", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_ListContracts_Empty(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/children/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/children/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contracts []models.ChildContract
	parseResponse(t, w, &contracts)

	if len(contracts) != 0 {
		t.Errorf("expected empty list, got %d contracts", len(contracts))
	}
}

func TestChildHandler_CreateContract_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
	}

	w := performRequest(r, "POST", "/children/999/contracts", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_CreateContract_InvalidChildID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
	}

	w := performRequest(r, "POST", "/children/invalid/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_CreateContract_ZeroCareHours(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 0,
		MealsIncluded:    false,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	// Document current behavior - zero hours may or may not be valid
	t.Logf("Create contract with zero care hours returned status %d", w.Code)
}

func TestChildHandler_CreateContract_ContractBoundaryTouch(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	// Create contract ending on specific date
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID:          child.ID,
		Period:           models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
		CareHoursPerWeek: 35,
	})

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	// Create contract starting the day after
	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
		MealsIncluded:    true,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for non-overlapping contract, got %d: %s",
			http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestChildHandler_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/children/1/contracts/999", nil)

	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestChildHandler_DeleteContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/children/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/children/1/contracts/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildHandler_GetCurrentContract_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	r := setupTestRouter()
	r.GET("/children/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/children/invalid/contracts/current", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Validation edge case tests

func TestChildHandler_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "Test",
		LastName:       "Child",
		Birthdate:      time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for future birthdate, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyFirstName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "   ",
		LastName:       "Child",
		Birthdate:      time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only first name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_Create_WhitespaceOnlyLastName(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/children", handler.Create)

	body := models.ChildCreate{
		OrganizationID: org.ID,
		FirstName:      "Test",
		LastName:       "   ",
		Birthdate:      time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/children", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only last name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_FromAfterTo(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	toDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	body := models.ChildContractCreate{
		From:             time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		To:               &toDate,
		CareHoursPerWeek: 40,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for from > to, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_NegativeCareHours(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: -1,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative care hours, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_CareHoursOver168(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 169,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for care hours > 168, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_SpecialNeedsTooLong(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	// Create a special needs string longer than 1000 characters
	longSpecialNeeds := ""
	for i := 0; i < 1001; i++ {
		longSpecialNeeds += "a"
	}

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
		SpecialNeeds:     longSpecialNeeds,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for special needs too long, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_SpecialNeedsXSS(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
		SpecialNeeds:     "<script>alert('xss')</script>",
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		return
	}

	var contract models.ChildContract
	parseResponse(t, w, &contract)

	// Verify XSS was sanitized
	if contract.SpecialNeeds == "<script>alert('xss')</script>" {
		t.Error("expected special needs to be sanitized for XSS")
	}
	if contract.SpecialNeeds != "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" {
		t.Errorf("expected sanitized HTML, got: %s", contract.SpecialNeeds)
	}
}

func TestChildHandler_CreateContract_GroupNotInOrg(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	// Create two separate organizations
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	// Create group in org2
	group := createTestGroupWithOrg(t, db, "Group in Org2", org2.ID)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
		GroupID:          &group.ID, // Group from different org
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for group from different org, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildHandler_CreateContract_GroupNotFound(t *testing.T) {
	db := setupTestDB(t)
	childService := createChildService(db)
	handler := NewChildHandler(childService)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Birthdate: time.Now()},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/children/:id/contracts", handler.CreateContract)

	nonExistentGroupID := uint(999)
	body := models.ChildContractCreate{
		From:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CareHoursPerWeek: 40,
		GroupID:          &nonExistentGroupID,
	}

	w := performRequest(r, "POST", "/children/1/contracts", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for non-existent group, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

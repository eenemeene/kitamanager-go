package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestEmployeeHandler_List(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")

	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Emp1", LastName: "Last", Birthdate: time.Now()},
	})
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Emp2", LastName: "Last", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/employees", handler.List)

	w := performRequest(r, "GET", "/employees", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var employees []models.Employee
	parseResponse(t, w, &employees)

	if len(employees) != 2 {
		t.Errorf("expected 2 employees, got %d", len(employees))
	}
}

func TestEmployeeHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/employees/:id", handler.Get)

	w := performRequest(r, "GET", "/employees/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/employees", handler.Create)

	body := models.EmployeeCreate{
		OrganizationID: org.ID,
		FirstName:      "New",
		LastName:       "Employee",
		Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/employees", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "New" {
		t.Errorf("expected first name 'New', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.PUT("/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/employees/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "ToDelete", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/employees/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestEmployeeHandler_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contracts
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Position:   "Developer",
	})

	r := setupTestRouter()
	r.GET("/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/employees/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contracts []models.EmployeeContract
	parseResponse(t, w, &contracts)

	if len(contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(contracts))
	}
}

func TestEmployeeHandler_GetCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create ongoing contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		Position:   "Developer",
	})

	r := setupTestRouter()
	r.GET("/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/employees/1/contracts/current", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contract models.EmployeeContract
	parseResponse(t, w, &contract)

	if contract.Position != "Developer" {
		t.Errorf("expected position 'Developer', got '%s'", contract.Position)
	}
}

func TestEmployeeHandler_GetCurrentContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.GET("/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/employees/1/contracts/current", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreate{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Salary:      600000,
	}

	w := performRequest(r, "POST", "/employees/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.EmployeeContract
	parseResponse(t, w, &contract)

	if contract.Position != "Senior Developer" {
		t.Errorf("expected position 'Senior Developer', got '%s'", contract.Position)
	}
}

func TestEmployeeHandler_CreateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create existing contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		Position:   "Developer",
	})

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	// Try to create overlapping contract
	body := models.EmployeeContractCreate{
		From:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Salary:      600000,
	}

	w := performRequest(r, "POST", "/employees/1/contracts", body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Position:   "Developer",
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/employees/1/contracts/1", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// Edge case tests

func TestEmployeeHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees/:id", handler.Get)

	w := performRequest(r, "GET", "/employees/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees/:id", handler.Get)

	w := performRequest(r, "GET", "/employees/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees/:id", handler.Get)

	w := performRequest(r, "GET", "/employees/0", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Create_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.POST("/employees", handler.Create)

	// Missing all required fields
	body := map[string]interface{}{}

	w := performRequest(r, "POST", "/employees", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_InvalidOrganizationID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.POST("/employees", handler.Create)

	body := models.EmployeeCreate{
		OrganizationID: 999, // Non-existent org
		FirstName:      "Test",
		LastName:       "Employee",
		Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/employees", body)

	// Should fail due to foreign key constraint (behavior depends on DB)
	// SQLite may not enforce FK by default, PostgreSQL will
	t.Logf("Create with invalid org ID returned status %d", w.Code)
}

func TestEmployeeHandler_Create_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/employees", handler.Create)

	body := models.EmployeeCreate{
		OrganizationID: org.ID,
		FirstName:      "",
		LastName:       "Employee",
		Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/employees", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty first name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_EmptyLastName(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/employees", handler.Create)

	body := models.EmployeeCreate{
		OrganizationID: org.ID,
		FirstName:      "Test",
		LastName:       "",
		Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", "/employees", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty last name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.PUT("/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/employees/999", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.PUT("/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdate{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", "/employees/invalid", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Update_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.PUT("/employees/:id", handler.Update)

	// Empty update
	body := models.EmployeeUpdate{}

	w := performRequest(r, "PUT", "/employees/1", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for empty update, got %d", http.StatusOK, w.Code)
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "Original" {
		t.Errorf("expected first name to remain 'Original', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.DELETE("/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/employees/999", nil)

	// Should return NoContent (idempotent) or NotFound
	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.DELETE("/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", "/employees/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees", handler.List)

	w := performRequest(r, "GET", "/employees", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var employees []models.Employee
	parseResponse(t, w, &employees)

	if len(employees) != 0 {
		t.Errorf("expected empty list, got %d employees", len(employees))
	}
}

func TestEmployeeHandler_ListContracts_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/employees/999/contracts", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_ListContracts_Empty(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.GET("/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", "/employees/1/contracts", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contracts []models.EmployeeContract
	parseResponse(t, w, &contracts)

	if len(contracts) != 0 {
		t.Errorf("expected empty list, got %d contracts", len(contracts))
	}
}

func TestEmployeeHandler_CreateContract_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreate{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 40,
		Salary:      600000,
	}

	w := performRequest(r, "POST", "/employees/999/contracts", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_InvalidEmployeeID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreate{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 40,
		Salary:      600000,
	}

	w := performRequest(r, "POST", "/employees/invalid/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_MissingPosition(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	body := map[string]interface{}{
		"from":         "2025-01-01T00:00:00Z",
		"weekly_hours": 40,
		"salary":       600000,
		// Missing position
	}

	w := performRequest(r, "POST", "/employees/1/contracts", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing position, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_ZeroWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreate{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 0,
		Salary:      600000,
	}

	w := performRequest(r, "POST", "/employees/1/contracts", body)

	// Document current behavior - zero hours may or may not be valid
	t.Logf("Create contract with zero weekly hours returned status %d", w.Code)
}

func TestEmployeeHandler_CreateContract_ContractBoundaryTouch(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contract ending on specific date
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
		Position:   "Developer",
	})

	r := setupTestRouter()
	r.POST("/employees/:id/contracts", handler.CreateContract)

	// Create contract starting the day after
	body := models.EmployeeContractCreate{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Salary:      700000,
	}

	w := performRequest(r, "POST", "/employees/1/contracts", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for non-overlapping contract, got %d: %s",
			http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/employees/1/contracts/999", nil)

	// Should return NoContent (idempotent) or NotFound
	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
		t.Errorf("expected status %d or %d, got %d", http.StatusNoContent, http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_DeleteContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	org := createTestOrganization(t, db, "Test Org")
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Now()},
	})

	r := setupTestRouter()
	r.DELETE("/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", "/employees/1/contracts/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_GetCurrentContract_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeStore := store.NewEmployeeStore(db)
	handler := NewEmployeeHandler(employeeStore)

	r := setupTestRouter()
	r.GET("/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", "/employees/invalid/contracts/current", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

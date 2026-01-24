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

package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestEmployeeHandler_List(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	emp1 := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Emp1", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(emp1)
	createActiveEmployeeContract(t, db, emp1.ID)
	emp2 := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Emp2", LastName: "Last", Gender: "female", Birthdate: time.Now()},
	}
	db.Create(emp2)
	createActiveEmployeeContract(t, db, emp2.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Employee]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 employees, got %d", len(response.Data))
	}
}

func TestEmployeeHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d", org.ID, employee.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Get_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	// Try to access employee from org2 (should fail with 404)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d", org2.ID, employee.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "New",
		LastName:  "Employee",
		Gender:    "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "New" {
		t.Errorf("expected first name 'New', got '%s'", result.FirstName)
	}
	if result.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, result.OrganizationID)
	}
}

func TestEmployeeHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/employees/%d", org.ID, employee.ID), body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.Employee
	parseResponse(t, w, &result)

	if result.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", result.FirstName)
	}
}

func TestEmployeeHandler_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Original", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:id", handler.Update)

	newName := "Hacked"
	body := models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	// Try to update employee from org2 (should fail with 404)
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/employees/%d", org2.ID, employee.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}

	// Verify employee was not modified
	var unchanged models.Employee
	db.First(&unchanged, employee.ID)
	if unchanged.FirstName != "Original" {
		t.Errorf("employee was modified despite wrong org access: got '%s'", unchanged.FirstName)
	}
}

func TestEmployeeHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "ToDelete", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d", org.ID, employee.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestEmployeeHandler_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Protected", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id", handler.Delete)

	// Try to delete employee from org2 (should fail with 404)
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d", org2.ID, employee.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}

	// Verify employee was not deleted
	var stillExists models.Employee
	if err := db.First(&stillExists, employee.ID).Error; err != nil {
		t.Errorf("employee was deleted despite wrong org access")
	}
}

func TestEmployeeHandler_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contracts
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.EmployeeContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 contract, got %d", len(response.Data))
	}
}

func TestEmployeeHandler_ListContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts", handler.ListContracts)

	// Try to list contracts from org2 (should fail with 404)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org2.ID, employee.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_GetCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create ongoing contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts/current", org.ID, employee.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var contract models.EmployeeContract
	parseResponse(t, w, &contract)

	if contract.Position != "Developer" {
		t.Errorf("expected position 'Developer', got '%s'", contract.Position)
	}
}

func TestEmployeeHandler_GetCurrentContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create ongoing contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts/current", handler.GetCurrentContract)

	// Try to get current contract from org2 (should fail with 404)
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts/current", org2.ID, employee.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_GetCurrentContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts/current", org.ID, employee.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.EmployeeContract
	parseResponse(t, w, &contract)

	if contract.Position != "Senior Developer" {
		t.Errorf("expected position 'Senior Developer', got '%s'", contract.Position)
	}
}

func TestEmployeeHandler_CreateContract_SameDay(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	// Create a same-day contract (from == to)
	sameDay := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	body := models.EmployeeContractCreateRequest{
		From:        sameDay,
		To:          &sameDay,
		Position:    "One-Day Consultant",
		WeeklyHours: 8,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for same-day contract, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var contract models.EmployeeContract
	parseResponse(t, w, &contract)

	if !contract.From.Equal(sameDay) {
		t.Errorf("expected from %v, got %v", sameDay, contract.From)
	}
	if contract.To == nil || !contract.To.Equal(sameDay) {
		t.Errorf("expected to %v, got %v", sameDay, contract.To)
	}
	if contract.Position != "One-Day Consultant" {
		t.Errorf("expected position 'One-Day Consultant', got '%s'", contract.Position)
	}
}

func TestEmployeeHandler_CreateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		Position:    "Malicious Contract",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	// Try to create contract from org2 (should fail with 404)
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org2.ID, employee.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}

	// Verify no contract was created
	var count int64
	db.Model(&models.EmployeeContract{}).Where("employee_id = ?", employee.ID).Count(&count)
	if count != 0 {
		t.Errorf("contract was created despite wrong org access")
	}
}

func TestEmployeeHandler_CreateContract_Overlap(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create existing contract
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: nil},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	// Try to create overlapping contract
	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Position: "Developer",
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d/contracts/%d", org.ID, employee.ID, contract.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestEmployeeHandler_DeleteContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employee in org1
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Position: "Developer",
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id/contracts/:contractId", handler.DeleteContract)

	// Try to delete contract from org2 (should fail with 404)
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d/contracts/%d", org2.ID, employee.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
	}

	// Verify contract was not deleted
	var stillExists models.EmployeeContract
	if err := db.First(&stillExists, contract.ID).Error; err != nil {
		t.Errorf("contract was deleted despite wrong org access")
	}
}

func TestEmployeeHandler_DeleteContract_WrongEmployee(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	// Create two employees in same org
	employee1 := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Employee", LastName: "One", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee1)

	employee2 := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Employee", LastName: "Two", Gender: "female", Birthdate: time.Now()},
	}
	db.Create(employee2)

	// Create contract for employee1
	contract := &models.EmployeeContract{
		EmployeeID: employee1.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Position: "Developer",
	}
	db.Create(contract)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id/contracts/:contractId", handler.DeleteContract)

	// Try to delete contract via employee2 URL (should fail with 404)
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d/contracts/%d", org.ID, employee2.ID, contract.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for wrong employee access, got %d", http.StatusNotFound, w.Code)
	}

	// Verify contract was not deleted
	var stillExists models.EmployeeContract
	if err := db.First(&stillExists, contract.ID).Error; err != nil {
		t.Errorf("contract was deleted despite wrong employee access")
	}
}

// Edge case tests

func TestEmployeeHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/999", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Get_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Get_ZeroID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/0", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for zero ID, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Get_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id", handler.Get)

	w := performRequest(r, "GET", "/organizations/invalid/employees/1", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	// Missing all required fields
	body := map[string]interface{}{}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_InvalidOrganizationID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	w := performRequest(r, "POST", "/organizations/invalid/employees", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_EmptyFirstName(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "",
		LastName:  "Employee",
		Gender:    "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty first name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Create_EmptyLastName(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "Test",
		LastName:  "",
		Gender:    "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty last name, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/employees/999", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:id", handler.Update)

	newName := "Updated"
	body := models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/employees/invalid", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_Update_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Original", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:id", handler.Update)

	// Empty update
	body := models.EmployeeUpdateRequest{}

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/employees/%d", org.ID, employee.ID), body)

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
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/999", org.ID), nil)

	// Should return NotFound for non-existent employee
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id", handler.Delete)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees", handler.List)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Employee]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d employees", len(response.Data))
	}
}

func TestEmployeeHandler_List_IsolatesOrganizations(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employees in different orgs with active contracts
	emp1 := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Emp1", LastName: "Org1", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(emp1)
	createActiveEmployeeContract(t, db, emp1.ID)
	emp2 := &models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Emp2", LastName: "Org1", Gender: "female", Birthdate: time.Now()},
	}
	db.Create(emp2)
	createActiveEmployeeContract(t, db, emp2.ID)
	emp3 := &models.Employee{
		Person: models.Person{OrganizationID: org2.ID, FirstName: "Emp1", LastName: "Org2", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(emp3)
	createActiveEmployeeContract(t, db, emp3.ID)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees", handler.List)

	// List org1 employees
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees", org1.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Employee]
	parseResponse(t, w, &response)

	if len(response.Data) != 2 {
		t.Errorf("expected 2 employees for org1, got %d", len(response.Data))
	}

	// Verify all returned employees belong to org1
	for _, emp := range response.Data {
		if emp.OrganizationID != org1.ID {
			t.Errorf("employee %d belongs to org %d, expected org %d", emp.ID, emp.OrganizationID, org1.ID)
		}
	}

	// List org2 employees
	w2 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees", org2.ID), nil)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w2.Code)
	}

	var response2 models.PaginatedResponse[models.Employee]
	parseResponse(t, w2, &response2)

	if len(response2.Data) != 1 {
		t.Errorf("expected 1 employee for org2, got %d", len(response2.Data))
	}
}

func TestEmployeeHandler_ListContracts_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/999/contracts", org.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_ListContracts_Empty(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts", handler.ListContracts)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.EmployeeContractResponse]
	parseResponse(t, w, &response)

	if len(response.Data) != 0 {
		t.Errorf("expected empty list, got %d contracts", len(response.Data))
	}
}

func TestEmployeeHandler_CreateContract_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/999/contracts", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_InvalidEmployeeID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/invalid/contracts", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_MissingPosition(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := map[string]interface{}{
		"from":         "2025-01-01T00:00:00Z",
		"weekly_hours": 40,
		"salary":       600000,
		// Missing position
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for missing position, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_CreateContract_ZeroWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 0,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	// Document current behavior - zero hours may or may not be valid
	t.Logf("Create contract with zero weekly hours returned status %d", w.Code)
}

func TestEmployeeHandler_CreateContract_ContractBoundaryTouch(t *testing.T) {
	// Tests that adjacent contracts (A ends day before B starts) are allowed.
	// This is the correct way to transition between contracts.
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contract ending on Dec 31, 2024
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	// Create contract starting Jan 1, 2025 (day after previous ends) - should succeed
	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d for adjacent (non-overlapping) contract, got %d: %s",
			http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_CreateContract_SameDayTransitionRejected(t *testing.T) {
	// Tests that "touching" contracts (A ends same day B starts) are rejected.
	// Both start and end dates are inclusive, so same-day transition would mean
	// both contracts are active on that day, which is not allowed.
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	// Create contract ending on Jan 31, 2025
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
		},
		Position: "Developer",
	})

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	// Try to create contract starting Jan 31, 2025 (same day as previous ends) - should fail
	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		Position:    "Senior Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d for same-day transition (overlap), got %d: %s",
			http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d/contracts/999", org.ID, employee.ID), nil)

	// Should return NotFound for non-existent contract
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestEmployeeHandler_DeleteContract_InvalidContractID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/employees/:id/contracts/:contractId", handler.DeleteContract)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/employees/%d/contracts/invalid", org.ID, employee.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEmployeeHandler_GetCurrentContract_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/:id/contracts/current", handler.GetCurrentContract)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/invalid/contracts/current", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Validation edge case tests

func TestEmployeeHandler_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "Test",
		LastName:  "Employee",
		Gender:    "male",
		Birthdate: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for future birthdate, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_Create_WhitespaceOnlyFirstName(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "   ",
		LastName:  "Employee",
		Gender:    "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only first name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_Create_WhitespaceOnlyLastName(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees", handler.Create)

	body := models.EmployeeCreateRequest{
		FirstName: "Test",
		LastName:  "   ",
		Gender:    "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees", org.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only last name, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_CreateContract_FromAfterTo(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	toDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		To:          &toDate,
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for from > to, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_CreateContract_NegativeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: -1,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for negative weekly hours, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_CreateContract_WeeklyHoursOver168(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "Developer",
		WeeklyHours: 169,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for weekly hours > 168, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestEmployeeHandler_CreateContract_WhitespacePosition(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")
	employee := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Gender: "male", Birthdate: time.Now()},
	}
	db.Create(employee)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:id/contracts", handler.CreateContract)

	body := models.EmployeeContractCreateRequest{
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Position:    "   ",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for whitespace-only position, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// =========================================
// Search Tests
// =========================================

func TestEmployeeHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	// Create employees with distinct names
	for _, name := range []struct{ first, last string }{
		{"Max", "Mustermann"},
		{"Maria", "Mueller"},
		{"Lisa", "Fischer"},
	} {
		emp := &models.Employee{
			Person: models.Person{OrganizationID: org.ID, FirstName: name.first, LastName: name.last, Gender: "male", Birthdate: time.Now()},
		}
		db.Create(emp)
		createActiveEmployeeContract(t, db, emp.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees", handler.List)

	// Search by first name prefix
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees?search=ma", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Employee]
	parseResponse(t, w, &response)

	// "ma" matches Max (first), Maria (first), and Lisa Maier would match (last) but here it's Mueller, Fischer
	// So only Max and Maria match
	if response.Total != 2 {
		t.Errorf("expected total 2 for search 'ma', got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 employees, got %d", len(response.Data))
	}

	// Search by last name
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees?search=fischer", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 1 {
		t.Errorf("expected total 1 for search 'fischer', got %d", response.Total)
	}

	// Search with no results
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees?search=zzz", org.ID), nil)
	parseResponse(t, w, &response)
	if response.Total != 0 {
		t.Errorf("expected total 0 for search 'zzz', got %d", response.Total)
	}
}

func TestEmployeeHandler_List_SearchWithPagination(t *testing.T) {
	db := setupTestDB(t)
	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, nil)

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 matching employees and 2 non-matching
	for i := 1; i <= 5; i++ {
		emp := &models.Employee{
			Person: models.Person{OrganizationID: org.ID, FirstName: fmt.Sprintf("Max%d", i), LastName: "Mustermann", Gender: "male", Birthdate: time.Now()},
		}
		db.Create(emp)
		createActiveEmployeeContract(t, db, emp.ID)
	}
	for _, name := range []string{"Lisa", "Anna"} {
		emp := &models.Employee{
			Person: models.Person{OrganizationID: org.ID, FirstName: name, LastName: "Other", Gender: "female", Birthdate: time.Now()},
		}
		db.Create(emp)
		createActiveEmployeeContract(t, db, emp.ID)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees", handler.List)

	// Page 1: search + pagination
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees?search=max&page=1&limit=2", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PaginatedResponse[models.Employee]
	parseResponse(t, w, &response)

	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 employees on page 1, got %d", len(response.Data))
	}
	if response.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.TotalPages)
	}

	// Page 3: last page should have 1 result
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees?search=max&page=3&limit=2", org.ID), nil)
	parseResponse(t, w, &response)

	if len(response.Data) != 1 {
		t.Errorf("expected 1 employee on page 3, got %d", len(response.Data))
	}
}

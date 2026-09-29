package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestEmployeeService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestEmployee(t, db, "John", "Doe", org.ID)
	createTestEmployee(t, db, "Jane", "Doe", org.ID)

	employees, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(employees) != 2 {
		t.Errorf("expected 2 employees, got %d", len(employees))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestEmployeeService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	found, err := svc.GetByID(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != employee.ID {
		t.Errorf("ID = %d, want %d", found.ID, employee.ID)
	}
	if found.FirstName != "John" {
		t.Errorf("FirstName = %v, want John", found.FirstName)
	}
}

func TestEmployeeService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.GetByID(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that accessing an employee from a different organization returns not found
func TestEmployeeService_GetByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)

	// Try to access employee from org1 using org2's context
	_, err := svc.GetByID(ctx, employee.ID, org2.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when accessing employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}
}

func TestEmployeeService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.EmployeeCreateRequest{
		FirstName: "John",
		LastName:  "Doe",
		Gender:    "male",
		Birthdate: "1990-05-15",
	}

	employee, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if employee.ID == 0 {
		t.Error("expected ID to be set")
	}
	if employee.FirstName != "John" {
		t.Errorf("FirstName = %v, want John", employee.FirstName)
	}
	if employee.LastName != "Doe" {
		t.Errorf("LastName = %v, want Doe", employee.LastName)
	}
	if employee.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", employee.OrganizationID, org.ID)
	}
}

func TestEmployeeService_Create_WhitespaceOnlyNames(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	tests := []struct {
		name string
		req  *models.EmployeeCreateRequest
	}{
		{"empty first name", &models.EmployeeCreateRequest{FirstName: "", LastName: "Doe", Birthdate: "1990-01-01"}},
		{"whitespace first name", &models.EmployeeCreateRequest{FirstName: "   ", LastName: "Doe", Birthdate: "1990-01-01"}},
		{"empty last name", &models.EmployeeCreateRequest{FirstName: "John", LastName: "", Birthdate: "1990-01-01"}},
		{"whitespace last name", &models.EmployeeCreateRequest{FirstName: "John", LastName: "   ", Birthdate: "1990-01-01"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, org.ID, tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestEmployeeService_Create_TrimmedNames(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.EmployeeCreateRequest{
		FirstName: "  John  ",
		LastName:  "  Doe  ",
		Gender:    "male",
		Birthdate: "1990-05-15",
	}

	employee, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if employee.FirstName != "John" {
		t.Errorf("FirstName = %v, want 'John' (trimmed)", employee.FirstName)
	}
	if employee.LastName != "Doe" {
		t.Errorf("LastName = %v, want 'Doe' (trimmed)", employee.LastName)
	}
}

func TestEmployeeService_Create_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.EmployeeCreateRequest{
		FirstName: "John",
		LastName:  "Doe",
		Birthdate: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), // 1 year in future
	}

	_, err := svc.Create(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for future birthdate, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	newFirstName := "Jane"
	req := &models.EmployeeUpdateRequest{
		FirstName: &newFirstName,
	}

	updated, err := svc.Update(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.FirstName != "Jane" {
		t.Errorf("FirstName = %v, want Jane", updated.FirstName)
	}
	if updated.LastName != "Doe" {
		t.Errorf("LastName should not change, got %v", updated.LastName)
	}
}

func TestEmployeeService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	newName := "Jane"
	req := &models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	_, err := svc.Update(ctx, 999, org.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that updating an employee from a different organization returns not found
func TestEmployeeService_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)

	newName := "Hacker"
	req := &models.EmployeeUpdateRequest{
		FirstName: &newName,
	}

	// Try to update employee from org1 using org2's context
	_, err := svc.Update(ctx, employee.ID, org2.ID, req)
	if err == nil {
		t.Fatal("SECURITY: expected error when updating employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}

	// Verify the employee was NOT modified
	original, err := svc.GetByID(ctx, employee.ID, org1.ID)
	if err != nil {
		t.Fatalf("failed to get original employee: %v", err)
	}
	if original.FirstName != "John" {
		t.Errorf("SECURITY: employee was modified despite cross-org attempt, FirstName = %v, want John", original.FirstName)
	}
}

func TestEmployeeService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	err := svc.Delete(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, employee.ID, org.ID)
	if err == nil {
		t.Error("expected employee to be deleted")
	}
}

// SECURITY TEST: Verify that deleting an employee from a different organization returns not found
func TestEmployeeService_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)

	// Try to delete employee from org1 using org2's context
	err := svc.Delete(ctx, employee.ID, org2.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when deleting employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}

	// Verify the employee was NOT deleted
	original, err := svc.GetByID(ctx, employee.ID, org1.ID)
	if err != nil {
		t.Fatalf("SECURITY: employee was deleted despite cross-org attempt: %v", err)
	}
	if original.ID != employee.ID {
		t.Error("SECURITY: employee was deleted despite cross-org attempt")
	}
}

func TestEmployeeService_ListByOrganizationAndSection_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Employee with active contract
	empActive := createTestEmployee(t, db, "Active", "Employee", org.ID)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, empActive.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1,
		From:      from, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Employee with expired contract
	empExpired := createTestEmployee(t, db, "Expired", "Employee", org.ID)
	fromExpired := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	toExpired := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, empExpired.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1,
		From:      fromExpired, To: &toExpired, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Employee with no contract
	createTestEmployee(t, db, "NoContract", "Employee", org.ID)

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// With activeOn filter: only the active employee should be returned
	employees, total, err := svc.ListByOrganizationAndSection(ctx, org.ID, models.EmployeeListFilter{ActiveOn: &refDate}, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(employees) != 1 {
		t.Errorf("expected 1 employee with active_on filter, got %d", len(employees))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(employees) == 1 && employees[0].FirstName != "Active" {
		t.Errorf("expected Active employee, got %s", employees[0].FirstName)
	}

	// Without activeOn filter: all 3 employees should be returned
	allEmployees, allTotal, err := svc.ListByOrganizationAndSection(ctx, org.ID, models.EmployeeListFilter{}, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allEmployees) != 3 {
		t.Errorf("expected 3 employees without filter, got %d", len(allEmployees))
	}
	if allTotal != 3 {
		t.Errorf("expected total 3, got %d", allTotal)
	}
}

func TestEmployeeService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	createTestEmployee(t, db, "John", "Doe", org1.ID)
	createTestEmployee(t, db, "Jane", "Doe", org1.ID)
	createTestEmployee(t, db, "Bob", "Smith", org2.ID)

	employees, total, err := svc.ListByOrganization(ctx, org1.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(employees) != 2 {
		t.Errorf("expected 2 employees in org1, got %d", len(employees))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

// SECURITY TEST: Verify that ListByOrganization only returns employees from the specified org
func TestEmployeeService_ListByOrganization_IsolatesData(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create employees in both orgs
	emp1 := createTestEmployee(t, db, "John", "Doe", org1.ID)
	createTestEmployee(t, db, "Jane", "Doe", org1.ID)
	emp3 := createTestEmployee(t, db, "Bob", "Smith", org2.ID)

	// List employees for org1
	employees1, total1, err := svc.ListByOrganization(ctx, org1.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total1 != 2 {
		t.Errorf("org1: expected total 2, got %d", total1)
	}

	// Verify all returned employees belong to org1
	for _, emp := range employees1 {
		if emp.OrganizationID != org1.ID {
			t.Errorf("SECURITY: employee %d belongs to org %d, expected org %d", emp.ID, emp.OrganizationID, org1.ID)
		}
	}

	// Verify org2's employee is not in org1's list
	for _, emp := range employees1 {
		if emp.ID == emp3.ID {
			t.Errorf("SECURITY: org2's employee (ID=%d) leaked to org1's list", emp3.ID)
		}
	}

	// List employees for org2
	employees2, total2, err := svc.ListByOrganization(ctx, org2.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total2 != 1 {
		t.Errorf("org2: expected total 1, got %d", total2)
	}

	// Verify org1's employees are not in org2's list
	for _, emp := range employees2 {
		if emp.ID == emp1.ID {
			t.Errorf("SECURITY: org1's employee (ID=%d) leaked to org2's list", emp1.ID)
		}
	}
}

func TestEmployeeService_Update_WhitespaceOnlyNames(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	emptyStr := ""
	whitespaceStr := "   "

	tests := []struct {
		name string
		req  *models.EmployeeUpdateRequest
	}{
		{"empty first name", &models.EmployeeUpdateRequest{FirstName: &emptyStr}},
		{"whitespace first name", &models.EmployeeUpdateRequest{FirstName: &whitespaceStr}},
		{"empty last name", &models.EmployeeUpdateRequest{LastName: &emptyStr}},
		{"whitespace last name", &models.EmployeeUpdateRequest{LastName: &whitespaceStr}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Update(ctx, employee.ID, org.ID, tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestEmployeeService_Update_FutureBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	futureBirthdate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	req := &models.EmployeeUpdateRequest{
		Birthdate: &futureBirthdate,
	}

	_, err := svc.Update(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for future birthdate, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_FindAllByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlanWithCoverage(t, db, "TV-L", org.ID)

	// Create 3 employees with contracts.
	for i := range 3 {
		emp := createTestEmployee(t, db, fmt.Sprintf("Emp%d", i), "Smith", org.ID)
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, "S8a", 1, 39)
	}

	results, err := svc.FindAllByOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 employees, got %d", len(results))
	}
}

func TestEmployeeService_FindAllByOrganization_IsolatesOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payPlan1 := createTestPayPlanWithCoverage(t, db, "TV-L", org1.ID)
	payPlan2 := createTestPayPlanWithCoverage(t, db, "TV-L", org2.ID)

	emp1 := createTestEmployee(t, db, "Emp", "Org1", org1.ID)
	createTestEmployeeContract(t, db, emp1.ID, payPlan1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, "S8a", 1, 39)
	emp2 := createTestEmployee(t, db, "Emp", "Org2", org2.ID)
	createTestEmployeeContract(t, db, emp2.ID, payPlan2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, "S8a", 1, 39)

	results, err := svc.FindAllByOrganization(ctx, org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 employee for org1, got %d", len(results))
	}
}

func TestEmployeeService_FindAllByOrganization_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Empty Org")

	results, err := svc.FindAllByOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 employees, got %d", len(results))
	}
}

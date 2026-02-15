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

func TestEmployeeService_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            &to,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.ID == 0 {
		t.Error("expected ID to be set")
	}
	if contract.EmployeeID != employee.ID {
		t.Errorf("EmployeeID = %d, want %d", contract.EmployeeID, employee.ID)
	}
	if contract.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", contract.StaffCategory)
	}
	if contract.WeeklyHours != 40 {
		t.Errorf("WeeklyHours = %v, want 40", contract.WeeklyHours)
	}
	if contract.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", contract.PayPlanID, payPlan.ID)
	}
}

func TestEmployeeService_CreateContract_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	_, err := svc.CreateContract(ctx, 999, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent employee, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that creating a contract for an employee from a different organization returns not found
func TestEmployeeService_CreateContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org2.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "supplementary",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	// Try to create contract for employee from org1 using org2's context
	_, err := svc.CreateContract(ctx, employee.ID, org2.ID, req)
	if err == nil {
		t.Fatal("SECURITY: expected error when creating contract for employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}

	// Verify no contract was created
	contracts, _, err := svc.ListContracts(ctx, employee.ID, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 0 {
		t.Errorf("SECURITY: contract was created despite cross-org attempt, got %d contracts", len(contracts))
	}
}

func TestEmployeeService_CreateContract_EmptyStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for empty staff category, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "invalid_category",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Before from

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            &to,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for invalid period (to before from), got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_OverlappingContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create first contract
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req1)
	if err != nil {
		t.Fatalf("first contract: expected no error, got %v", err)
	}

	// Try to create overlapping contract
	from2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Overlaps with first
	to2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from2,
		To:            &to2,
		StaffCategory: "qualified",
		WeeklyHours:   35,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	_, err = svc.CreateContract(ctx, employee.ID, org.ID, req2)
	if err == nil {
		t.Fatal("expected error for overlapping contract, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestEmployeeService_CreateContract_OngoingContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// No 'to' date means ongoing contract
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            nil,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.To != nil {
		t.Errorf("To = %v, want nil (ongoing)", contract.To)
	}
}

func TestEmployeeService_CreateContract_ValidStaffCategories(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	validCategories := []string{"qualified", "supplementary", "non_pedagogical"}

	for i, cat := range validCategories {
		employee := createTestEmployee(t, db, "John", fmt.Sprintf("Doe%d", i), org.ID)

		from := time.Date(2024+i, 1, 1, 0, 0, 0, 0, time.UTC)
		req := &models.EmployeeContractCreateRequest{
			SectionID:     1,
			From:          from,
			StaffCategory: cat,
			WeeklyHours:   40,
			Grade:         "S8a", Step: 3,
			PayPlanID: payPlan.ID,
		}

		contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
		if err != nil {
			t.Fatalf("expected no error for staff category %q, got %v", cat, err)
		}
		if contract.StaffCategory != cat {
			t.Errorf("StaffCategory = %v, want %v", contract.StaffCategory, cat)
		}
	}
}

func TestEmployeeService_CreateContract_SectionNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     99999,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err == nil {
		t.Fatal("expected error for non-existent section, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_SectionFromWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org1.ID)

	// Get org2's default section
	var org2Section models.Section
	db.Where("organization_id = ?", org2.ID).First(&org2Section)

	_, err := svc.CreateContract(ctx, employee.ID, org1.ID, &models.EmployeeContractCreateRequest{
		SectionID:     org2Section.ID,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err == nil {
		t.Fatal("expected error for section from wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create two contracts
	from1 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{SectionID: 1, From: from1, To: &to1, StaffCategory: "supplementary", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, _ = svc.CreateContract(ctx, employee.ID, org.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreateRequest{SectionID: 1, From: from2, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, _ = svc.CreateContract(ctx, employee.ID, org.ID, req2)

	contracts, _, err := svc.ListContracts(ctx, employee.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("expected 2 contracts, got %d", len(contracts))
	}
}

func TestEmployeeService_ListContracts_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, _, err := svc.ListContracts(ctx, 999, org.ID, 100, 0)
	if err == nil {
		t.Fatal("expected error for non-existent employee, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that listing contracts from a different organization returns not found
func TestEmployeeService_ListContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org1.ID)

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, _ = svc.CreateContract(ctx, employee.ID, org1.ID, req)

	// Try to list contracts for employee from org1 using org2's context
	_, _, err := svc.ListContracts(ctx, employee.ID, org2.ID, 100, 0)
	if err == nil {
		t.Fatal("SECURITY: expected error when listing contracts for employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}
}

func TestEmployeeService_ListByOrganizationAndSection_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Employee with active contract
	empActive := createTestEmployee(t, db, "Active", "Employee", org.ID)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, empActive.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1,
		From:      from, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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
		From:      fromExpired, To: &toExpired, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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

// SECURITY TEST: Verify GetCurrentContract returns not found for wrong org
func TestEmployeeService_GetCurrentContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org1.ID)

	// Create an active (ongoing) contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, _ = svc.CreateContract(ctx, employee.ID, org1.ID, req)

	// Try to get current contract for employee from org1 using org2's context
	_, err := svc.GetCurrentContract(ctx, employee.ID, org2.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when getting current contract for employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}
}

// SECURITY TEST: Verify DeleteContract returns not found for wrong org
func TestEmployeeService_DeleteContract_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org1.ID)

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to delete contract for employee from org1 using org2's context
	err = svc.DeleteContract(ctx, contract.ID, employee.ID, org2.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when deleting contract for employee from wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}

	// Verify the contract was NOT deleted
	contracts, _, err := svc.ListContracts(ctx, employee.ID, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Errorf("SECURITY: contract was deleted despite cross-org attempt, got %d contracts", len(contracts))
	}
}

func TestEmployeeService_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Delete the contract
	err = svc.DeleteContract(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	contracts, _, err := svc.ListContracts(ctx, employee.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 0 {
		t.Errorf("expected 0 contracts after deletion, got %d", len(contracts))
	}
}

func TestEmployeeService_DeleteContract_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	err := svc.DeleteContract(ctx, 999, employee.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent contract, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that a contract belonging to another employee cannot be deleted
func TestEmployeeService_DeleteContract_WrongEmployee(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee1 := createTestEmployee(t, db, "John", "Doe", org.ID)
	employee2 := createTestEmployee(t, db, "Jane", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a contract for employee1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee1.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to delete employee1's contract using employee2's ID
	err = svc.DeleteContract(ctx, contract.ID, employee2.ID, org.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when deleting contract with wrong employee ID, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}

	// Verify the contract was NOT deleted
	contracts, _, err := svc.ListContracts(ctx, employee1.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Errorf("SECURITY: contract was deleted despite wrong employee ID, got %d contracts", len(contracts))
	}
}

func TestEmployeeService_GetCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create an ongoing contract (no end date)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	created, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Get current contract
	current, err := svc.GetCurrentContract(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current.ID != created.ID {
		t.Errorf("ID = %d, want %d", current.ID, created.ID)
	}
	if current.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", current.StaffCategory)
	}
}

func TestEmployeeService_GetCurrentContract_NoActiveContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create an expired contract
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Get current contract (should fail - all contracts expired)
	_, err = svc.GetCurrentContract(ctx, employee.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for no active contract, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
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

func TestEmployeeService_UpdateContract_StaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newCategory := "supplementary"
	updateReq := &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.StaffCategory != "supplementary" {
		t.Errorf("StaffCategory = %v, want supplementary", updated.StaffCategory)
	}
}

func TestEmployeeService_UpdateContract_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	invalidCategory := "invalid_category"
	updateReq := &models.EmployeeContractUpdateRequest{
		StaffCategory: &invalidCategory,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// =====================================================================
// PayPlan ID validation tests
// =====================================================================

func TestEmployeeService_CreateContract_WithPayPlanID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE 2024", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", contract.PayPlanID, payPlan.ID)
	}
}

func TestEmployeeService_CreateContract_PayPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: 99999, // Non-existent pay plan
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent payplan_id, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// SECURITY TEST: Verify that using a pay plan from a different organization is rejected
func TestEmployeeService_CreateContract_PayPlanWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlanOrg2 := createTestPayPlan(t, db, "TVoD-SuE Org2", org2.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlanOrg2.ID, // Pay plan belongs to org2, employee belongs to org1
	}

	_, err := svc.CreateContract(ctx, employee.ID, org1.ID, req)
	if err == nil {
		t.Fatal("SECURITY: expected error when using pay plan from different org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Verify no contract was created
	contracts, _, err := svc.ListContracts(ctx, employee.ID, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 0 {
		t.Errorf("SECURITY: contract was created with wrong org's pay plan, got %d contracts", len(contracts))
	}
}

func TestEmployeeService_UpdateContract_PayPlanID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan1 := createTestPayPlan(t, db, "TVoD-SuE 2023", org.ID)
	payPlan2 := createTestPayPlan(t, db, "TVoD-SuE 2024", org.ID)

	// Create contract with payPlan1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan1.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	if contract.PayPlanID != payPlan1.ID {
		t.Fatalf("PayPlanID = %d, want %d", contract.PayPlanID, payPlan1.ID)
	}

	// Update to payPlan2
	newPayPlanID := payPlan2.ID
	updateReq := &models.EmployeeContractUpdateRequest{
		PayPlanID: &newPayPlanID,
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.PayPlanID != payPlan2.ID {
		t.Errorf("PayPlanID = %d, want %d", updated.PayPlanID, payPlan2.ID)
	}
}

func TestEmployeeService_UpdateContract_PayPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update to a non-existent pay plan
	nonExistentID := uint(99999)
	updateReq := &models.EmployeeContractUpdateRequest{
		PayPlanID: &nonExistentID,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
	if err == nil {
		t.Fatal("expected error for non-existent payplan_id, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Verify the contract was NOT updated
	fetched, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get contract: %v", err)
	}
	if fetched.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID should not have changed, got %d, want %d", fetched.PayPlanID, payPlan.ID)
	}
}

// SECURITY TEST: Verify that updating to a pay plan from a different organization is rejected
func TestEmployeeService_UpdateContract_PayPlanWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan1 := createTestPayPlan(t, db, "TVoD-SuE Org1", org1.ID)
	payPlanOrg2 := createTestPayPlan(t, db, "TVoD-SuE Org2", org2.ID)

	// Create contract with org1's pay plan
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan1.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update to org2's pay plan
	wrongOrgPayPlanID := payPlanOrg2.ID
	updateReq := &models.EmployeeContractUpdateRequest{
		PayPlanID: &wrongOrgPayPlanID,
	}

	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org1.ID, updateReq)
	if err == nil {
		t.Fatal("SECURITY: expected error when updating to pay plan from different org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Verify the contract was NOT updated
	fetched, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org1.ID)
	if err != nil {
		t.Fatalf("failed to get contract: %v", err)
	}
	if fetched.PayPlanID != payPlan1.ID {
		t.Errorf("SECURITY: PayPlanID was changed despite wrong org, got %d, want %d", fetched.PayPlanID, payPlan1.ID)
	}
}

func TestEmployeeService_CreateContract_PayPlanIDResponse(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE 2024", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify response includes payplan_id
	if contract.PayPlanID != payPlan.ID {
		t.Errorf("response PayPlanID = %d, want %d", contract.PayPlanID, payPlan.ID)
	}

	// Verify it's also in the list response
	contracts, _, err := svc.ListContracts(ctx, employee.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list contracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].PayPlanID != payPlan.ID {
		t.Errorf("list response PayPlanID = %d, want %d", contracts[0].PayPlanID, payPlan.ID)
	}

	// Verify it's in the GetContractByID response
	fetched, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get contract: %v", err)
	}
	if fetched.PayPlanID != payPlan.ID {
		t.Errorf("get response PayPlanID = %d, want %d", fetched.PayPlanID, payPlan.ID)
	}

	// Verify it's in the GetCurrentContract response
	current, err := svc.GetCurrentContract(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get current contract: %v", err)
	}
	if current.PayPlanID != payPlan.ID {
		t.Errorf("current contract PayPlanID = %d, want %d", current.PayPlanID, payPlan.ID)
	}
}

func TestEmployeeService_UpdateContract_PayPlanIDNotChangedWhenOmitted(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update without providing PayPlanID (should keep original)
	newCategory := "supplementary"
	updateReq := &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID should not change when omitted, got %d, want %d", updated.PayPlanID, payPlan.ID)
	}
	if updated.StaffCategory != "supplementary" {
		t.Errorf("StaffCategory = %v, want supplementary", updated.StaffCategory)
	}
}

func TestEmployeeService_CreateContract_PayPlanIDZero(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: 0, // Zero value (not set)
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for zero payplan_id, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// =========================================
// Amend-specific UpdateContract Tests
// =========================================

func TestEmployeeService_UpdateContract_AmendChangeSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section1 := createTestSection(t, db, "Krippe", org.ID, false)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section1.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		SectionID: &section2.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// New contract created
	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend creates new contract)")
	}
	if !updated.From.Truncate(24 * time.Hour).Equal(today) {
		t.Errorf("new contract From = %v, want %v", updated.From, today)
	}
	if updated.SectionID != section2.ID {
		t.Errorf("SectionID = %d, want %d", updated.SectionID, section2.ID)
	}
	// Employee-specific fields carried over
	if updated.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", updated.StaffCategory)
	}
	if updated.Grade != "S8a" {
		t.Errorf("Grade = %v, want S8a", updated.Grade)
	}
	if updated.Step != 3 {
		t.Errorf("Step = %d, want 3", updated.Step)
	}
	if updated.WeeklyHours != 39 {
		t.Errorf("WeeklyHours = %v, want 39", updated.WeeklyHours)
	}
	if updated.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", updated.PayPlanID, payPlan.ID)
	}
	if updated.EmployeeID != employee.ID {
		t.Errorf("EmployeeID = %d, want %d", updated.EmployeeID, employee.ID)
	}

	// Verify old contract was closed
	old, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get old contract: %v", err)
	}
	if old.To == nil {
		t.Fatal("old contract To should not be nil after amend")
	}
	if !old.To.Truncate(24 * time.Hour).Equal(yesterday) {
		t.Errorf("old contract To = %v, want %v", old.To, yesterday)
	}
}

func TestEmployeeService_UpdateContract_AmendChangeStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newCategory := "supplementary"
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.StaffCategory != "supplementary" {
		t.Errorf("StaffCategory = %v, want supplementary", updated.StaffCategory)
	}
}

func TestEmployeeService_UpdateContract_AmendChangePayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan1 := createTestPayPlan(t, db, "TVoD-SuE 2023", org.ID)
	payPlan2 := createTestPayPlan(t, db, "TVoD-SuE 2024", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan1.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		PayPlanID: &payPlan2.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.PayPlanID != payPlan2.ID {
		t.Errorf("PayPlanID = %d, want %d", updated.PayPlanID, payPlan2.ID)
	}
}

func TestEmployeeService_UpdateContract_AmendChangeGradeStep(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S11b"
	newStep := 5
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		Grade: &newGrade,
		Step:  &newStep,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.Grade != "S11b" {
		t.Errorf("Grade = %v, want S11b", updated.Grade)
	}
	if updated.Step != 5 {
		t.Errorf("Step = %d, want 5", updated.Step)
	}
}

func TestEmployeeService_UpdateContract_AmendChangeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newHours := 30.0
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		WeeklyHours: &newHours,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.WeeklyHours != 30.0 {
		t.Errorf("WeeklyHours = %v, want 30", updated.WeeklyHours)
	}
}

func TestEmployeeService_UpdateContract_AmendAllFieldsCarryOver(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID:  payPlan.ID,
		Properties: models.ContractProperties{"benefit": "bonus"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update with no fields changed (empty update) → still creates new contract via amend
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID == contract.ID {
		t.Error("expected new contract ID (amend)")
	}
	if updated.SectionID != section.ID {
		t.Errorf("SectionID = %d, want %d", updated.SectionID, section.ID)
	}
	if updated.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", updated.StaffCategory)
	}
	if updated.WeeklyHours != 39 {
		t.Errorf("WeeklyHours = %v, want 39", updated.WeeklyHours)
	}
	if updated.Grade != "S8a" {
		t.Errorf("Grade = %v, want S8a", updated.Grade)
	}
	if updated.Step != 3 {
		t.Errorf("Step = %d, want 3", updated.Step)
	}
	if updated.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", updated.PayPlanID, payPlan.ID)
	}
	if updated.Properties["benefit"] != "bonus" {
		t.Errorf("Properties should carry over, got %v", updated.Properties)
	}
}

func TestEmployeeService_UpdateContract_EndedContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	past := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	pastEnd := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          past,
		To:            &pastEnd,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newCategory := "supplementary"
	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	})
	if err == nil {
		t.Fatal("expected error for ended contract, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_AmendFromIgnored(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	requestedFrom := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		From: &requestedFrom,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if !updated.From.Truncate(24 * time.Hour).Equal(today) {
		t.Errorf("From = %v, want today (%v) — From should be ignored in amend mode", updated.From, today)
	}
}

func TestEmployeeService_UpdateContract_AmendOverlapConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	past := today.AddDate(0, -3, 0)

	// Create ongoing contract starting in the past (qualifies for amend)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create first contract: %v", err)
	}

	// Insert a blocking contract directly in DB (bypass overlap validation)
	futureEnd := today.AddDate(1, 0, 0)
	blockingContract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: today, To: &futureEnd},
			SectionID: section.ID,
		},
		StaffCategory: "qualified",
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	}
	if err := db.Create(blockingContract).Error; err != nil {
		t.Fatalf("failed to insert blocking contract: %v", err)
	}

	// Try to amend first contract:
	// Amend closes old (To=yesterday), creates new (from=today, ongoing) → overlaps with blocking
	newCategory := "supplementary"
	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	})
	if err == nil {
		t.Fatal("expected overlap conflict error, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_InPlace_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	tomorrow := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          tomorrow,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := tomorrow.AddDate(0, 0, 7)
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		From: &newFrom,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.ID != contract.ID {
		t.Errorf("ID = %d, want %d (should be in-place update)", updated.ID, contract.ID)
	}
	if !updated.From.Equal(newFrom) {
		t.Errorf("From = %v, want %v", updated.From, newFrom)
	}
}

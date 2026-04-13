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

// SECURITY TEST: Verify GetCurrentRecord returns not found for wrong org
func TestEmployeeService_GetCurrentRecord_WrongOrg(t *testing.T) {
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
	_, err := svc.GetCurrentRecord(ctx, employee.ID, org2.ID)
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

func TestEmployeeService_GetCurrentRecord(t *testing.T) {
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
	current, err := svc.GetCurrentRecord(ctx, employee.ID, org.ID)
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

func TestEmployeeService_GetCurrentRecord_NoActiveContract(t *testing.T) {
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
	_, err = svc.GetCurrentRecord(ctx, employee.ID, org.ID)
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

	// Verify it's in the GetCurrentRecord response
	current, err := svc.GetCurrentRecord(ctx, employee.ID, org.ID)
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

// =========================================
// Edge Case Tests: Amend Field Preservation
// =========================================

// After amend: verify state consistency (old closed, new active, list shows both)
func TestEmployeeService_UpdateContract_AmendStateConsistency(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVÖD", org.ID)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change staff category
	newCategory := "supplementary"
	newContract, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		StaffCategory: &newCategory,
	})
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}

	// List should show 2 contracts
	contracts, total, err := svc.ListContracts(ctx, employee.ID, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("ListContracts failed: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts after amend, got %d", len(contracts))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	// Old contract should be closed
	oldContract, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get old contract: %v", err)
	}
	if oldContract.To == nil {
		t.Fatal("old contract To should not be nil")
	}
	if !oldContract.To.Truncate(24 * time.Hour).Equal(yesterday) {
		t.Errorf("old contract To = %v, want %v", oldContract.To, yesterday)
	}

	// GetCurrentRecord should return the new contract
	current, err := svc.GetCurrentRecord(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("GetCurrentRecord failed: %v", err)
	}
	if current.ID != newContract.ID {
		t.Errorf("GetCurrentRecord returned ID %d, want %d", current.ID, newContract.ID)
	}
	if current.StaffCategory != "supplementary" {
		t.Errorf("current contract StaffCategory = %s, want supplementary", current.StaffCategory)
	}
}

// Amend on ongoing contract: new contract should also have nil To
func TestEmployeeService_UpdateContract_AmendPreservesOngoingTo(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVÖD", org.ID)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
		// No To — ongoing
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newHours := float64(30)
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		WeeklyHours: &newHours,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.To != nil {
		t.Errorf("new contract To should be nil (ongoing), got %v", updated.To)
	}
}

// Amend on contract with specific To: To carries over when not in request
func TestEmployeeService_UpdateContract_AmendPreservesToWhenNotInRequest(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)
	payPlan := createTestPayPlan(t, db, "TVÖD", org.ID)

	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -3, 0)
	endDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 6, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		To:            &endDate,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S9"
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		Grade: &newGrade,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.To == nil {
		t.Fatal("new contract To should not be nil")
	}
	if !updated.To.Truncate(24 * time.Hour).Equal(endDate) {
		t.Errorf("To = %v, want %v (carried over from original)", updated.To, endDate)
	}
}

// =========================================
// Edge Case Tests: Contract Creation Boundaries
// =========================================

// Adjacent contracts (touching, not overlapping) should succeed
func TestEmployeeService_CreateContract_AdjacentContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVÖD", org.ID)

	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("first contract: %v", err)
	}

	// Jul 1 (day after Jun 30) — should succeed
	from2 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from2,
		To:            &to2,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("adjacent contract should succeed, got: %v", err)
	}
}

// Overlapping on single day (inclusive boundaries) should fail
func TestEmployeeService_CreateContract_OverlapOnSameDay(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVÖD", org.ID)

	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("first contract: %v", err)
	}

	// Starts on Jun 30 — same day as contract 1 ends — should fail
	from2 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from2,
		To:            &to2,
		StaffCategory: "qualified",
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err == nil {
		t.Fatal("expected overlap error for same-day boundary, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// Delete non-existent contract
func TestEmployeeService_DeleteContract_NotFoundByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	err := svc.DeleteContract(ctx, 99999, employee.ID, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent contract, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =========================================
// Nullable field clearing tests (employee contracts)
// =========================================

func TestEmployeeService_UpdateContract_ClearNullableTo(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "Test Pay Plan", org.ID)
	section := getDefaultSection(t, db, org.ID)

	// Create contract with To set (use future date to trigger in-place update)
	from := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2050, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          from,
		To:            &to,
		SectionID:     section.ID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if contract.To == nil {
		t.Fatal("setup: To should be set")
	}

	// Clear To by sending nil (simulates frontend sending null to make open-ended)
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		To: nil,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.To != nil {
		t.Errorf("To should be nil after clearing, got %v", updated.To)
	}

	// Verify persistence
	refetched, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.To != nil {
		t.Errorf("To should be nil after re-fetch, got %v", refetched.To)
	}
}

func TestEmployeeService_UpdateContract_ClearNullableProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "Test Pay Plan", org.ID)
	section := getDefaultSection(t, db, org.ID)

	// Create contract with Properties set (use future date to trigger in-place update)
	from := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          from,
		SectionID:     section.ID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     payPlan.ID,
		Properties:    models.ContractProperties{"role": "deputy"},
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if contract.Properties == nil {
		t.Fatal("setup: Properties should be set")
	}

	// Clear Properties by sending nil
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		Properties: nil,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Properties != nil {
		t.Errorf("Properties should be nil after clearing, got %v", updated.Properties)
	}

	// Verify persistence
	refetched, err := svc.GetContractByID(ctx, contract.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.Properties != nil {
		t.Errorf("Properties should be nil after re-fetch, got %v", refetched.Properties)
	}
}

func TestEmployeeService_Import_NewEmployee(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "Mustermann",
				Gender:    "male",
				Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						To:            &to,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FirstName != "Max" {
		t.Errorf("FirstName = %v, want Max", results[0].FirstName)
	}
	if len(results[0].Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(results[0].Contracts))
	}
	c := results[0].Contracts[0]
	if c.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", c.StaffCategory)
	}
	if c.WeeklyHours != 39 {
		t.Errorf("WeeklyHours = %v, want 39", c.WeeklyHours)
	}
	if c.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", c.PayPlanID, payPlan.ID)
	}
}

func TestEmployeeService_Import_UpsertReplacesContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name
	birthdate := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)

	baseContract := models.EmployeeContractResponse{
		SectionName:   &sectionName,
		PayPlanName:   &payPlanName,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39,
	}

	// First import: one contract.
	c1 := baseContract
	c1.From = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lisa",
				LastName:  "Schulz",
				Gender:    "female",
				Birthdate: birthdate,
				Contracts: []models.EmployeeContractResponse{c1},
			},
		},
	}
	results1, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if len(results1[0].Contracts) != 1 {
		t.Fatalf("first import: expected 1 contract, got %d", len(results1[0].Contracts))
	}
	employeeID := results1[0].ID

	// Second import: two contracts replace the original.
	c2a := baseContract
	c2a.From = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to2a := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	c2a.To = &to2a

	c2b := baseContract
	c2b.From = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	data2 := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lisa",
				LastName:  "Schulz",
				Gender:    "female",
				Birthdate: birthdate,
				Contracts: []models.EmployeeContractResponse{c2a, c2b},
			},
		},
	}
	results2, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if results2[0].ID != employeeID {
		t.Errorf("expected same employee ID %d, got %d", employeeID, results2[0].ID)
	}
	if len(results2[0].Contracts) != 2 {
		t.Errorf("second import: expected 2 contracts, got %d", len(results2[0].Contracts))
	}
}

func TestEmployeeService_Import_SectionAutoCreation(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	payPlanName := payPlan.Name
	newSectionName := "Elementar"

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Tom",
				LastName:  "Klein",
				Gender:    "male",
				Birthdate: time.Date(1988, 7, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &newSectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify section was auto-created.
	var section models.Section
	if err := db.Where("organization_id = ? AND name = ?", org.ID, newSectionName).First(&section).Error; err != nil {
		t.Fatalf("auto-created section not found: %v", err)
	}
	if results[0].Contracts[0].SectionID != section.ID {
		t.Errorf("contract SectionID = %d, want %d", results[0].Contracts[0].SectionID, section.ID)
	}
}

func TestEmployeeService_Import_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Import(ctx, org.ID, &models.EmployeeImportExportData{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingName(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "",
				Gender:    "male",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "Mustermann",
				Gender:    "male",
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing birthdate, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingContractFrom(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	payPlanName := payPlan.Name
	sectionName := section.Name

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Jan",
				LastName:  "Bauer",
				Gender:    "male",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing contract From, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	sectionName := section.Name
	missingPlan := "NonExistent Plan"

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Kim",
				LastName:  "Lehmann",
				Gender:    "female",
				Birthdate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &missingPlan,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing pay plan, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lea",
				LastName:  "Richter",
				Gender:    "female",
				Birthdate: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "invalid_category",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_NegativeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Nico",
				LastName:  "Wolf",
				Gender:    "male",
				Birthdate: time.Date(1993, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   -5,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for negative weekly hours, got nil")
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
	payPlan := createTestPayPlan(t, db, "TV-L", org.ID)

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
	payPlan1 := createTestPayPlan(t, db, "TV-L", org1.ID)
	payPlan2 := createTestPayPlan(t, db, "TV-L", org2.ID)

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

func TestEmployeeService_Import_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // To before From

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Ole",
				LastName:  "Krause",
				Gender:    "male",
				Birthdate: time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						To:            &to,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for invalid period (to before from), got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_GetContractByID_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	created, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	found, err := svc.GetContractByID(ctx, created.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("ID = %d, want %d", found.ID, created.ID)
	}
	if found.Grade != "S8a" {
		t.Errorf("Grade = %v, want S8a", found.Grade)
	}
	if found.WeeklyHours != 40 {
		t.Errorf("WeeklyHours = %v, want 40", found.WeeklyHours)
	}
}

// SECURITY TEST: Verify that accessing a contract belonging to a different employee returns not found
func TestEmployeeService_GetContractByID_WrongEmployee(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee1 := createTestEmployee(t, db, "John", "Doe", org.ID)
	employee2 := createTestEmployee(t, db, "Jane", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee1.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to access employee1's contract using employee2's ID
	_, err = svc.GetContractByID(ctx, contract.ID, employee2.ID, org.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when accessing contract with wrong employee ID, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound, got %v", err)
	}
}

// SECURITY TEST: Verify that accessing a contract for an employee in a different org returns not found
func TestEmployeeService_GetContractByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org1.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to access contract using org2's context
	_, err = svc.GetContractByID(ctx, contract.ID, employee.ID, org2.ID)
	if err == nil {
		t.Fatal("SECURITY: expected error when accessing contract from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("SECURITY: expected ErrNotFound (not ErrForbidden to prevent info leak), got %v", err)
	}
}

func TestEmployeeService_ListContracts_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create 5 non-overlapping contracts
	for i := range 5 {
		from := time.Date(2020+i, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2020+i, 12, 31, 0, 0, 0, 0, time.UTC)
		createTestEmployeeContract(t, db, employee.ID, payPlan.ID, from, &to, "S8a", 3, 40.0)
	}

	// Request page with limit=2
	contracts, total, err := svc.ListContracts(ctx, employee.ID, org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(contracts) != 2 {
		t.Errorf("expected 2 results on first page, got %d", len(contracts))
	}
}

func TestEmployeeService_CreateContract_WeeklyHoursNegative(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: -1, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for negative weekly hours, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_GradeTrimmed(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "  S8a  ", Step: 3, PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.Grade != "S8a" {
		t.Errorf("Grade = %q, want %q (trimmed)", contract.Grade, "S8a")
	}
}

func TestEmployeeService_UpdateContract_InPlace_ChangeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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

	if updated.WeeklyHours != 30 {
		t.Errorf("WeeklyHours = %v, want 30", updated.WeeklyHours)
	}
	// Verify the same contract ID was updated (in-place, not amend)
	if updated.ID != contract.ID {
		t.Errorf("expected in-place update (same ID %d), got new ID %d", contract.ID, updated.ID)
	}
}

func TestEmployeeService_UpdateContract_InPlace_ChangeGradeStep(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S8b"
	newStep := 5
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		Grade: &newGrade,
		Step:  &newStep,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Grade != "S8b" {
		t.Errorf("Grade = %v, want S8b", updated.Grade)
	}
	if updated.Step != 5 {
		t.Errorf("Step = %v, want 5", updated.Step)
	}
	if updated.ID != contract.ID {
		t.Errorf("expected in-place update (same ID %d), got new ID %d", contract.ID, updated.ID)
	}
}

func TestEmployeeService_UpdateContract_InPlace_InvalidWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	negativeHours := -5.0
	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		WeeklyHours: &negativeHours,
	})
	if err == nil {
		t.Fatal("expected error for negative weekly hours, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_InPlace_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	invalidCategory := "invalid"
	_, err = svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		StaffCategory: &invalidCategory,
	})
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// In-place update with nil To clears the end date (makes contract ongoing).
// This is by design: nullable fields are always assigned so the frontend can clear them by sending null.
func TestEmployeeService_UpdateContract_InPlace_NilToClearsEndDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract with an explicit end date
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 6, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}
	if contract.To == nil {
		t.Fatal("expected contract to have an end date")
	}

	// Update WeeklyHours with nil To — should clear the end date
	newHours := 35.0
	updated, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		WeeklyHours: &newHours,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// To should be nil (cleared) because nil To = "make open-ended"
	if updated.To != nil {
		t.Errorf("To should be nil (cleared), got %v", *updated.To)
	}
}

func TestEmployeeService_UpdateContract_AmendGradeTrimmed(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	// Create a past-dated ongoing contract (amend mode)
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: 40, Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	gradeWithSpaces := "  S8a  "
	amended, err := svc.UpdateContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractUpdateRequest{
		Grade: &gradeWithSpaces,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Amend creates a new contract
	if amended.ID == contract.ID {
		t.Errorf("expected amend to create new contract, but got same ID %d", amended.ID)
	}
	if amended.Grade != "S8a" {
		t.Errorf("Grade = %q, want %q (trimmed)", amended.Grade, "S8a")
	}
}

// --- Batch Update Contract Tests ---

func TestEmployeeService_BatchUpdateContracts_ShiftBoundary(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	aTo := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	contractA := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &aTo, "S8a", 3, 39)

	bTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contractB := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &bTo, "S8a", 3, 39)

	// Shift boundary earlier
	newATo := time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC)
	newBFrom := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	results, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contractA.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: &newATo}},
			{ID: contractB.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{From: &newBFrom, To: &bTo}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].To.Equal(newATo) {
		t.Errorf("contract A To = %v, want %v", results[0].To, newATo)
	}
	if !results[1].From.Equal(newBFrom) {
		t.Errorf("contract B From = %v, want %v", results[1].From, newBFrom)
	}
}

func TestEmployeeService_BatchUpdateContracts_AllFields(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section2 := createTestSection(t, db, "Section2", org.ID, false)
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	payplan2 := createTestPayPlan(t, db, "TestPP2", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, "S8a", 3, 39)

	newFrom := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	newTo := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	newCategory := "supplementary"
	newGrade := "S4"
	newStep := 5
	newHours := 30.0

	results, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{
				From:          &newFrom,
				To:            &newTo,
				SectionID:     &section2.ID,
				StaffCategory: &newCategory,
				Grade:         &newGrade,
				Step:          &newStep,
				WeeklyHours:   &newHours,
				PayPlanID:     &payplan2.ID,
			}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	r := results[0]
	if !r.From.Equal(newFrom) {
		t.Errorf("From = %v, want %v", r.From, newFrom)
	}
	if r.SectionID != section2.ID {
		t.Errorf("SectionID = %d, want %d", r.SectionID, section2.ID)
	}
	if r.StaffCategory != newCategory {
		t.Errorf("StaffCategory = %q, want %q", r.StaffCategory, newCategory)
	}
	if r.Grade != newGrade {
		t.Errorf("Grade = %q, want %q", r.Grade, newGrade)
	}
	if r.Step != newStep {
		t.Errorf("Step = %d, want %d", r.Step, newStep)
	}
	if r.WeeklyHours != newHours {
		t.Errorf("WeeklyHours = %f, want %f", r.WeeklyHours, newHours)
	}
	if r.PayPlanID != payplan2.ID {
		t.Errorf("PayPlanID = %d, want %d", r.PayPlanID, payplan2.ID)
	}
}

func TestEmployeeService_BatchUpdateContracts_PastContracts_NoAmendMode(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	aTo := time.Date(2020, 6, 30, 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), &aTo, "S8a", 3, 39)

	newTo := time.Date(2020, 8, 31, 0, 0, 0, 0, time.UTC)
	results, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: &newTo}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error for past contract batch update, got %v", err)
	}
	if !results[0].To.Equal(newTo) {
		t.Errorf("To = %v, want %v", results[0].To, newTo)
	}
}

func TestEmployeeService_BatchUpdateContracts_DuplicateIDs(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: 1, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
			{ID: 1, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate IDs, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_ContractNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: 9999, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_WrongEmployee(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	emp1 := createTestEmployee(t, db, "John", "Doe", org.ID)
	emp2 := createTestEmployee(t, db, "Jane", "Doe", org.ID)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, emp1.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, "S8a", 3, 39)

	_, err := svc.BatchUpdateContracts(ctx, emp2.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
		},
	})
	if err == nil {
		t.Fatal("expected error for wrong employee, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)

	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org2.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: 1, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
		},
	})
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_InvalidPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	payplan1 := createTestPayPlan(t, db, "PP1", org1.ID)
	payplan2 := createTestPayPlan(t, db, "PP2", org2.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, employee.ID, payplan1.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, "S8a", 3, 39)

	// Try to use pay plan from different org
	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org1.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{PayPlanID: &payplan2.ID, To: &to}},
		},
	})
	if err == nil {
		t.Fatal("expected error for wrong-org pay plan, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &to, "S8a", 3, 39)

	badCategory := "invalid_category"
	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{StaffCategory: &badCategory, To: &to}},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_OverlapConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	aTo := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	contractA := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &aTo, "S8a", 3, 39)

	bTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contractB := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &bTo, "S8a", 3, 39)

	// Make them overlap
	newATo := time.Date(2024, 9, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contractA.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: &newATo}},
			{ID: contractB.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: &bTo}},
		},
	})
	if err == nil {
		t.Fatal("expected overlap error, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payplan := createTestPayPlan(t, db, "TestPP", org.ID)
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	aTo := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	contractA := createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &aTo, "S8a", 3, 39)

	bTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestEmployeeContract(t, db, employee.ID, payplan.ID,
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &bTo, "S8a", 3, 39)

	// First update valid, second references nonexistent contract
	newATo := time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contractA.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: &newATo}},
			{ID: 9999, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{}},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify contract A was NOT changed (rolled back)
	resp, err := svc.GetContractByID(ctx, contractA.ID, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get contract A: %v", err)
	}
	if !resp.To.Equal(aTo) {
		t.Errorf("contract A To = %v, want %v (should be unchanged after rollback)", resp.To, aTo)
	}
}

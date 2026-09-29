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

func TestEmployeeService_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            &to,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org2.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "supplementary",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "invalid_category",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Before from

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            &to,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create first contract
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
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
		WeeklyHours:   float64Ptr(35),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// No 'to' date means ongoing contract
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		To:            nil,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	validCategories := []string{"qualified", "supplementary", "non_pedagogical"}

	for i, cat := range validCategories {
		employee := createTestEmployee(t, db, "John", fmt.Sprintf("Doe%d", i), org.ID)

		from := time.Date(2024+i, 1, 1, 0, 0, 0, 0, time.UTC)
		req := &models.EmployeeContractCreateRequest{
			SectionID:     1,
			From:          from,
			StaffCategory: cat,
			WeeklyHours:   float64Ptr(40),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     99999,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   float64Ptr(39),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org1.ID)

	// Get org2's default section
	var org2Section models.Section
	db.Where("organization_id = ?", org2.ID).First(&org2Section)

	_, err := svc.CreateContract(ctx, employee.ID, org1.ID, &models.EmployeeContractCreateRequest{
		SectionID:     org2Section.ID,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   float64Ptr(39),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create two contracts
	from1 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{SectionID: 1, From: from1, To: &to1, StaffCategory: "supplementary", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	_, _ = svc.CreateContract(ctx, employee.ID, org.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreateRequest{SectionID: 1, From: from2, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org1.ID)

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
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

// SECURITY TEST: Verify GetCurrentRecord returns not found for wrong org
func TestEmployeeService_GetCurrentRecord_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	employee := createTestEmployee(t, db, "John", "Doe", org1.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org1.ID)

	// Create an active (ongoing) contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org1.ID)

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to delete contract for employee from org1 using org2's context
	err = svc.DeleteContract(ctx, contract.ID, employee.ID, org2.ID, nil)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Delete the contract
	err = svc.DeleteContract(ctx, contract.ID, employee.ID, org.ID, nil)
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

	err := svc.DeleteContract(ctx, 999, employee.ID, org.ID, nil)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a contract for employee1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
	contract, err := svc.CreateContract(ctx, employee1.ID, org.ID, req)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to delete employee1's contract using employee2's ID
	err = svc.DeleteContract(ctx, contract.ID, employee2.ID, org.ID, nil)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create an ongoing contract (no end date)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create an expired contract
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{SectionID: 1, From: from, To: &to, StaffCategory: "qualified", WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID}
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

func TestEmployeeService_UpdateContract_StaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newCategory := "supplementary"
	updateReq := &models.EmployeeContractCorrectRequest{
		StaffCategory: models.OptOf(newCategory),
	}

	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(40),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	invalidCategory := "invalid_category"
	updateReq := &models.EmployeeContractCorrectRequest{
		StaffCategory: models.OptOf(invalidCategory),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
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

// =========================================
// Amend-specific UpdateContract Tests
// =========================================

func TestEmployeeService_UpdateContract_AmendChangeSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section1 := createTestSection(t, db, "Krippe", org.ID, false)
	section2 := createTestSection(t, db, "Elementar", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section1.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		SectionID:     models.OptOf(section2.ID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

	today := models.Today()
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newCategory := "supplementary"
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		StaffCategory: models.OptOf(newCategory),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan1 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2023", org.ID)
	payPlan2 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2024", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan1.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		PayPlanID:     models.OptOf(payPlan2.ID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S11b"
	newStep := 5
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		Grade:         models.OptOf(newGrade),
		Step:          models.OptOf(newStep),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newHours := 30.0
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		WeeklyHours:   models.OptOf(newHours),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID:  payPlan.ID,
		Properties: models.ContractProperties{"benefit": "bonus"},
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update with no fields changed (empty update) → still creates new contract via amend
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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

func TestEmployeeService_UpdateContract_AmendOverlapConflict(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	today := models.Today()
	past := today.AddDate(0, -3, 0)
	originalEnd := today.AddDate(0, 0, 30)

	// Original contract: started in the past (amend mode), bounded so we can
	// place a non-overlapping blocker after it (required by the EXCLUDE
	// constraint added in migration 000022).
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		To:            &originalEnd,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create first contract: %v", err)
	}

	// Blocker contract: disjoint from the original, but in the way of an
	// extended amend.
	blockerStart := originalEnd.AddDate(0, 0, 1)
	blockerEnd := blockerStart.AddDate(0, 3, 0)
	if _, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          blockerStart,
		To:            &blockerEnd,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}); err != nil {
		t.Fatalf("failed to create blocker contract: %v", err)
	}

	// Amend the original AND extend its To past the blocker. Amend creates a
	// new contract from today with the extended To — that range overlaps the
	// blocker, so the overlap check inside amendContractTx must produce 409.
	extendedEnd := blockerStart.AddDate(0, 1, 0)
	newCategory := "supplementary"
	_, err = svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		StaffCategory: models.OptOf(newCategory),
		To:            models.OptOf(extendedEnd),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)
	section := createTestSection(t, db, "Krippe", org.ID, false)

	tomorrow := models.Today().AddDate(0, 0, 1)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          tomorrow,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newFrom := tomorrow.AddDate(0, 0, 7)
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		From: models.OptOf(newFrom),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVÖD", org.ID)

	past := models.Today().AddDate(0, -3, 0)
	today := models.Today()
	yesterday := today.AddDate(0, 0, -1)

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Amend: change staff category
	newCategory := "supplementary"
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		StaffCategory: models.OptOf(newCategory),
	})
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}
	newContract := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVÖD", org.ID)

	past := models.Today().AddDate(0, -3, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
		// No To — ongoing
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newHours := float64(30)
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		WeeklyHours:   models.OptOf(newHours),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVÖD", org.ID)

	past := models.Today().AddDate(0, -3, 0)
	endDate := models.Today().AddDate(0, 6, 0)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     section.ID,
		From:          past,
		To:            &endDate,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S9"
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		Grade:         models.OptOf(newGrade),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	updated := &amendResp.Created

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVÖD", org.ID)

	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
		WeeklyHours:   float64Ptr(39),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVÖD", org.ID)

	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from1,
		To:            &to1,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
		WeeklyHours:   float64Ptr(39),
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

	err := svc.DeleteContract(ctx, 99999, employee.ID, org.ID, nil)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "Test Pay Plan", org.ID)
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
		WeeklyHours:   float64Ptr(39),
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if contract.To == nil {
		t.Fatal("setup: To should be set")
	}

	// Clear To by sending nil (simulates frontend sending null to make open-ended)
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		To: models.OptNull[time.Time](),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "Test Pay Plan", org.ID)
	section := getDefaultSection(t, db, org.ID)

	// Create contract with Properties set (use future date to trigger in-place update)
	from := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          from,
		SectionID:     section.ID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   float64Ptr(39),
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
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		Properties: models.OptNull[models.ContractProperties](),
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

func TestEmployeeService_GetContractByID_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	created, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee1.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org1.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, To: &to, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(-1), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "  S8a  ", Step: 3, PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.Grade != "S8a" {
		t.Errorf("Grade = %q, want %q (trimmed)", contract.Grade, "S8a")
	}
}

// Pay-plan coverage validation: contract create must fail loudly when the
// (PayPlanID, Grade, Step) tuple does not resolve to an entry. Without this
// check, the misconfiguration is silent until CalculateSalary or step
// promotion runs much later.

func TestEmployeeService_CreateContract_RejectsMissingGrade(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S99a", // not in the seeded coverage
		Step:          3,
		PayPlanID:     payPlan.ID,
	}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected create to fail because grade is not in pay plan")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_RejectsMissingStep(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a",
		Step:          9, // coverage seeds steps 1..6
		PayPlanID:     payPlan.ID,
	}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected create to fail because step is not in pay plan")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_RejectsNoPeriodAtFrom(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)

	// Pay plan whose only period is in the future relative to the contract's From.
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)
	period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 400000, nil)

	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected create to fail because no period covers the From date")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_AllowsUnpinnedGradeStep(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	// Plan exists but no period yet — that's fine because the contract
	// doesn't pin a grade/step, so no salary lookup will happen.
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID)

	req := &models.EmployeeContractCreateRequest{
		SectionID: 1, From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		StaffCategory: "qualified", WeeklyHours: float64Ptr(39),
		Grade: "", Step: 0, // unpinned
		PayPlanID: payPlan.ID,
	}
	if _, err := svc.CreateContract(ctx, employee.ID, org.ID, req); err != nil {
		t.Fatalf("unpinned (grade, step) should be allowed, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_InPlace_SkipsCoverageWhenTupleUnchanged(t *testing.T) {
	// A legacy contract whose (grade, step) is no longer in the pay plan
	// should still accept updates that don't touch the tuple — touching
	// unrelated fields shouldn't break already-saved data.
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	payPlan := createTestPayPlan(t, db, "TVoD-SuE", org.ID) // no coverage

	// Direct-insert a contract bypassing the service (simulates legacy data
	// that pre-dates the validation, or coverage that was deleted later).
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID, from, nil, "S8a", 3, 40.0)

	newHours := 30.0
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		WeeklyHours: models.OptOf(newHours),
	})
	if err != nil {
		t.Fatalf("update should succeed when only weekly_hours changes, got %v", err)
	}
	if updated.WeeklyHours != 30 {
		t.Errorf("WeeklyHours = %v, want 30", updated.WeeklyHours)
	}
}

func TestEmployeeService_UpdateContract_InPlace_RejectsChangeToMissingGrade(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "Anna", "Schmidt", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	bogus := "S99a"
	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		Grade: models.OptOf(bogus),
	})
	if err == nil {
		t.Fatal("expected update to reject missing grade")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_InPlace_ChangeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newHours := 30.0
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		WeeklyHours: models.OptOf(newHours),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	newGrade := "S8b"
	newStep := 5
	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		Grade: models.OptOf(newGrade),
		Step:  models.OptOf(newStep),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	negativeHours := -5.0
	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		WeeklyHours: models.OptOf(negativeHours),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a future-dated contract (in-place mode)
	from := time.Now().AddDate(1, 0, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	invalidCategory := "invalid"
	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractCorrectRequest{
		StaffCategory: models.OptOf(invalidCategory),
	})
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_UpdateContract_AmendGradeTrimmed(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	// Create a past-dated ongoing contract (amend mode)
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, &models.EmployeeContractCreateRequest{
		SectionID: 1, From: from, StaffCategory: "qualified",
		WeeklyHours: float64Ptr(40), Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	gradeWithSpaces := "  S8a  "
	amendResp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID, &models.EmployeeContractAmendRequest{
		EffectiveFrom: models.Today(),
		Grade:         models.OptOf(gradeWithSpaces),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	amended := &amendResp.Created

	// Amend creates a new contract
	if amended.ID == contract.ID {
		t.Errorf("expected amend to create new contract, but got same ID %d", amended.ID)
	}
	if amended.Grade != "S8a" {
		t.Errorf("Grade = %q, want %q (trimmed)", amended.Grade, "S8a")
	}
}

// --- Batch Update Contract Tests ---

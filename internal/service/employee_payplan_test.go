package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// =====================================================================
// PayPlan ID validation tests
// =====================================================================

func TestEmployeeService_CreateContract_WithPayPlanID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2024", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
		WeeklyHours:   float64Ptr(39),
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
	payPlanOrg2 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE Org2", org2.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
	payPlan1 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2023", org.ID)
	payPlan2 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2024", org.ID)

	// Create contract with payPlan1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
	updateReq := &models.EmployeeContractCorrectRequest{
		PayPlanID: models.OptOf(newPayPlanID),
	}

	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update to a non-existent pay plan
	nonExistentID := uint(99999)
	updateReq := &models.EmployeeContractCorrectRequest{
		PayPlanID: models.OptOf(nonExistentID),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
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
	payPlan1 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE Org1", org1.ID)
	payPlanOrg2 := createTestPayPlanWithCoverage(t, db, "TVoD-SuE Org2", org2.ID)

	// Create contract with org1's pay plan
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan1.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org1.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Try to update to org2's pay plan
	wrongOrgPayPlanID := payPlanOrg2.ID
	updateReq := &models.EmployeeContractCorrectRequest{
		PayPlanID: models.OptOf(wrongOrgPayPlanID),
	}

	_, err = svc.CorrectContract(ctx, contract.ID, employee.ID, org1.ID, updateReq)
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE 2024", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
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
	payPlan := createTestPayPlanWithCoverage(t, db, "TVoD-SuE", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createReq := &models.EmployeeContractCreateRequest{
		SectionID:     1,
		From:          from,
		StaffCategory: "qualified",
		WeeklyHours:   float64Ptr(39),
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, createReq)
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Update without providing PayPlanID (should keep original)
	newCategory := "supplementary"
	updateReq := &models.EmployeeContractCorrectRequest{
		StaffCategory: models.OptOf(newCategory),
	}

	updated, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, updateReq)
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
		WeeklyHours:   float64Ptr(39),
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

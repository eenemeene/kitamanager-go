package service

import (
	"context"
	"errors"
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
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
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
		{"empty first name", &models.EmployeeCreateRequest{FirstName: "", LastName: "Doe", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace first name", &models.EmployeeCreateRequest{FirstName: "   ", LastName: "Doe", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"empty last name", &models.EmployeeCreateRequest{FirstName: "John", LastName: "", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace last name", &models.EmployeeCreateRequest{FirstName: "John", LastName: "   ", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
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
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
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
		Birthdate: time.Now().AddDate(1, 0, 0), // 1 year in future
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

// REGRESSION TEST: Updating section_id must persist when employee has a preloaded section association.
// Previously, GORM's Save() would override section_id from the preloaded Section association,
// causing the update to silently fail.
func TestEmployeeService_Update_SectionID(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section1 := createTestSection(t, db, "Section A", org.ID, false)
	section2 := createTestSection(t, db, "Section B", org.ID, false)

	// Create employee in section1
	sid1 := section1.ID
	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Maria",
			LastName:       "Weber",
			Birthdate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			SectionID:      &sid1,
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create test employee: %v", err)
	}

	// Update section_id to section2
	newSectionID := section2.ID
	updated, err := svc.Update(ctx, employee.ID, org.ID, &models.EmployeeUpdateRequest{SectionID: &newSectionID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.SectionID == nil || *updated.SectionID != section2.ID {
		t.Errorf("response SectionID = %v, want %d", updated.SectionID, section2.ID)
	}

	// Verify it persisted
	refetched, err := svc.GetByID(ctx, employee.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error on re-fetch, got %v", err)
	}
	if refetched.SectionID == nil || *refetched.SectionID != section2.ID {
		t.Errorf("persisted SectionID = %v, want %d", refetched.SectionID, section2.ID)
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

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	req := &models.EmployeeContractCreateRequest{
		From:        from,
		To:          &to,
		Position:    "Teacher",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3, // 50000.00 in cents
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
	if contract.Position != "Teacher" {
		t.Errorf("Position = %v, want Teacher", contract.Position)
	}
	if contract.WeeklyHours != 40 {
		t.Errorf("WeeklyHours = %v, want 40", contract.WeeklyHours)
	}
}

func TestEmployeeService_CreateContract_EmployeeNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		Position:    "Teacher",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
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

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		Position:    "Hacker",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
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

func TestEmployeeService_CreateContract_EmptyPosition(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		Position:    "",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for empty position, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_CreateContract_WhitespaceOnlyPosition(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		Position:    "   ",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for whitespace-only position, got nil")
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

	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Before from

	req := &models.EmployeeContractCreateRequest{
		From:        from,
		To:          &to,
		Position:    "Teacher",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
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

	// Create first contract
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{
		From:        from1,
		To:          &to1,
		Position:    "Teacher",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}
	_, err := svc.CreateContract(ctx, employee.ID, org.ID, req1)
	if err != nil {
		t.Fatalf("first contract: expected no error, got %v", err)
	}

	// Try to create overlapping contract
	from2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Overlaps with first
	to2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreateRequest{
		From:        from2,
		To:          &to2,
		Position:    "Senior Teacher",
		WeeklyHours: 35,
		Grade:       "S8a", Step: 3,
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

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// No 'to' date means ongoing contract
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		To:          nil,
		Position:    "Teacher",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.To != nil {
		t.Errorf("To = %v, want nil (ongoing)", contract.To)
	}
}

func TestEmployeeService_CreateContract_TrimmedPosition(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{
		From:        from,
		Position:    "  Teacher  ",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.Position != "Teacher" {
		t.Errorf("Position = %v, want 'Teacher' (trimmed)", contract.Position)
	}
}

func TestEmployeeService_ListContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	// Create two contracts
	from1 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to1 := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
	req1 := &models.EmployeeContractCreateRequest{From: from1, To: &to1, Position: "Junior", WeeklyHours: 40, Grade: "S8a", Step: 3}
	_, _ = svc.CreateContract(ctx, employee.ID, org.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreateRequest{From: from2, Position: "Senior", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Employee with active contract
	empActive := createTestEmployee(t, db, "Active", "Employee", org.ID)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, empActive.ID, org.ID, &models.EmployeeContractCreateRequest{
		From: from, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Employee with expired contract
	empExpired := createTestEmployee(t, db, "Expired", "Employee", org.ID)
	fromExpired := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	toExpired := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateContract(ctx, empExpired.ID, org.ID, &models.EmployeeContractCreateRequest{
		From: fromExpired, To: &toExpired, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	// Employee with no contract
	createTestEmployee(t, db, "NoContract", "Employee", org.ID)

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// With activeOn filter: only the active employee should be returned
	employees, total, err := svc.ListByOrganizationAndSection(ctx, org.ID, nil, &refDate, "", 100, 0)
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
	allEmployees, allTotal, err := svc.ListByOrganizationAndSection(ctx, org.ID, nil, nil, "", 100, 0)
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

	// Create an active (ongoing) contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Create a contract for org1's employee
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, To: &to, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Create a contract
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, To: &to, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Create a contract for employee1
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, To: &to, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	// Create an ongoing contract (no end date)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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
	if current.Position != "Teacher" {
		t.Errorf("Position = %v, want Teacher", current.Position)
	}
}

func TestEmployeeService_GetCurrentContract_NoActiveContract(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	// Create an expired contract
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreateRequest{From: from, To: &to, Position: "Teacher", WeeklyHours: 40, Grade: "S8a", Step: 3}
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

	futureBirthdate := time.Now().AddDate(1, 0, 0)
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

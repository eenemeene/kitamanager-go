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

	found, err := svc.GetByID(ctx, employee.ID)
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

	_, err := svc.GetByID(ctx, 999)
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

func TestEmployeeService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.EmployeeCreate{
		FirstName: "John",
		LastName:  "Doe",
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
		req  *models.EmployeeCreate
	}{
		{"empty first name", &models.EmployeeCreate{FirstName: "", LastName: "Doe", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace first name", &models.EmployeeCreate{FirstName: "   ", LastName: "Doe", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"empty last name", &models.EmployeeCreate{FirstName: "John", LastName: "", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
		{"whitespace last name", &models.EmployeeCreate{FirstName: "John", LastName: "   ", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}},
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

	req := &models.EmployeeCreate{
		FirstName: "  John  ",
		LastName:  "  Doe  ",
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

	req := &models.EmployeeCreate{
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
	req := &models.EmployeeUpdate{
		FirstName: &newFirstName,
	}

	updated, err := svc.Update(ctx, employee.ID, req)
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

	newName := "Jane"
	req := &models.EmployeeUpdate{
		FirstName: &newName,
	}

	_, err := svc.Update(ctx, 999, req)
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

func TestEmployeeService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	err := svc.Delete(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, employee.ID)
	if err == nil {
		t.Error("expected employee to be deleted")
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

	req := &models.EmployeeContractCreate{
		From:        from,
		To:          &to,
		Position:    "Teacher",
		WeeklyHours: 40,
		Salary:      5000000, // 50000.00 in cents
	}

	contract, err := svc.CreateContract(ctx, employee.ID, req)
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

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreate{
		From:        from,
		Position:    "Teacher",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	_, err := svc.CreateContract(ctx, 999, req)
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

func TestEmployeeService_CreateContract_EmptyPosition(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	employee := createTestEmployee(t, db, "John", "Doe", org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req := &models.EmployeeContractCreate{
		From:        from,
		Position:    "",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	_, err := svc.CreateContract(ctx, employee.ID, req)
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
	req := &models.EmployeeContractCreate{
		From:        from,
		Position:    "   ",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	_, err := svc.CreateContract(ctx, employee.ID, req)
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

	req := &models.EmployeeContractCreate{
		From:        from,
		To:          &to,
		Position:    "Teacher",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	_, err := svc.CreateContract(ctx, employee.ID, req)
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
	req1 := &models.EmployeeContractCreate{
		From:        from1,
		To:          &to1,
		Position:    "Teacher",
		WeeklyHours: 40,
		Salary:      5000000,
	}
	_, err := svc.CreateContract(ctx, employee.ID, req1)
	if err != nil {
		t.Fatalf("first contract: expected no error, got %v", err)
	}

	// Try to create overlapping contract
	from2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Overlaps with first
	to2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreate{
		From:        from2,
		To:          &to2,
		Position:    "Senior Teacher",
		WeeklyHours: 35,
		Salary:      6000000,
	}

	_, err = svc.CreateContract(ctx, employee.ID, req2)
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
	req := &models.EmployeeContractCreate{
		From:        from,
		To:          nil,
		Position:    "Teacher",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, req)
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
	req := &models.EmployeeContractCreate{
		From:        from,
		Position:    "  Teacher  ",
		WeeklyHours: 40,
		Salary:      5000000,
	}

	contract, err := svc.CreateContract(ctx, employee.ID, req)
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
	req1 := &models.EmployeeContractCreate{From: from1, To: &to1, Position: "Junior", WeeklyHours: 40, Salary: 4000000}
	_, _ = svc.CreateContract(ctx, employee.ID, req1)

	from2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	req2 := &models.EmployeeContractCreate{From: from2, Position: "Senior", WeeklyHours: 40, Salary: 5000000}
	_, _ = svc.CreateContract(ctx, employee.ID, req2)

	contracts, err := svc.ListContracts(ctx, employee.ID)
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

	_, err := svc.ListContracts(ctx, 999)
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

package store

import (
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func datePtr(t time.Time) *time.Time {
	return &t
}

func TestPeriodStore_GetCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	store := NewEmployeeStore(db)

	// Create an ongoing contract (no end date)
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		Position:    "Developer",
		WeeklyHours: 40,
		Salary:      500000,
	}
	db.Create(contract)

	current, err := store.Contracts.GetCurrentContract(employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current == nil {
		t.Fatal("expected to find current contract")
	}

	if current.ID != contract.ID {
		t.Errorf("expected contract ID %d, got %d", contract.ID, current.ID)
	}
}

func TestPeriodStore_GetCurrentContract_NoContract(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	store := NewEmployeeStore(db)

	current, err := store.Contracts.GetCurrentContract(employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current != nil {
		t.Error("expected nil for employee without contract")
	}
}

func TestPeriodStore_GetContractOn(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	store := NewEmployeeStore(db)

	// Create a contract from Jan 1 to Dec 31, 2024
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		Position:    "Developer",
		WeeklyHours: 40,
		Salary:      500000,
	}
	db.Create(contract)

	tests := []struct {
		name    string
		date    time.Time
		wantID  *uint
		wantNil bool
	}{
		{
			name:    "date before contract",
			date:    time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			wantNil: true,
		},
		{
			name:   "first day of contract (inclusive)",
			date:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantID: &contract.ID,
		},
		{
			name:   "middle of contract",
			date:   time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			wantID: &contract.ID,
		},
		{
			name:   "last day of contract (inclusive)",
			date:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			wantID: &contract.ID,
		},
		{
			name:    "day after contract ends",
			date:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := store.Contracts.GetContractOn(employee.ID, tt.date)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if found != nil {
					t.Errorf("expected nil, got contract ID %d", found.ID)
				}
			} else {
				if found == nil {
					t.Error("expected contract, got nil")
				} else if found.ID != *tt.wantID {
					t.Errorf("expected contract ID %d, got %d", *tt.wantID, found.ID)
				}
			}
		})
	}
}

func TestPeriodStore_GetHistory(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	store := NewEmployeeStore(db)

	// Create contracts in non-chronological order to test sorting
	contract2 := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		Position: "Senior Developer",
	}
	db.Create(contract2)

	contract1 := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		Position: "Junior Developer",
	}
	db.Create(contract1)

	history, err := store.Contracts.GetHistory(employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(history))
	}

	// Should be sorted by from_date ASC
	if history[0].Position != "Junior Developer" {
		t.Errorf("expected first contract to be 'Junior Developer', got '%s'", history[0].Position)
	}
	if history[1].Position != "Senior Developer" {
		t.Errorf("expected second contract to be 'Senior Developer', got '%s'", history[1].Position)
	}
}

func TestPeriodStore_ValidateNoOverlap_NoExisting(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	store := NewEmployeeStore(db)

	// No existing contracts, should always succeed
	err := store.Contracts.ValidateNoOverlap(
		employee.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
		nil,
	)
	if err != nil {
		t.Errorf("expected no error for first contract, got %v", err)
	}
}

func TestPeriodStore_ValidateNoOverlap_Overlapping(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	// Create existing contract: 2024-01-01 to 2024-12-31
	existing := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		Position: "Existing",
	}
	db.Create(existing)

	store := NewEmployeeStore(db)

	tests := []struct {
		name        string
		from        time.Time
		to          *time.Time
		shouldError bool
	}{
		{
			name:        "completely before existing (no overlap)",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "completely after existing (no overlap)",
			from:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "adjacent before (no overlap - day before from)",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "adjacent after (no overlap - day after to)",
			from:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: false,
		},
		{
			name:        "overlaps at start",
			from:        time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "overlaps at end",
			from:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "completely within existing",
			from:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "completely contains existing",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "exact same dates",
			from:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "ongoing contract overlapping with existing",
			from:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Contracts.ValidateNoOverlap(employee.ID, tt.from, tt.to, nil)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.shouldError && err != nil && !errors.Is(err, ErrContractOverlap) {
				t.Errorf("expected ErrContractOverlap, got %v", err)
			}
		})
	}
}

func TestPeriodStore_ValidateNoOverlap_ExcludeID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	// Create existing contract
	existing := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		Position: "Existing",
	}
	db.Create(existing)

	store := NewEmployeeStore(db)

	// When updating the same contract, it should be excluded from overlap check
	err := store.Contracts.ValidateNoOverlap(
		employee.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		&existing.ID, // exclude this contract from check
	)

	if err != nil {
		t.Errorf("expected no error when excluding own ID, got %v", err)
	}
}

func TestPeriodStore_ValidateNoOverlap_OngoingContracts(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	// Create existing ongoing contract (no end date)
	existing := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil, // ongoing
		},
		Position: "Existing Ongoing",
	}
	db.Create(existing)

	store := NewEmployeeStore(db)

	tests := []struct {
		name        string
		from        time.Time
		to          *time.Time
		shouldError bool
	}{
		{
			name:        "before ongoing (no overlap)",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "overlaps with ongoing",
			from:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "after start of ongoing (overlaps)",
			from:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: true,
		},
		{
			name:        "another ongoing starting same day",
			from:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Contracts.ValidateNoOverlap(employee.ID, tt.from, tt.to, nil)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestPeriodStore_HasActiveContract(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	// Create contract: 2024-01-01 to 2024-12-31
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		Position: "Developer",
	}
	db.Create(contract)

	store := NewEmployeeStore(db)

	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "before contract",
			date:     time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "during contract",
			date:     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "after contract",
			date:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasActive, err := store.Contracts.HasActiveContract(employee.ID, tt.date)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if hasActive != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, hasActive)
			}
		})
	}
}

func TestPeriodStore_CloseCurrentContract(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	// Create ongoing contract
	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil, // ongoing
		},
		Position: "Developer",
	}
	db.Create(contract)

	store := NewEmployeeStore(db)

	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	err := store.Contracts.CloseCurrentContract(employee.ID, endDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify contract was closed
	var updated models.EmployeeContract
	db.First(&updated, contract.ID)

	if updated.To == nil {
		t.Error("expected contract to have end date set")
	} else if !updated.To.Equal(endDate) {
		t.Errorf("expected end date %v, got %v", endDate, *updated.To)
	}
}

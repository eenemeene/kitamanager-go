package store

import (
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// datePtr is defined in testutil_test.go

func TestPeriodStore_GetCurrentRecord(t *testing.T) {
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
	}
	db.Create(contract)

	current, err := store.Contracts().GetCurrentRecord(ctx, employee.ID)
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

func TestPeriodStore_GetCurrentRecord_NoContract(t *testing.T) {
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

	current, err := store.Contracts().GetCurrentRecord(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current != nil {
		t.Error("expected nil for employee without contract")
	}
}

func TestPeriodStore_GetRecordOn(t *testing.T) {
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
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
			found, err := store.Contracts().GetRecordOn(ctx, employee.ID, tt.date)
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

func TestPeriodStore_ListRecords(t *testing.T) {
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "supplementary",
	}
	db.Create(contract2)

	contract1 := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
	}
	db.Create(contract1)

	history, err := store.Contracts().ListRecords(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(history))
	}

	// Should be sorted by from_date ASC
	if history[0].StaffCategory != "qualified" {
		t.Errorf("expected first contract to be 'qualified', got '%s'", history[0].StaffCategory)
	}
	if history[1].StaffCategory != "supplementary" {
		t.Errorf("expected second contract to be 'supplementary', got '%s'", history[1].StaffCategory)
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
	err := store.Contracts().ValidateNoOverlap(ctx,
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
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
			err := store.Contracts().ValidateNoOverlap(ctx, employee.ID, tt.from, tt.to, nil)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.shouldError && err != nil && !errors.Is(err, ErrPeriodOverlap) {
				t.Errorf("expected ErrPeriodOverlap, got %v", err)
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
	}
	db.Create(existing)

	store := NewEmployeeStore(db)

	// When updating the same contract, it should be excluded from overlap check
	err := store.Contracts().ValidateNoOverlap(ctx,
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil, // ongoing
			},
		},
		StaffCategory: "qualified",
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
			err := store.Contracts().ValidateNoOverlap(ctx, employee.ID, tt.from, tt.to, nil)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestPeriodStore_HasActiveRecord(t *testing.T) {
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
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
			hasActive, err := store.Contracts().HasActiveRecord(ctx, employee.ID, tt.date)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if hasActive != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, hasActive)
			}
		})
	}
}

func TestPeriodStore_GetRecordOn_ConsecutiveContracts(t *testing.T) {
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

	// Contract A: Jan 1 to Jan 10
	contractA := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
	}
	db.Create(contractA)

	// Contract B: Jan 11 onwards (consecutive, no gap)
	contractB := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "supplementary",
		WeeklyHours:   30,
		Grade:         "S8b", Step: 1,
	}
	db.Create(contractB)

	tests := []struct {
		name    string
		date    time.Time
		wantID  *uint
		wantNil bool
	}{
		{
			name:    "before both contracts",
			date:    time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			wantNil: true,
		},
		{
			name:   "day 9 - within contract A",
			date:   time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC),
			wantID: &contractA.ID,
		},
		{
			name:   "day 10 - last day of contract A (inclusive)",
			date:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			wantID: &contractA.ID,
		},
		{
			name:   "day 11 - first day of contract B",
			date:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
			wantID: &contractB.ID,
		},
		{
			name:   "day 20 - within contract B",
			date:   time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			wantID: &contractB.ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := store.Contracts().GetRecordOn(ctx, employee.ID, tt.date)
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

func TestPeriodStore_GetRecordOn_GapBetweenContracts(t *testing.T) {
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

	// Contract A: Jan 1 to Jan 2
	contractA := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
	}
	db.Create(contractA)

	// Contract B: Jan 4 onwards (gap on Jan 3)
	contractB := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "supplementary",
		WeeklyHours:   30,
		Grade:         "S8b", Step: 1,
	}
	db.Create(contractB)

	tests := []struct {
		name    string
		date    time.Time
		wantID  *uint
		wantNil bool
	}{
		{
			name:   "day 1 - within contract A",
			date:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantID: &contractA.ID,
		},
		{
			name:   "day 2 - last day of contract A",
			date:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			wantID: &contractA.ID,
		},
		{
			name:    "day 3 - gap between contracts (no active contract)",
			date:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			wantNil: true,
		},
		{
			name:   "day 4 - first day of contract B",
			date:   time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			wantID: &contractB.ID,
		},
		{
			name:   "day 10 - well within contract B",
			date:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			wantID: &contractB.ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := store.Contracts().GetRecordOn(ctx, employee.ID, tt.date)
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

func TestPeriodStore_HasActiveRecord_GapBetweenContracts(t *testing.T) {
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

	// Contract A: Jan 1 to Jan 2
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
			},
		},
		StaffCategory: "qualified",
	})

	// Contract B: Jan 4 onwards
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "supplementary",
	})

	store := NewEmployeeStore(db)

	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{"day 2 - last day of A", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), true},
		{"day 3 - gap (no active)", time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), false},
		{"day 4 - first day of B", time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasActive, err := store.Contracts().HasActiveRecord(ctx, employee.ID, tt.date)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasActive != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, hasActive)
			}
		})
	}
}

func TestPeriodStore_CloseCurrentRecord(t *testing.T) {
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil, // ongoing
			},
		},
		StaffCategory: "qualified",
	}
	db.Create(contract)

	store := NewEmployeeStore(db)

	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	err := store.Contracts().CloseCurrentRecord(ctx, employee.ID, endDate)
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

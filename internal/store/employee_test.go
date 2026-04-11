package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestEmployeeStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Max",
			LastName:       "Mustermann",
			Birthdate:      time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	err := store.Create(ctx, employee)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if employee.ID == 0 {
		t.Error("expected employee ID to be set")
	}
}

func TestEmployeeStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create employees directly
	employee1 := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "First",
			LastName:       "Employee",
			Birthdate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	employee2 := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Second",
			LastName:       "Employee",
			Birthdate:      time.Date(1991, 2, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(employee1)
	db.Create(employee2)

	employees, total, err := store.FindAll(ctx, 100, 0)
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

func TestEmployeeStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(employee)

	found, err := store.FindByID(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", found.FirstName)
	}
}

func TestEmployeeStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Emp1", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Emp2", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})
	db.Create(&models.Employee{
		Person: models.Person{OrganizationID: org2.ID, FirstName: "Emp3", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})

	employees, total, err := store.FindByOrganization(ctx, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(employees) != 2 {
		t.Errorf("expected 2 employees for org1, got %d", len(employees))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestEmployeeStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Original",
			LastName:       "Name",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	employee.FirstName = "Updated"
	err := store.Update(ctx, employee)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(ctx, employee.ID)
	if found.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", found.FirstName)
	}
}

func TestEmployeeStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "ToDelete",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	err := store.Delete(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(ctx, employee.ID)
	if err == nil {
		t.Error("expected error finding deleted employee")
	}
}

func TestEmployeeStore_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}

	err := store.CreateContract(ctx, contract)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.ID == 0 {
		t.Error("expected contract ID to be set")
	}
}

func TestEmployeeStore_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}
	db.Create(contract)

	err := store.DeleteContract(ctx, contract.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindContractByID(ctx, contract.ID)
	if err == nil {
		t.Error("expected error finding deleted contract")
	}
}

func TestEmployeeStore_FindByOrganizationAndSection_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)

	// Employee with active contract on refDate
	empActive := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Active", LastName: "Employee", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(empActive)
	db.Create(&models.EmployeeContract{
		EmployeeID:    empActive.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})

	// Employee with expired contract
	empExpired := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Expired", LastName: "Employee", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(empExpired)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID:    empExpired.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: &to}},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})

	// Employee with future contract
	empFuture := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Future", LastName: "Employee", Birthdate: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(empFuture)
	db.Create(&models.EmployeeContract{
		EmployeeID:    empFuture.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})

	// Employee with no contract
	empNoContract := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "NoContract", LastName: "Employee", Birthdate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(empNoContract)

	// Query with activeOn filter
	employees, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, "", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(employees) != 1 {
		t.Errorf("expected 1 employee with active contract, got %d", len(employees))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(employees) == 1 && employees[0].FirstName != "Active" {
		t.Errorf("expected Active employee, got %s", employees[0].FirstName)
	}

	// Query without activeOn (should return all 4 employees)
	allEmployees, allTotal, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allEmployees) != 4 {
		t.Errorf("expected 4 employees without filter, got %d", len(allEmployees))
	}
	if allTotal != 4 {
		t.Errorf("expected total 4, got %d", allTotal)
	}
}

func TestEmployeeStore_FindByOrganizationAndSection_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create employees with distinct names
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Max", LastName: "Mustermann", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Maria", LastName: "Mueller", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Lisa", LastName: "Maier", Birthdate: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Search by first name prefix (case-insensitive)
	_, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "ma", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// "Ma" matches Max (first), Maria (first), and Lisa Maier (last)
	if total != 3 {
		t.Errorf("expected total 3 for search 'ma', got %d", total)
	}

	// Search by last name
	employees, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "mustermann", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for search 'mustermann', got %d", total)
	}
	if len(employees) == 1 && employees[0].FirstName != "Max" {
		t.Errorf("expected Max, got %s", employees[0].FirstName)
	}

	// Search with no results
	employees, total, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "zzz", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0 for search 'zzz', got %d", total)
	}
	if len(employees) != 0 {
		t.Errorf("expected 0 employees for search 'zzz', got %d", len(employees))
	}

	// Empty search returns all
	_, total, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3 for empty search, got %d", total)
	}
}

func TestEmployeeStore_FindByOrganizationAndSection_SearchWithPagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create 5 employees with "Ma" prefix and 2 without
	for i := 1; i <= 5; i++ {
		db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: fmt.Sprintf("Max%d", i), LastName: "Mustermann", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}})
	}
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Lisa", LastName: "Fischer", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Anna", LastName: "Weber", Birthdate: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Page 1 of search results (limit=2)
	employees, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "max", nil, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5 for search 'max', got %d", total)
	}
	if len(employees) != 2 {
		t.Errorf("expected 2 employees on page 1, got %d", len(employees))
	}

	// Page 2
	employees, _, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "max", nil, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 2 {
		t.Errorf("expected 2 employees on page 2, got %d", len(employees))
	}

	// Page 3 (last)
	employees, _, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, "max", nil, 2, 4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Errorf("expected 1 employee on page 3, got %d", len(employees))
	}
}

func TestEmployeeStore_FindByOrganizationAndSection_SearchWithActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Max with active contract
	max := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Max", LastName: "Mustermann", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(max)
	db.Create(&models.EmployeeContract{EmployeeID: max.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 40, PayPlanID: payPlan.ID})

	// Maria without contract
	db.Create(&models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Maria", LastName: "Mueller", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Search "m" + activeOn: only Max has an active contract
	employees, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, "m", nil, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for search 'm' with activeOn, got %d", total)
	}
	if len(employees) == 1 && employees[0].FirstName != "Max" {
		t.Errorf("expected Max, got %s", employees[0].FirstName)
	}
}

func TestEmployeeStore_DeleteAlsoDeletesContracts(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Employee",
			Birthdate:      time.Now(),
		},
	}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		Grade:         "S8a", Step: 3,
		PayPlanID: payPlan.ID,
	}
	db.Create(contract)
	contractID := contract.ID

	err := store.Delete(ctx, employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify contract is also deleted
	_, err = store.FindContractByID(ctx, contractID)
	if err == nil {
		t.Error("expected contract to be deleted with employee")
	}
}

// --- FindByOrganizationInDateRange tests ---

func TestEmployeeStore_FindByOrganizationInDateRange_Basic(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	// Employee with contract in Jan-Jun 2024
	emp1 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Anna", LastName: "Mueller", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp1)
	endDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp1.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate}},
		StaffCategory: "qualified",
		WeeklyHours:   30,
		PayPlanID:     payplan.ID,
	})

	// Employee with contract starting Jul 2024 (outside Jan-Jun range)
	emp2 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Bob", LastName: "Schmidt", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp2)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp2.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payplan.ID,
	})

	// Query Jan-Jun 2024 => only emp1
	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if employees[0].ID != emp1.ID {
		t.Errorf("expected employee %d, got %d", emp1.ID, employees[0].ID)
	}
	// Contracts should be preloaded
	if len(employees[0].Contracts) != 1 {
		t.Errorf("expected 1 preloaded contract, got %d", len(employees[0].Contracts))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_OngoingContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	emp := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Clara", LastName: "Ongoing", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "qualified",
		WeeklyHours:   35,
		PayPlanID:     payplan.ID,
	})

	// Ongoing contract should appear in any future range
	rangeStart := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2030, 12, 1, 0, 0, 0, 0, time.UTC)
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_SectionFilter(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	// Create second section
	section2 := &models.Section{OrganizationID: org.ID, Name: "Section B"}
	db.Create(section2)

	emp1 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "A", LastName: "SectionDefault", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp1)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp1.ID,
		BaseContract:  models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "qualified",
		WeeklyHours:   30,
		PayPlanID:     payplan.ID,
	})

	emp2 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "B", LastName: "SectionB", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp2)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp2.ID,
		BaseContract:  models.BaseContract{SectionID: section2.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
		StaffCategory: "supplementary",
		WeeklyHours:   20,
		PayPlanID:     payplan.ID,
	})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Filter by section2 => only emp2
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, &section2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if employees[0].ID != emp2.ID {
		t.Errorf("expected employee %d, got %d", emp2.ID, employees[0].ID)
	}
	// Preloaded contracts should only include section2's contracts
	if len(employees[0].Contracts) != 1 {
		t.Errorf("expected 1 preloaded contract, got %d", len(employees[0].Contracts))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Empty Org")

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 0 {
		t.Errorf("expected 0 employees, got %d", len(employees))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_OrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section1 := getDefaultSectionID(t, db, org1.ID)
	section2 := getDefaultSectionID(t, db, org2.ID)
	payplan1 := createTestPayPlan(t, db, org1.ID)
	payplan2 := createTestPayPlan(t, db, org2.ID)

	emp1 := &models.Employee{Person: models.Person{OrganizationID: org1.ID, FirstName: "Org1", LastName: "Employee", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp1)
	db.Create(&models.EmployeeContract{EmployeeID: emp1.ID, BaseContract: models.BaseContract{SectionID: section1, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 30, PayPlanID: payplan1.ID})

	emp2 := &models.Employee{Person: models.Person{OrganizationID: org2.ID, FirstName: "Org2", LastName: "Employee", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp2)
	db.Create(&models.EmployeeContract{EmployeeID: emp2.ID, BaseContract: models.BaseContract{SectionID: section2, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 40, PayPlanID: payplan2.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Query org1 => only emp1
	employees, err := store.FindByOrganizationInDateRange(ctx, org1.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if employees[0].ID != emp1.ID {
		t.Errorf("expected employee %d, got %d", emp1.ID, employees[0].ID)
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_MultipleContracts(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	emp := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Multi", LastName: "Contract", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp)

	// Two contracts: Jan-Mar and Jul-Dec
	endDate1 := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate1}}, StaffCategory: "qualified", WeeklyHours: 25, PayPlanID: payplan.ID})
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 35, PayPlanID: payplan.ID})

	// Query full year => employee returned once with both contracts preloaded
	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee (not duplicated), got %d", len(employees))
	}
	if len(employees[0].Contracts) != 2 {
		t.Errorf("expected 2 preloaded contracts, got %d", len(employees[0].Contracts))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_StaffCategoryFilter(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	// 3 employees with different categories
	empQualified := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Anna", LastName: "Qualified", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(empQualified)
	db.Create(&models.EmployeeContract{EmployeeID: empQualified.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 39, PayPlanID: payplan.ID})

	empSupplementary := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Bob", LastName: "Supplementary", Birthdate: time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(empSupplementary)
	db.Create(&models.EmployeeContract{EmployeeID: empSupplementary.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "supplementary", WeeklyHours: 30, PayPlanID: payplan.ID})

	empNonPed := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Clara", LastName: "NonPed", Birthdate: time.Date(1988, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(empNonPed)
	db.Create(&models.EmployeeContract{EmployeeID: empNonPed.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "non_pedagogical", WeeklyHours: 20, PayPlanID: payplan.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Filter qualified only
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{"qualified"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if employees[0].ID != empQualified.ID {
		t.Errorf("expected qualified employee, got ID %d", employees[0].ID)
	}
	if len(employees[0].Contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(employees[0].Contracts))
	}

	// Filter multiple categories (qualified + supplementary)
	employees, err = store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{"qualified", "supplementary"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(employees))
	}

	// No filter (nil) => all 3
	employees, err = store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 3 {
		t.Fatalf("expected 3 employees, got %d", len(employees))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_StaffCategoryFilter_PreloadedContracts(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	// Employee with two contracts: one qualified, one non-pedagogical (category change mid-year)
	emp := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Dana", LastName: "Mixed", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp)
	endDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate}}, StaffCategory: "qualified", WeeklyHours: 39, PayPlanID: payplan.ID})
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "non_pedagogical", WeeklyHours: 20, PayPlanID: payplan.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Filter qualified => employee returned, but only the qualified contract is preloaded
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{"qualified"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if len(employees[0].Contracts) != 1 {
		t.Fatalf("expected 1 preloaded contract (only qualified), got %d", len(employees[0].Contracts))
	}
	if employees[0].Contracts[0].StaffCategory != "qualified" {
		t.Errorf("expected qualified contract, got %q", employees[0].Contracts[0].StaffCategory)
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_StaffCategoryFilter_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	emp := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Eva", LastName: "Qualified", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp)
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 39, PayPlanID: payplan.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Filter for category that no one has
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{"non_pedagogical"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 0 {
		t.Errorf("expected 0 employees, got %d", len(employees))
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_StaffCategoryAndSectionCombined(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionA := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	sectionB := &models.Section{OrganizationID: org.ID, Name: "Section B"}
	db.Create(sectionB)

	// emp1: qualified in section A
	emp1 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "A", LastName: "QualA", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp1)
	db.Create(&models.EmployeeContract{EmployeeID: emp1.ID, BaseContract: models.BaseContract{SectionID: sectionA, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 39, PayPlanID: payplan.ID})

	// emp2: qualified in section B
	emp2 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "B", LastName: "QualB", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp2)
	db.Create(&models.EmployeeContract{EmployeeID: emp2.ID, BaseContract: models.BaseContract{SectionID: sectionB.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 30, PayPlanID: payplan.ID})

	// emp3: supplementary in section B
	emp3 := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "C", LastName: "SuppB", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp3)
	db.Create(&models.EmployeeContract{EmployeeID: emp3.ID, BaseContract: models.BaseContract{SectionID: sectionB.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "supplementary", WeeklyHours: 20, PayPlanID: payplan.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Filter: qualified + section B => only emp2
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{"qualified"}, &sectionB.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(employees))
	}
	if employees[0].ID != emp2.ID {
		t.Errorf("expected employee %d, got %d", emp2.ID, employees[0].ID)
	}
}

func TestEmployeeStore_FindByOrganizationInDateRange_EmptyStaffCategories(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
	org := createTestOrganization(t, db, "Test Org")
	sectionID := getDefaultSectionID(t, db, org.ID)
	payplan := createTestPayPlan(t, db, org.ID)

	emp := &models.Employee{Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Employee", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emp)
	db.Create(&models.EmployeeContract{EmployeeID: emp.ID, BaseContract: models.BaseContract{SectionID: sectionID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, StaffCategory: "qualified", WeeklyHours: 39, PayPlanID: payplan.ID})

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	// Empty slice should behave like nil (no filter)
	employees, err := store.FindByOrganizationInDateRange(ctx, org.ID, rangeStart, rangeEnd, []string{}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(employees) != 1 {
		t.Errorf("expected 1 employee (empty slice = no filter), got %d", len(employees))
	}
}

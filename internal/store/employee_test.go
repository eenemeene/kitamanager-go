package store

import (
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

	err := store.Create(employee)
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

	employees, total, err := store.FindAll(100, 0)
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

	found, err := store.FindByID(employee.ID)
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

	employees, total, err := store.FindByOrganization(org1.ID, 100, 0)
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
	err := store.Update(employee)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(employee.ID)
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

	err := store.Delete(employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(employee.ID)
	if err == nil {
		t.Error("expected error finding deleted employee")
	}
}

func TestEmployeeStore_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
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

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
		},
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}

	err := store.CreateContract(contract)
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
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}
	db.Create(contract)

	err := store.DeleteContract(contract.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindContractByID(contract.ID)
	if err == nil {
		t.Error("expected error finding deleted contract")
	}
}

func TestEmployeeStore_DeleteAlsoDeletesContracts(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmployeeStore(db)
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

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		Position:    "Developer",
		WeeklyHours: 40,
		Grade:       "S8a", Step: 3,
	}
	db.Create(contract)
	contractID := contract.ID

	err := store.Delete(employee.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify contract is also deleted
	_, err = store.FindContractByID(contractID)
	if err == nil {
		t.Error("expected contract to be deleted with employee")
	}
}

package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Emma",
			LastName:       "Schmidt",
			Birthdate:      time.Date(2020, 3, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	err := store.Create(child)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if child.ID == 0 {
		t.Error("expected child ID to be set")
	}
}

func TestChildStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child1", LastName: "Last", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child2", LastName: "Last", Birthdate: time.Now()},
	})

	children, total, err := store.FindAll(100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestChildStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	found, err := store.FindByID(child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got '%s'", found.FirstName)
	}
}

func TestChildStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Child1", LastName: "Last", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Child2", LastName: "Last", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org2.ID, FirstName: "Child3", LastName: "Last", Birthdate: time.Now()},
	})

	children, total, err := store.FindByOrganization(org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children for org1, got %d", len(children))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestChildStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Original",
			LastName:       "Name",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	child.FirstName = "Updated"
	err := store.Update(child)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(child.ID)
	if found.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got '%s'", found.FirstName)
	}
}

func TestChildStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "ToDelete",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	err := store.Delete(child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(child.ID)
	if err == nil {
		t.Error("expected error finding deleted child")
	}
}

func TestChildStore_CreateContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		Attributes: []string{"ganztags"},
	}

	err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contract.ID == 0 {
		t.Error("expected contract ID to be set")
	}
}

func TestChildStore_DeleteContract(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
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

func TestChildStore_DeleteAlsoDeletesContracts(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(contract)
	contractID := contract.ID

	err := store.Delete(child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify contract is also deleted
	_, err = store.FindContractByID(contractID)
	if err == nil {
		t.Error("expected contract to be deleted with child")
	}
}

func TestChildStore_ContractOverlapValidation(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	// Create existing contract
	existing := &models.ChildContract{
		ChildID: child.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
	}
	db.Create(existing)

	// Try to create overlapping contract
	err := store.Contracts().ValidateNoOverlap(
		child.ID,
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		datePtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		nil,
	)

	if err == nil {
		t.Error("expected overlap error")
	}

	// Non-overlapping contract should succeed
	err = store.Contracts().ValidateNoOverlap(
		child.ID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("expected no error for non-overlapping contract, got %v", err)
	}
}

func TestChildStore_FindByOrganizationWithContractOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")
	org2 := createTestOrganization(t, db, "Other Org")

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)

	// Child with active contract on refDate
	childActive := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Active",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childActive)
	db.Create(&models.ChildContract{
		ChildID:    childActive.ID,
		Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		Attributes: []string{"ganztags"},
	})

	// Child with ongoing contract (no end date)
	childOngoing := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Ongoing",
			LastName:       "Child",
			Birthdate:      time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childOngoing)
	db.Create(&models.ChildContract{
		ChildID:    childOngoing.ID,
		Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		Attributes: []string{"halbtags", "ndh"},
	})

	// Child with expired contract (before refDate)
	childExpired := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Expired",
			LastName:       "Child",
			Birthdate:      time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childExpired)
	db.Create(&models.ChildContract{
		ChildID: childExpired.ID,
		Period: models.Period{
			From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
	})

	// Child with future contract (after refDate)
	childFuture := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Future",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childFuture)
	db.Create(&models.ChildContract{
		ChildID: childFuture.ID,
		Period:  models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	})

	// Child with no contract
	childNoContract := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "NoContract",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childNoContract)

	// Child in different organization with active contract
	childOtherOrg := &models.Child{
		Person: models.Person{
			OrganizationID: org2.ID,
			FirstName:      "OtherOrg",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(childOtherOrg)
	db.Create(&models.ChildContract{
		ChildID: childOtherOrg.ID,
		Period:  models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	// Query for children with active contracts on refDate
	children, err := store.FindByOrganizationWithContractOn(org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should return only childActive and childOngoing
	if len(children) != 2 {
		t.Errorf("expected 2 children with active contracts, got %d", len(children))
		for _, c := range children {
			t.Logf("  - %s %s (ID: %d)", c.FirstName, c.LastName, c.ID)
		}
	}

	// Verify each child has contracts loaded
	for _, child := range children {
		if len(child.Contracts) == 0 {
			t.Errorf("expected child %s to have contracts loaded", child.FirstName)
		}
	}

	// Verify correct children are returned
	names := make(map[string]bool)
	for _, c := range children {
		names[c.FirstName] = true
	}
	if !names["Active"] || !names["Ongoing"] {
		t.Errorf("expected Active and Ongoing children, got names: %v", names)
	}
}

func TestChildStore_ContractAttributes_JSONSerialization(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	tests := []struct {
		name       string
		attributes []string
	}{
		{
			name:       "empty attributes",
			attributes: []string{},
		},
		{
			name:       "single attribute",
			attributes: []string{"ganztags"},
		},
		{
			name:       "multiple attributes",
			attributes: []string{"ganztags", "ndh", "integration_a"},
		},
		{
			name:       "attribute with special chars",
			attributes: []string{"type-a", "type_b", "type.c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := &models.ChildContract{
				ChildID: child.ID,
				Period: models.Period{
					From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Attributes: tt.attributes,
			}

			err := store.CreateContract(contract)
			if err != nil {
				t.Fatalf("failed to create contract: %v", err)
			}

			// Retrieve the contract from database
			retrieved, err := store.FindContractByID(contract.ID)
			if err != nil {
				t.Fatalf("failed to retrieve contract: %v", err)
			}

			// Verify attributes round-trip correctly
			if len(retrieved.Attributes) != len(tt.attributes) {
				t.Errorf("expected %d attributes, got %d", len(tt.attributes), len(retrieved.Attributes))
			}

			for i, attr := range tt.attributes {
				if i >= len(retrieved.Attributes) {
					break
				}
				if retrieved.Attributes[i] != attr {
					t.Errorf("attribute[%d]: expected %q, got %q", i, attr, retrieved.Attributes[i])
				}
			}

			// Cleanup for next test
			_ = store.DeleteContract(contract.ID)
		})
	}
}

func TestChildStore_ContractAttributes_NilHandling(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Now(),
		},
	}
	db.Create(child)

	// Create contract with nil attributes
	contract := &models.ChildContract{
		ChildID: child.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Attributes: nil,
	}

	err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("failed to create contract with nil attributes: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.FindContractByID(contract.ID)
	if err != nil {
		t.Fatalf("failed to retrieve contract: %v", err)
	}

	// nil or empty slice should be acceptable
	if len(retrieved.Attributes) != 0 {
		t.Errorf("expected nil or empty attributes, got %v", retrieved.Attributes)
	}
}

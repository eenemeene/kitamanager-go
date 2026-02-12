package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

var ctx = context.Background()

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

	err := store.Create(ctx, child)
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
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child1", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Child2", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})

	children, total, err := store.FindAll(ctx, 100, 0)
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

	found, err := store.FindByID(ctx, child.ID)
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
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Child1", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Child2", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})
	db.Create(&models.Child{
		Person: models.Person{OrganizationID: org2.ID, FirstName: "Child3", LastName: "Last", Gender: "male", Birthdate: time.Now()},
	})

	children, total, err := store.FindByOrganization(ctx, org1.ID, 100, 0)
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
	err := store.Update(ctx, child)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(ctx, child.ID)
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

	err := store.Delete(ctx, child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(ctx, child.ID)
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   nil,
			},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}

	err := store.CreateContract(ctx, contract)
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	db.Create(contract)
	contractID := contract.ID

	err := store.Delete(ctx, child.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify contract is also deleted
	_, err = store.FindContractByID(ctx, contractID)
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
	}
	db.Create(existing)

	// Try to create overlapping contract
	err := store.Contracts().ValidateNoOverlap(ctx,
		child.ID,
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		datePtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		nil,
	)

	if err == nil {
		t.Error("expected overlap error")
	}

	// Non-overlapping contract should succeed
	err = store.Contracts().ValidateNoOverlap(ctx,
		child.ID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
		nil,
	)

	if err != nil {
		t.Errorf("expected no error for non-overlapping contract, got %v", err)
	}
}

func TestChildStore_FindByOrganizationWithActiveOn(t *testing.T) {
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
		ChildID: childActive.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
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
		ChildID: childOngoing.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "halbtag", "supplements": []string{"ndh"}},
		},
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
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
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
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		},
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
		BaseContract: models.BaseContract{
			Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	// Query for children with active contracts on refDate
	children, err := store.FindByOrganizationWithActiveOn(ctx, org.ID, refDate)
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

func TestChildStore_FindByOrganizationAndSection_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	refDate := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)

	// Child with active contract on refDate
	childActive := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Active", LastName: "Child", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childActive)
	db.Create(&models.ChildContract{
		ChildID:      childActive.ID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
	})

	// Child with expired contract (before refDate)
	childExpired := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Expired", LastName: "Child", Birthdate: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childExpired)
	db.Create(&models.ChildContract{
		ChildID:      childExpired.ID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))}},
	})

	// Child with future contract (after refDate)
	childFuture := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Future", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childFuture)
	db.Create(&models.ChildContract{
		ChildID:      childFuture.ID,
		BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}},
	})

	// Child with no contract
	childNoContract := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "NoContract", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childNoContract)

	// Query with activeOn filter
	children, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, nil, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 1 {
		t.Errorf("expected 1 child with active contract, got %d", len(children))
		for _, c := range children {
			t.Logf("  - %s %s (ID: %d)", c.FirstName, c.LastName, c.ID)
		}
	}

	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	if len(children) == 1 && children[0].FirstName != "Active" {
		t.Errorf("expected Active child, got %s", children[0].FirstName)
	}

	// Query without activeOn (should return all 4 children)
	allChildren, allTotal, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(allChildren) != 4 {
		t.Errorf("expected 4 children without filter, got %d", len(allChildren))
	}
	if allTotal != 4 {
		t.Errorf("expected total 4, got %d", allTotal)
	}
}

func TestChildStore_FindByOrganizationAndSection_ActiveOn_Pagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Create 3 children with active contracts
	for i := 0; i < 3; i++ {
		child := &models.Child{
			Person: models.Person{OrganizationID: org.ID, FirstName: "Active", LastName: "Child", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		db.Create(child)
		db.Create(&models.ChildContract{
			ChildID:      child.ID,
			BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
		})
	}

	// Create 1 child without contract
	noContract := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "NoContract", LastName: "Child", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(noContract)

	// Paginate: limit=2, offset=0 should return 2 of 3 total
	children, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, nil, "", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children (page 1), got %d", len(children))
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}

	// Page 2: limit=2, offset=2 should return 1 remaining
	children2, _, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, nil, "", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(children2) != 1 {
		t.Errorf("expected 1 child (page 2), got %d", len(children2))
	}
}

func TestChildStore_FindByOrganizationAndSection_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create children with distinct names
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Emma", LastName: "Schmidt", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Liam", LastName: "Mueller", Birthdate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Emilia", LastName: "Fischer", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Search by first name prefix (case-insensitive)
	children, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "em", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2 for search 'em', got %d", total)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children for search 'em', got %d", len(children))
	}

	// Search by last name
	children, total, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "schmidt", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for search 'schmidt', got %d", total)
	}
	if len(children) == 1 && children[0].FirstName != "Emma" {
		t.Errorf("expected Emma, got %s", children[0].FirstName)
	}

	// Search with no results
	children, total, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "zzz", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0 for search 'zzz', got %d", total)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children for search 'zzz', got %d", len(children))
	}

	// Empty search returns all
	_, total, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3 for empty search, got %d", total)
	}
}

func TestChildStore_FindByOrganizationAndSection_SearchWithPagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create 5 children with "Em" prefix and 2 without
	for i := 1; i <= 5; i++ {
		db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: fmt.Sprintf("Emma%d", i), LastName: "Schmidt", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}})
	}
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Liam", LastName: "Mueller", Birthdate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}})
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Noah", LastName: "Fischer", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Page 1 of search results (limit=2)
	children, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "emma", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5 for search 'emma', got %d", total)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children on page 1, got %d", len(children))
	}

	// Page 2
	children, _, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "emma", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children on page 2, got %d", len(children))
	}

	// Page 3 (last)
	children, _, err = store.FindByOrganizationAndSection(ctx, org.ID, nil, nil, nil, "emma", 2, 4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(children) != 1 {
		t.Errorf("expected 1 child on page 3, got %d", len(children))
	}
}

func TestChildStore_FindByOrganizationAndSection_SearchWithActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewChildStore(db)
	org := createTestOrganization(t, db, "Test Org")

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Emma with active contract
	emma := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Emma", LastName: "Schmidt", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(emma)
	db.Create(&models.ChildContract{ChildID: emma.ID, BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}}})

	// Emilia without contract
	db.Create(&models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Emilia", LastName: "Fischer", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}})

	// Search "em" + activeOn: only Emma has an active contract
	children, total, err := store.FindByOrganizationAndSection(ctx, org.ID, nil, &refDate, nil, "em", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for search 'em' with activeOn, got %d", total)
	}
	if len(children) == 1 && children[0].FirstName != "Emma" {
		t.Errorf("expected Emma, got %s", children[0].FirstName)
	}
}

func TestChildStore_ContractProperties_JSONSerialization(t *testing.T) {
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
		properties models.ContractProperties
	}{
		{
			name:       "nil properties",
			properties: nil,
		},
		{
			name:       "empty properties",
			properties: models.ContractProperties{},
		},
		{
			name:       "scalar property",
			properties: models.ContractProperties{"care_type": "ganztag"},
		},
		{
			name:       "array property",
			properties: models.ContractProperties{"supplements": []string{"ndh", "mss"}},
		},
		{
			name:       "mixed properties",
			properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := &models.ChildContract{
				ChildID: child.ID,
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					Properties: tt.properties,
				},
			}

			err := store.CreateContract(ctx, contract)
			if err != nil {
				t.Fatalf("failed to create contract: %v", err)
			}

			// Retrieve the contract from database
			retrieved, err := store.FindContractByID(ctx, contract.ID)
			if err != nil {
				t.Fatalf("failed to retrieve contract: %v", err)
			}

			// Verify properties round-trip correctly
			if tt.properties == nil {
				if len(retrieved.Properties) != 0 {
					t.Errorf("expected nil or empty properties, got %v", retrieved.Properties)
				}
			} else {
				// Check scalar property
				if careType := tt.properties.GetScalarProperty("care_type"); careType != "" {
					if retrieved.Properties.GetScalarProperty("care_type") != careType {
						t.Errorf("care_type mismatch: expected %q, got %q", careType, retrieved.Properties.GetScalarProperty("care_type"))
					}
				}
			}

			// Cleanup for next test
			_ = store.DeleteContract(ctx, contract.ID)
		})
	}
}

// datePtr returns a pointer to a time.Time
func datePtr(t time.Time) *time.Time {
	return &t
}

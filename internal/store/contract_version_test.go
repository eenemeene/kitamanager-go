package store

// Optimistic concurrency on contract updates.
//
// Contract writes previously used a bare GORM Save() with RowsAffected ignored,
// so two people editing the same contract silently last-write-wins — and since a
// contract's care type and supplements decide its funding, the lost update
// quietly changes money. These tests pin the guard.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

func TestChildStore_UpdateContract_BumpsVersion(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewChildStore(db)
	ctx := context.Background()

	org := testutil.CreateTestOrganization(t, db, "Version Org")
	section := testutil.CreateTestSection(t, db, "Nest", org.ID, false)
	child := testutil.CreateTestChild(t, db, "Version", "Child", org.ID)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "halbtag"},
		},
	}
	if err := s.CreateContract(ctx, contract); err != nil {
		t.Fatalf("create: %v", err)
	}
	if contract.Version != 1 {
		t.Fatalf("new contract version = %d, want 1", contract.Version)
	}

	contract.Properties = models.ContractProperties{"care_type": "ganztag"}
	if err := s.UpdateContract(ctx, contract); err != nil {
		t.Fatalf("update: %v", err)
	}
	if contract.Version != 2 {
		t.Errorf("version after update = %d, want 2", contract.Version)
	}

	reloaded, err := s.FindContractByID(ctx, contract.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Version != 2 {
		t.Errorf("persisted version = %d, want 2", reloaded.Version)
	}
	if reloaded.Properties["care_type"] != "ganztag" {
		t.Errorf("the update did not persist: %v", reloaded.Properties)
	}
}

// The case that used to lose data: two writers read the same contract, both
// save. The second must be refused rather than silently overwriting the first.
func TestChildStore_UpdateContract_StaleWriteRejected(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewChildStore(db)
	ctx := context.Background()

	org := testutil.CreateTestOrganization(t, db, "Stale Org")
	section := testutil.CreateTestSection(t, db, "Nest", org.ID, false)
	child := testutil.CreateTestChild(t, db, "Stale", "Child", org.ID)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
		},
	}
	if err := s.CreateContract(ctx, contract); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Two editors load the same row.
	first, err := s.FindContractByID(ctx, contract.ID)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	second, err := s.FindContractByID(ctx, contract.ID)
	if err != nil {
		t.Fatalf("load second: %v", err)
	}

	// Editor one wins.
	first.Properties = models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"}
	if err := s.UpdateContract(ctx, first); err != nil {
		t.Fatalf("first update: %v", err)
	}

	// Editor two, still holding version 1, must be refused.
	second.Properties = models.ContractProperties{"care_type": "teilzeit"}
	err = s.UpdateContract(ctx, second)
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale update err = %v, want ErrVersionConflict", err)
	}

	// The winner's data must survive intact — including the supplement the loser
	// would have dropped.
	reloaded, err := s.FindContractByID(ctx, contract.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag (the stale write must not land)", reloaded.Properties["care_type"])
	}
	if reloaded.Properties["ndh"] != "ndh" {
		t.Errorf("the stale write dropped a supplement: %v", reloaded.Properties)
	}
	if reloaded.Version != 2 {
		t.Errorf("version = %d, want 2 — a refused write must not bump it", reloaded.Version)
	}

	// The rejected in-memory object keeps its original version, so a caller can
	// reload and retry without having to repair it.
	if second.Version != 1 {
		t.Errorf("refused write left version = %d on the caller's object, want 1", second.Version)
	}
}

func TestEmployeeStore_UpdateContract_StaleWriteRejected(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewEmployeeStore(db)
	ctx := context.Background()

	org := testutil.CreateTestOrganization(t, db, "Emp Stale Org")
	section := testutil.CreateTestSection(t, db, "Nest", org.ID, false)
	payPlan := testutil.CreateTestPayPlan(t, db, "TVöD", org.ID)
	employee := testutil.CreateTestEmployee(t, db, "Stale", "Employee", org.ID)

	contract := &models.EmployeeContract{
		EmployeeID:    employee.ID,
		PayPlanID:     payPlan.ID,
		Grade:         "S8a",
		Step:          2,
		WeeklyHours:   30,
		StaffCategory: "qualified",
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section.ID,
		},
	}
	if err := s.CreateContract(ctx, contract); err != nil {
		t.Fatalf("create: %v", err)
	}

	first, _ := s.FindContractByID(ctx, contract.ID)
	second, _ := s.FindContractByID(ctx, contract.ID)

	first.WeeklyHours = 35
	if err := s.UpdateContract(ctx, first); err != nil {
		t.Fatalf("first update: %v", err)
	}

	second.WeeklyHours = 20
	if err := s.UpdateContract(ctx, second); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale update err = %v, want ErrVersionConflict", err)
	}

	reloaded, _ := s.FindContractByID(ctx, contract.ID)
	if reloaded.WeeklyHours != 35 {
		t.Errorf("weekly_hours = %v, want 35 — the stale write must not land", reloaded.WeeklyHours)
	}
}

// The delete's version guard exists for the window between the service checking
// the version and the DELETE running: if another writer lands in that window, the
// guard must match nothing rather than destroy the change. That window cannot be
// hit deterministically from the HTTP tests — the service's own check catches
// every stale version before the store is reached — so the guard is pinned here,
// at the level where it is the only thing standing between a concurrent edit and
// a contract that no longer exists.
func TestChildStore_DeleteContract_StaleVersionRefused(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewChildStore(db)
	ctx := context.Background()

	org := testutil.CreateTestOrganization(t, db, "Delete Guard Org")
	section := testutil.CreateTestSection(t, db, "Nest", org.ID, false)
	child := testutil.CreateTestChild(t, db, "Delete", "Guard", org.ID)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"},
		},
	}
	if err := s.CreateContract(ctx, contract); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Someone else edits it, so version 1 is stale.
	contract.Properties = models.ContractProperties{"care_type": "halbtag"}
	if err := s.UpdateContract(ctx, contract); err != nil {
		t.Fatalf("update: %v", err)
	}

	stale := int64(1)
	if err := s.DeleteContract(ctx, contract.ID, &stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale delete err = %v, want ErrVersionConflict", err)
	}
	if _, err := s.FindContractByID(ctx, contract.ID); err != nil {
		t.Fatalf("the contract must survive a refused delete: %v", err)
	}

	// The current version deletes it.
	current := contract.Version
	if err := s.DeleteContract(ctx, contract.ID, &current); err != nil {
		t.Fatalf("delete with current version: %v", err)
	}
	if _, err := s.FindContractByID(ctx, contract.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, find err = %v, want ErrNotFound", err)
	}

	// And an unguarded delete (nil version) stays available for the importer's
	// delete-then-recreate, where there is no client version to honour.
	other := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section.ID,
		},
	}
	if err := s.CreateContract(ctx, other); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if err := s.DeleteContract(ctx, other.ID, nil); err != nil {
		t.Fatalf("unguarded delete: %v", err)
	}
}

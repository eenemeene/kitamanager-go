package service

// Edge cases inherited from the deleted batch endpoint.
//
// PUT /contracts/batch had 29 tests. Most of what they pinned is covered by
// contract_intent_test.go or is no longer expressible (a seam move derives both
// sides, so "two updated contracts overlap each other" cannot be requested). The
// guarantees below are the ones that *did* apply to the new endpoints and were
// not otherwise tested, consolidated here rather than duplicated per endpoint:
//
//   - ownership: an unknown contract, another owner's contract, or another
//     organization's owner is 404 on every write, not just on the old batch
//   - validation on a correction: period order, section organization, a
//     single-day period, and every field at once
//   - employee validation: pay plan existence and organization, a grade the plan
//     has no entry for, an invalid staff category
//   - a refused seam move leaves both contracts exactly as they were
//
// Table-driven over the endpoints, because the interesting property is that these
// rules hold for *all* of them — which is precisely what the per-endpoint
// duplicates could not express.

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// Every contract write refuses a contract that is not the addressed child's, and
// a child that is not the addressed organization's. The old batch endpoint tested
// this for itself; the rule belongs to all of them.
func TestChildContractIntents_OwnershipIsEnforced(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Owner Org")
	otherOrg := createTestOrganization(t, db, "Other Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Owned", "Child", org.ID)
	stranger := createTestChild(t, db, "Other", "Child", org.ID)

	contract := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: date(2026, 1, 1)},
		SectionID: section.ID,
	}}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	const missingID = 99999
	writes := map[string]func(contractID, childID, orgID uint) error{
		"correct": func(cid, chid, oid uint) error {
			_, err := svc.CorrectContract(ctx, cid, chid, oid,
				&models.ChildContractCorrectRequest{SectionID: models.OptOf(section.ID)})
			return err
		},
		"amend": func(cid, chid, oid uint) error {
			_, err := svc.AmendContract(ctx, cid, chid, oid,
				&models.ChildContractAmendRequest{EffectiveFrom: date(2026, 6, 1)})
			return err
		},
		"end": func(cid, chid, oid uint) error {
			_, err := svc.EndContract(ctx, cid, chid, oid,
				&models.ContractEndRequest{To: models.OptOf(date(2026, 12, 31))})
			return err
		},
		"delete": func(cid, chid, oid uint) error {
			return svc.DeleteContract(ctx, cid, chid, oid, nil)
		},
	}

	cases := map[string]struct{ contractID, childID, orgID uint }{
		"unknown contract":          {missingID, child.ID, org.ID},
		"contract of another child": {contract.ID, stranger.ID, org.ID},
		"child of another org":      {contract.ID, child.ID, otherOrg.ID},
	}

	for name, write := range writes {
		for caseName, tc := range cases {
			t.Run(name+"/"+caseName, func(t *testing.T) {
				err := write(tc.contractID, tc.childID, tc.orgID)
				assertAppError(t, err, 404, "")
			})
		}
	}
}

// The employee side of the same rule.
func TestEmployeeContractIntents_OwnershipIsEnforced(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Emp Owner Org")
	otherOrg := createTestOrganization(t, db, "Emp Other Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Owned", "Employee", org.ID)
	stranger := createTestEmployee(t, db, "Other", "Employee", org.ID)

	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID,
		date(2026, 1, 1), nil, "S8a", 3, 30)
	_ = section

	const missingID = 99999
	writes := map[string]func(contractID, employeeID, orgID uint) error{
		"correct": func(cid, eid, oid uint) error {
			_, err := svc.CorrectContract(ctx, cid, eid, oid,
				&models.EmployeeContractCorrectRequest{WeeklyHours: models.OptOf(35.0)})
			return err
		},
		"amend": func(cid, eid, oid uint) error {
			_, err := svc.AmendContract(ctx, cid, eid, oid,
				&models.EmployeeContractAmendRequest{EffectiveFrom: date(2026, 6, 1)})
			return err
		},
		"end": func(cid, eid, oid uint) error {
			_, err := svc.EndContract(ctx, cid, eid, oid,
				&models.ContractEndRequest{To: models.OptOf(date(2026, 12, 31))})
			return err
		},
		"delete": func(cid, eid, oid uint) error {
			return svc.DeleteContract(ctx, cid, eid, oid, nil)
		},
	}
	cases := map[string]struct{ contractID, employeeID, orgID uint }{
		"unknown contract":             {missingID, employee.ID, org.ID},
		"contract of another employee": {contract.ID, stranger.ID, org.ID},
		"employee of another org":      {contract.ID, employee.ID, otherOrg.ID},
	}
	for name, write := range writes {
		for caseName, tc := range cases {
			t.Run(name+"/"+caseName, func(t *testing.T) {
				assertAppError(t, write(tc.contractID, tc.employeeID, tc.orgID), 404, "")
			})
		}
	}
}

// Period and section validation on a correction, plus the two shapes that are
// legal and easy to break: a single-day contract, and changing everything at once.
func TestChildService_Correct_Validation(t *testing.T) {
	f := setupChildIntent(t, "CorrectValid", date(2026, 1, 1), timePtr(date(2026, 12, 31)))
	ctx := context.Background()

	otherOrg := createTestOrganization(t, f.db, "Foreign Org")
	foreignSection := createTestSection(t, f.db, "Foreign", otherOrg.ID, false)

	t.Run("to before from is rejected", func(t *testing.T) {
		_, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
			&models.ChildContractCorrectRequest{To: models.OptOf(date(2025, 12, 31))})
		assertAppError(t, err, 400, "")
	})

	t.Run("a section from another organization is rejected", func(t *testing.T) {
		_, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
			&models.ChildContractCorrectRequest{SectionID: models.OptOf(foreignSection.ID)})
		assertAppError(t, err, 400, "")
	})

	t.Run("a single-day period is allowed", func(t *testing.T) {
		day := date(2026, 5, 1)
		resp, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
			&models.ChildContractCorrectRequest{From: models.OptOf(day), To: models.OptOf(day)})
		if err != nil {
			t.Fatalf("a contract covering exactly one day must be allowed: %v", err)
		}
		if !resp.From.Equal(day) || resp.To == nil || !resp.To.Equal(day) {
			t.Errorf("from/to = %v/%v, want both %v", resp.From, resp.To, day)
		}
	})

	t.Run("every field at once", func(t *testing.T) {
		other := createTestSection(t, f.db, "Elefanten", f.orgID, false)
		resp, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
			&models.ChildContractCorrectRequest{
				From:       models.OptOf(date(2026, 2, 1)),
				To:         models.OptOf(date(2026, 11, 30)),
				SectionID:  models.OptOf(other.ID),
				Properties: models.OptOf(models.ContractProperties{"care_type": "teilzeit", "ndh": "ndh"}),
			})
		if err != nil {
			t.Fatalf("correct every field: %v", err)
		}
		if !resp.From.Equal(date(2026, 2, 1)) || resp.To == nil || !resp.To.Equal(date(2026, 11, 30)) ||
			resp.SectionID != other.ID || resp.Properties["care_type"] != "teilzeit" {
			t.Errorf("not all fields applied: %+v", resp)
		}
	})
}

// Employee-specific validation, including a grade the pay plan has no entry for —
// the check that stops a typo becoming a salary calculation failure months later.
func TestEmployeeService_Correct_Validation(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Emp Valid Org")
	createTestSection(t, db, "Nest", org.ID, false)
	otherOrg := createTestOrganization(t, db, "Emp Foreign Org")
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	foreignPlan := createTestPayPlanWithCoverage(t, db, "Foreign", otherOrg.ID)
	employee := createTestEmployee(t, db, "Valid", "Employee", org.ID)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID,
		date(2026, 1, 1), nil, "S8a", 3, 30)

	cases := map[string]*models.EmployeeContractCorrectRequest{
		"unknown pay plan":                {PayPlanID: models.OptOf(uint(99999))},
		"pay plan of another org":         {PayPlanID: models.OptOf(foreignPlan.ID)},
		"grade the plan has no entry for": {Grade: models.OptOf("S99a")},
		"invalid staff category":          {StaffCategory: models.OptOf("wizard")},
		"weekly hours above the maximum":  {WeeklyHours: models.OptOf(200.0)},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID, req)
			assertAppError(t, err, 400, "")
		})
	}
}

// A refused seam move must leave both contracts exactly as they were. The batch
// endpoint had a rollback test for this; here the refusal happens before any
// write, and the observable guarantee is the same.
func TestChildService_MoveBoundary_RefusalChangesNothing(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Rollback Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Rollback", "Child", org.ID)

	earlier := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: date(2026, 1, 1), To: timePtr(date(2026, 3, 31))},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}}
	later := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: date(2026, 4, 1), To: timePtr(date(2026, 6, 30))},
		SectionID: section.ID,
	}}
	for _, c := range []*models.ChildContract{earlier, later} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// A stale version on the later side.
	_, err := svc.MoveContractBoundary(ctx, child.ID, org.ID,
		&models.ContractBoundaryMoveRequest{
			EarlierID: earlier.ID, LaterID: later.ID, At: date(2026, 3, 1),
			EarlierVersion: earlier.Version, LaterVersion: later.Version + 99,
		})
	assertAppError(t, err, 412, "")

	// Neither side moved, and the earlier contract's properties are intact.
	var e, l models.ChildContract
	if err := db.First(&e, earlier.ID).Error; err != nil {
		t.Fatalf("reload earlier: %v", err)
	}
	if err := db.First(&l, later.ID).Error; err != nil {
		t.Fatalf("reload later: %v", err)
	}
	if e.To == nil || !e.To.Equal(date(2026, 3, 31)) {
		t.Errorf("earlier to = %v, want 2026-03-31 unchanged", e.To)
	}
	if !l.From.Equal(date(2026, 4, 1)) {
		t.Errorf("later from = %v, want 2026-04-01 unchanged", l.From)
	}
	if e.Properties["care_type"] != "halbtag" {
		t.Errorf("earlier properties = %v, want unchanged", e.Properties)
	}
	if e.Version != earlier.Version || l.Version != later.Version {
		t.Errorf("versions moved on a refused write: %d/%d", e.Version, l.Version)
	}
}

// Unused import guard for time, which the date helper hides.
var _ = time.Now

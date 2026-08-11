package service

// Regression tests for partial contract updates.
//
// The timeline boundary drag (frontend/src/components/contracts/contract-timeline.tsx,
// handleBoundaryChange) sends a batch of two entries carrying *one date each* —
// no properties, no section. applyChildContractFields / applyEmployeeContractFields
// assigned Properties unconditionally from the request, so a dates-only entry
// replaced them with nothing: care_type and every supplement (NdH, QM/MSS,
// Integration A/B) were stripped off both adjacent contracts, silently
// recomputing months of funding at the base rate. Nothing failed, nothing was
// logged, and the docs actively recommend that drag for correcting dates.
//
// The rule these tests pin: an omitted properties field means "leave alone", an
// explicit empty object still clears.

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// A dates-only boundary drag must not touch funding properties.
func TestChildService_BatchUpdate_DatesOnly_PreservesProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "Batch Props Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Batch", "Child", org.ID)

	from := today.AddDate(-1, 0, 0)
	boundary := today.AddDate(0, -6, 0)

	older := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: from, To: timePtr(boundary.AddDate(0, 0, -1))},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
	}}
	newer := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: boundary, To: nil},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "ganztag", "integration": "integration a"},
	}}
	if err := db.Create(older).Error; err != nil {
		t.Fatalf("seed older: %v", err)
	}
	if err := db.Create(newer).Error; err != nil {
		t.Fatalf("seed newer: %v", err)
	}

	// Exactly the payload handleBoundaryChange produces: one date per entry.
	moved := today.AddDate(0, -3, 0)
	if _, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: older.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{To: timePtr(moved.AddDate(0, 0, -1))}},
			{ID: newer.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{From: &moved}},
		},
	}); err != nil {
		t.Fatalf("boundary drag: %v", err)
	}

	list, _, err := svc.ListContracts(ctx, child.ID, org.ID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, c := range list {
		switch c.ID {
		case older.ID:
			if c.Properties["care_type"] != "halbtag" || c.Properties["ndh"] != "ndh" {
				t.Errorf("older contract lost properties: %v", c.Properties)
			}
			if c.To == nil || !c.To.Equal(moved.AddDate(0, 0, -1)) {
				t.Errorf("older contract To = %v, want %s", c.To, moved.AddDate(0, 0, -1).Format("2006-01-02"))
			}
		case newer.ID:
			if c.Properties["care_type"] != "ganztag" || c.Properties["integration"] != "integration a" {
				t.Errorf("newer contract lost properties: %v", c.Properties)
			}
			if !c.From.Equal(moved) {
				t.Errorf("newer contract From = %s, want %s", c.From.Format("2006-01-02"), moved.Format("2006-01-02"))
			}
		}
	}
}

// An explicit empty object must still clear properties — the edit dialog needs
// to be able to remove every supplement.
func TestChildService_BatchUpdate_ExplicitEmptyProperties_Clears(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "Clear Props Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Clear", "Child", org.ID)

	c := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: today.AddDate(-1, 0, 0), To: nil},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"},
	}}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: c.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				Properties: models.ContractProperties{},
			}},
		},
	}); err != nil {
		t.Fatalf("clear: %v", err)
	}

	list, _, _ := svc.ListContracts(ctx, child.ID, org.ID, 50, 0)
	for _, got := range list {
		if got.ID != c.ID {
			continue
		}
		if _, still := got.Properties["ndh"]; still {
			t.Errorf("explicit empty properties did not clear: %v", got.Properties)
		}
	}
}

// Same guarantee for employee contracts, whose timeline drag sends the same
// dates-only payload.
func TestEmployeeService_BatchUpdate_DatesOnly_PreservesProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "Emp Batch Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Emp", "Batch", org.ID)

	from := today.AddDate(-1, 0, 0)
	boundary := today.AddDate(0, -6, 0)

	older := &models.EmployeeContract{EmployeeID: employee.ID, PayPlanID: payPlan.ID,
		Grade: "S8a", Step: 2, WeeklyHours: 30, StaffCategory: string(models.StaffCategoryQualified),
		BaseContract: models.BaseContract{
			Period:     models.Period{From: from, To: timePtr(boundary.AddDate(0, 0, -1))},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"note": "first phase"},
		}}
	newer := &models.EmployeeContract{EmployeeID: employee.ID, PayPlanID: payPlan.ID,
		Grade: "S8a", Step: 3, WeeklyHours: 35, StaffCategory: string(models.StaffCategoryQualified),
		BaseContract: models.BaseContract{
			Period:     models.Period{From: boundary, To: nil},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"note": "second phase"},
		}}
	if err := db.Create(older).Error; err != nil {
		t.Fatalf("seed older: %v", err)
	}
	if err := db.Create(newer).Error; err != nil {
		t.Fatalf("seed newer: %v", err)
	}

	moved := today.AddDate(0, -3, 0)
	if _, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: older.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{To: timePtr(moved.AddDate(0, 0, -1))}},
			{ID: newer.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{From: &moved}},
		},
	}); err != nil {
		t.Fatalf("boundary drag: %v", err)
	}

	list, _, err := svc.ListContracts(ctx, employee.ID, org.ID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, c := range list {
		if len(c.Properties) == 0 {
			t.Errorf("employee contract %d lost properties on a dates-only drag", c.ID)
		}
		// The salary-bearing fields were already conditional; assert they held.
		if c.WeeklyHours == 0 || c.Grade == "" {
			t.Errorf("contract %d lost salary fields: hours=%v grade=%q", c.ID, c.WeeklyHours, c.Grade)
		}
	}
}

// The API contract for `to`: omitting it clears it. That is deliberate — it is
// how a contract is set back to ongoing (see BatchUpdateContracts_MakeOngoing) —
// but it means any partial caller MUST send `to` explicitly to keep it. Pinned
// here because getting this wrong broke the timeline drag.
func TestChildService_BatchUpdate_OmittedTo_ClearsIt(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "OmitTo Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "OmitTo", "Child", org.ID)

	closed := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: today.AddDate(-1, 0, 0), To: timePtr(today.AddDate(0, -6, 0))},
		SectionID: section.ID,
	}}
	if err := db.Create(closed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	newFrom := today.AddDate(-1, -1, 0)
	if _, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: closed.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{From: &newFrom}},
		},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := svc.GetContractByID(ctx, closed.ID, child.ID, org.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.To != nil {
		t.Errorf("To = %v, want nil — omitting `to` must clear it (documented contract)", got.To)
	}
}

// A three-contract timeline A|B|C: dragging the OLDER (A|B) boundary must work.
// The timeline now sends both dates for both contracts, so B keeps its `to` and
// does not collide with C. Sending only `from` for B wiped that `to`, made B
// ongoing and produced a 409 — dragging any but the newest boundary failed.
func TestChildService_BatchUpdate_ThreeContracts_DragOlderBoundary(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "Three Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Three", "Child", org.ID)

	aFrom := today.AddDate(-1, 0, 0)
	b1 := today.AddDate(0, -8, 0) // A|B boundary
	b2 := today.AddDate(0, -4, 0) // B|C boundary

	a := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: aFrom, To: timePtr(b1.AddDate(0, 0, -1))},
		SectionID: section.ID, Properties: models.ContractProperties{"care_type": "halbtag"}}}
	b := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: b1, To: timePtr(b2.AddDate(0, 0, -1))},
		SectionID: section.ID, Properties: models.ContractProperties{"care_type": "teilzeit"}}}
	c := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: b2, To: nil},
		SectionID: section.ID, Properties: models.ContractProperties{"care_type": "ganztag"}}}
	for _, x := range []*models.ChildContract{a, b, c} {
		if err := db.Create(x).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// The payload handleBoundaryChange now produces: both dates, both contracts.
	moved := today.AddDate(0, -10, 0)
	bTo := b2.AddDate(0, 0, -1)
	if _, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: a.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				From: &aFrom, To: timePtr(moved.AddDate(0, 0, -1))}},
			{ID: b.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				From: &moved, To: &bTo}},
		},
	}); err != nil {
		t.Fatalf("dragging the older boundary of a 3-contract timeline failed: %v", err)
	}

	list, _, err := svc.ListContracts(ctx, child.ID, org.ID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 contracts, got %d", len(list))
	}
	for _, x := range list {
		if len(x.Properties) == 0 {
			t.Errorf("contract %d lost properties during the drag", x.ID)
		}
		switch x.ID {
		case a.ID:
			if x.To == nil || !x.To.Equal(moved.AddDate(0, 0, -1)) {
				t.Errorf("A.To = %v, want %s", x.To, moved.AddDate(0, 0, -1).Format("2006-01-02"))
			}
		case b.ID:
			if !x.From.Equal(moved) {
				t.Errorf("B.From = %s, want %s", x.From.Format("2006-01-02"), moved.Format("2006-01-02"))
			}
			if x.To == nil || !x.To.Equal(bTo) {
				t.Errorf("B.To = %v, want %s — the third contract must not be collided with", x.To, bTo.Format("2006-01-02"))
			}
		case c.ID:
			if x.To != nil || !x.From.Equal(b2) {
				t.Errorf("C was modified: from=%s to=%v", x.From.Format("2006-01-02"), x.To)
			}
		}
	}
}

// Employee equivalent of the three-contract drag. The timeline component is
// shared between children and employees, so the same payload shape must work
// here — and the salary-bearing fields must survive untouched.
func TestEmployeeService_BatchUpdate_ThreeContracts_DragOlderBoundary(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()
	today := models.Today()

	org := createTestOrganization(t, db, "Emp Three Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Emp", "Three", org.ID)

	aFrom := today.AddDate(-1, 0, 0)
	b1 := today.AddDate(0, -8, 0)
	b2 := today.AddDate(0, -4, 0)

	mk := func(from time.Time, to *time.Time, hours float64, step int, note string) *models.EmployeeContract {
		return &models.EmployeeContract{EmployeeID: employee.ID, PayPlanID: payPlan.ID,
			Grade: "S8a", Step: step, WeeklyHours: hours,
			StaffCategory: string(models.StaffCategoryQualified),
			BaseContract: models.BaseContract{
				Period: models.Period{From: from, To: to}, SectionID: section.ID,
				Properties: models.ContractProperties{"note": note}}}
	}
	a := mk(aFrom, timePtr(b1.AddDate(0, 0, -1)), 30, 1, "phase a")
	b := mk(b1, timePtr(b2.AddDate(0, 0, -1)), 35, 2, "phase b")
	c := mk(b2, nil, 39, 3, "phase c")
	for _, x := range []*models.EmployeeContract{a, b, c} {
		if err := db.Create(x).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	moved := today.AddDate(0, -10, 0)
	bTo := b2.AddDate(0, 0, -1)
	if _, err := svc.BatchUpdateContracts(ctx, employee.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: a.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{
				From: &aFrom, To: timePtr(moved.AddDate(0, 0, -1))}},
			{ID: b.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{
				From: &moved, To: &bTo}},
		},
	}); err != nil {
		t.Fatalf("dragging the older boundary of a 3-contract employee timeline failed: %v", err)
	}

	list, _, err := svc.ListContracts(ctx, employee.ID, org.ID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 contracts, got %d", len(list))
	}
	for _, x := range list {
		if len(x.Properties) == 0 {
			t.Errorf("contract %d lost properties", x.ID)
		}
		if x.WeeklyHours == 0 || x.Grade == "" || x.Step == 0 {
			t.Errorf("contract %d lost salary fields: hours=%v grade=%q step=%d",
				x.ID, x.WeeklyHours, x.Grade, x.Step)
		}
		if x.ID == b.ID && (x.To == nil || !x.To.Equal(bTo)) {
			t.Errorf("B.To = %v, want %s", x.To, bTo.Format("2006-01-02"))
		}
	}
}

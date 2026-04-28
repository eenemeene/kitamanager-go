package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ===========================================================================
// H3: employee contract create-time birthdate guard
// ===========================================================================

func TestEmployeeService_CreateContract_BeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	emp.Birthdate = time.Date(2005, 5, 15, 0, 0, 0, 0, time.UTC)
	if err := db.Save(emp).Error; err != nil {
		t.Fatalf("failed to save employee: %v", err)
	}

	payPlan := createTestPayPlanWithCoverage(t, db, "PP", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// Contract starting before birthdate must fail.
	_, err := svc.CreateContract(ctx, emp.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          time.Date(1989, 1, 1, 0, 0, 0, 0, time.UTC),
		SectionID:     sectionID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})
	if err == nil {
		t.Fatal("expected error for contract from before birthdate, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Contract starting on the birthdate is accepted.
	c, err := svc.CreateContract(ctx, emp.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          emp.Birthdate,
		SectionID:     sectionID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})
	if err != nil {
		t.Fatalf("expected success for contract on birthdate, got %v", err)
	}
	if c == nil {
		t.Fatal("expected contract response, got nil")
	}
}

func TestEmployeeService_CreateContract_ToBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	emp.Birthdate = time.Date(2005, 5, 15, 0, 0, 0, 0, time.UTC)
	db.Save(emp)

	payPlan := createTestPayPlanWithCoverage(t, db, "PP", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// To before birthdate is structurally invalid (forces From even earlier or
	// inverts the period). Reject explicitly so the diagnostic is precise.
	to := time.Date(1989, 12, 31, 0, 0, 0, 0, time.UTC)
	from := time.Date(1989, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, emp.ID, org.ID, &models.EmployeeContractCreateRequest{
		From:          from,
		To:            &to,
		SectionID:     sectionID,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   40,
		PayPlanID:     payPlan.ID,
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// ===========================================================================
// H4: birthdate is also enforced on update paths (not just create)
// ===========================================================================

func TestChildService_UpdateContract_FromMovedBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	child.Birthdate = time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// Create a future-starting contract (so amend mode is "in place").
	from := time.Now().UTC().AddDate(0, 0, 30).Truncate(24 * time.Hour)
	created, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		From:      from,
		SectionID: sectionID,
	})
	if err != nil {
		t.Fatalf("setup: create contract: %v", err)
	}

	// Now move From earlier than birthdate via Update — must be rejected.
	bad := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.UpdateContract(ctx, created.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		From: &bad,
	})
	if err == nil {
		t.Fatal("expected error when moving From earlier than birthdate via Update, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// And the persisted From was not corrupted.
	var got models.ChildContract
	if err := db.First(&got, created.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !got.From.Equal(from) {
		t.Errorf("From persisted = %v, want unchanged %v", got.From, from)
	}
}

func TestChildService_BatchUpdateContracts_FromMovedBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	child.Birthdate = time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)
	db.Save(child)
	sectionID := getDefaultSection(t, db, org.ID).ID

	a := mustCreateChildContract(t, ctx, svc, child.ID, org.ID, sectionID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		datePtr(2025, time.June, 30))
	b := mustCreateChildContract(t, ctx, svc, child.ID, org.ID, sectionID,
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), nil)

	bad := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: a.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{From: &bad}},
			{ID: b.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{}},
		},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}

	// Rollback semantics: neither contract changed.
	var aGot, bGot models.ChildContract
	db.First(&aGot, a.ID)
	db.First(&bGot, b.ID)
	if !aGot.From.Equal(a.From) {
		t.Errorf("contract A From = %v, want unchanged %v", aGot.From, a.From)
	}
	if bGot.From.IsZero() || !bGot.From.Equal(b.From) {
		t.Errorf("contract B From = %v, want unchanged %v", bGot.From, b.From)
	}
}

func TestEmployeeService_UpdateContract_FromMovedBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	emp.Birthdate = time.Date(2005, 5, 15, 0, 0, 0, 0, time.UTC)
	db.Save(emp)
	payPlan := createTestPayPlanWithCoverage(t, db, "PP", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// Future-starting contract → in-place update path.
	from := time.Now().UTC().AddDate(0, 0, 30).Truncate(24 * time.Hour)
	created, err := svc.CreateContract(ctx, emp.ID, org.ID, &models.EmployeeContractCreateRequest{
		From: from, SectionID: sectionID, StaffCategory: "qualified",
		Grade: "S8a", Step: 1, WeeklyHours: 40, PayPlanID: payPlan.ID,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	bad := time.Date(1989, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.UpdateContract(ctx, created.ID, emp.ID, org.ID, &models.EmployeeContractUpdateRequest{
		From: &bad,
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_BatchUpdateContracts_FromMovedBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	emp.Birthdate = time.Date(2005, 5, 15, 0, 0, 0, 0, time.UTC)
	db.Save(emp)
	payPlan := createTestPayPlanWithCoverage(t, db, "PP", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	a := mustCreateEmployeeContract(t, ctx, svc, emp.ID, org.ID, sectionID, payPlan.ID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		datePtr(2025, time.June, 30))

	bad := time.Date(1989, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.BatchUpdateContracts(ctx, emp.ID, org.ID, &models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: a.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{From: &bad}},
		},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// Whitebox: the helper itself must be timezone-tolerant so a from sent at
// midnight in a non-UTC zone (semantically the same calendar day as a
// UTC-stored birthdate) is not rejected.
func TestValidateContractDatesAfterBirthdate_TimezoneTolerance(t *testing.T) {
	birthdate := time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)

	// Same calendar day, but wall-clock midnight in a +02:00 zone is
	// 22:00 UTC the previous day. Without truncation, raw time.Before
	// would say from < birthdate. The truncated comparison must agree
	// "same day".
	cest := time.FixedZone("CEST", 2*60*60)
	from := time.Date(2022, 6, 15, 0, 0, 0, 0, cest)
	if err := validateContractDatesAfterBirthdate(from, nil, birthdate); err != nil {
		t.Errorf("same calendar date in CEST should not be rejected, got %v", err)
	}

	// Calendar day strictly before birthdate is rejected even in UTC.
	beforeUTC := time.Date(2022, 6, 14, 0, 0, 0, 0, time.UTC)
	if err := validateContractDatesAfterBirthdate(beforeUTC, nil, birthdate); !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for date before birthdate, got %v", err)
	}
}

// ===========================================================================
// M7: amendContractTx uses the caller's `today`, not its own time.Now()
// ===========================================================================

// amendContractTx receives `today` from the caller. This test passes a fixed
// value and verifies (a) the new contract's From equals exactly that today,
// (b) the old contract's To equals exactly today-1, with no drift caused by
// re-reading time.Now() inside the helper (the bug M7 closes).
func TestAmendContractTx_UsesCallerToday(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	child.Birthdate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Save(child)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// Create a contract that started yesterday so a real Update goes into
	// amend mode — that path passes today=models.TruncateToDate(time.Now())
	// to amendContractTx. We then read back what the helper produced and
	// assert today/yesterday agree to within one day.
	yesterday := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	old, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		From: yesterday, SectionID: sectionID,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	updated, err := svc.UpdateContract(ctx, old.ID, child.ID, org.ID, &models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "ganztag"},
	})
	if err != nil {
		t.Fatalf("amend update: %v", err)
	}

	// The new contract's From and the closure of the old contract's To must
	// be exactly one day apart — i.e. the helper used a single `today`.
	var oldRow models.ChildContract
	if err := db.First(&oldRow, old.ID).Error; err != nil {
		t.Fatalf("read old: %v", err)
	}
	if oldRow.To == nil {
		t.Fatal("old contract To not set after amend")
	}
	gap := updated.From.Sub(models.TruncateToDate(*oldRow.To))
	if gap != 24*time.Hour {
		t.Errorf("new.From - old.To = %v, want 24h (today vs yesterday consistency)", gap)
	}
}

// ===========================================================================
// H1+M6: DEFERRABLE EXCLUDE constraint at the DB layer
// ===========================================================================

// Direct db.Create bypasses the service layer's pre-check entirely. The DB
// EXCLUDE is the only thing that prevents corruption here. Without the
// constraint this test passes trivially; with it the second insert errors.
func TestChildContract_DBExclusionConstraint_DirectInsertBypass(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	first := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(2025, time.June, 30),
			},
		},
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Overlapping: starts on the last day of `first`. Under the inclusive
	// `[from,to]` domain semantics this is an overlap, and the constraint's
	// daterange(from, COALESCE(to+1,'infinity'),'[)') formulation must catch
	// it. Constraint is DEFERRABLE INITIALLY DEFERRED — inside an implicit
	// (autocommit) statement-level write the violation surfaces immediately.
	overlap := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
				To:   datePtr(2025, time.December, 31),
			},
		},
	}
	err := db.Create(overlap).Error
	if err == nil {
		t.Fatal("expected exclusion-violation, got nil — constraint not in place?")
	}
	if !store.IsExclusionViolation(err) {
		t.Errorf("expected sqlstate 23P01, got %v", err)
	}
}

// Companion test for employees — same shape, different table.
func TestEmployeeContract_DBExclusionConstraint_DirectInsertBypass(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID
	payPlan := createTestPayPlan(t, db, "PP", org.ID)

	first := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(2025, time.June, 30),
			},
		},
		StaffCategory: "qualified",
		PayPlanID:     payPlan.ID,
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("first create: %v", err)
	}

	overlap := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period: models.Period{
				From: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
				To:   datePtr(2025, time.December, 31),
			},
		},
		StaffCategory: "qualified",
		PayPlanID:     payPlan.ID,
	}
	err := db.Create(overlap).Error
	if err == nil {
		t.Fatal("expected exclusion-violation, got nil")
	}
	if !store.IsExclusionViolation(err) {
		t.Errorf("expected sqlstate 23P01, got %v", err)
	}
}

// Concurrent service-level CreateContract calls for the same child: the race
// the application-level pre-check cannot catch under READ COMMITTED. With the
// DB EXCLUDE constraint exactly one wins, the other gets a 409; without it
// both would persist.
//
// Skipped if -short — the goroutine spin-up matters less than testing it
// thoroughly when the suite runs full.
func TestChildService_CreateContract_ConcurrentRaceLosesExactlyOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}
	db := setupTestDB(t)
	svc := createChildService(db)
	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// Both attempts request the exact same period. With overlap protection
	// only one can persist; the other must fail with 409. This is the bug
	// the EXCLUDE constraint exists to close.
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := datePtr(2025, time.December, 31)

	const N = 8
	var wg sync.WaitGroup
	results := make([]error, N)
	start := make(chan struct{})
	for i := range N {
		wg.Go(func() {
			<-start
			_, err := svc.CreateContract(context.Background(), child.ID, org.ID, &models.ChildContractCreateRequest{
				From: from, To: to, SectionID: sectionID,
			})
			results[i] = err
		})
	}
	close(start)
	wg.Wait()

	successes, conflicts, others := 0, 0, []error{}
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, apperror.ErrConflict):
			conflicts++
		default:
			others = append(others, err)
		}
	}
	if len(others) > 0 {
		t.Fatalf("unexpected error categories: %v", others)
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 successful create across %d racers, got %d", N, successes)
	}
	if conflicts != N-1 {
		t.Errorf("expected %d conflicts, got %d", N-1, conflicts)
	}

	// Persisted state must show exactly one row.
	var count int64
	db.Model(&models.ChildContract{}).Where("child_id = ?", child.ID).Count(&count)
	if count != 1 {
		t.Errorf("persisted child_contracts rows = %d, want 1", count)
	}
}

// Same race for employees.
func TestEmployeeService_CreateContract_ConcurrentRaceLosesExactlyOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "PP", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := datePtr(2025, time.December, 31)

	const N = 8
	var wg sync.WaitGroup
	results := make([]error, N)
	start := make(chan struct{})
	for i := range N {
		wg.Go(func() {
			<-start
			_, err := svc.CreateContract(context.Background(), emp.ID, org.ID, &models.EmployeeContractCreateRequest{
				From: from, To: to, SectionID: sectionID,
				StaffCategory: "qualified", Grade: "S8a", Step: 1,
				WeeklyHours: 40, PayPlanID: payPlan.ID,
			})
			results[i] = err
		})
	}
	close(start)
	wg.Wait()

	successes, conflicts := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, apperror.ErrConflict):
			conflicts++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d", successes)
	}
	if conflicts != N-1 {
		t.Errorf("expected %d conflicts, got %d", N-1, conflicts)
	}
}

// BatchUpdateContracts must succeed when swapping adjacent ranges — phase 1
// transiently overlaps, phase 2 fixes it, and the deferred constraint sees a
// consistent state at COMMIT. Without DEFERRABLE INITIALLY DEFERRED this
// would fail on phase 1's first save.
func TestChildService_BatchUpdateContracts_DeferredConstraintAllowsSwap(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	child := createTestChild(t, db, "Lia", "Becker", org.ID)
	child.Birthdate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Save(child)
	sectionID := getDefaultSection(t, db, org.ID).ID

	// A: Jan–Jun, B: Aug–Dec, with a one-month gap so the desired final
	// state is A: Jan–Jul, B: Jul-Aug→Dec — that is, A grows and B retracts;
	// the deliberately-tricky case is the per-row save order: extending A
	// first creates a transient overlap with B until B's From is shifted.
	a := mustCreateChildContract(t, ctx, svc, child.ID, org.ID, sectionID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		datePtr(2025, time.June, 30))
	b := mustCreateChildContract(t, ctx, svc, child.ID, org.ID, sectionID,
		time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		datePtr(2025, time.December, 31))

	// Final state: A ends 2025-07-31, B starts 2025-08-01.
	aNewTo := datePtr(2025, time.July, 31)
	bNewFrom := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	// Note phase-1 save order extends A first (A's new To = 2025-07-31 still
	// doesn't overlap B's untouched From of 2025-08-01, so this batch alone
	// wouldn't actually trip the constraint — see the next subtest for the
	// stronger swap that requires DEFERRED).
	_, err := svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: a.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{To: aNewTo}},
			{ID: b.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{From: &bNewFrom}},
		},
	})
	if err != nil {
		t.Fatalf("batch update should succeed, got %v", err)
	}

	// Now the stronger case: extend A INTO B's range simultaneously with
	// shifting B forward. Phase 1 saves A first → transient overlap.
	// Without DEFERRABLE the very first INSERT/UPDATE fails. With it,
	// commit sees the reconciled state and succeeds.
	aPushTo := datePtr(2025, time.September, 30)              // A grows past B's old From
	bPushFrom := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC) // B retreats past A's new To
	_, err = svc.BatchUpdateContracts(ctx, child.ID, org.ID, &models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: a.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{To: aPushTo}},
			{ID: b.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{From: &bPushFrom}},
		},
	})
	if err != nil {
		t.Fatalf("DEFERRED batch swap should succeed, got %v", err)
	}

	var aRow, bRow models.ChildContract
	db.First(&aRow, a.ID)
	db.First(&bRow, b.ID)
	if aRow.To == nil || !aRow.To.Equal(*aPushTo) {
		t.Errorf("A.To = %v, want %v", aRow.To, aPushTo)
	}
	if !bRow.From.Equal(bPushFrom) {
		t.Errorf("B.From = %v, want %v", bRow.From, bPushFrom)
	}
}

// mapContractDeferredOverlap is the bridge between the raw 23P01 sqlstate the
// driver returns and the apperror.Conflict the API produces. Make sure the
// mapping is exact and other errors pass through.
func TestMapContractDeferredOverlap_Mapping(t *testing.T) {
	t.Run("23P01 maps to Conflict", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23P01", Message: "conflicting key value"}
		mapped := mapContractDeferredOverlap(err)
		if !errors.Is(mapped, apperror.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", mapped)
		}
	})
	t.Run("nil passes through", func(t *testing.T) {
		if mapContractDeferredOverlap(nil) != nil {
			t.Error("nil must remain nil")
		}
	})
	t.Run("other pg errors pass through unchanged", func(t *testing.T) {
		fk := &pgconn.PgError{Code: "23503"}
		if got := mapContractDeferredOverlap(fk); got != fk {
			t.Errorf("FK violation should pass through, got %v", got)
		}
	})
	t.Run("apperror.Conflict from inner pre-check passes through", func(t *testing.T) {
		// The closure-side ErrPeriodOverlap path returns apperror.Conflict
		// already; mapper must not double-wrap.
		inner := apperror.Conflict("already converted")
		got := mapContractDeferredOverlap(inner)
		if got != inner {
			t.Errorf("inner Conflict must pass through unchanged, got %v", got)
		}
	})
}

// ===========================================================================
// M9: DB-level CHECK constraints on employee_contracts
// ===========================================================================

func TestEmployeeContract_DBCheckConstraint_StaffCategory(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID
	payPlan := createTestPayPlan(t, db, "PP", org.ID)

	bad := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		StaffCategory: "vp_of_culture", // not in the enum
		PayPlanID:     payPlan.ID,
	}
	err := db.Create(bad).Error
	if err == nil {
		t.Fatal("expected CHECK violation for invalid staff_category")
	}
	if !isCheckViolation(err) {
		t.Errorf("expected sqlstate 23514, got %v", err)
	}
}

func TestEmployeeContract_DBCheckConstraint_StepBounds(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID
	payPlan := createTestPayPlan(t, db, "PP", org.ID)

	cases := []struct {
		name string
		step int
		from time.Time
	}{
		{"step negative", -1, time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"step above bound", 11, time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &models.EmployeeContract{
				EmployeeID: emp.ID,
				BaseContract: models.BaseContract{
					SectionID: sectionID,
					Period:    models.Period{From: tc.from},
				},
				StaffCategory: "qualified",
				Step:          tc.step,
				PayPlanID:     payPlan.ID,
			}
			err := db.Create(c).Error
			if err == nil {
				t.Fatal("expected CHECK violation, got nil")
			}
			if !isCheckViolation(err) {
				t.Errorf("expected sqlstate 23514, got %v", err)
			}
		})
	}

	// Boundary values 0 and 10 are accepted. Bounded, disjoint date ranges so
	// neither the EXCLUDE constraint nor open-ended-overlap is the test's gate.
	bounds := []struct {
		step int
		from time.Time
		to   *time.Time
	}{
		{0, time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC), datePtr(2030, time.December, 31)},
		{10, time.Date(2031, time.January, 1, 0, 0, 0, 0, time.UTC), datePtr(2031, time.December, 31)},
	}
	for _, ok := range bounds {
		c := &models.EmployeeContract{
			EmployeeID: emp.ID,
			BaseContract: models.BaseContract{
				SectionID: sectionID,
				Period:    models.Period{From: ok.from, To: ok.to},
			},
			StaffCategory: "qualified",
			Step:          ok.step,
			PayPlanID:     payPlan.ID,
		}
		if err := db.Create(c).Error; err != nil {
			t.Errorf("step=%d should be accepted, got %v", ok.step, err)
		}
	}
}

func TestEmployeeContract_DBCheckConstraint_WeeklyHoursBounds(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	emp := createTestEmployee(t, db, "Anna", "Becker", org.ID)
	sectionID := getDefaultSection(t, db, org.ID).ID
	payPlan := createTestPayPlan(t, db, "PP", org.ID)

	cases := []struct {
		name  string
		hours float64
	}{
		{"negative hours", -0.1},
		{"above 168", 168.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &models.EmployeeContract{
				EmployeeID: emp.ID,
				BaseContract: models.BaseContract{
					SectionID: sectionID,
					Period:    models.Period{From: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
				},
				StaffCategory: "qualified",
				WeeklyHours:   tc.hours,
				PayPlanID:     payPlan.ID,
			}
			err := db.Create(c).Error
			if err == nil {
				t.Fatal("expected CHECK violation, got nil")
			}
			if !isCheckViolation(err) {
				t.Errorf("expected sqlstate 23514, got %v", err)
			}
		})
	}

	// Boundary value 168 is accepted.
	c := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)},
		},
		StaffCategory: "qualified",
		WeeklyHours:   168,
		PayPlanID:     payPlan.ID,
	}
	if err := db.Create(c).Error; err != nil {
		t.Errorf("168 hours/week should be accepted, got %v", err)
	}
}

// ===========================================================================
// helpers
// ===========================================================================

func mustCreateChildContract(t *testing.T, ctx context.Context, svc *ChildService, childID, orgID, sectionID uint, from time.Time, to *time.Time) *models.ChildContractResponse {
	t.Helper()
	resp, err := svc.CreateContract(ctx, childID, orgID, &models.ChildContractCreateRequest{
		From: from, To: to, SectionID: sectionID,
	})
	if err != nil {
		t.Fatalf("setup: create child contract %v..%v: %v", from, to, err)
	}
	return resp
}

func mustCreateEmployeeContract(t *testing.T, ctx context.Context, svc *EmployeeService, empID, orgID, sectionID, payPlanID uint, from time.Time, to *time.Time) *models.EmployeeContractResponse {
	t.Helper()
	resp, err := svc.CreateContract(ctx, empID, orgID, &models.EmployeeContractCreateRequest{
		From: from, To: to, SectionID: sectionID,
		StaffCategory: "qualified", Grade: "S8a", Step: 1,
		WeeklyHours: 40, PayPlanID: payPlanID,
	})
	if err != nil {
		t.Fatalf("setup: create employee contract %v..%v: %v", from, to, err)
	}
	return resp
}

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

package service

// Tests for the intent-based contract operations.
//
// These pin the guarantees the old PUT/batch surface could not make:
//
//   - a field omitted from a correction is left alone (the old PUT cleared `to`
//     and `properties` by omission, which stripped care_type and every funding
//     supplement off a contract and silently rebilled it at the base rate)
//   - an amendment starts where the caller says, including in the past, so a
//     late Bescheid is one call instead of amend-then-drag
//   - a boundary move takes one date and derives both sides, so it cannot clear
//     the neighbour's end date or wipe its properties
//   - null is a value, distinct from absent, everywhere it means something
//
// Both happy and unhappy paths: the point of naming intents is that the illegal
// ones become nameable, so they are asserted rather than left to the DB.

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// assertAppError checks the HTTP status and, when given, the machine-readable
// error code — a 409 for an overlap and a 409 for a stale write need different
// handling in the client, so the code is part of the contract.
func assertAppError(t *testing.T, err error, wantStatus int, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is not an *apperror.AppError: %v", err)
	}
	if appErr.Code != wantStatus {
		t.Errorf("status = %d, want %d (err: %v)", appErr.Code, wantStatus, err)
	}
	if wantCode != "" && appErr.GetErrorCode() != wantCode {
		t.Errorf("error code = %q, want %q (err: %v)", appErr.GetErrorCode(), wantCode, err)
	}
}

// childIntentFixture is a child with one open-ended contract carrying funding
// properties, which is the state most of these operations start from.
type childIntentFixture struct {
	orgID     uint
	childID   uint
	sectionID uint
	contract  *models.ChildContract
	svc       *ChildService
	db        *gorm.DB
}

func setupChildIntent(t *testing.T, name string, from time.Time, to *time.Time) *childIntentFixture {
	t.Helper()
	db := setupTestDB(t)
	svc := createChildService(db)

	org := createTestOrganization(t, db, name+" Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, name, "Child", org.ID)

	contract := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: from, To: to},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"},
	}}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	return &childIntentFixture{org.ID, child.ID, section.ID, contract, svc, db}
}

// ---------------------------------------------------------------- correct ----

// The core promise: omitting a field leaves it exactly as it was. This is the
// payload the kanban board sends to move a child between sections.
func TestChildService_Correct_OmittedFieldsUntouched(t *testing.T) {
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Correct", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)
	ctx := context.Background()

	other := createTestSection(t, f.db, "Elefanten", f.orgID, false)

	resp, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractCorrectRequest{SectionID: models.OptOf(other.ID)})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}

	if resp.SectionID != other.ID {
		t.Errorf("section_id = %d, want %d", resp.SectionID, other.ID)
	}
	if resp.To == nil || !resp.To.Equal(end) {
		t.Errorf("to = %v, want %v — an omitted field must not be cleared", resp.To, end)
	}
	if resp.Properties["care_type"] != "ganztag" || resp.Properties["ndh"] != "ndh" {
		t.Errorf("properties = %v, want care_type and ndh preserved", resp.Properties)
	}
}

// Null is the only way to clear, and it has to actually work.
func TestChildService_Correct_ExplicitNullClears(t *testing.T) {
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Clear", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)
	ctx := context.Background()

	resp, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractCorrectRequest{
			To:         models.OptNull[time.Time](),
			Properties: models.OptNull[models.ContractProperties](),
		})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	if resp.To != nil {
		t.Errorf("to = %v, want nil", resp.To)
	}
	if len(resp.Properties) != 0 {
		t.Errorf("properties = %v, want empty", resp.Properties)
	}
}

// from and section_id have no meaningful null: a contract always has both. The
// request is refused rather than quietly ignored, so a client bug surfaces.
func TestChildService_Correct_NullRejectedWhereMeaningless(t *testing.T) {
	f := setupChildIntent(t, "NullReject", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)
	ctx := context.Background()

	for name, req := range map[string]*models.ChildContractCorrectRequest{
		"from":       {From: models.OptNull[time.Time]()},
		"section_id": {SectionID: models.OptNull[uint]()},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID, req)
			assertAppError(t, err, 400, "")
		})
	}
}

// A correction may touch a contract that already ended — correcting history is
// the whole point, and the per-field audit diff is what makes it safe. The old
// PUT refused this with "cannot update a contract that has already ended",
// which is why fixing a typo in last year's dates needed the batch endpoint.
func TestChildService_Correct_EndedContractAllowed(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	ended := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Ended", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &ended)
	ctx := context.Background()

	// The old surface on the same contract, for contrast.
	if _, err := f.svc.UpdateContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractUpdateRequest{To: &ended}); err == nil {
		t.Error("UpdateContract accepted an ended contract; this test's premise is stale")
	}

	corrected := time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)
	resp, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractCorrectRequest{To: models.OptOf(corrected)})
	if err != nil {
		t.Fatalf("correct an ended contract: %v", err)
	}
	if resp.To == nil || !resp.To.Equal(corrected) {
		t.Errorf("to = %v, want %v", resp.To, corrected)
	}
}

// Extending a contract over its successor is a conflict the client can act on
// (reload and pick a different date), so it gets the overlap code rather than a
// bare 409.
func TestChildService_Correct_OverlapReportsContractOverlap(t *testing.T) {
	boundary := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Overlap", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		timePtr(boundary.AddDate(0, 0, -1)))
	ctx := context.Background()

	successor := &models.ChildContract{ChildID: f.childID, BaseContract: models.BaseContract{
		Period:    models.Period{From: boundary},
		SectionID: f.sectionID,
	}}
	if err := f.db.Create(successor).Error; err != nil {
		t.Fatalf("seed successor: %v", err)
	}

	_, err := f.svc.CorrectContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractCorrectRequest{To: models.OptOf(boundary.AddDate(0, 1, 0))})
	assertAppError(t, err, 409, apperror.CodeContractConflict)
}

// ------------------------------------------------------------------ amend ----

// The gap this whole redesign started from: recording a change that took effect
// in the past, in one call. The old PUT ignored the request's `from` and always
// started the successor today.
func TestChildService_Amend_BackdatedInOneCall(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	f := setupChildIntent(t, "Backdate", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)
	ctx := context.Background()

	seam := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	resp, err := f.svc.AmendContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractAmendRequest{
			EffectiveFrom: seam,
			Properties:    models.OptOf(models.ContractProperties{"care_type": "halbtag"}),
		})
	if err != nil {
		t.Fatalf("amend: %v", err)
	}

	if !resp.Created.From.Equal(seam) {
		t.Errorf("successor from = %v, want %v (the requested date, not today)", resp.Created.From, seam)
	}
	wantClose := seam.AddDate(0, 0, -1)
	if resp.Closed.To == nil || !resp.Closed.To.Equal(wantClose) {
		t.Errorf("predecessor to = %v, want %v", resp.Closed.To, wantClose)
	}
	if resp.Created.Properties["care_type"] != "halbtag" {
		t.Errorf("successor care_type = %v, want halbtag", resp.Created.Properties["care_type"])
	}
	if resp.Closed.ID == resp.Created.ID {
		t.Error("closed and created are the same contract")
	}
}

// The common case: a new Bescheid changes the care type and nothing else, so
// everything unmentioned has to carry over — including the supplements that
// decide what the Kita is paid.
func TestChildService_Amend_InheritsOmittedFields(t *testing.T) {
	end := time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Inherit", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)
	ctx := context.Background()

	resp, err := f.svc.AmendContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractAmendRequest{EffectiveFrom: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("amend: %v", err)
	}

	if resp.Created.Properties["care_type"] != "ganztag" || resp.Created.Properties["ndh"] != "ndh" {
		t.Errorf("successor properties = %v, want the predecessor's", resp.Created.Properties)
	}
	if resp.Created.SectionID != f.sectionID {
		t.Errorf("successor section = %d, want %d", resp.Created.SectionID, f.sectionID)
	}
	if resp.Created.To == nil || !resp.Created.To.Equal(end) {
		t.Errorf("successor to = %v, want the predecessor's %v", resp.Created.To, end)
	}
}

// An explicit null on `to` means open-ended, which is different from inheriting.
func TestChildService_Amend_ToNullMakesSuccessorOpenEnded(t *testing.T) {
	end := time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "OpenEnd", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)
	ctx := context.Background()

	resp, err := f.svc.AmendContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractAmendRequest{
			EffectiveFrom: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			To:            models.OptNull[time.Time](),
		})
	if err != nil {
		t.Fatalf("amend: %v", err)
	}
	if resp.Created.To != nil {
		t.Errorf("successor to = %v, want nil", resp.Created.To)
	}
}

// Amending at or before the contract's own start would close it with an empty or
// negative period. That request is a correction, and the error says so.
func TestChildService_Amend_SeamNotAfterStartRejected(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "SeamStart", from, nil)
	ctx := context.Background()

	for name, seam := range map[string]time.Time{
		"same day as start": from,
		"before start":      from.AddDate(0, 0, -1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.svc.AmendContract(ctx, f.contract.ID, f.childID, f.orgID,
				&models.ChildContractAmendRequest{EffectiveFrom: seam})
			assertAppError(t, err, 400, "")
		})
	}
}

// Amending a contract that already ended would *extend* it to the day before the
// seam — silently covering months it never covered, which for a child contract
// means months that were already billed.
func TestChildService_Amend_AlreadyEndedRejected(t *testing.T) {
	ended := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "AmendEnded", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &ended)
	ctx := context.Background()

	_, err := f.svc.AmendContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ChildContractAmendRequest{EffectiveFrom: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)})
	assertAppError(t, err, 400, "")
}

// -------------------------------------------------------------------- end ----

func TestChildService_End_SetsThenReopens(t *testing.T) {
	f := setupChildIntent(t, "End", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)
	ctx := context.Background()

	leaving := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	resp, err := f.svc.EndContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ContractEndRequest{To: models.OptOf(leaving)})
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if resp.To == nil || !resp.To.Equal(leaving) {
		t.Fatalf("to = %v, want %v", resp.To, leaving)
	}

	// Undoing a departure: an explicit null, not an omission.
	resp, err = f.svc.EndContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ContractEndRequest{To: models.OptNull[time.Time]()})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if resp.To != nil {
		t.Errorf("to = %v, want nil after reopening", resp.To)
	}
}

// An absent `to` is not a decision. The old surface read it as "make ongoing",
// which is how an unrelated partial update erased a departure date.
func TestChildService_End_MissingToRejected(t *testing.T) {
	f := setupChildIntent(t, "EndMissing", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)

	_, err := f.svc.EndContract(context.Background(), f.contract.ID, f.childID, f.orgID,
		&models.ContractEndRequest{})
	assertAppError(t, err, 400, "")
}

func TestChildService_End_ReopeningOverSuccessorConflicts(t *testing.T) {
	boundary := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	f := setupChildIntent(t, "Reopen", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		timePtr(boundary.AddDate(0, 0, -1)))
	ctx := context.Background()

	successor := &models.ChildContract{ChildID: f.childID, BaseContract: models.BaseContract{
		Period:    models.Period{From: boundary},
		SectionID: f.sectionID,
	}}
	if err := f.db.Create(successor).Error; err != nil {
		t.Fatalf("seed successor: %v", err)
	}

	_, err := f.svc.EndContract(ctx, f.contract.ID, f.childID, f.orgID,
		&models.ContractEndRequest{To: models.OptNull[time.Time]()})
	assertAppError(t, err, 409, apperror.CodeContractConflict)
}

// --------------------------------------------------------------- boundary ----

// The regression that broke funding twice, now structurally impossible: a
// three-contract timeline where the middle seam moves. The old client had to
// send four dates for two contracts; omitting the neighbour's `to` made it
// ongoing and collided with the third contract (409), and omitting its
// properties stripped care_type and every supplement.
func TestChildService_MoveBoundary_ThreeContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Seam Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Seam", "Child", org.ID)

	// 2026: Jan–Mar | Apr–Jun | Jul–onwards
	first := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: date(2026, 1, 1), To: timePtr(date(2026, 3, 31))},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
	}}
	second := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: date(2026, 4, 1), To: timePtr(date(2026, 6, 30))},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "ganztag", "integration": "integration a"},
	}}
	third := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: date(2026, 7, 1)},
		SectionID:  section.ID,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}}
	for _, c := range []*models.ChildContract{first, second, third} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Move the first/second seam back a month.
	resp, err := svc.MoveContractBoundary(ctx, child.ID, org.ID,
		&models.ContractBoundaryMoveRequest{
			EarlierID: first.ID, LaterID: second.ID, At: date(2026, 3, 1),
			EarlierVersion: first.Version, LaterVersion: second.Version,
		})
	if err != nil {
		t.Fatalf("move boundary: %v", err)
	}

	if !resp.Later.From.Equal(date(2026, 3, 1)) {
		t.Errorf("later from = %v, want 2026-03-01", resp.Later.From)
	}
	if resp.Earlier.To == nil || !resp.Earlier.To.Equal(date(2026, 2, 28)) {
		t.Errorf("earlier to = %v, want 2026-02-28", resp.Earlier.To)
	}
	// The neighbour's own end date is untouched — this is the 409 that used to
	// hit every child with three or more contracts.
	if resp.Later.To == nil || !resp.Later.To.Equal(date(2026, 6, 30)) {
		t.Errorf("later to = %v, want 2026-06-30 unchanged", resp.Later.To)
	}
	// And the funding properties on both sides survive.
	if resp.Earlier.Properties["care_type"] != "halbtag" || resp.Earlier.Properties["ndh"] != "ndh" {
		t.Errorf("earlier properties = %v, want preserved", resp.Earlier.Properties)
	}
	if resp.Later.Properties["care_type"] != "ganztag" || resp.Later.Properties["integration"] != "integration a" {
		t.Errorf("later properties = %v, want preserved", resp.Later.Properties)
	}

	// The third contract is not involved and must be exactly as it was.
	var reloaded models.ChildContract
	if err := db.First(&reloaded, third.ID).Error; err != nil {
		t.Fatalf("reload third: %v", err)
	}
	if !reloaded.From.Equal(date(2026, 7, 1)) || reloaded.To != nil {
		t.Errorf("third contract changed: from=%v to=%v", reloaded.From, reloaded.To)
	}
}

func TestChildService_MoveBoundary_Rejections(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Seam Reject Org")
	section := createTestSection(t, db, "Nest", org.ID, false)
	child := createTestChild(t, db, "Reject", "Child", org.ID)

	// Adjacent pair plus a third contract separated by a gap in July.
	first := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: date(2026, 1, 1), To: timePtr(date(2026, 3, 31))},
		SectionID: section.ID,
	}}
	second := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: date(2026, 4, 1), To: timePtr(date(2026, 6, 30))},
		SectionID: section.ID,
	}}
	gapped := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:    models.Period{From: date(2026, 8, 1)},
		SectionID: section.ID,
	}}
	for _, c := range []*models.ChildContract{first, second, gapped} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	v := func(req *models.ContractBoundaryMoveRequest, earlier, later *models.ChildContract) *models.ContractBoundaryMoveRequest {
		req.EarlierVersion, req.LaterVersion = earlier.Version, later.Version
		return req
	}
	cases := map[string]*models.ContractBoundaryMoveRequest{
		// One seam, two ids — naming the same contract twice is meaningless.
		"same contract twice": v(&models.ContractBoundaryMoveRequest{EarlierID: first.ID, LaterID: first.ID, At: date(2026, 3, 1)}, first, first),
		// Swapped roles: the server will not guess which the caller meant.
		"wrong order": v(&models.ContractBoundaryMoveRequest{EarlierID: second.ID, LaterID: first.ID, At: date(2026, 3, 1)}, second, first),
		// A gap is two boundaries, not one seam; moving "the" seam would swallow it.
		"not adjacent": v(&models.ContractBoundaryMoveRequest{EarlierID: second.ID, LaterID: gapped.ID, At: date(2026, 7, 15)}, second, gapped),
		// Both sides must keep at least one day.
		"empties the earlier side": v(&models.ContractBoundaryMoveRequest{EarlierID: first.ID, LaterID: second.ID, At: date(2026, 1, 1)}, first, second),
		"empties the later side":   v(&models.ContractBoundaryMoveRequest{EarlierID: first.ID, LaterID: second.ID, At: date(2026, 7, 1)}, first, second),
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.MoveContractBoundary(ctx, child.ID, org.ID, req)
			assertAppError(t, err, 400, "")
		})
	}
}

// --------------------------------------------------------------- employee ----

// 0 weekly hours is a real contract — parental leave keeps the employee on the
// books with no hours. `binding:"required"` on a float64 rejects zero, so the
// old update request could not express it at all.
func TestEmployeeService_Correct_ZeroWeeklyHoursAccepted(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Zero Hours Org")
	createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Zero", "Hours", org.ID)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID, date(2026, 1, 1), nil, "S8a", 3, 30)

	resp, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID,
		&models.EmployeeContractCorrectRequest{WeeklyHours: models.OptOf(0.0)})
	if err != nil {
		t.Fatalf("correct to 0 hours: %v", err)
	}
	if resp.WeeklyHours != 0 {
		t.Errorf("weekly_hours = %v, want 0", resp.WeeklyHours)
	}
}

func TestEmployeeService_Correct_OmittedFieldsUntouched(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Emp Correct Org")
	createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Emp", "Correct", org.ID)
	end := date(2026, 12, 31)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID, date(2026, 1, 1), &end, "S8a", 3, 30)

	resp, err := svc.CorrectContract(ctx, contract.ID, employee.ID, org.ID,
		&models.EmployeeContractCorrectRequest{WeeklyHours: models.OptOf(35.0)})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	if resp.Grade != "S8a" || resp.Step != 3 {
		t.Errorf("grade/step = %s/%d, want S8a/3 untouched", resp.Grade, resp.Step)
	}
	if resp.To == nil || !resp.To.Equal(end) {
		t.Errorf("to = %v, want %v — an omitted field must not be cleared", resp.To, end)
	}
}

// A raise takes effect on a date, and the old terms stay on record for the
// months they applied to — that is what makes this an amend and not a correct.
func TestEmployeeService_Amend_ClosesPredecessorAndCreatesSuccessor(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Emp Amend Org")
	createTestSection(t, db, "Nest", org.ID, false)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Emp", "Amend", org.ID)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID, date(2026, 1, 1), nil, "S8a", 3, 30)

	seam := date(2026, 4, 1)
	resp, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID,
		&models.EmployeeContractAmendRequest{EffectiveFrom: seam, Step: models.OptOf(4)})
	if err != nil {
		t.Fatalf("amend: %v", err)
	}

	if resp.Closed.To == nil || !resp.Closed.To.Equal(date(2026, 3, 31)) {
		t.Errorf("predecessor to = %v, want 2026-03-31", resp.Closed.To)
	}
	if resp.Closed.Step != 3 {
		t.Errorf("predecessor step = %d, want 3 — the old terms must stay on record", resp.Closed.Step)
	}
	if !resp.Created.From.Equal(seam) || resp.Created.Step != 4 {
		t.Errorf("successor = from %v step %d, want %v step 4", resp.Created.From, resp.Created.Step, seam)
	}
	// Everything unmentioned carries over.
	if resp.Created.WeeklyHours != 30 || resp.Created.Grade != "S8a" {
		t.Errorf("successor hours/grade = %v/%s, want 30/S8a inherited", resp.Created.WeeklyHours, resp.Created.Grade)
	}
}

// Same "absent is not zero" distinction on the create path, which is where the
// problem started: `binding:"required"` on a float64 rejects 0, so a 0-hour
// contract could not be created at all. WeeklyHours is now a pointer, and the
// requirement is enforced where absent and 0 are still distinguishable.
func TestEmployeeService_CreateContract_WeeklyHoursPresence(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Create Hours Org")
	sectionID := createTestSection(t, db, "Nest", org.ID, false).ID
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD", org.ID)
	employee := createTestEmployee(t, db, "Create", "Hours", org.ID)

	base := func() *models.EmployeeContractCreateRequest {
		return &models.EmployeeContractCreateRequest{
			From: date(2026, 1, 1), SectionID: sectionID, StaffCategory: "qualified",
			Grade: "S8a", Step: 3, PayPlanID: payPlan.ID,
		}
	}

	t.Run("zero is accepted", func(t *testing.T) {
		req := base()
		req.WeeklyHours = float64Ptr(0)
		resp, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
		if err != nil {
			t.Fatalf("create with 0 hours: %v", err)
		}
		if resp.WeeklyHours != 0 {
			t.Errorf("weekly_hours = %v, want 0", resp.WeeklyHours)
		}
	})

	t.Run("absent is rejected", func(t *testing.T) {
		req := base()
		req.From = date(2027, 1, 1) // avoid overlapping the contract above
		_, err := svc.CreateContract(ctx, employee.ID, org.ID, req)
		assertAppError(t, err, 400, "")
	})
}

// The pay-plan coverage check has to run at the successor's start date, not at
// today: a pay plan that only starts in 2026 cannot price a contract backdated
// into 2025, and the old path (checking at today) accepted exactly that.
func TestEmployeeService_Amend_PayPlanCoverageAnchoredAtSeam(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Anchor Org")
	createTestSection(t, db, "Nest", org.ID, false)

	// A plan that only covers 2026 onward.
	payPlan := createTestPayPlan(t, db, "TVöD 2026", org.ID)
	period := createTestPayPlanPeriod(t, db, payPlan.ID, date(2026, 1, 1), nil, 39.0)
	createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 300000, nil)

	employee := createTestEmployee(t, db, "Anchor", "Employee", org.ID)
	contract := createTestEmployeeContract(t, db, employee.ID, payPlan.ID, date(2026, 2, 1), nil, "S8a", 3, 30)

	// Amending back into 2025 must fail: the plan does not cover that date, even
	// though it covers today.
	_, err := svc.AmendContract(ctx, contract.ID, employee.ID, org.ID,
		&models.EmployeeContractAmendRequest{EffectiveFrom: date(2025, 6, 1)})
	if err == nil {
		t.Fatal("amend into a date the pay plan does not cover was accepted")
	}
}

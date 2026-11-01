package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// A month whose bill carries no row for a child reads as a full-rate
// shortfall, and the issue list marks it as work to do. That is right when the
// money never came -- and wrong when the Senate settled the gap later, which
// is how it settles a lapsed Gutschein: nothing for months, then the whole
// backlog in one bill as correction rows carrying the months they settle.
//
// Observed on real data before this existed: one child absent from seven
// consecutive bills, 13.473,36 EUR of corrections in the eighth, and seven
// months reported as 13.482,58 EUR of deficit to chase.
//
// These tests pin the fix and, as much, pin what it must NOT do: the amounts
// stay exactly as they were, because corrections offset the difference nowhere
// in this comparison and two other tests assert the five categories still sum
// to it.

// settledFixture builds the shape every test here needs: a child with a
// contract active across both months, a bill for `gapMonth` that does not
// carry the child, and a later bill that does and whose rows are attributed
// back to `gapMonth`.
type settledFixture struct {
	db       *gorm.DB
	svc      *GovernmentFundingBillService
	orgID    uint
	childID  uint
	gapMonth time.Time
}

func newSettledFixture(t *testing.T, email, voucher string, settleAmount int, settleMonth time.Month, settleYear int) settledFixture {
	t.Helper()
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", email, "password")
	setupFundingRates(t, db)

	child, _ := createChildWithVoucher(t, db, "Lena", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 4, 10, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil,
		models.ContractProperties{"care_type": "ganztag"})

	gap := firstOf(2025, time.May)

	// The gap month's own bill: somebody else entirely, so our child falls to
	// the calc-only path.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.May, []models.GovernmentFundingBillChild{{
		VoucherNumber: "GB-19999999999-01", ChildName: "Andere,Person", BirthDate: "01.20", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, RowIndex: 0, BillingMonth: &gap},
		},
	}})

	// The settling bill: its own regular row plus a correction attributed
	// back to the gap month.
	own := firstOf(settleYear, settleMonth)
	rows := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, RowIndex: 0, BillingMonth: &own},
	}
	if settleAmount != 0 {
		rows = append(rows, models.GovernmentFundingBillPayment{
			Key: "care_type", Value: "ganztag", Amount: settleAmount,
			RowType: models.RowTypeCorrection, RowIndex: 1, BillingMonth: &gap,
		})
	}
	createBillFixture(t, db, org.ID, user.ID, settleYear, settleMonth, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel,Lena", BirthDate: "04.21", District: 1,
		Payments: rows,
	}})

	return settledFixture{db: db, svc: svc, orgID: org.ID, childID: child.ID, gapMonth: gap}
}

// gapChild returns the comparison row for the gap month's child, and the
// summary built over just that month.
func (f settledFixture) gapChild(t *testing.T) (models.FundingComparisonChild, models.FundingComparisonSummary) {
	t.Helper()
	comp, err := f.svc.CompareByDate(context.Background(), f.orgID, f.gapMonth)
	if err != nil {
		t.Fatalf("CompareByDate: %v", err)
	}
	for _, ch := range comp.Children {
		if ch.ChildID != nil && *ch.ChildID == f.childID {
			return ch, BuildComparisonSummary([]models.FundingComparisonResponse{*comp})
		}
	}
	t.Fatalf("child %d not in the %s comparison", f.childID, f.gapMonth.Format("2006-01"))
	return models.FundingComparisonChild{}, models.FundingComparisonSummary{}
}

func issueFor(t *testing.T, s models.FundingComparisonSummary, childID uint) models.FundingComparisonIssueSummary {
	t.Helper()
	for _, i := range s.Issues {
		if i.ChildID != nil && *i.ChildID == childID {
			return i
		}
	}
	t.Fatalf("no issue for child %d", childID)
	return models.FundingComparisonIssueSummary{}
}

// assertInvariant re-checks the property the whole design rests on: the five
// categories still sum to the difference. If a settlement ever starts moving
// an amount rather than a flag, this is what catches it.
func assertInvariant(t *testing.T, s models.FundingComparisonSummary) {
	t.Helper()
	sum := 0
	for _, c := range s.Categories {
		sum += c.TotalAmount
	}
	if sum != s.TotalDifference {
		t.Errorf("invariant broken: sum(categories)=%d != total_difference=%d", sum, s.TotalDifference)
	}
}

// Fully settled: nothing left to chase, and not a cent of the reported
// shortfall moves.
func TestSettledLater_FullySettledIsNotActionable(t *testing.T) {
	f := newSettledFixture(t, "settled1@example.com", "GB-11000000001-01", 120000, time.August, 2025)
	child, summary := f.gapChild(t)

	if child.Status != "calc_only" {
		t.Fatalf("status = %q, want calc_only", child.Status)
	}
	if child.SettledLaterAmount == nil || *child.SettledLaterAmount != 120000 {
		t.Fatalf("settled_later_amount = %v, want 120000", child.SettledLaterAmount)
	}
	if len(child.SettledByBillMonths) != 1 || child.SettledByBillMonths[0] != "2025-08-01" {
		t.Errorf("settled_by_bill_months = %v, want [2025-08-01]", child.SettledByBillMonths)
	}

	issue := issueFor(t, summary, f.childID)
	if issue.Actionable {
		t.Error("a month paid in full is still marked actionable; this is what sends an operator after money already received")
	}
	if issue.SettledAmount != 120000 {
		t.Errorf("settled_amount = %d, want 120000", issue.SettledAmount)
	}
	if issue.TotalAmount != -120000 {
		t.Errorf("total_amount = %d, want -120000: the month was underfunded at the time and the bars must keep summing to the difference",
			issue.TotalAmount)
	}
	assertInvariant(t, summary)
}

// Partly settled: the remainder is still real work.
func TestSettledLater_PartialSettlementStaysActionable(t *testing.T) {
	f := newSettledFixture(t, "settled2@example.com", "GB-11000000002-01", 70000, time.August, 2025)
	child, summary := f.gapChild(t)

	if child.SettledLaterAmount == nil || *child.SettledLaterAmount != 70000 {
		t.Fatalf("settled_later_amount = %v, want 70000", child.SettledLaterAmount)
	}
	issue := issueFor(t, summary, f.childID)
	if !issue.Actionable {
		t.Error("70000 of a 120000 month was paid; the remaining 50000 is still work")
	}
	assertInvariant(t, summary)
}

// A clawback attributed to the gap month makes the month worse, not better.
// Getting this sign wrong would mark a month settled by taking money away.
func TestSettledLater_NegativeCorrectionStaysActionable(t *testing.T) {
	f := newSettledFixture(t, "settled3@example.com", "GB-11000000003-01", -30000, time.August, 2025)
	child, summary := f.gapChild(t)

	if child.SettledLaterAmount == nil || *child.SettledLaterAmount != -30000 {
		t.Fatalf("settled_later_amount = %v, want -30000", child.SettledLaterAmount)
	}
	issue := issueFor(t, summary, f.childID)
	if !issue.Actionable {
		t.Error("a clawback attributed to the month cannot settle it")
	}
	assertInvariant(t, summary)
}

// The settling bill predates the gap month -- a Vorschuss. Direction is
// irrelevant; the question is which month the row is about.
func TestSettledLater_EarlierBillAlsoSettles(t *testing.T) {
	f := newSettledFixture(t, "settled4@example.com", "GB-11000000004-01", 120000, time.February, 2025)
	child, _ := f.gapChild(t)
	if child.SettledLaterAmount == nil || *child.SettledLaterAmount != 120000 {
		t.Fatalf("settled_later_amount = %v, want 120000 from a bill before the gap month", child.SettledLaterAmount)
	}
}

// Rows carrying no billing_month -- every bill imported before the column was
// populated -- COALESCE to their own bill's month and must never settle an
// earlier one. This is the property that makes a partially re-imported
// database safe: the failure mode is always "no settlement found".
func TestSettledLater_RowsWithoutAttributionNeverSettle(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "settled5@example.com", "password")
	setupFundingRates(t, db)

	const voucher = "GB-11000000005-01"
	child, _ := createChildWithVoucher(t, db, "Lena", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 4, 10, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil,
		models.ContractProperties{"care_type": "ganztag"})

	gap := firstOf(2025, time.May)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.May, []models.GovernmentFundingBillChild{{
		VoucherNumber: "GB-19999999999-01", ChildName: "Andere,Person", BirthDate: "01.20", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, BillingMonth: &gap},
		},
	}})
	// Correction with a NIL billing month, the shape of every pre-migration row.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.August, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel,Lena", BirthDate: "04.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, BillingMonth: nil},
		},
	}})

	comp, err := svc.CompareByDate(context.Background(), org.ID, gap)
	if err != nil {
		t.Fatalf("CompareByDate: %v", err)
	}
	for _, ch := range comp.Children {
		if ch.ChildID != nil && *ch.ChildID == child.ID {
			if ch.SettledLaterAmount != nil {
				t.Errorf("settled_later_amount = %d; a row with no month of its own belongs to its own bill and must not settle an earlier month",
					*ch.SettledLaterAmount)
			}
			return
		}
	}
	t.Fatal("child not in the comparison")
}

// Another child's correction, attributed to the same month, must not offset
// this one. The query joins through child_vouchers for exactly this reason.
func TestSettledLater_AnotherChildsCorrectionDoesNotSettle(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "settled6@example.com", "password")
	setupFundingRates(t, db)

	absent, _ := createChildWithVoucher(t, db, "Lena", "Beispiel", org.ID, section.ID,
		"GB-11000000006-01", time.Date(2021, 4, 10, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil, models.ContractProperties{"care_type": "ganztag"})
	createChildWithVoucher(t, db, "Tim", "Beispiel", org.ID, section.ID,
		"GB-11000000007-01", time.Date(2021, 3, 3, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil, models.ContractProperties{"care_type": "ganztag"})

	gap := firstOf(2025, time.May)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.May, []models.GovernmentFundingBillChild{{
		VoucherNumber: "GB-11000000007-01", ChildName: "Beispiel,Tim", BirthDate: "03.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, BillingMonth: &gap},
		},
	}})
	// Tim is settled for May in August; Lena must not be.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.August, []models.GovernmentFundingBillChild{{
		VoucherNumber: "GB-11000000007-01", ChildName: "Beispiel,Tim", BirthDate: "03.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, BillingMonth: &gap},
		},
	}})

	comp, err := svc.CompareByDate(context.Background(), org.ID, gap)
	if err != nil {
		t.Fatalf("CompareByDate: %v", err)
	}
	for _, ch := range comp.Children {
		if ch.ChildID != nil && *ch.ChildID == absent.ID && ch.SettledLaterAmount != nil {
			t.Fatalf("another child's correction settled this one: %d", *ch.SettledLaterAmount)
		}
	}
}

// A child with no Gutschein on record cannot be matched to anything, so
// nothing can settle it and it stays actionable. Without this the lookup
// could appear to "work" by finding nothing for the wrong reason.
func TestSettledLater_ChildWithoutVoucherStaysActionable(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "settled7@example.com", "password")
	setupFundingRates(t, db)

	child := createTestChild(t, db, "Ohne", "Gutschein", org.ID)
	createTestChildContract(t, db, child.ID,
		firstOf(2025, time.January), nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	gap := firstOf(2025, time.May)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.May, []models.GovernmentFundingBillChild{{
		VoucherNumber: "GB-19999999999-01", ChildName: "Andere,Person", BirthDate: "01.20", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, BillingMonth: &gap},
		},
	}})

	comp, err := svc.CompareByDate(context.Background(), org.ID, gap)
	if err != nil {
		t.Fatalf("CompareByDate: %v", err)
	}
	s := BuildComparisonSummary([]models.FundingComparisonResponse{*comp})
	issue := issueFor(t, s, child.ID)
	if !issue.Actionable {
		t.Error("a child with no Gutschein on record has nothing that could have settled it")
	}
	if issue.SettledAmount != 0 {
		t.Errorf("settled_amount = %d, want 0", issue.SettledAmount)
	}
}

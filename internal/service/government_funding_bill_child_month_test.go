package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// firstOf is a terse month constructor for the fixtures below.
func firstOf(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}

// renewalFixture builds the ISBJ voucher-renewal shape: one child holding two
// Gutscheine, both appearing in the SAME bill.
//
// This is what the Senate does on a renewal. The months already paid under the
// old Gutschein are reversed on it (negative correction rows) and re-booked
// under the new one (positive correction rows), while the new Gutschein also
// carries the regular row for the bill's own month. Three of the twenty-two
// Abrechnungen this was derived from carry exactly this shape, so it is routine
// rather than exotic.
//
// The month in question is funded ONCE, by the regular row on the new voucher.
// A comparison that walks bill-child rows instead of children counts the
// contract's calculated rate once per row and reports a month-sized deficit
// that nobody is owed.
func renewalFixture(t *testing.T, email string) (*GovernmentFundingBillService, *models.Organization, *models.Child, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", email, "password")
	setupFundingRates(t, db)

	const oldVoucher = "GB-10000000001-04"
	const newVoucher = "GB-10000000001-08"

	child, _ := createChildWithVoucher(t, db, "Lena", "Beispiel", org.ID, section.ID,
		oldVoucher, time.Date(2021, 4, 15, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil,
		models.ContractProperties{"care_type": "ganztag"})
	if err := db.Create(&models.ChildVoucher{
		ChildID: child.ID, VoucherNumber: newVoucher, FirstSeen: firstOf(2025, time.August),
	}).Error; err != nil {
		t.Fatalf("create second voucher: %v", err)
	}

	jun, jul, aug := firstOf(2025, time.June), firstOf(2025, time.July), firstOf(2025, time.August)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.August, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: oldVoucher, ChildName: "Beispiel, Lena", BirthDate: "04.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: -100000, RowType: models.RowTypeCorrection, RowIndex: 0, BillingMonth: &jun},
				{Key: "care_type", Value: "ganztag", Amount: -100000, RowType: models.RowTypeCorrection, RowIndex: 1, BillingMonth: &jul},
			},
		},
		{
			VoucherNumber: newVoucher, ChildName: "Beispiel, Lena", BirthDate: "04.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, RowIndex: 0, BillingMonth: &jun},
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, RowIndex: 1, BillingMonth: &jul},
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, RowIndex: 2, BillingMonth: &aug},
			},
		},
	})
	return svc, org, child, context.Background()
}

func TestComparePeriod_VoucherRenewalCountsMonthOnce(t *testing.T) {
	svc, org, child, ctx := renewalFixture(t, "childmonth1@example.com")

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}

	// One child, one month, one rate. The two bill rows are one enrolment.
	if comp.ChildrenCount != 1 {
		t.Errorf("children_count = %d, want 1 (two Gutscheine are one child)", comp.ChildrenCount)
	}
	if len(comp.Children) != 1 {
		t.Fatalf("len(children) = %d, want 1", len(comp.Children))
	}
	if comp.BillTotal != 120000 {
		t.Errorf("bill_total = %d, want 120000 (August's regular row)", comp.BillTotal)
	}
	if comp.CalcTotal != 120000 {
		t.Errorf("calculated_total = %d, want 120000 (the month's rate, counted once)", comp.CalcTotal)
	}
	if comp.Difference != 0 {
		t.Errorf("difference = %d, want 0 — August was funded in full", comp.Difference)
	}
	// -200000 reversed + 240000 re-booked: the genuine retroactive uplift.
	if comp.CorrectionTotal != 40000 {
		t.Errorf("correction_total = %d, want 40000", comp.CorrectionTotal)
	}

	got := comp.Children[0]
	if got.ChildID == nil || *got.ChildID != child.ID {
		t.Errorf("child_id = %v, want %d", got.ChildID, child.ID)
	}
	if got.Status != "match" {
		t.Errorf("status = %q, want \"match\"", got.Status)
	}
	if len(got.VoucherNumbers) != 2 {
		t.Errorf("voucher_numbers = %v, want both Gutscheine", got.VoucherNumbers)
	}
	// The primary voucher is the one that actually carried the month's money.
	if got.VoucherNumber != "GB-10000000001-08" {
		t.Errorf("voucher_number = %q, want the Gutschein holding the regular row", got.VoucherNumber)
	}
	for _, p := range got.Properties {
		if p.Mismatch == models.MismatchMissing {
			t.Errorf("property %s:%s reported as missing; the month was billed", p.Key, p.Value)
		}
	}
}

// The summary is what the deficit bars and the actionable issue list are built
// from, so the double count has to be gone there too — and it must not leave an
// actionable "chase the Senate" item for money that was paid.
func TestBuildComparisonSummary_VoucherRenewalRaisesNoIssue(t *testing.T) {
	svc, org, _, ctx := renewalFixture(t, "childmonth2@example.com")

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	s := BuildComparisonSummary([]models.FundingComparisonResponse{*comp})

	if s.TotalCalculated != 120000 {
		t.Errorf("total_calculated = %d, want 120000", s.TotalCalculated)
	}
	if s.TotalDifference != 0 {
		t.Errorf("total_difference = %d, want 0", s.TotalDifference)
	}
	for _, issue := range s.Issues {
		if issue.Actionable {
			t.Errorf("actionable issue raised for a fully funded month: %+v", issue)
		}
	}
	catSum := 0
	for _, c := range s.Categories {
		catSum += c.TotalAmount
	}
	if catSum != s.TotalDifference {
		t.Errorf("invariant broken: sum(categories)=%d != total_difference=%d", catSum, s.TotalDifference)
	}
}

func TestChildBillingHistory_TwoVouchersInOneBill(t *testing.T) {
	svc, org, child, ctx := renewalFixture(t, "childmonth3@example.com")

	hist, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}
	if len(hist.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 — one bill is one row, whatever its Gutschein count", len(hist.Entries))
	}
	e := hist.Entries[0]
	if e.BillTotal != 120000 {
		t.Errorf("entry bill_total = %d, want 120000", e.BillTotal)
	}
	if e.CalcTotal == nil || *e.CalcTotal != 120000 {
		t.Errorf("entry calculated_total = %v, want 120000", e.CalcTotal)
	}
	if e.Difference == nil || *e.Difference != 0 {
		t.Errorf("entry difference = %v, want 0", e.Difference)
	}
	if e.RunningDifference != 0 {
		t.Errorf("running_difference = %d, want 0", e.RunningDifference)
	}
	if hist.TotalCalculated != 120000 {
		t.Errorf("total_calculated = %d, want 120000", hist.TotalCalculated)
	}
	if hist.TotalDifference != 0 {
		t.Errorf("total_difference = %d, want 0", hist.TotalDifference)
	}
}

// The same defect without any correction row in sight: a mid-month renewal
// where the Senate splits the month pro rata across the two Gutscheine. Both
// rows are regular, so the billed side is right either way, but the calculated
// side is still counted once per row. Not observed in the sample bills; it
// follows from the same code path.
func TestComparePeriod_ProRataSplitAcrossVouchers(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "childmonth4@example.com", "password")
	ctx := context.Background()
	setupFundingRates(t, db)

	const oldVoucher = "GB-10000000005-01"
	const newVoucher = "GB-10000000005-02"
	child, _ := createChildWithVoucher(t, db, "Tim", "Beispiel", org.ID, section.ID,
		oldVoucher, time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil,
		models.ContractProperties{"care_type": "ganztag"})
	if err := db.Create(&models.ChildVoucher{
		ChildID: child.ID, VoucherNumber: newVoucher, FirstSeen: firstOf(2025, time.September),
	}).Error; err != nil {
		t.Fatalf("create second voucher: %v", err)
	}

	sep := firstOf(2025, time.September)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.September, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: oldVoucher, ChildName: "Beispiel, Tim", BirthDate: "02.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 60000, RowType: models.RowTypeRegular, BillingMonth: &sep},
			},
		},
		{
			VoucherNumber: newVoucher, ChildName: "Beispiel, Tim", BirthDate: "02.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 60000, RowType: models.RowTypeRegular, BillingMonth: &sep},
			},
		},
	})

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	if comp.ChildrenCount != 1 {
		t.Errorf("children_count = %d, want 1", comp.ChildrenCount)
	}
	if comp.BillTotal != 120000 {
		t.Errorf("bill_total = %d, want 120000 (both halves of the month)", comp.BillTotal)
	}
	if comp.CalcTotal != 120000 {
		t.Errorf("calculated_total = %d, want 120000", comp.CalcTotal)
	}
	if comp.Difference != 0 {
		t.Errorf("difference = %d, want 0", comp.Difference)
	}
}

// A bill row carrying only corrections for earlier months, for a child whose
// contract is active in the bill's month. The month really was not funded, so
// the figure stands -- but the cause is "not billed", not "the Senate has the
// wrong care type on file", and the two lead an operator to different work.
func TestComparePeriod_CorrectionOnlyEntryIsNotBilled(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "notbilled1@example.com", "password")
	ctx := context.Background()
	setupFundingRates(t, db)

	const voucher = "GB-10000000002-02"
	createChildWithVoucher(t, db, "Noah", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 3, 10, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.May), nil,
		models.ContractProperties{"care_type": "ganztag"})

	may, jun := firstOf(2025, time.May), firstOf(2025, time.June)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.October, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel, Noah", BirthDate: "03.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, RowIndex: 0, BillingMonth: &may},
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeCorrection, RowIndex: 1, BillingMonth: &jun},
		},
	}})

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	if len(comp.Children) != 1 {
		t.Fatalf("len(children) = %d, want 1", len(comp.Children))
	}
	if got := comp.Children[0].Status; got != "not_billed" {
		t.Errorf("status = %q, want \"not_billed\"", got)
	}
	// The figure is unchanged: October carried no money for this child.
	if comp.Children[0].Difference == nil || *comp.Children[0].Difference != -120000 {
		t.Errorf("difference = %v, want -120000", comp.Children[0].Difference)
	}
	if comp.DifferenceCount != 1 {
		t.Errorf("difference_count = %d, want 1", comp.DifferenceCount)
	}

	s := BuildComparisonSummary([]models.FundingComparisonResponse{*comp})
	byCat := make(map[string]models.FundingComparisonCategorySummary, len(s.Categories))
	catSum := 0
	for _, c := range s.Categories {
		byCat[c.Category] = c
		catSum += c.TotalAmount
	}
	if _, ok := byCat["property_mismatch"]; ok {
		t.Errorf("not-billed month classified as property_mismatch: %+v", byCat["property_mismatch"])
	}
	if _, ok := byCat["rate_difference"]; ok {
		t.Errorf("not-billed month leaked into rate_difference: %+v", byCat["rate_difference"])
	}
	nb, ok := byCat["not_billed"]
	if !ok {
		t.Fatalf("no not_billed category; got %+v", s.Categories)
	}
	if nb.TotalAmount != -120000 {
		t.Errorf("not_billed amount = %d, want -120000", nb.TotalAmount)
	}
	if nb.ChildCount != 1 {
		t.Errorf("not_billed child_count = %d, want 1", nb.ChildCount)
	}
	if !nb.Actionable {
		t.Error("not_billed must be actionable: somebody has to ask why the child was not billed")
	}
	if catSum != s.TotalDifference {
		t.Errorf("invariant broken: sum(categories)=%d != total_difference=%d", catSum, s.TotalDifference)
	}

	if len(s.Issues) != 1 {
		t.Fatalf("len(issues) = %d, want 1", len(s.Issues))
	}
	if s.Issues[0].Category != "not_billed" || s.Issues[0].PropertyKey != "" {
		t.Errorf("issue = %+v, want a not_billed issue with no property key", s.Issues[0])
	}
}

// The same bill row for a child whose contract had already ended: the Senate
// settling a departed child retroactively. There is no month to be short of,
// so this stays bill_only and contributes nothing to the difference.
func TestComparePeriod_DepartedChildStaysBillOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "notbilled2@example.com", "password")
	ctx := context.Background()
	setupFundingRates(t, db)

	const voucher = "GB-10000000003-13"
	left := time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "Emil", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2020, 2, 2, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), &left,
		models.ContractProperties{"care_type": "ganztag"})

	jan := firstOf(2025, time.January)
	createBillFixture(t, db, org.ID, user.ID, 2026, time.April, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel, Emil", BirthDate: "02.20", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 2365, RowType: models.RowTypeCorrection, BillingMonth: &jan},
		},
	}})

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	if got := comp.Children[0].Status; got != "bill_only" {
		t.Errorf("status = %q, want \"bill_only\"", got)
	}
	if comp.Difference != 0 {
		t.Errorf("difference = %d, want 0 — a departed child is owed no month", comp.Difference)
	}
	if comp.CorrectionTotal != 2365 {
		t.Errorf("correction_total = %d, want 2365", comp.CorrectionTotal)
	}
}

// The same booking has to reach the comparison summary, which is what the
// deficit analysis reconciles against. It is not part of the difference: no
// child rows stand behind it, so there is nothing calculated to set it against.
func TestBuildComparisonSummary_CarriesManualCorrectionBooking(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "manualcorr2@example.com", "password")
	ctx := context.Background()
	setupFundingRates(t, db)

	const voucher = "GB-10000000006-01"
	createChildWithVoucher(t, db, "Ida", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 5, 5, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.January), nil,
		models.ContractProperties{"care_type": "ganztag"})

	nov := firstOf(2025, time.November)
	period := createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel, Ida", BirthDate: "05.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, BillingMonth: &nov},
		},
	}})
	if err := db.Model(period).Update("correction_booking", 1823085).Error; err != nil {
		t.Fatalf("set correction_booking: %v", err)
	}

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	if comp.ManualCorrectionTotal != 1823085 {
		t.Errorf("manual_correction_total = %d, want 1823085", comp.ManualCorrectionTotal)
	}
	if comp.Difference != 0 {
		t.Errorf("difference = %d, want 0 — a manual booking has no calculated counterpart", comp.Difference)
	}

	s := BuildComparisonSummary([]models.FundingComparisonResponse{*comp})
	if s.TotalManualCorrections != 1823085 {
		t.Errorf("total_manual_corrections = %d, want 1823085", s.TotalManualCorrections)
	}
	if s.TotalDifference != 0 {
		t.Errorf("total_difference = %d, want 0", s.TotalDifference)
	}
	catSum := 0
	for _, c := range s.Categories {
		catSum += c.TotalAmount
	}
	if catSum != s.TotalDifference {
		t.Errorf("invariant broken: sum(categories)=%d != total_difference=%d", catSum, s.TotalDifference)
	}
}

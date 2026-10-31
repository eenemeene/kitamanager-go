package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// A bill for a month no funding period covers. The comparison cannot produce a
// calculated figure for anybody, and before this it reported the whole bill as
// that much overfunding -- while the same child on the billing-history page
// correctly said nothing could be computed. Only that path ever read the flag.
//
// Reachable by closing the open-ended funding period, which the rates UI
// allows, and then every bill from that month on reads as a surplus the size of
// itself.
func TestComparePeriod_NoFundingPeriodReportsRatherThanInventsASurplus(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "nofunding1@example.com", "password")
	ctx := context.Background()

	// A funding period that ended long before the bill's month.
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	lapsed := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &lapsed, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 120000, -1, -1)

	const voucher = "GB-10000000007-01"
	createChildWithVoucher(t, db, "Rosa", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 7, 7, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"})

	nov := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel, Rosa", BirthDate: "07.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, BillingMonth: &nov},
		},
	}})

	comp, err := svc.CompareLatest(ctx, org.ID)
	if err != nil {
		t.Fatalf("CompareLatest() error = %v", err)
	}
	if len(comp.Children) != 1 {
		t.Fatalf("len(children) = %d, want 1", len(comp.Children))
	}
	child := comp.Children[0]
	if child.Status != "no_funding_config" {
		t.Errorf("status = %q, want \"no_funding_config\"", child.Status)
	}
	if child.CalcTotal != nil {
		t.Errorf("calculated_total = %d, want nil: no rate could be computed", *child.CalcTotal)
	}
	if child.Difference != nil {
		t.Errorf("difference = %d, want nil", *child.Difference)
	}
	if comp.NoFundingConfigCount != 1 {
		t.Errorf("no_funding_config_count = %d, want 1", comp.NoFundingConfigCount)
	}
	// The whole point: no invented surplus, at either level.
	if comp.Difference != 0 {
		t.Errorf("difference = %d, want 0 — the bill is unmeasured, not a surplus", comp.Difference)
	}
	if comp.BillTotal != 0 || comp.CalcTotal != 0 {
		t.Errorf("bill_total/calculated_total = %d/%d, want 0/0: an unmeasurable child belongs to neither",
			comp.BillTotal, comp.CalcTotal)
	}
	if comp.DifferenceCount != 0 {
		t.Errorf("difference_count = %d, want 0", comp.DifferenceCount)
	}

	s := BuildComparisonSummary([]models.FundingComparisonResponse{*comp})
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

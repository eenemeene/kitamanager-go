package service

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ============================================================
// computeChildComparison unit tests
// ============================================================

func TestComputeChildComparison_BillOnly(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular},
	}
	result := computeChildComparison(childComparisonInput{
		BillPayments: payments,
		Contract:     nil,
		BillDate:     time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		LabelMap:     make(map[string]string),
	})
	if result.Status != "bill_only" {
		t.Errorf("expected status bill_only, got %q", result.Status)
	}
	if result.BillTotal != 120000 {
		t.Errorf("expected bill_total 120000, got %d", result.BillTotal)
	}
	if result.CalcTotal != nil {
		t.Error("expected calc_total nil for bill_only")
	}
	if result.Age != nil {
		t.Error("expected age nil for bill_only")
	}
}

func TestComputeChildComparison_MatchWithFunding(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular},
	}
	birthdate := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	contract := &models.ChildContract{
		BaseContract: models.BaseContract{
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}

	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := models.GovernmentFundingPeriod{
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		},
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 120000, MinAge: intPtr(0), MaxAge: intPtr(8)},
		},
	}

	result := computeChildComparison(childComparisonInput{
		BillPayments:   payments,
		Contract:       contract,
		Birthdate:      &birthdate,
		BillDate:       time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		FundingPeriods: []models.GovernmentFundingPeriod{fundingPeriod},
		LabelMap:       make(map[string]string),
	})

	if result.Status != "match" {
		t.Errorf("expected status match, got %q", result.Status)
	}
	if result.CalcTotal == nil || *result.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %v", result.CalcTotal)
	}
	if result.Difference == nil || *result.Difference != 0 {
		t.Errorf("expected difference 0, got %v", result.Difference)
	}
	if result.Age == nil || *result.Age != 4 {
		t.Errorf("expected age 4, got %v", result.Age)
	}
}

func TestComputeChildComparison_DifferenceWithFunding(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 100000, RowType: models.RowTypeRegular},
	}
	birthdate := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	contract := &models.ChildContract{
		BaseContract: models.BaseContract{
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := models.GovernmentFundingPeriod{
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		},
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 120000, MinAge: intPtr(0), MaxAge: intPtr(8)},
		},
	}

	result := computeChildComparison(childComparisonInput{
		BillPayments:   payments,
		Contract:       contract,
		Birthdate:      &birthdate,
		BillDate:       time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		FundingPeriods: []models.GovernmentFundingPeriod{fundingPeriod},
		LabelMap:       make(map[string]string),
	})

	if result.Status != "difference" {
		t.Errorf("expected status difference, got %q", result.Status)
	}
	if result.Difference == nil || *result.Difference != -20000 {
		t.Errorf("expected difference -20000, got %v", result.Difference)
	}
}

func TestComputeChildComparison_NoFundingConfig(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular},
	}
	birthdate := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	contract := &models.ChildContract{
		BaseContract: models.BaseContract{
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}

	result := computeChildComparison(childComparisonInput{
		BillPayments:   payments,
		Contract:       contract,
		Birthdate:      &birthdate,
		BillDate:       time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		FundingPeriods: nil, // no funding config
		LabelMap:       make(map[string]string),
	})

	if !result.NoFundingConfig {
		t.Error("expected NoFundingConfig to be true")
	}
	// CalcTotal should be set to 0 (contract matched but no rates)
	if result.CalcTotal == nil || *result.CalcTotal != 0 {
		t.Errorf("expected calc_total 0, got %v", result.CalcTotal)
	}
}

func TestComputeChildComparison_CorrectionSeparated(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular},
		{Key: "care_type", Value: "ganztag", Amount: 5000, RowType: models.RowTypeCorrection},
	}
	result := computeChildComparison(childComparisonInput{
		BillPayments: payments,
		Contract:     nil,
		BillDate:     time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		LabelMap:     make(map[string]string),
	})
	if result.BillTotal != 120000 {
		t.Errorf("expected bill_total 120000 (regular only), got %d", result.BillTotal)
	}
	if result.CorrectionTotal != 5000 {
		t.Errorf("expected correction_total 5000, got %d", result.CorrectionTotal)
	}
}

// ============================================================
// classifyMismatches unit tests
// ============================================================

func TestClassifyMismatches_NoMismatch(t *testing.T) {
	bill := map[string]int{"care_type:ganztag": 120000, "parent:meals": -2300}
	calc := map[string]int{"care_type:ganztag": 120000, "parent:meals": -2300}
	result := classifyMismatches(bill, calc)
	if len(result) != 0 {
		t.Errorf("expected no mismatches, got %v", result)
	}
}

func TestClassifyMismatches_AmountDiffOnly(t *testing.T) {
	// Same key:value on both sides but different amounts — NOT a property mismatch
	bill := map[string]int{"care_type:ganztag": 100000}
	calc := map[string]int{"care_type:ganztag": 120000}
	result := classifyMismatches(bill, calc)
	if len(result) != 0 {
		t.Errorf("expected no mismatches (amount diff only), got %v", result)
	}
}

func TestClassifyMismatches_Different(t *testing.T) {
	// Bill has teilzeit, calc has ganztag for the same base key
	bill := map[string]int{"care_type:teilzeit": 90000}
	calc := map[string]int{"care_type:ganztag": 120000}
	result := classifyMismatches(bill, calc)
	if result["care_type:teilzeit"] != models.MismatchDifferent {
		t.Errorf("expected care_type:teilzeit = different, got %q", result["care_type:teilzeit"])
	}
	if result["care_type:ganztag"] != models.MismatchDifferent {
		t.Errorf("expected care_type:ganztag = different, got %q", result["care_type:ganztag"])
	}
}

func TestClassifyMismatches_Additional(t *testing.T) {
	// Bill has integration but calc does not
	bill := map[string]int{"care_type:ganztag": 120000, "integration:integration b": 330000}
	calc := map[string]int{"care_type:ganztag": 120000}
	result := classifyMismatches(bill, calc)
	if result["integration:integration b"] != models.MismatchAdditional {
		t.Errorf("expected integration:integration b = additional, got %q", result["integration:integration b"])
	}
	// care_type should NOT be a mismatch
	if result["care_type:ganztag"] != models.MismatchNone {
		t.Errorf("expected care_type:ganztag = none, got %q", result["care_type:ganztag"])
	}
}

func TestClassifyMismatches_Missing(t *testing.T) {
	// Calc has integration but bill does not
	bill := map[string]int{"care_type:ganztag": 120000}
	calc := map[string]int{"care_type:ganztag": 120000, "integration:integration b": 330000}
	result := classifyMismatches(bill, calc)
	if result["integration:integration b"] != models.MismatchMissing {
		t.Errorf("expected integration:integration b = missing, got %q", result["integration:integration b"])
	}
}

func TestClassifyMismatches_MixedScenario(t *testing.T) {
	// care_type: different (bill=teilzeit, calc=ganztag)
	// integration: additional in bill (not in calc)
	// parent:meals: same on both — no mismatch
	bill := map[string]int{
		"care_type:teilzeit":        90000,
		"integration:integration a": 165000,
		"parent:meals":              -2300,
	}
	calc := map[string]int{
		"care_type:ganztag": 120000,
		"parent:meals":      -2300,
	}
	result := classifyMismatches(bill, calc)

	if result["care_type:teilzeit"] != models.MismatchDifferent {
		t.Errorf("care_type:teilzeit = %q, want different", result["care_type:teilzeit"])
	}
	if result["care_type:ganztag"] != models.MismatchDifferent {
		t.Errorf("care_type:ganztag = %q, want different", result["care_type:ganztag"])
	}
	if result["integration:integration a"] != models.MismatchAdditional {
		t.Errorf("integration:integration a = %q, want additional", result["integration:integration a"])
	}
	if result["parent:meals"] != models.MismatchNone {
		t.Errorf("parent:meals = %q, want none", result["parent:meals"])
	}
}

func TestClassifyMismatches_SurchargesExcluded(t *testing.T) {
	// parent:care exists only in bill — should NOT be flagged because parent is not a contract property
	// ndh:ndh exists only in bill — should NOT be flagged (surcharge)
	bill := map[string]int{
		"care_type:ganztag": 120000,
		"parent:care":       0,
		"parent:meals":      -2300,
		"ndh:ndh":           8000,
		"qm/mss:qm/mss":     5000,
	}
	calc := map[string]int{
		"care_type:ganztag": 120000,
		"parent:meals":      -2300,
	}
	result := classifyMismatches(bill, calc)

	// None of these surcharge keys should be flagged
	for _, kv := range []string{"parent:care", "parent:meals", "ndh:ndh", "qm/mss:qm/mss"} {
		if result[kv] != models.MismatchNone {
			t.Errorf("%s = %q, want none (surcharges should be excluded)", kv, result[kv])
		}
	}
	// care_type should also be none since both sides agree
	if result["care_type:ganztag"] != models.MismatchNone {
		t.Errorf("care_type:ganztag = %q, want none", result["care_type:ganztag"])
	}
}

// TestClassifyMismatches_SurchargeOnlyOneSideLogs asserts that bill-vs-calc
// surcharge drift (a key appearing only on one side) emits a warn log. The
// UI still does not flag these in the comparison matrix — by design — but
// operators need some signal when their funding config has drifted.
func TestClassifyMismatches_SurchargeOnlyOneSideLogs(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(oldLogger)

	bill := map[string]int{"ndh:ndh": 5000}
	calc := map[string]int{}
	classifyMismatches(bill, calc)
	if !bytes.Contains(buf.Bytes(), []byte("surcharge key appears only on one side")) {
		t.Errorf("expected surcharge-drift warning, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("bill")) {
		t.Errorf("expected log to report side=bill, got: %s", buf.String())
	}
}

// TestBuildBillOnlyProperties_DedupesAndAggregates asserts that a bill-only
// child with multiple payments sharing the same (key, value) produces one
// row in the comparison — the previous implementation produced one row per
// raw payment, giving the UI duplicate lines.
func TestBuildBillOnlyProperties_DedupesAndAggregates(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 100000, RowType: models.RowTypeRegular},
		{Key: "care_type", Value: "ganztag", Amount: 20000, RowType: models.RowTypeCorrection},
		{Key: "parent", Value: "care", Amount: 0, RowType: models.RowTypeRegular},
	}
	labelMap := map[string]string{"care_type:ganztag": "Ganztag"}

	got := buildBillOnlyProperties(payments, labelMap)

	if len(got) != 2 {
		t.Fatalf("expected 2 deduped rows, got %d: %+v", len(got), got)
	}
	var ganztag *models.FundingComparisonAmount
	for i := range got {
		if got[i].Key == "care_type" && got[i].Value == "ganztag" {
			ganztag = &got[i]
		}
	}
	if ganztag == nil {
		t.Fatalf("care_type:ganztag row missing, got: %+v", got)
	}
	if ganztag.BillAmount == nil || *ganztag.BillAmount != 120000 {
		t.Errorf("expected aggregated bill amount 120000, got %+v", ganztag.BillAmount)
	}
	if ganztag.Difference != 120000 {
		t.Errorf("expected difference 120000, got %d", ganztag.Difference)
	}
	if ganztag.Label != "Ganztag" {
		t.Errorf("expected label 'Ganztag', got %q", ganztag.Label)
	}
}

// TestClassifyMismatches_SurchargeOnBothSidesNoLog asserts no warning when
// a surcharge key is present on both sides (the normal case).
func TestClassifyMismatches_SurchargeOnBothSidesNoLog(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(oldLogger)

	bill := map[string]int{"ndh:ndh": 5000}
	calc := map[string]int{"ndh:ndh": 5000}
	classifyMismatches(bill, calc)
	if bytes.Contains(buf.Bytes(), []byte("surcharge key appears")) {
		t.Errorf("no warning expected when surcharge is on both sides, got: %s", buf.String())
	}
}

func TestBuildComparisonSummary_Empty(t *testing.T) {
	s := BuildComparisonSummary(nil)
	if s.MonthCount != 0 {
		t.Errorf("expected 0 months, got %d", s.MonthCount)
	}
	if len(s.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(s.Categories))
	}
	if len(s.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(s.Issues))
	}
}

func TestBuildComparisonSummary_InvariantCategorySumEqualsDifference(t *testing.T) {
	// Response-level totals must match sum of children's amounts for invariant to hold.
	// Children: matched(bill=100000,calc=105000) + bill_only(80000) + calc_only(75000) + match(320000,320000)
	// → BillTotal = 100000+80000+0+320000 = 500000
	// → CalcTotal = 105000+0+75000+320000 = 500000
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:        "2025-01-01",
			BillTotal:       500000,
			CalcTotal:       500000,
			CorrectionTotal: 10000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-1",
					ChildName:     "Matched, Child",
					ChildID:       uintPtr(1),
					Status:        "difference",
					BillTotal:     100000,
					CalcTotal:     intPtr(105000),
					Properties: []models.FundingComparisonAmount{
						{Key: "care_type", Value: "ganztag", Difference: -3000},                                           // rate diff
						{Key: "integration", Value: "integration a", Difference: -2000, Mismatch: models.MismatchMissing}, // mismatch
					},
				},
				{
					VoucherNumber: "V-2",
					ChildName:     "Bill, Only",
					Status:        "bill_only",
					BillTotal:     80000,
				},
				{
					VoucherNumber: "V-3",
					ChildName:     "Calc, Only",
					ChildID:       uintPtr(3),
					Status:        "calc_only",
					CalcTotal:     intPtr(75000),
				},
				{
					VoucherNumber: "V-4",
					ChildName:     "Perfect, Match",
					ChildID:       uintPtr(4),
					Status:        "match",
					BillTotal:     320000,
					CalcTotal:     intPtr(320000),
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	// Invariant: sum of category amounts == total_difference
	catSum := 0
	for _, c := range s.Categories {
		catSum += c.TotalAmount
	}
	if catSum != s.TotalDifference {
		t.Errorf("invariant broken: sum(categories)=%d != total_difference=%d", catSum, s.TotalDifference)
	}
	// rate_diff(-3000) + mismatch(-2000) + bill_only(+80000) + calc_only(-75000) = 0
	if s.TotalDifference != 0 {
		t.Errorf("expected total_difference 0, got %d", s.TotalDifference)
	}
	if s.TotalCorrections != 10000 {
		t.Errorf("expected total_corrections 10000, got %d", s.TotalCorrections)
	}

	// Verify individual categories
	catMap := make(map[string]int)
	for _, c := range s.Categories {
		catMap[c.Category] = c.TotalAmount
	}
	if catMap["rate_difference"] != -3000 {
		t.Errorf("expected rate_difference -3000, got %d", catMap["rate_difference"])
	}
	if catMap["property_mismatch"] != -2000 {
		t.Errorf("expected property_mismatch -2000, got %d", catMap["property_mismatch"])
	}
	if catMap["bill_only"] != 80000 {
		t.Errorf("expected bill_only 80000, got %d", catMap["bill_only"])
	}
	if catMap["calc_only"] != -75000 {
		t.Errorf("expected calc_only -75000, got %d", catMap["calc_only"])
	}
}

func TestBuildComparisonSummary_DeduplicatesByChildID(t *testing.T) {
	// Same child_id=1, two different vouchers, same missing property across 2 months
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:  "2025-01-01",
			BillTotal: 100000,
			CalcTotal: 110000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-OLD",
					ChildName:     "Doe,Jane",
					ChildID:       uintPtr(1),
					Status:        "difference",
					BillTotal:     100000,
					CalcTotal:     intPtr(110000),
					Properties: []models.FundingComparisonAmount{
						{Key: "integration", Value: "integration b", Difference: -10000, Mismatch: models.MismatchMissing},
					},
				},
			},
		},
		{
			BillFrom:  "2025-02-01",
			BillTotal: 100000,
			CalcTotal: 110000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-NEW",
					ChildName:     "Doe, Jane",
					ChildID:       uintPtr(1),
					Status:        "difference",
					BillTotal:     100000,
					CalcTotal:     intPtr(110000),
					Properties: []models.FundingComparisonAmount{
						{Key: "integration", Value: "integration b", Difference: -10000, Mismatch: models.MismatchMissing},
					},
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	// Should be 1 issue (deduped by child_id), not 2
	mismatchIssues := 0
	for _, issue := range s.Issues {
		if issue.Category == "property_mismatch" {
			mismatchIssues++
			if issue.MonthCount != 2 {
				t.Errorf("expected 2 months, got %d", issue.MonthCount)
			}
			if issue.TotalAmount != -20000 {
				t.Errorf("expected total -20000, got %d", issue.TotalAmount)
			}
		}
	}
	if mismatchIssues != 1 {
		t.Errorf("expected 1 deduped mismatch issue, got %d", mismatchIssues)
	}
}

func TestBuildComparisonSummary_FiltersZeroAmountIssues(t *testing.T) {
	// "different" mismatch where bill-side and calc-side amounts cancel out
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:  "2025-01-01",
			BillTotal: 100000,
			CalcTotal: 100000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-1",
					ChildName:     "Zero, Impact",
					ChildID:       uintPtr(1),
					Status:        "difference",
					BillTotal:     100000,
					CalcTotal:     intPtr(100000),
					Properties: []models.FundingComparisonAmount{
						{Key: "integration", Value: "integration", BillAmount: intPtr(50000), Difference: 50000, Mismatch: models.MismatchDifferent},
						{Key: "integration", Value: "integration a", CalcAmount: intPtr(50000), Difference: -50000, Mismatch: models.MismatchDifferent},
					},
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	if len(s.Issues) != 0 {
		t.Errorf("expected 0 issues (zero-amount filtered), got %d", len(s.Issues))
	}
}

func TestBuildComparisonSummary_SortsByChildImpactThenIssueImpact(t *testing.T) {
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:  "2025-01-01",
			BillTotal: 0,
			CalcTotal: 350000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-SMALL",
					ChildName:     "Small, Impact",
					ChildID:       uintPtr(1),
					Status:        "difference",
					Properties: []models.FundingComparisonAmount{
						{Key: "care_type", Value: "ganztag", Difference: -1000, Mismatch: models.MismatchMissing},
					},
				},
				{
					VoucherNumber: "V-BIG",
					ChildName:     "Big, Impact",
					ChildID:       uintPtr(2),
					Status:        "calc_only",
					CalcTotal:     intPtr(200000),
				},
				{
					VoucherNumber: "V-MED",
					ChildName:     "Medium, Impact",
					ChildID:       uintPtr(3),
					Status:        "calc_only",
					CalcTotal:     intPtr(50000),
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	if len(s.Issues) < 3 {
		t.Fatalf("expected 3 issues, got %d", len(s.Issues))
	}
	// Biggest impact child first
	if s.Issues[0].ChildName != "Big, Impact" {
		t.Errorf("expected first issue for 'Big, Impact', got %q", s.Issues[0].ChildName)
	}
	if s.Issues[1].ChildName != "Medium, Impact" {
		t.Errorf("expected second issue for 'Medium, Impact', got %q", s.Issues[1].ChildName)
	}
	if s.Issues[2].ChildName != "Small, Impact" {
		t.Errorf("expected third issue for 'Small, Impact', got %q", s.Issues[2].ChildName)
	}
}

func TestBuildComparisonSummary_CorrectionOnlyChildNotCounted(t *testing.T) {
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:        "2025-01-01",
			BillTotal:       0,
			CalcTotal:       0,
			CorrectionTotal: 50000,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber:   "V-CORR",
					ChildName:       "Correction, Only",
					Status:          "bill_only",
					BillTotal:       0,
					CorrectionTotal: 50000,
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	if s.TotalCorrections != 50000 {
		t.Errorf("expected corrections 50000, got %d", s.TotalCorrections)
	}
	// bill_only with 0 regular amount should not produce an issue
	for _, issue := range s.Issues {
		if issue.Category == "bill_only" {
			t.Errorf("unexpected bill_only issue for correction-only child")
		}
	}
}

// TestAttributedCorrections covers the figure the deficit analysis needs to
// reconcile with the Kita year row above it: corrections that APPLY to the
// window, wherever the bill carrying them arrived.
//
// BuildComparisonSummary can only see the bills inside the window, so its
// TotalCorrections misses a correction for one of these months that arrived in
// a later bill -- which is the normal case, a correction being retroactive by
// definition.
func TestAttributedCorrections(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Attr Corr Org")
	user := createTestUser(t, db, "User", "attr_corr@example.com", "password")
	ctx := context.Background()

	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	augEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	// An August bill: its own regular row, plus a correction about July.
	bill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: aug, To: &augEnd},
		FileName:       "aug.xlsx",
		FileSha256:     "attr-corr-aug",
		FacilityName:   "Kita Sonnenschein",
		FacilityTotal:  93150,
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-12345678901-02",
				ChildName:     "Musterkind, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 94650,
						RowType: models.RowTypeRegular, BillingMonth: &aug},
					{Key: "care_type", Value: "ganztag", Amount: -1500,
						RowType: models.RowTypeCorrection, BillingMonth: &jul},
				},
			},
		},
	}
	if err := db.Create(bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	// A Kita-year window ending in July. The August bill falls outside it
	// entirely, so an arrival-keyed reading sees no corrections at all.
	from := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	got, err := svc.AttributedCorrections(ctx, org.ID, from, to)
	if err != nil {
		t.Fatalf("AttributedCorrections() error = %v", err)
	}
	// July has no bill of its own, so the correction is an orphan: real money,
	// but no calculated figure for that month to set it against.
	if got.Billed != 0 {
		t.Errorf("billed corrections for 25/26 = %d, want 0 (July has no bill)", got.Billed)
	}
	if got.Orphan != -1500 {
		t.Errorf("orphan corrections for 25/26 = %d, want -1500 (the August bill corrects July)", got.Orphan)
	}

	// The next Kita year contains the bill but not the month it corrects.
	nextFrom := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	nextTo := time.Date(2027, 7, 31, 0, 0, 0, 0, time.UTC)
	gotNext, err := svc.AttributedCorrections(ctx, org.ID, nextFrom, nextTo)
	if err != nil {
		t.Fatalf("AttributedCorrections() error = %v", err)
	}
	if gotNext.Billed != 0 || gotNext.Orphan != 0 {
		t.Errorf("attributed corrections for 26/27 = %+v, want both 0 (its only correction is about July)", gotNext)
	}
}

// TestAttributedCorrections_BilledMonthIsNotAnOrphan is the other half of the
// split: the same July correction, but now July has a bill of its own, so the
// month is evaluable and the correction reconciles against it.
//
// The split has to be made on "did a bill arrive for this month", because that
// is the same fact the Kita year row uses to decide a month counts towards its
// difference. Splitting on anything else -- whether the month has attributed
// regular money, say -- puts a late-registration month on the wrong side.
func TestAttributedCorrections_BilledMonthIsNotAnOrphan(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Attr Corr Billed Org")
	user := createTestUser(t, db, "User", "attr_corr_billed@example.com", "password")
	ctx := context.Background()

	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	julEnd := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	augEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	billFor := func(from, to time.Time, hash string, payments []models.GovernmentFundingBillPayment) *models.GovernmentFundingBillPeriod {
		return &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: from, To: &to},
			FileName:       hash + ".xlsx",
			FileSha256:     hash,
			FacilityName:   "Kita Sonnenschein",
			FacilityTotal:  94650,
			CreatedBy:      &user.ID,
			Children: []models.GovernmentFundingBillChild{{
				VoucherNumber: "GB-12345678901-02",
				ChildName:     "Musterkind, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments:      payments,
			}},
		}
	}

	julBill := billFor(jul, julEnd, "attr-corr-billed-jul", []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 94650, RowType: models.RowTypeRegular, BillingMonth: &jul},
	})
	augBill := billFor(aug, augEnd, "attr-corr-billed-aug", []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 94650, RowType: models.RowTypeRegular, BillingMonth: &aug},
		{Key: "care_type", Value: "ganztag", Amount: -1500, RowType: models.RowTypeCorrection, BillingMonth: &jul},
	})
	for _, b := range []*models.GovernmentFundingBillPeriod{julBill, augBill} {
		if err := db.Create(b).Error; err != nil {
			t.Fatalf("create bill %s: %v", b.FileSha256, err)
		}
	}

	from := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	got, err := svc.AttributedCorrections(ctx, org.ID, from, to)
	if err != nil {
		t.Fatalf("AttributedCorrections() error = %v", err)
	}
	if got.Billed != -1500 {
		t.Errorf("billed corrections = %d, want -1500 (July has a bill of its own)", got.Billed)
	}
	if got.Orphan != 0 {
		t.Errorf("orphan corrections = %d, want 0", got.Orphan)
	}
}

// TestBuildComparisonSummary_CategoriesPlusCorrectionsReconcile pins the
// identity the deficit analysis now prints on screen: the category bars
// decompose the regular-only difference, and adding the corrections gives the
// figure the Kita year row shows. The two used to disagree with nothing saying
// why.
func TestBuildComparisonSummary_CategoriesPlusCorrectionsReconcile(t *testing.T) {
	comparisons := []models.FundingComparisonResponse{
		{
			BillFrom:        "2026-07-01",
			BillTotal:       300000,
			CalcTotal:       310000,
			CorrectionTotal: -1500,
			Children: []models.FundingComparisonChild{
				{
					VoucherNumber: "V-1",
					ChildName:     "Rate, Difference",
					ChildID:       uintPtr(1),
					Status:        "difference",
					BillTotal:     300000,
					CalcTotal:     intPtr(310000),
					Properties: []models.FundingComparisonAmount{
						{Key: "care_type", Value: "ganztag", BillAmount: intPtr(300000),
							CalcAmount: intPtr(310000), Difference: -10000},
					},
				},
			},
		},
	}

	s := BuildComparisonSummary(comparisons)

	catSum := 0
	for _, c := range s.Categories {
		catSum += c.TotalAmount
	}
	if catSum != s.TotalDifference {
		t.Fatalf("sum(categories)=%d != total_difference=%d", catSum, s.TotalDifference)
	}
	if s.TotalDifference != -10000 {
		t.Errorf("total_difference = %d, want -10000", s.TotalDifference)
	}
	// Corrections sit outside the decomposition -- they are not a category of
	// defect -- but the reconciled figure has to include them.
	if s.TotalCorrections != -1500 {
		t.Errorf("total_corrections = %d, want -1500", s.TotalCorrections)
	}
	if reconciled := catSum + s.TotalCorrections; reconciled != -11500 {
		t.Errorf("categories + corrections = %d, want -11500", reconciled)
	}
	// Not set by the pure function: it has no window to attribute against.
	if s.TotalCorrectionsAttributed != nil {
		t.Errorf("TotalCorrectionsAttributed = %v, want nil from the pure builder", *s.TotalCorrectionsAttributed)
	}
}

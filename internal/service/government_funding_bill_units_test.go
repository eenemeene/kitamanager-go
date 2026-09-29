package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestComputeFileHash(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		content := []byte("hello world")
		hash1, err := ComputeFileHash(bytes.NewReader(content))
		if err != nil {
			t.Fatalf("ComputeFileHash() error = %v", err)
		}
		hash2, err := ComputeFileHash(bytes.NewReader(content))
		if err != nil {
			t.Fatalf("ComputeFileHash() error = %v", err)
		}
		if hash1 != hash2 {
			t.Error("expected same hash for same content")
		}
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		hash1, _ := ComputeFileHash(bytes.NewReader([]byte("hello")))
		hash2, _ := ComputeFileHash(bytes.NewReader([]byte("world")))
		if hash1 == hash2 {
			t.Error("expected different hashes for different content")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		hash, err := ComputeFileHash(bytes.NewReader([]byte{}))
		if err != nil {
			t.Fatalf("ComputeFileHash() error = %v", err)
		}
		if hash == "" {
			t.Error("expected non-empty hash for empty content")
		}
		// SHA-256 of empty is well-known
		if hash != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Errorf("unexpected empty hash: %s", hash)
		}
	})

	t.Run("correct sha256 length", func(t *testing.T) {
		hash, _ := ComputeFileHash(bytes.NewReader([]byte("test")))
		if len(hash) != 64 {
			t.Errorf("expected 64 char hex string, got %d chars", len(hash))
		}
	})
}

func TestLastDayOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "November 2025 (30 days)",
			input:    time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February 2024 (leap year)",
			input:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February 2025 (non-leap year)",
			input:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "December 2025 (31 days)",
			input:    time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "January 2026 (31 days)",
			input:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lastDayOfMonth(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("lastDayOfMonth(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatToDate(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		result := formatToDate(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("non-nil returns formatted", func(t *testing.T) {
		d := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
		result := formatToDate(&d)
		if result != "2025-11-30" {
			t.Errorf("expected '2025-11-30', got %q", result)
		}
	})
}

func TestConvertBillAmounts(t *testing.T) {
	amounts := []isbj.SettlementAmount{
		{Key: "care_type", Value: "ganztag", Amount: 89000},
		{Key: "qm/mss", Value: "qm/mss", Amount: 5531},
	}

	result := convertBillAmounts(amounts)

	if len(result) != 2 {
		t.Fatalf("expected 2 amounts, got %d", len(result))
	}
	if result[0].Key != "care_type" || result[0].Value != "ganztag" || result[0].Amount != 89000 {
		t.Errorf("result[0] = %+v, want care_type/ganztag/89000", result[0])
	}
	if result[1].Key != "qm/mss" || result[1].Amount != 5531 {
		t.Errorf("result[1] = %+v, want qm/mss/5531", result[1])
	}
}

func TestConvertBillAmounts_Empty(t *testing.T) {
	result := convertBillAmounts(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 amounts for nil input, got %d", len(result))
	}

	result = convertBillAmounts([]isbj.SettlementAmount{})
	if len(result) != 0 {
		t.Errorf("expected 0 amounts for empty input, got %d", len(result))
	}
}

func TestBuildResponse_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "buildresp1@example.com", "password")
	ctx := context.Background()

	// Create a persisted bill period so buildResponse has a valid periodID.
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "sha256",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup: %v", err)
	}

	converted := &isbj.ConvertedSettlement{
		FacilityName:      "Kita Test",
		FacilityTotal:     100000,
		ContractBooking:   90000,
		CorrectionBooking: 10000,
		ChildrenCount:     0,
		Children:          nil,
	}

	resp, err := svc.buildResponse(ctx, org.ID, period.ID, period.From, converted)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}
	if resp.FacilityName != "Kita Test" {
		t.Errorf("FacilityName = %q, want %q", resp.FacilityName, "Kita Test")
	}
	if resp.FacilityTotal != 100000 {
		t.Errorf("FacilityTotal = %d, want 100000", resp.FacilityTotal)
	}
	if resp.ChildrenCount != 0 {
		t.Errorf("ChildrenCount = %d, want 0", resp.ChildrenCount)
	}
	if resp.MatchedCount != 0 {
		t.Errorf("MatchedCount = %d, want 0", resp.MatchedCount)
	}
}

func TestBuildResponse_WithMatchedChild(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "buildresp2@example.com", "password")
	ctx := context.Background()

	// Create a child with a voucher-numbered contract.
	childBirthdate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	createChildWithVoucherAndContract(t, db, "Max", "Musterkind", org.ID, "GB-12345678901-01", childBirthdate, nil)

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "sha256",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup: %v", err)
	}

	converted := &isbj.ConvertedSettlement{
		FacilityName:  "Kita Test",
		FacilityTotal: 141331,
		ChildrenCount: 1,
		Children: []isbj.ConvertedChild{
			{
				VoucherNumber: "GB-12345678901-01",
				ChildName:     "Musterkind, Max",
				BirthDate:     "01.20",
				District:      1,
				TotalAmount:   141331,
				Rows: []isbj.ConvertedChildRow{
					{
						TotalRowAmount: 141331,
						Amounts: []isbj.SettlementAmount{
							{Key: "care_type", Value: "ganztag", Amount: 89000},
							{Key: "qm/mss", Value: "qm/mss", Amount: 5531},
						},
					},
				},
			},
		},
	}

	resp, err := svc.buildResponse(ctx, org.ID, period.ID, period.From, converted)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if resp.ChildrenCount != 1 {
		t.Errorf("ChildrenCount = %d, want 1", resp.ChildrenCount)
	}
	if resp.MatchedCount != 1 {
		t.Errorf("MatchedCount = %d, want 1", resp.MatchedCount)
	}
	if resp.UnmatchedCount != 0 {
		t.Errorf("UnmatchedCount = %d, want 0", resp.UnmatchedCount)
	}
	if !resp.Children[0].Matched {
		t.Error("expected child to be matched")
	}
	if resp.Children[0].ChildID == nil {
		t.Error("expected ChildID to be set for matched child")
	}
}

func TestBuildResponse_UnmatchedChild(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "buildresp3@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "sha256",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup: %v", err)
	}

	converted := &isbj.ConvertedSettlement{
		FacilityName:  "Kita Test",
		FacilityTotal: 50000,
		ChildrenCount: 1,
		Children: []isbj.ConvertedChild{
			{
				VoucherNumber: "GB-NONEXISTENT-01",
				ChildName:     "Unknown, Child",
				TotalAmount:   50000,
				Rows: []isbj.ConvertedChildRow{
					{
						TotalRowAmount: 50000,
						Amounts: []isbj.SettlementAmount{
							{Key: "care_type", Value: "ganztag", Amount: 50000},
						},
					},
				},
			},
		},
	}

	resp, err := svc.buildResponse(ctx, org.ID, period.ID, period.From, converted)
	if err != nil {
		t.Fatalf("buildResponse() error = %v", err)
	}

	if resp.MatchedCount != 0 {
		t.Errorf("MatchedCount = %d, want 0", resp.MatchedCount)
	}
	if resp.UnmatchedCount != 1 {
		t.Errorf("UnmatchedCount = %d, want 1", resp.UnmatchedCount)
	}
	if resp.Children[0].Matched {
		t.Error("expected child to NOT be matched")
	}
}

func TestBillPaymentsToAmountMap(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular},
		{Key: "ndh", Value: "ndh", Amount: 8000, RowType: models.RowTypeRegular},
		{Key: "care_type", Value: "ganztag", Amount: 5000, RowType: models.RowTypeRegular},
	}

	amounts, total := billPaymentsToAmountMap(payments, "")
	if total != 133000 {
		t.Errorf("expected total 133000, got %d", total)
	}
	if amounts["care_type:ganztag"] != 125000 {
		t.Errorf("expected care_type:ganztag=125000, got %d", amounts["care_type:ganztag"])
	}
	if amounts["ndh:ndh"] != 8000 {
		t.Errorf("expected ndh:ndh=8000, got %d", amounts["ndh:ndh"])
	}
}

func TestBillPaymentsToAmountMap_Empty(t *testing.T) {
	amounts, total := billPaymentsToAmountMap(nil, "")
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(amounts) != 0 {
		t.Errorf("expected empty map, got %d entries", len(amounts))
	}
}

func TestBillPaymentsToAmountMap_FilterByRowType(t *testing.T) {
	payments := []models.GovernmentFundingBillPayment{
		{Key: "care_type", Value: "ganztag", Amount: 96950, RowType: models.RowTypeRegular},
		{Key: "parent", Value: "meals", Amount: -2300, RowType: models.RowTypeRegular},
		{Key: "care_type", Value: "ganztag", Amount: 23856, RowType: models.RowTypeCorrection},
		{Key: "care_type", Value: "ganztag", Amount: 3199, RowType: models.RowTypeCorrection},
	}

	// Regular only
	regAmounts, regTotal := billPaymentsToAmountMap(payments, models.RowTypeRegular)
	if regTotal != 94650 { // 96950 - 2300
		t.Errorf("regular total: expected 94650, got %d", regTotal)
	}
	if regAmounts["care_type:ganztag"] != 96950 {
		t.Errorf("regular care_type:ganztag: expected 96950, got %d", regAmounts["care_type:ganztag"])
	}

	// Corrections only
	corrAmounts, corrTotal := billPaymentsToAmountMap(payments, models.RowTypeCorrection)
	if corrTotal != 27055 { // 23856 + 3199
		t.Errorf("correction total: expected 27055, got %d", corrTotal)
	}
	if corrAmounts["care_type:ganztag"] != 27055 {
		t.Errorf("correction care_type:ganztag: expected 27055, got %d", corrAmounts["care_type:ganztag"])
	}

	// All (empty filter)
	_, allTotal := billPaymentsToAmountMap(payments, "")
	if allTotal != 121705 { // 96950 - 2300 + 23856 + 3199
		t.Errorf("all total: expected 121705, got %d", allTotal)
	}
}

func TestCalcAmountsFromFunding(t *testing.T) {
	period := &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 120000},
			{Key: "ndh", Value: "ndh", Payment: 8000},
			{Key: "care_type", Value: "halbtag", Payment: 60000}, // won't match
		},
	}

	props := models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"}
	amounts, total := calcAmountsFromFunding(5, props, period)

	if total != 128000 {
		t.Errorf("expected total 128000, got %d", total)
	}
	if amounts["care_type:ganztag"] != 120000 {
		t.Errorf("expected care_type:ganztag=120000, got %d", amounts["care_type:ganztag"])
	}
	if amounts["ndh:ndh"] != 8000 {
		t.Errorf("expected ndh:ndh=8000, got %d", amounts["ndh:ndh"])
	}
}

func TestCalcAmountsFromFunding_NilPeriod(t *testing.T) {
	props := models.ContractProperties{"care_type": "ganztag"}
	amounts, total := calcAmountsFromFunding(5, props, nil)

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(amounts) != 0 {
		t.Errorf("expected empty map, got %d entries", len(amounts))
	}
}

// ============================================================
// Auto-discovery and name/birth parsing tests
// ============================================================

func TestParseBillChildName(t *testing.T) {
	tests := []struct {
		input     string
		wantFirst string
		wantLast  string
	}{
		{"Mevissen,Magnus Morgan", "Magnus Morgan", "Mevissen"},
		{"Hardt,Alva", "Alva", "Hardt"},
		{"Silva Elgueda,Caetano", "Caetano", "Silva Elgueda"},
		{"Conde Kleppe,Yanosh Rio", "Yanosh Rio", "Conde Kleppe"},
		{"Beetz,Wilda", "Wilda", "Beetz"},
		{"NoComma", "", "NoComma"},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			first, last := parseBillChildName(tt.input)
			if first != tt.wantFirst || last != tt.wantLast {
				t.Errorf("parseBillChildName(%q) = (%q, %q), want (%q, %q)",
					tt.input, first, last, tt.wantFirst, tt.wantLast)
			}
		})
	}
}

func TestParseBillBirthMonth(t *testing.T) {
	tests := []struct {
		input     string
		wantMonth time.Month
		wantYear  int
		wantErr   bool
	}{
		{"06.20", 6, 2020, false},
		{"01.23", 1, 2023, false},
		{"12.99", 12, 2099, false},
		{"", 0, 0, true},
		{"invalid", 0, 0, true},
		{"13.20", 0, 0, true},
		{"00.20", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			month, year, err := parseBillBirthMonth(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got (%d, %d)", month, year)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if month != tt.wantMonth || year != tt.wantYear {
				t.Errorf("got (%d, %d), want (%d, %d)", month, year, tt.wantMonth, tt.wantYear)
			}
		})
	}
}

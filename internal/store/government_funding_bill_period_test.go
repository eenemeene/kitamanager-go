package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGovernmentFundingBillPeriodStore_Create(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billtest@example.com")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          "Abrechnung_11-25.xlsx",
		FileSha256:        "abc123def456",
		FacilityName:      "Kita Sonnenschein",
		FacilityTotal:     500000,
		ContractBooking:   480000,
		CorrectionBooking: 20000,
		CreatedBy:         user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-12345678901-02",
				ChildName:     "Mustermann, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
					{Key: "ndh", Value: "ndh", Amount: 5000},
				},
			},
			{
				VoucherNumber: "GB-98765432109-01",
				ChildName:     "Müller, Anna",
				BirthDate:     "03.21",
				District:      3,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "halbtag", Amount: 80000},
				},
			},
		},
	}

	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if period.ID == 0 {
		t.Error("expected period ID to be set after create")
	}

	// Verify nested records were created
	var childCount int64
	db.Model(&models.GovernmentFundingBillChild{}).Where("period_id = ?", period.ID).Count(&childCount)
	if childCount != 2 {
		t.Errorf("expected 2 children, got %d", childCount)
	}

	var paymentCount int64
	db.Model(&models.GovernmentFundingBillPayment{}).
		Joins("JOIN government_funding_bill_children ON government_funding_bill_children.id = government_funding_bill_payments.child_id").
		Where("government_funding_bill_children.period_id = ?", period.ID).
		Count(&paymentCount)
	if paymentCount != 3 {
		t.Errorf("expected 3 payments, got %d", paymentCount)
	}
}

func TestGovernmentFundingBillPeriodStore_CreateEmptyChildren(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billtest2@example.com")
	ctx := context.Background()

	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          "empty.xlsx",
		FileSha256:        "emptyhash",
		FacilityName:      "Kita Leer",
		FacilityTotal:     0,
		ContractBooking:   0,
		CorrectionBooking: 0,
		CreatedBy:         user.ID,
	}

	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() with empty children error = %v", err)
	}

	if period.ID == 0 {
		t.Error("expected period ID to be set")
	}
}

func TestGovernmentFundingBillPeriodStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billtest3@example.com")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          "test.xlsx",
		FileSha256:        "hash123",
		FacilityName:      "Kita Test",
		FacilityTotal:     300000,
		ContractBooking:   280000,
		CorrectionBooking: 20000,
		CreatedBy:         user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-11111111111-01",
				ChildName:     "Kind, Eins",
				BirthDate:     "05.19",
				District:      2,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 150000},
					{Key: "ndh", Value: "ndh", Amount: 10000},
				},
			},
		},
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	found, err := s.FindByID(ctx, period.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != period.ID {
		t.Errorf("expected ID %d, got %d", period.ID, found.ID)
	}
	if found.FacilityName != "Kita Test" {
		t.Errorf("expected facility name 'Kita Test', got %q", found.FacilityName)
	}
	if found.OrganizationID != org.ID {
		t.Errorf("expected org ID %d, got %d", org.ID, found.OrganizationID)
	}

	// Verify children are preloaded
	if len(found.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(found.Children))
	}
	if found.Children[0].VoucherNumber != "GB-11111111111-01" {
		t.Errorf("expected voucher 'GB-11111111111-01', got %q", found.Children[0].VoucherNumber)
	}

	// Verify payments are preloaded
	if len(found.Children[0].Payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(found.Children[0].Payments))
	}
}

func TestGovernmentFundingBillPeriodStore_FindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	ctx := context.Background()

	_, err := s.FindByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
}

func TestGovernmentFundingBillPeriodStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "Test User", "billtest4@example.com")
	ctx := context.Background()

	// Create 3 periods for org1, 1 for org2
	for i := range 3 {
		month := time.Month(i + 1)
		to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
		p := &models.GovernmentFundingBillPeriod{
			OrganizationID: org1.ID,
			Period:         models.Period{From: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC), To: &to},
			FileName:       fmt.Sprintf("file_%d.xlsx", i),
			FileSha256:     fmt.Sprintf("hash_%d", i),
			FacilityName:   "Kita",
			CreatedBy:      user.ID,
		}
		if err := s.Create(ctx, p); err != nil {
			t.Fatalf("setup: Create() error = %v", err)
		}
	}
	toOrg2 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	if err := s.Create(ctx, &models.GovernmentFundingBillPeriod{
		OrganizationID: org2.ID,
		Period:         models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &toOrg2},
		FileName:       "other.xlsx",
		FileSha256:     "otherhash",
		FacilityName:   "Other Kita",
		CreatedBy:      user.ID,
	}); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	t.Run("returns only org1 periods", func(t *testing.T) {
		periods, total, err := s.FindByOrganization(ctx, org1.ID, 10, 0)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(periods) != 3 {
			t.Errorf("expected 3 periods, got %d", len(periods))
		}
	})

	t.Run("returns only org2 periods", func(t *testing.T) {
		periods, total, err := s.FindByOrganization(ctx, org2.ID, 10, 0)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(periods) != 1 {
			t.Errorf("expected 1 period, got %d", len(periods))
		}
	})

	t.Run("pagination limit", func(t *testing.T) {
		periods, total, err := s.FindByOrganization(ctx, org1.ID, 2, 0)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(periods) != 2 {
			t.Errorf("expected 2 periods (limit), got %d", len(periods))
		}
	})

	t.Run("pagination offset", func(t *testing.T) {
		periods, total, err := s.FindByOrganization(ctx, org1.ID, 10, 2)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(periods) != 1 {
			t.Errorf("expected 1 period (offset 2 of 3), got %d", len(periods))
		}
	})

	t.Run("ordered by from_date descending", func(t *testing.T) {
		periods, _, err := s.FindByOrganization(ctx, org1.ID, 10, 0)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		for i := 1; i < len(periods); i++ {
			if periods[i].From.After(periods[i-1].From) {
				t.Errorf("periods not ordered by from_date DESC: %v > %v", periods[i].From, periods[i-1].From)
			}
		}
	})

	t.Run("empty for unknown org", func(t *testing.T) {
		periods, total, err := s.FindByOrganization(ctx, 99999, 10, 0)
		if err != nil {
			t.Fatalf("FindByOrganization() error = %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
		if len(periods) != 0 {
			t.Errorf("expected 0 periods, got %d", len(periods))
		}
	})
}

func TestGovernmentFundingBillPeriodStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billtest5@example.com")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "delete-test.xlsx",
		FileSha256:     "deletehash",
		FacilityName:   "Kita Delete",
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-00000000000-01",
				ChildName:     "Delete, Child",
				BirthDate:     "01.22",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
				},
			},
		},
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	if err := s.Delete(ctx, period.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify period is deleted
	_, err := s.FindByID(ctx, period.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}

	// Verify cascade delete of children
	var childCount int64
	db.Model(&models.GovernmentFundingBillChild{}).Where("period_id = ?", period.ID).Count(&childCount)
	if childCount != 0 {
		t.Errorf("expected 0 children after cascade delete, got %d", childCount)
	}
}

func TestGovernmentFundingBillPeriodStore_FindByIDChildrenOrdered(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billtest6@example.com")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "order-test.xlsx",
		FileSha256:     "orderhash",
		FacilityName:   "Kita Order",
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-00000000001-01", ChildName: "Alpha, Child", BirthDate: "01.20", District: 1},
			{VoucherNumber: "GB-00000000002-01", ChildName: "Beta, Child", BirthDate: "02.20", District: 2},
			{VoucherNumber: "GB-00000000003-01", ChildName: "Gamma, Child", BirthDate: "03.20", District: 3},
		},
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	found, err := s.FindByID(ctx, period.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if len(found.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(found.Children))
	}

	// Verify children are ordered by ID ASC (insertion order)
	for i := 1; i < len(found.Children); i++ {
		if found.Children[i].ID <= found.Children[i-1].ID {
			t.Errorf("children not ordered by ID ASC: %d <= %d", found.Children[i].ID, found.Children[i-1].ID)
		}
	}
}

func TestGovernmentFundingBillPeriodStore_FindByOrganizationAndVoucherNumber(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	org2 := createTestOrganization(t, db, "Other Org")
	user := createTestUser(t, db, "Test User", "billvoucher@example.com")
	ctx := context.Background()

	// Create 3 bill periods for org, child "GB-VOUCHER-01" appears in 2 of them
	toNov := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period1 := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &toNov},
		FileName:       "nov.xlsx", FileSha256: "hash1", FacilityName: "Kita A", CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-VOUCHER-01", ChildName: "Kind, Eins", BirthDate: "01.20", District: 1},
			{VoucherNumber: "GB-VOUCHER-02", ChildName: "Kind, Zwei", BirthDate: "03.21", District: 2},
		},
	}
	toDec := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), To: &toDec},
		FileName:       "dec.xlsx", FileSha256: "hash2", FacilityName: "Kita A", CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-VOUCHER-01", ChildName: "Kind, Eins", BirthDate: "01.20", District: 1},
		},
	}
	toOct := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
	period3 := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: &toOct},
		FileName:       "oct.xlsx", FileSha256: "hash3", FacilityName: "Kita B", CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-VOUCHER-02", ChildName: "Kind, Zwei", BirthDate: "03.21", District: 2},
		},
	}
	// Bill in different org with same voucher
	toNov2 := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	periodOtherOrg := &models.GovernmentFundingBillPeriod{
		OrganizationID: org2.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &toNov2},
		FileName:       "other.xlsx", FileSha256: "hash4", FacilityName: "Kita Other", CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-VOUCHER-01", ChildName: "Kind, Eins", BirthDate: "01.20", District: 1},
		},
	}

	for _, p := range []*models.GovernmentFundingBillPeriod{period1, period2, period3, periodOtherOrg} {
		if err := s.Create(ctx, p); err != nil {
			t.Fatalf("setup: Create() error = %v", err)
		}
	}

	t.Run("returns bills containing the voucher", func(t *testing.T) {
		results, err := s.FindByOrganizationAndVoucherNumber(ctx, org.ID, "GB-VOUCHER-01")
		if err != nil {
			t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 appearances, got %d", len(results))
		}
		// Should be ordered by from_date ASC
		if results[0].BillFrom != "2025-11-01" {
			t.Errorf("expected first bill_from '2025-11-01', got %q", results[0].BillFrom)
		}
		if results[1].BillFrom != "2025-12-01" {
			t.Errorf("expected second bill_from '2025-12-01', got %q", results[1].BillFrom)
		}
		if results[0].BillID != period1.ID {
			t.Errorf("expected first bill_id %d, got %d", period1.ID, results[0].BillID)
		}
		if results[1].BillID != period2.ID {
			t.Errorf("expected second bill_id %d, got %d", period2.ID, results[1].BillID)
		}
		if results[0].FacilityName != "Kita A" {
			t.Errorf("expected facility_name 'Kita A', got %q", results[0].FacilityName)
		}
	})

	t.Run("does not return bills from other organizations", func(t *testing.T) {
		results, err := s.FindByOrganizationAndVoucherNumber(ctx, org.ID, "GB-VOUCHER-01")
		if err != nil {
			t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
		}
		for _, r := range results {
			if r.BillID == periodOtherOrg.ID {
				t.Errorf("should not include bill from other org (ID %d)", periodOtherOrg.ID)
			}
		}
	})

	t.Run("returns only bills with that specific voucher", func(t *testing.T) {
		results, err := s.FindByOrganizationAndVoucherNumber(ctx, org.ID, "GB-VOUCHER-02")
		if err != nil {
			t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 appearances for VOUCHER-02, got %d", len(results))
		}
	})

	t.Run("returns empty for unknown voucher", func(t *testing.T) {
		results, err := s.FindByOrganizationAndVoucherNumber(ctx, org.ID, "GB-NONEXISTENT-01")
		if err != nil {
			t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 appearances for unknown voucher, got %d", len(results))
		}
	})

	t.Run("returns empty for unknown organization", func(t *testing.T) {
		results, err := s.FindByOrganizationAndVoucherNumber(ctx, 99999, "GB-VOUCHER-01")
		if err != nil {
			t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 appearances for unknown org, got %d", len(results))
		}
	})
}

func TestGovernmentFundingBillPeriodStore_FindFacilityTotalsByOrganizationInDateRange(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	org2 := createTestOrganization(t, db, "Other Org")
	user := createTestUser(t, db, "Test User", "billtotals@example.com")
	ctx := context.Background()

	// Create bills: Jan, Feb, Mar for org; Jan for org2
	for _, m := range []time.Month{time.January, time.February, time.March} {
		to := time.Date(2025, m+1, 0, 0, 0, 0, 0, time.UTC)
		p := &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: time.Date(2025, m, 1, 0, 0, 0, 0, time.UTC), To: &to},
			FileName:       fmt.Sprintf("file_%d.xlsx", m), FileSha256: fmt.Sprintf("hash_%d", m), FacilityName: "Kita",
			FacilityTotal: int(m) * 100000, // Jan=100000, Feb=200000, Mar=300000
			CreatedBy:     user.ID,
		}
		if err := s.Create(ctx, p); err != nil {
			t.Fatalf("setup: Create() error = %v", err)
		}
	}
	toJanOrg2 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	if err := s.Create(ctx, &models.GovernmentFundingBillPeriod{
		OrganizationID: org2.ID,
		Period:         models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &toJanOrg2},
		FileName:       "other.xlsx", FileSha256: "otherhash", FacilityName: "Other",
		FacilityTotal: 999999,
		CreatedBy:     user.ID,
	}); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	t.Run("returns all bills in range", func(t *testing.T) {
		result, err := s.FindFacilityTotalsByOrganizationInDateRange(ctx, org.ID,
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if result[jan] != 100000 {
			t.Errorf("expected Jan=100000, got %d", result[jan])
		}
		feb := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
		if result[feb] != 200000 {
			t.Errorf("expected Feb=200000, got %d", result[feb])
		}
		mar := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
		if result[mar] != 300000 {
			t.Errorf("expected Mar=300000, got %d", result[mar])
		}
	})

	t.Run("partial range", func(t *testing.T) {
		result, err := s.FindFacilityTotalsByOrganizationInDateRange(ctx, org.ID,
			time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("excludes other org", func(t *testing.T) {
		result, err := s.FindFacilityTotalsByOrganizationInDateRange(ctx, org.ID,
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if result[jan] != 100000 {
			t.Errorf("expected 100000 (not org2's 999999), got %d", result[jan])
		}
	})

	t.Run("empty for no bills in range", func(t *testing.T) {
		result, err := s.FindFacilityTotalsByOrganizationInDateRange(ctx, org.ID,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 entries, got %d", len(result))
		}
	})

	t.Run("sums multiple bills in same month", func(t *testing.T) {
		// Add a second bill for January (correction)
		toJan2 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		if err := s.Create(ctx, &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), To: &toJan2},
			FileName:       "correction.xlsx", FileSha256: "corrhash", FacilityName: "Kita",
			FacilityTotal: 50000,
			CreatedBy:     user.ID,
		}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		result, err := s.FindFacilityTotalsByOrganizationInDateRange(ctx, org.ID,
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if result[jan] != 150000 {
			t.Errorf("expected 150000 (100000 + 50000), got %d", result[jan])
		}
	})
}

// TestGovernmentFundingBillPeriodStore_FindByOrganizationAndVoucherNumber_DuplicateInSameBill
// tests the edge case where the same voucher number appears multiple times in a single bill
// (e.g. correction rows). The JOIN will produce one row per child entry, so the query may
// return duplicate bill IDs. This test verifies the actual behavior.
func TestGovernmentFundingBillPeriodStore_FindByOrganizationAndVoucherNumber_DuplicateInSameBill(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Test User", "billdup@example.com")
	ctx := context.Background()

	// Create a bill where the same voucher appears as two separate child rows
	// (this can happen with correction entries in ISBJ files)
	toNov := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &toNov},
		FileName:       "dup.xlsx", FileSha256: "duphash", FacilityName: "Kita Dup",
		CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-DUPVOUCHER-01", ChildName: "Dup, Child", BirthDate: "01.20", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
			{VoucherNumber: "GB-DUPVOUCHER-01", ChildName: "Dup, Child", BirthDate: "01.20", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: -50000}}},
		},
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("setup: Create() error = %v", err)
	}

	results, err := s.FindByOrganizationAndVoucherNumber(ctx, org.ID, "GB-DUPVOUCHER-01")
	if err != nil {
		t.Fatalf("FindByOrganizationAndVoucherNumber() error = %v", err)
	}

	// DISTINCT in the query should deduplicate: only 1 result for the 1 bill
	if len(results) != 1 {
		t.Fatalf("expected 1 result (deduplicated), got %d", len(results))
	}
	if results[0].BillID != period.ID {
		t.Errorf("expected bill_id %d, got %d", period.ID, results[0].BillID)
	}
	if results[0].FacilityName != "Kita Dup" {
		t.Errorf("expected facility_name 'Kita Dup', got %q", results[0].FacilityName)
	}
}

func TestGovernmentFundingBillPeriodStore_ExistsByOrgAndHash(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "hash_exists@example.com")
	ctx := context.Background()

	// Initially no bills exist.
	exists, err := s.ExistsByOrgAndHash(ctx, org.ID, "hashA")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if exists {
		t.Error("expected false for non-existent hash")
	}

	// Create a bill.
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "hashA",
		FacilityName:   "Kita",
		CreatedBy:      user.ID,
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Now it should exist.
	exists, err = s.ExistsByOrgAndHash(ctx, org.ID, "hashA")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if !exists {
		t.Error("expected true for existing hash")
	}

	// Different hash should not exist.
	exists, err = s.ExistsByOrgAndHash(ctx, org.ID, "hashB")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if exists {
		t.Error("expected false for different hash")
	}

	// Same hash, different org should not exist.
	org2 := createTestOrganization(t, db, "Org 2")
	exists, err = s.ExistsByOrgAndHash(ctx, org2.ID, "hashA")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if exists {
		t.Error("expected false for same hash in different org")
	}
}

func TestGovernmentFundingBillPeriodStore_ExistsByOrgAndMonth(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "month_exists@example.com")
	ctx := context.Background()

	nov := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	dec := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)

	// Initially no bills exist.
	exists, err := s.ExistsByOrgAndMonth(ctx, org.ID, nov)
	if err != nil {
		t.Fatalf("ExistsByOrgAndMonth() error = %v", err)
	}
	if exists {
		t.Error("expected false for non-existent month")
	}

	// Create a bill for November.
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: nov, To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "hash1",
		FacilityName:   "Kita",
		CreatedBy:      user.ID,
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// November should exist.
	exists, err = s.ExistsByOrgAndMonth(ctx, org.ID, nov)
	if err != nil {
		t.Fatalf("ExistsByOrgAndMonth() error = %v", err)
	}
	if !exists {
		t.Error("expected true for existing month")
	}

	// December should not exist.
	exists, err = s.ExistsByOrgAndMonth(ctx, org.ID, dec)
	if err != nil {
		t.Fatalf("ExistsByOrgAndMonth() error = %v", err)
	}
	if exists {
		t.Error("expected false for different month")
	}

	// Same month, different org should not exist.
	org2 := createTestOrganization(t, db, "Org 2")
	exists, err = s.ExistsByOrgAndMonth(ctx, org2.ID, nov)
	if err != nil {
		t.Fatalf("ExistsByOrgAndMonth() error = %v", err)
	}
	if exists {
		t.Error("expected false for same month in different org")
	}
}

func TestGovernmentFundingBillPeriodStore_ExistsByOrgAndHash_AfterDelete(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "hash_delete@example.com")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "delhash",
		FacilityName:   "Kita",
		CreatedBy:      user.ID,
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Exists before delete.
	exists, err := s.ExistsByOrgAndHash(ctx, org.ID, "delhash")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if !exists {
		t.Error("expected true before delete")
	}

	// Delete.
	if err := s.Delete(ctx, period.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// No longer exists after delete.
	exists, err = s.ExistsByOrgAndHash(ctx, org.ID, "delhash")
	if err != nil {
		t.Fatalf("ExistsByOrgAndHash() error = %v", err)
	}
	if exists {
		t.Error("expected false after delete")
	}
}

func TestGovernmentFundingBillPeriodStore_ExistsByOrgAndMonth_AfterDelete(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "month_delete@example.com")
	ctx := context.Background()

	nov := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: nov, To: &to},
		FileName:       "test.xlsx",
		FileSha256:     "hash_month_del",
		FacilityName:   "Kita",
		CreatedBy:      user.ID,
	}
	if err := s.Create(ctx, period); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete.
	if err := s.Delete(ctx, period.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// No longer exists after delete.
	exists, err := s.ExistsByOrgAndMonth(ctx, org.ID, nov)
	if err != nil {
		t.Fatalf("ExistsByOrgAndMonth() error = %v", err)
	}
	if exists {
		t.Error("expected false after delete")
	}
}

func TestGovernmentFundingBillPeriodStore_FindChildEntriesByOrgAndVoucherNumbers(t *testing.T) {
	db := setupTestDB(t)
	s := NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billstore_entries@example.com")
	ctx := context.Background()

	voucher1 := "GB-11111111111-01"
	voucher2 := "GB-22222222222-01"

	// Create two bills with overlapping children
	for _, month := range []time.Month{10, 11} {
		from := time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
		period := &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: from, To: &to},
			FileName:       fmt.Sprintf("bill_%d.xlsx", month),
			FileSha256:     fmt.Sprintf("hash_%d", month),
			FacilityName:   "Test Kita",
			CreatedBy:      user.ID,
			Children: []models.GovernmentFundingBillChild{
				{
					VoucherNumber: voucher1,
					ChildName:     "Kind Eins",
					BirthDate:     "01.20",
					District:      1,
					Payments: []models.GovernmentFundingBillPayment{
						{Key: "care_type", Value: "ganztag", Amount: 120000},
						{Key: "ndh", Value: "ndh", Amount: 8000},
					},
				},
				{
					VoucherNumber: voucher2,
					ChildName:     "Kind Zwei",
					BirthDate:     "06.21",
					District:      2,
					Payments: []models.GovernmentFundingBillPayment{
						{Key: "care_type", Value: "halbtag", Amount: 60000},
					},
				},
			},
		}
		if err := db.Create(period).Error; err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	t.Run("single voucher", func(t *testing.T) {
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org.ID, []string{voucher1})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 entries for voucher1, got %d", len(results))
		}
		// Should be chronological
		if results[0].BillFrom.Month() != 10 {
			t.Errorf("expected first entry month 10, got %d", results[0].BillFrom.Month())
		}
		if results[1].BillFrom.Month() != 11 {
			t.Errorf("expected second entry month 11, got %d", results[1].BillFrom.Month())
		}
		// Payments should be preloaded
		if len(results[0].Child.Payments) != 2 {
			t.Errorf("expected 2 payments, got %d", len(results[0].Child.Payments))
		}
	})

	t.Run("multiple vouchers", func(t *testing.T) {
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org.ID, []string{voucher1, voucher2})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(results) != 4 {
			t.Fatalf("expected 4 entries total, got %d", len(results))
		}
	})

	t.Run("no matching voucher", func(t *testing.T) {
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org.ID, []string{"GB-00000000000-00"})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 entries, got %d", len(results))
		}
	})

	t.Run("empty voucher list", func(t *testing.T) {
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org.ID, []string{})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if results != nil {
			t.Errorf("expected nil, got %v", results)
		}
	})

	t.Run("org isolation", func(t *testing.T) {
		org2 := createTestOrganization(t, db, "Other Org")
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org2.ID, []string{voucher1})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 entries for other org, got %d", len(results))
		}
	})

	t.Run("period metadata populated", func(t *testing.T) {
		results, err := s.FindChildEntriesByOrgAndVoucherNumbers(ctx, org.ID, []string{voucher1})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		for _, r := range results {
			if r.BillPeriodID == 0 {
				t.Error("expected non-zero BillPeriodID")
			}
			if r.FacilityName != "Test Kita" {
				t.Errorf("expected FacilityName 'Test Kita', got %q", r.FacilityName)
			}
			if r.BillTo == nil {
				t.Error("expected BillTo to be set")
			}
		}
	})
}

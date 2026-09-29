package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestChildBillingHistory_EmptyNoVoucher(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	ctx := context.Background()

	// Child with no voucher number on contract
	child := createTestChildWithContract(t, db, "Anna", "Müller", org.ID, getDefaultSection(t, db, org.ID).ID)

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if result.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.ChildID)
	}
	if len(result.VoucherNumbers) != 0 {
		t.Errorf("expected 0 voucher numbers, got %d", len(result.VoucherNumbers))
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
	if result.TotalBilled != 0 {
		t.Errorf("expected total_billed 0, got %d", result.TotalBilled)
	}
}

func TestChildBillingHistory_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	ctx := context.Background()

	_, err := svc.ChildBillingHistory(ctx, 99999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent child, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestChildBillingHistory_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	ctx := context.Background()

	child := createTestChild(t, db, "Max", "Test", org1.ID)

	_, err := svc.ChildBillingHistory(ctx, child.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestChildBillingHistory_SingleBillMatch(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist1@example.com", "password")
	ctx := context.Background()

	// Set up funding config
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)
	createTestFundingProperty(t, db, fundingPeriod.ID, "ndh", "ndh", 8000, -1, -1)

	voucher := "GB-11111111111-01"
	child, _ := createChildWithVoucher(t, db, "Max", "Mustermann", org.ID, section.ID, voucher,
		time.Date(2020, 3, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"},
	)

	// Create bill for Nov 2025
	createBillFixture(t, db, org.ID, user.ID, 2025, 11, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Mustermann, Max",
			BirthDate:     "03.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
				{Key: "ndh", Value: "ndh", Amount: 8000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if result.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.ChildID)
	}
	if result.ChildName != "Max Mustermann" {
		t.Errorf("expected child_name 'Max Mustermann', got %q", result.ChildName)
	}
	if len(result.VoucherNumbers) != 1 || result.VoucherNumbers[0] != voucher {
		t.Errorf("expected voucher numbers [%s], got %v", voucher, result.VoucherNumbers)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if entry.BillFrom != "2025-11-01" {
		t.Errorf("expected bill_from '2025-11-01', got %q", entry.BillFrom)
	}
	if entry.Status != "match" {
		t.Errorf("expected status 'match', got %q", entry.Status)
	}
	if entry.BillTotal != 128000 {
		t.Errorf("expected bill_total 128000, got %d", entry.BillTotal)
	}
	if entry.CalcTotal == nil || *entry.CalcTotal != 128000 {
		t.Errorf("expected calculated_total 128000, got %v", entry.CalcTotal)
	}
	if entry.Difference == nil || *entry.Difference != 0 {
		t.Errorf("expected difference 0, got %v", entry.Difference)
	}
	if entry.Age == nil {
		t.Fatal("expected age to be set")
	}
	if *entry.Age != 5 { // Born 2020-03-10, bill date 2025-11-01 → age 5
		t.Errorf("expected age 5, got %d", *entry.Age)
	}
	if entry.ContractID == nil {
		t.Error("expected contract_id to be set")
	}
	if len(entry.Properties) < 2 {
		t.Errorf("expected at least 2 properties, got %d", len(entry.Properties))
	}

	// Check aggregated totals
	if result.TotalBilled != 128000 {
		t.Errorf("expected total_billed 128000, got %d", result.TotalBilled)
	}
	if result.TotalCalculated != 128000 {
		t.Errorf("expected total_calculated 128000, got %d", result.TotalCalculated)
	}
	if result.TotalDifference != 0 {
		t.Errorf("expected total_difference 0, got %d", result.TotalDifference)
	}
}

func TestChildBillingHistory_MultipleBillsChronological(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist2@example.com", "password")
	ctx := context.Background()

	// Set up funding config
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-22222222222-01"
	child, _ := createChildWithVoucher(t, db, "Emma", "Schmidt", org.ID, section.ID, voucher,
		time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Create bills for 3 consecutive months
	for month := time.Month(9); month <= 11; month++ {
		createBillFixture(t, db, org.ID, user.ID, 2025, month, []models.GovernmentFundingBillChild{
			{
				VoucherNumber: voucher,
				ChildName:     "Schmidt, Emma",
				BirthDate:     "06.21",
				District:      3,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 120000},
				},
			},
		})
	}

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.Entries))
	}

	// Verify chronological order
	expectedMonths := []string{"2025-09-01", "2025-10-01", "2025-11-01"}
	for i, entry := range result.Entries {
		if entry.BillFrom != expectedMonths[i] {
			t.Errorf("entry[%d]: expected bill_from %q, got %q", i, expectedMonths[i], entry.BillFrom)
		}
		if entry.Status != "match" {
			t.Errorf("entry[%d]: expected status 'match', got %q", i, entry.Status)
		}
	}

	// 3 months * 120000 = 360000
	if result.TotalBilled != 360000 {
		t.Errorf("expected total_billed 360000, got %d", result.TotalBilled)
	}
	if result.TotalCalculated != 360000 {
		t.Errorf("expected total_calculated 360000, got %d", result.TotalCalculated)
	}
	if result.TotalDifference != 0 {
		t.Errorf("expected total_difference 0, got %d", result.TotalDifference)
	}
}

func TestChildBillingHistory_BillingDifference(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist3@example.com", "password")
	ctx := context.Background()

	// Set up funding config: ganztag costs 120000
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-33333333333-01"
	child, _ := createChildWithVoucher(t, db, "Leo", "Weber", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bill with wrong amount (110000 instead of 120000)
	createBillFixture(t, db, org.ID, user.ID, 2025, 10, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Weber, Leo",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 110000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if entry.Status != "difference" {
		t.Errorf("expected status 'difference', got %q", entry.Status)
	}
	if entry.BillTotal != 110000 {
		t.Errorf("expected bill_total 110000, got %d", entry.BillTotal)
	}
	if entry.CalcTotal == nil || *entry.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %v", entry.CalcTotal)
	}
	if entry.Difference == nil || *entry.Difference != -10000 {
		t.Errorf("expected difference -10000, got %v", entry.Difference)
	}

	if result.TotalDifference != -10000 {
		t.Errorf("expected total_difference -10000, got %d", result.TotalDifference)
	}
}

func TestChildBillingHistory_NoContract(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist4@example.com", "password")
	ctx := context.Background()

	voucher := "GB-44444444444-01"
	// Contract ends before the bill date
	contractEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	child, _ := createChildWithVoucher(t, db, "Mia", "Fischer", org.ID, section.ID, voucher,
		time.Date(2020, 5, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bill for Nov 2025 — after contract ended
	createBillFixture(t, db, org.ID, user.ID, 2025, 11, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Fischer, Mia",
			BirthDate:     "05.20",
			District:      2,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if entry.Status != "no_contract" {
		t.Errorf("expected status 'no_contract', got %q", entry.Status)
	}
	if entry.ContractID != nil {
		t.Error("expected contract_id to be nil for no_contract")
	}
	if entry.CalcTotal != nil {
		t.Error("expected calc_total to be nil for no_contract")
	}
	if entry.Age != nil {
		t.Error("expected age to be nil for no_contract")
	}
	// Bill total should still be tracked
	if entry.BillTotal != 120000 {
		t.Errorf("expected bill_total 120000, got %d", entry.BillTotal)
	}
}

func TestChildBillingHistory_NoFundingConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist5@example.com", "password")
	ctx := context.Background()

	// No funding config created for this org's state

	voucher := "GB-55555555555-01"
	child, _ := createChildWithVoucher(t, db, "Paul", "Becker", org.ID, section.ID, voucher,
		time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	createBillFixture(t, db, org.ID, user.ID, 2025, 10, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Becker, Paul",
			BirthDate:     "08.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if entry.Status != "no_funding_config" {
		t.Errorf("expected status 'no_funding_config', got %q", entry.Status)
	}
	if entry.CalcTotal != nil {
		t.Error("expected calc_total to be nil when no funding config")
	}
	if entry.Age == nil {
		t.Error("expected age to be set even without funding config")
	}
}

func TestChildBillingHistory_MultipleVouchers(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist6@example.com", "password")
	ctx := context.Background()

	// Set up funding config
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	// Child with two contracts, different voucher numbers (contract renewal)
	voucher1 := "GB-66666666666-01"
	voucher2 := "GB-66666666666-02"
	birthdate := time.Date(2020, 4, 15, 0, 0, 0, 0, time.UTC)

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Sophie",
			LastName:       "Braun",
			Birthdate:      birthdate,
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Create child_voucher entries
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: voucher1, FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: voucher2, FirstSeen: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)})

	// First contract: Jan-Jun 2025
	contract1End := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &contract1End},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}).Error; err != nil {
		t.Fatalf("create contract1: %v", err)
	}

	// Second contract: Jul 2025 onwards
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}).Error; err != nil {
		t.Fatalf("create contract2: %v", err)
	}

	// Bill with voucher1 in June
	createBillFixture(t, db, org.ID, user.ID, 2025, 6, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher1,
			ChildName:     "Braun, Sophie",
			BirthDate:     "04.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	// Bill with voucher2 in September
	createBillFixture(t, db, org.ID, user.ID, 2025, 9, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher2,
			ChildName:     "Braun, Sophie",
			BirthDate:     "04.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	// VoucherNumbers shows all full vouchers seen in bills (both suffixes)
	if len(result.VoucherNumbers) != 2 {
		t.Errorf("expected 2 voucher numbers from bills, got %d: %v", len(result.VoucherNumbers), result.VoucherNumbers)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}

	// First entry: June with voucher1
	if result.Entries[0].VoucherNumber != voucher1 {
		t.Errorf("entry[0]: expected voucher %s, got %s", voucher1, result.Entries[0].VoucherNumber)
	}
	if result.Entries[0].Status != "match" {
		t.Errorf("entry[0]: expected status 'match', got %q", result.Entries[0].Status)
	}

	// Second entry: September with voucher2
	if result.Entries[1].VoucherNumber != voucher2 {
		t.Errorf("entry[1]: expected voucher %s, got %s", voucher2, result.Entries[1].VoucherNumber)
	}
	if result.Entries[1].Status != "match" {
		t.Errorf("entry[1]: expected status 'match', got %q", result.Entries[1].Status)
	}

	// Aggregated totals
	if result.TotalBilled != 240000 {
		t.Errorf("expected total_billed 240000, got %d", result.TotalBilled)
	}
}

func TestChildBillingHistory_FundingConfigChanges(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist7@example.com", "password")
	ctx := context.Background()

	// Set up funding config with rate change mid-year
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")

	// Period 1: Jan-Jun 2025 at 110000
	period1End := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	fundingPeriod1 := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &period1End, 39.0)
	createTestFundingProperty(t, db, fundingPeriod1.ID, "care_type", "ganztag", 110000, -1, -1)

	// Period 2: Jul 2025 onwards at 120000
	fundingPeriod2 := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod2.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-77777777777-01"
	child, _ := createChildWithVoucher(t, db, "Lina", "Hoffmann", org.ID, section.ID, voucher,
		time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// June bill with old rate
	createBillFixture(t, db, org.ID, user.ID, 2025, 6, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Hoffmann, Lina",
			BirthDate:     "01.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 110000},
			},
		},
	})

	// September bill with new rate
	createBillFixture(t, db, org.ID, user.ID, 2025, 9, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Hoffmann, Lina",
			BirthDate:     "01.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}

	// June: billed 110000, calc 110000 → match
	june := result.Entries[0]
	if june.Status != "match" {
		t.Errorf("June: expected 'match', got %q", june.Status)
	}
	if june.BillTotal != 110000 {
		t.Errorf("June: expected bill_total 110000, got %d", june.BillTotal)
	}
	if june.CalcTotal == nil || *june.CalcTotal != 110000 {
		t.Errorf("June: expected calc_total 110000, got %v", june.CalcTotal)
	}

	// September: billed 120000, calc 120000 → match
	sep := result.Entries[1]
	if sep.Status != "match" {
		t.Errorf("September: expected 'match', got %q", sep.Status)
	}
	if sep.BillTotal != 120000 {
		t.Errorf("September: expected bill_total 120000, got %d", sep.BillTotal)
	}
	if sep.CalcTotal == nil || *sep.CalcTotal != 120000 {
		t.Errorf("September: expected calc_total 120000, got %v", sep.CalcTotal)
	}
}

func TestChildBillingHistory_OrganizationIsolation(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section1 := getDefaultSection(t, db, org1.ID)
	section2 := getDefaultSection(t, db, org2.ID)
	user := createTestUser(t, db, "User", "billing_hist8@example.com", "password")
	ctx := context.Background()

	voucher1 := "GB-88888888881-01"
	voucher2 := "GB-88888888882-01"

	// Different voucher numbers for children in different orgs
	child1, _ := createChildWithVoucher(t, db, "Child", "Org1", org1.ID, section1.ID, voucher1,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	createChildWithVoucher(t, db, "Child", "Org2", org2.ID, section2.ID, voucher2,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bills in org1
	createBillFixture(t, db, org1.ID, user.ID, 2025, 10, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher1,
			ChildName:     "Org1, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 100000},
			},
		},
	})

	// Bills in org2
	createBillFixture(t, db, org2.ID, user.ID, 2025, 10, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher2,
			ChildName:     "Org2, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 200000},
			},
		},
	})
	createBillFixture(t, db, org2.ID, user.ID, 2025, 11, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher2,
			ChildName:     "Org2, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 200000},
			},
		},
	})

	// Child1 in org1 should only see org1's bill
	result, err := svc.ChildBillingHistory(ctx, child1.ID, org1.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry for org1, got %d", len(result.Entries))
	}
	if result.Entries[0].BillTotal != 100000 {
		t.Errorf("expected bill_total 100000 (org1), got %d", result.Entries[0].BillTotal)
	}
}

func TestChildBillingHistory_NoBills(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	ctx := context.Background()

	voucher := "GB-99999999999-01"
	child, _ := createChildWithVoucher(t, db, "Felix", "Klein", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// No bills uploaded
	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.VoucherNumbers) != 1 {
		t.Errorf("expected 1 voucher number, got %d", len(result.VoucherNumbers))
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
	if result.TotalBilled != 0 {
		t.Errorf("expected total_billed 0, got %d", result.TotalBilled)
	}
}

func TestChildBillingHistory_MixedStatuses(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_hist9@example.com", "password")
	ctx := context.Background()

	// Funding config only covers Jul 2025 onwards
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-10101010101-01"
	// Contract active Jul-Dec 2025
	contractStart := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	contractEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	child, _ := createChildWithVoucher(t, db, "Tim", "Wolf", org.ID, section.ID, voucher,
		time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
		contractStart, &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// June bill: voucher in bill but contract not yet active → no_contract
	createBillFixture(t, db, org.ID, user.ID, 2025, 6, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher, ChildName: "Wolf, Tim", BirthDate: "02.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	// August bill: contract active, funding config available, amounts match → match
	createBillFixture(t, db, org.ID, user.ID, 2025, 8, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher, ChildName: "Wolf, Tim", BirthDate: "02.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	// October bill: contract active, funding config available, wrong amount → difference
	createBillFixture(t, db, org.ID, user.ID, 2025, 10, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher, ChildName: "Wolf, Tim", BirthDate: "02.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 115000},
			},
		},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.Entries))
	}

	// Entry 0: June → no_contract
	if result.Entries[0].Status != "no_contract" {
		t.Errorf("June: expected 'no_contract', got %q", result.Entries[0].Status)
	}

	// Entry 1: August → match
	if result.Entries[1].Status != "match" {
		t.Errorf("August: expected 'match', got %q", result.Entries[1].Status)
	}

	// Entry 2: October → difference
	if result.Entries[2].Status != "difference" {
		t.Errorf("October: expected 'difference', got %q", result.Entries[2].Status)
	}

	// Totals
	if result.TotalBilled != 355000 { // 120000 + 120000 + 115000
		t.Errorf("expected total_billed 355000, got %d", result.TotalBilled)
	}
}

func TestChildBillingHistory_RunningDifference(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_running@example.com", "password")
	ctx := context.Background()

	// Funding config: ganztag = 120000
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-77777777777-77"
	child, _ := createChildWithVoucher(t, db, "Test", "Running", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Month 1: underpaid by 100 (bill=119900, calc=120000, diff=-100)
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Running, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 119900},
			}},
	})

	// Month 2: underpaid by 100 again (running = -200)
	createBillFixture(t, db, org.ID, user.ID, 2025, 2, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Running, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 119900},
			}},
	})

	// Month 3: correction - overpaid by 200 to compensate (running = 0)
	createBillFixture(t, db, org.ID, user.ID, 2025, 3, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Running, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120200},
			}},
	})

	// Month 4: exact match (running stays 0)
	createBillFixture(t, db, org.ID, user.ID, 2025, 4, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Running, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result.Entries))
	}

	// Verify running differences
	expectedRunning := []int{-100, -200, 0, 0}
	for i, expected := range expectedRunning {
		if result.Entries[i].RunningDifference != expected {
			t.Errorf("entry[%d]: expected running_difference %d, got %d",
				i, expected, result.Entries[i].RunningDifference)
		}
	}
}

func TestChildBillingHistory_RunningDifferenceSkipsNoContract(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_running2@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-88888888888-88"
	// Contract only covers Feb onwards
	contractStart := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	child, _ := createChildWithVoucher(t, db, "Test", "Skip", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		contractStart, nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Month 1: no contract (bill exists but diff is nil → running stays 0)
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Skip, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 100000},
			}},
	})

	// Month 2: has contract, underpaid by 500 (running = -500)
	createBillFixture(t, db, org.ID, user.ID, 2025, 2, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Skip, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 119500},
			}},
	})

	// Month 3: exact match (running stays -500)
	createBillFixture(t, db, org.ID, user.ID, 2025, 3, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Skip, Test", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.Entries))
	}

	// Entry 0: no_contract → running stays 0
	if result.Entries[0].Status != "no_contract" {
		t.Errorf("entry[0]: expected no_contract, got %s", result.Entries[0].Status)
	}
	if result.Entries[0].RunningDifference != 0 {
		t.Errorf("entry[0]: expected running 0, got %d", result.Entries[0].RunningDifference)
	}

	// Entry 1: diff = -500, running = -500
	if result.Entries[1].RunningDifference != -500 {
		t.Errorf("entry[1]: expected running -500, got %d", result.Entries[1].RunningDifference)
	}

	// Entry 2: diff = 0, running stays -500
	if result.Entries[2].RunningDifference != -500 {
		t.Errorf("entry[2]: expected running -500, got %d", result.Entries[2].RunningDifference)
	}
}

func TestChildBillingHistory_CrossSuffixVoucherMatching(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billing_cross@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	// Child has contract with voucher suffix -08
	currentVoucher := "GB-56589936966-08"
	child, _ := createChildWithVoucher(t, db, "Cross", "Suffix", org.ID, section.ID, currentVoucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Add older voucher suffixes to child_vouchers
	oldVoucher := "GB-56589936966-02"
	midVoucher := "GB-56589936966-04"
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: oldVoucher, FirstSeen: time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC)})
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: midVoucher, FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})

	// Bills exist with OLDER suffix -02
	createBillFixture(t, db, org.ID, user.ID, 2024, 10, []models.GovernmentFundingBillChild{
		{VoucherNumber: oldVoucher, ChildName: "Suffix, Cross", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	// Bill with another suffix -04
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: midVoucher, ChildName: "Suffix, Cross", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	// Bill with current suffix -08
	createBillFixture(t, db, org.ID, user.ID, 2025, 6, []models.GovernmentFundingBillChild{
		{VoucherNumber: currentVoucher, ChildName: "Suffix, Cross", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	result, err := svc.ChildBillingHistory(ctx, child.ID, org.ID)
	if err != nil {
		t.Fatalf("ChildBillingHistory() error = %v", err)
	}

	// All 3 bills should be found despite different suffixes
	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 entries (across suffixes -02, -04, -08), got %d", len(result.Entries))
	}

	// All should have matched contracts (not no_contract)
	for i, entry := range result.Entries {
		if entry.Status == "no_contract" {
			t.Errorf("entry[%d] (%s, voucher %s): expected matched, got no_contract",
				i, entry.BillFrom, entry.VoucherNumber)
		}
	}
}

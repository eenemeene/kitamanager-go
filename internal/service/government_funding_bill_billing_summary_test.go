package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ============================================================
// ChildrenBillingSummary tests
// ============================================================

func TestChildrenBillingSummary_Empty(t *testing.T) {
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

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("ChildrenBillingSummary() error = %v", err)
	}
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(result.Children))
	}
}

func TestChildrenBillingSummary_SingleChildMatch(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary1@example.com", "password")
	ctx := context.Background()

	// Funding config
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-SUMMARY-001"
	createChildWithVoucher(t, db, "Max", "Test", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Two months of bills, both matching
	for _, month := range []time.Month{1, 2} {
		createBillFixture(t, db, org.ID, user.ID, 2025, month, []models.GovernmentFundingBillChild{
			{VoucherNumber: voucher, ChildName: "Test, Max", BirthDate: "01.20", District: 1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 120000},
				}},
		})
	}

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("ChildrenBillingSummary() error = %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	child := result.Children[0]
	if child.TotalBilled != 240000 {
		t.Errorf("expected total_billed 240000, got %d", child.TotalBilled)
	}
	if child.TotalCalculated != 240000 {
		t.Errorf("expected total_calculated 240000, got %d", child.TotalCalculated)
	}
	if child.TotalDifference != 0 {
		t.Errorf("expected total_difference 0, got %d", child.TotalDifference)
	}
	if child.BillCount != 2 {
		t.Errorf("expected bill_count 2, got %d", child.BillCount)
	}
}

func TestChildrenBillingSummary_Difference(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary2@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-SUMMARY-002"
	createChildWithVoucher(t, db, "Emma", "Diff", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bill with underpayment
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Diff, Emma", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 119000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	if result.Children[0].TotalDifference != -1000 {
		t.Errorf("expected total_difference -1000, got %d", result.Children[0].TotalDifference)
	}
}

func TestChildrenBillingSummary_MultipleChildren(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary3@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "halbtag", 60000, -1, -1)

	v1 := "GB-30000000003-01"
	v2 := "GB-30000000004-01"
	child1, _ := createChildWithVoucher(t, db, "A", "Child", org.ID, section.ID, v1,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)
	child2, _ := createChildWithVoucher(t, db, "B", "Child", org.ID, section.ID, v2,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "halbtag"},
	)

	// Both children in same bill
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: v1, ChildName: "Child, A", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
		{VoucherNumber: v2, ChildName: "Child, B", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "halbtag", Amount: 60000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(result.Children))
	}

	// Find each child in results
	summaryMap := make(map[uint]models.ChildBillingSummaryEntry)
	for _, c := range result.Children {
		summaryMap[c.ChildID] = c
	}

	c1 := summaryMap[child1.ID]
	if c1.TotalBilled != 120000 || c1.TotalCalculated != 120000 || c1.TotalDifference != 0 {
		t.Errorf("child1: billed=%d calc=%d diff=%d", c1.TotalBilled, c1.TotalCalculated, c1.TotalDifference)
	}
	c2 := summaryMap[child2.ID]
	if c2.TotalBilled != 60000 || c2.TotalCalculated != 60000 || c2.TotalDifference != 0 {
		t.Errorf("child2: billed=%d calc=%d diff=%d", c2.TotalBilled, c2.TotalCalculated, c2.TotalDifference)
	}
}

func TestChildrenBillingSummary_MultipleVouchers(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary4@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	v1 := "GB-40000000004-01"
	v2 := "GB-40000000005-01"

	// Child with two contracts (voucher renewal)
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Multi",
			LastName:       "Voucher",
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Create child_voucher entries for both vouchers
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: v1, FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: v2, FirstSeen: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)})

	c1End := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: &c1End},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill with v1 in Jan
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: v1, ChildName: "Voucher, Multi", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})
	// Bill with v2 in Aug
	createBillFixture(t, db, org.ID, user.ID, 2025, 8, []models.GovernmentFundingBillChild{
		{VoucherNumber: v2, ChildName: "Voucher, Multi", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Should aggregate both vouchers into one child entry
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child (aggregated), got %d", len(result.Children))
	}
	if result.Children[0].ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.Children[0].ChildID)
	}
	if result.Children[0].TotalBilled != 240000 {
		t.Errorf("expected total_billed 240000, got %d", result.Children[0].TotalBilled)
	}
	if result.Children[0].TotalCalculated != 240000 {
		t.Errorf("expected total_calculated 240000, got %d", result.Children[0].TotalCalculated)
	}
	if result.Children[0].BillCount != 2 {
		t.Errorf("expected bill_count 2, got %d", result.Children[0].BillCount)
	}
}

func TestChildrenBillingSummary_NoContract(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary5@example.com", "password")
	ctx := context.Background()

	// Bill with a voucher that has no matching contract
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-ORPHAN-001", ChildName: "Unknown, Child", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	// Orphan voucher should not appear in results (no child mapping)
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children for unmatched voucher, got %d", len(result.Children))
	}
}

func TestChildrenBillingSummary_OrgIsolation(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary6@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-ISO-001"
	createChildWithVoucher(t, db, "A", "Org1", org1.ID, section1.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	createBillFixture(t, db, org1.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Org1, A", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	// Org2 should see nothing
	result, err := svc.ChildrenBillingSummary(ctx, org2.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children for org2, got %d", len(result.Children))
	}
}

func TestChildrenBillingSummary_NoFundingConfig(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary7@example.com", "password")
	ctx := context.Background()

	// No funding config!
	voucher := "GB-NOFUND-001"
	createChildWithVoucher(t, db, "No", "Funding", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Funding, No", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	// Billed total exists, calculated is 0 (no funding config)
	if result.Children[0].TotalBilled != 120000 {
		t.Errorf("expected total_billed 120000, got %d", result.Children[0].TotalBilled)
	}
	if result.Children[0].TotalCalculated != 0 {
		t.Errorf("expected total_calculated 0, got %d", result.Children[0].TotalCalculated)
	}
	if result.Children[0].TotalDifference != 120000 {
		t.Errorf("expected total_difference 120000, got %d", result.Children[0].TotalDifference)
	}
}

func TestChildrenBillingSummary_ExpiredContract(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary8@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-EXPIRED-001"
	contractEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "Ex", "Pired", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bill in Jan (contract active) and Nov (contract expired)
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Pired, Ex", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})
	createBillFixture(t, db, org.ID, user.ID, 2025, 11, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Pired, Ex", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	// Billed 240000 (both months), but calculated only 120000 (Jan only, Nov expired)
	if result.Children[0].TotalBilled != 240000 {
		t.Errorf("expected total_billed 240000, got %d", result.Children[0].TotalBilled)
	}
	if result.Children[0].TotalCalculated != 120000 {
		t.Errorf("expected total_calculated 120000, got %d", result.Children[0].TotalCalculated)
	}
	if result.Children[0].TotalDifference != 120000 {
		t.Errorf("expected total_difference 120000, got %d", result.Children[0].TotalDifference)
	}
}

func TestChildrenBillingSummary_ContractMonths(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary_months@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-MONTHS-001"
	// Contract from Jan 1 to Jun 30 = 6 months
	contractEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "Coverage", "Test", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Only 2 bills (Jan, Feb) out of 6 contract months
	for _, month := range []time.Month{1, 2} {
		createBillFixture(t, db, org.ID, user.ID, 2025, month, []models.GovernmentFundingBillChild{
			{VoucherNumber: voucher, ChildName: "Test, Coverage", BirthDate: "01.20", District: 1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 120000},
				}},
		})
	}

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	if result.Children[0].BillCount != 2 {
		t.Errorf("expected bill_count 2, got %d", result.Children[0].BillCount)
	}
	if result.Children[0].ContractMonths != 6 {
		t.Errorf("expected contract_months 6, got %d", result.Children[0].ContractMonths)
	}
}

func TestChildrenBillingSummary_ContractMonthsCappedAtToday(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary_cap@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-CAP-001"
	// Contract from Jan 2026 to Dec 2030 (far in the future)
	contractEnd := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "Future", "Contract", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	createBillFixture(t, db, org.ID, user.ID, 2026, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Contract, Future", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	// Contract is Jan 2026 - Dec 2030 = 60 months total
	// But today is April 2026, so only ~4 months should be counted (Jan-Apr 2026)
	// The exact count depends on today's date, but it MUST be less than 60
	if result.Children[0].ContractMonths >= 60 {
		t.Errorf("expected contract_months < 60 (capped at today), got %d", result.Children[0].ContractMonths)
	}
	if result.Children[0].ContractMonths < 1 {
		t.Errorf("expected contract_months >= 1, got %d", result.Children[0].ContractMonths)
	}
}

func TestChildrenBillingSummary_FutureContractZeroMonths(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary_future@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-FUTURE-001"
	// Contract starts in 2028 (future)
	contractEnd := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "NotYet", "Started", org.ID, section.ID, voucher,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	// Bill in 2028 (even though we can't really have one yet, test the logic)
	createBillFixture(t, db, org.ID, user.ID, 2028, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: voucher, ChildName: "Started, NotYet", BirthDate: "01.24", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	// Contract starts in 2028, today is 2026 — should be 0 months
	if result.Children[0].ContractMonths != 0 {
		t.Errorf("expected contract_months 0 for future contract, got %d", result.Children[0].ContractMonths)
	}
}

func TestChildrenBillingSummary_ExpiredContractFullMonths(t *testing.T) {
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
	user := createTestUser(t, db, "User", "billsummary_expired@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	voucher := "GB-EXPIRED-002"
	// Contract Jan-Jun 2025 (already ended)
	contractEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	createChildWithVoucher(t, db, "Already", "Done", org.ID, section.ID, voucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd,
		models.ContractProperties{"care_type": "ganztag"},
	)

	for _, month := range []time.Month{1, 2, 3, 4, 5, 6} {
		createBillFixture(t, db, org.ID, user.ID, 2025, month, []models.GovernmentFundingBillChild{
			{VoucherNumber: voucher, ChildName: "Done, Already", BirthDate: "01.20", District: 1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
				}},
		})
	}

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	// Contract ended Jun 2025 (in the past), should count full 6 months
	if result.Children[0].ContractMonths != 6 {
		t.Errorf("expected contract_months 6 for expired contract, got %d", result.Children[0].ContractMonths)
	}
	if result.Children[0].BillCount != 6 {
		t.Errorf("expected bill_count 6, got %d", result.Children[0].BillCount)
	}
}

func TestChildrenBillingSummary_CrossSuffixAggregation(t *testing.T) {
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
	user := createTestUser(t, db, "User", "summary_cross@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, -1, -1)

	currentVoucher := "GB-11111111111-08"
	oldVoucher := "GB-11111111111-02"
	child, _ := createChildWithVoucher(t, db, "Agg", "Test", org.ID, section.ID, currentVoucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)
	// Also add the old voucher to child_vouchers
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: oldVoucher, FirstSeen: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)})

	// Bill with old suffix
	createBillFixture(t, db, org.ID, user.ID, 2024, 6, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-11111111111-02", ChildName: "Test, Agg", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})
	// Bill with current suffix
	createBillFixture(t, db, org.ID, user.ID, 2025, 1, []models.GovernmentFundingBillChild{
		{VoucherNumber: currentVoucher, ChildName: "Test, Agg", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: "regular"},
			}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child (aggregated across suffixes), got %d", len(result.Children))
	}

	if result.Children[0].ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result.Children[0].ChildID)
	}
	// Both bills aggregated: 120000 + 120000 = 240000
	if result.Children[0].TotalBilled != 240000 {
		t.Errorf("expected total_billed 240000, got %d", result.Children[0].TotalBilled)
	}
	if result.Children[0].BillCount != 2 {
		t.Errorf("expected bill_count 2, got %d", result.Children[0].BillCount)
	}
}

// Amending a contract closes the old one at To = yesterday and opens the
// successor at From = today, so on any day but the first of a month both cover
// that month. Summing each contract's span counted it twice, and amending is
// the ordinary way to change a care type -- the error grew with every amendment
// a child had ever had.
func TestChildrenBillingSummary_ContractMonths_AmendmentMonthCountedOnce(t *testing.T) {
	db := setupTestDB(t)
	svc := newBillSummaryService(db)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billsummary_amend@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 120000, -1, -1)

	// Jan 1 - Mar 14, then Mar 15 onward: the seam falls mid-March, so March is
	// covered by both. Four calendar months are touched: Jan, Feb, Mar, Apr.
	seamEnd := time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC)
	child, _ := createChildWithVoucher(t, db, "Amended", "Child", org.ID, section.ID, "GB-AMEND-001",
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), &seamEnd,
		models.ContractProperties{"care_type": "ganztag"})

	successorEnd := time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC), To: &successorEnd},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}).Error; err != nil {
		t.Fatalf("setup: successor contract: %v", err)
	}

	// The summary only reports children that appear in at least one bill, so
	// one is needed for the child to be in the result at all.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.January, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-AMEND-001", ChildName: "Child, Amended", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	// Jan, Feb, Mar, Apr. Summing the spans gives 3 + 2 = 5, counting March twice.
	if got := result.Children[0].ContractMonths; got != 4 {
		t.Errorf("contract_months = %d, want 4 (Jan-Apr, March covered by both contracts but counted once)", got)
	}
}

// A child can hold more than one voucher -- Berlin reissues a Gutschein on a
// deferral or a district change -- and both can appear in one bill. Counting
// bills per voucher and adding them up counted that bill twice.
func TestChildrenBillingSummary_BillCount_TwoVouchersInOneBill(t *testing.T) {
	db := setupTestDB(t)
	svc := newBillSummaryService(db)
	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "billsummary_twovouchers@example.com", "password")
	ctx := context.Background()

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 120000, -1, -1)

	oldVoucher, newVoucher := "GB-REISSUE-OLD", "GB-REISSUE-NEW"
	child, _ := createChildWithVoucher(t, db, "Reissued", "Voucher", org.ID, section.ID, oldVoucher,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"})

	if err := db.Create(&models.ChildVoucher{
		ChildID: child.ID, VoucherNumber: newVoucher,
		FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("setup: second voucher: %v", err)
	}

	// One bill listing the child under both vouchers.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.January, []models.GovernmentFundingBillChild{
		{VoucherNumber: oldVoucher, ChildName: "Voucher, Reissued", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 60000}}},
		{VoucherNumber: newVoucher, ChildName: "Voucher, Reissued", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 60000}}},
	})

	result, err := svc.ChildrenBillingSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}
	entry := result.Children[0]
	if entry.BillCount != 1 {
		t.Errorf("bill_count = %d, want 1 (one bill, listed under two of the child's vouchers)", entry.BillCount)
	}
	// Amounts still add across vouchers: each voucher's payment rows are its own.
	if entry.TotalBilled != 120000 {
		t.Errorf("total_billed = %d, want 120000 (both vouchers' rows belong to this child)", entry.TotalBilled)
	}
}

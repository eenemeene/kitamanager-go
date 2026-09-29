package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// ============================================================
// Compare tests
// ============================================================

// setupBillCompareService creates a service with all stores for Compare tests.
func setupBillCompareService(t *testing.T, db *gorm.DB) *GovernmentFundingBillService {
	t.Helper()
	return NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
}

// setupFundingRates creates government funding with a period and properties for comparison tests.
// Returns care_type ganztag: 150000 (age 0-2), 120000 (age 3+), ndh: 8000, qm/mss: 5000
func setupFundingRates(t *testing.T, db *gorm.DB) {
	t.Helper()
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	period := createTestFundingPeriod(t, db, funding.ID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

	// care_type ganztag: U3 (age 0-2) = 150000
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 150000, 0, 2)
	// care_type ganztag: Ü3 (age 3+) = 120000
	createTestFundingProperty(t, db, period.ID, "care_type", "ganztag", 120000, 3, -1)
	// care_type halbtag: all ages = 80000
	createTestFundingProperty(t, db, period.ID, "care_type", "halbtag", 80000, 0, -1)
	// ndh = 8000
	createTestFundingProperty(t, db, period.ID, "ndh", "ndh", 8000, 0, -1)
	// qm/mss = 5000
	createTestFundingProperty(t, db, period.ID, "qm/mss", "qm/mss", 5000, 0, -1)
	// integration = 12000
	createTestFundingProperty(t, db, period.ID, "integration", "integration", 12000, 0, -1)
}

// createBillPeriodForCompare creates a bill period with children for compare tests.
func createBillPeriodForCompare(t *testing.T, db *gorm.DB, orgID, userID uint, children []models.GovernmentFundingBillChild) *models.GovernmentFundingBillPeriod {
	t.Helper()
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: orgID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "compare-test.xlsx",
		FileSha256:     "comparehash",
		FacilityName:   "Kita Compare",
		FacilityTotal:  500000,
		CreatedBy:      &userID,
		Children:       children,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup: create bill period error = %v", err)
	}
	return period
}

// createChildWithVoucherAndContract creates a child with an active contract having the given voucher and properties.
func createChildWithVoucherAndContract(t *testing.T, db *gorm.DB, firstName, lastName string, orgID uint, voucher string, birthdate time.Time, props models.ContractProperties) *models.Child {
	t.Helper()
	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      firstName,
			LastName:       lastName,
			Birthdate:      birthdate,
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("setup: create child error = %v", err)
	}

	// Create child_voucher entry
	if err := db.Create(&models.ChildVoucher{
		ChildID: child.ID, VoucherNumber: voucher, FirstSeen: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("setup: create child voucher error = %v", err)
	}

	section := getDefaultSection(t, db, orgID)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: props,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("setup: create contract error = %v", err)
	}
	return child
}

// --- minimal stub stores for exercising comparePeriod directly ---
//
// Each stub embeds the full store interface (so it satisfies the type) but
// overrides only the methods the calc-only path actually calls; any
// unexpected call nil-panics, which is the signal we got the path wrong.

type stubCmpChildStore struct {
	store.ChildStorer
	children []models.Child
}

func (s stubCmpChildStore) FindByOrganizationWithActiveOn(_ context.Context, _ uint, _ time.Time) ([]models.Child, error) {
	return s.children, nil
}

type stubCmpVoucherStore struct {
	store.ChildVoucherStorer
}

func (stubCmpVoucherStore) FindVouchersByChildIDs(_ context.Context, _ []uint) ([]models.ChildVoucher, error) {
	return nil, nil
}

type stubCmpOrgStore struct {
	store.OrganizationStorer
}

func (stubCmpOrgStore) FindByID(_ context.Context, id uint) (*models.Organization, error) {
	return &models.Organization{State: string(models.StateBerlin)}, nil
}

type stubCmpFundingStore struct {
	store.GovernmentFundingStorer
	funding *models.GovernmentFunding
}

func (s stubCmpFundingStore) FindByStateWithDetails(_ context.Context, _ string, _ int, _ *time.Time) (*models.GovernmentFunding, error) {
	return s.funding, nil
}

// ============================================================
// ChildBillingHistory tests
// ============================================================

// createBillFixture creates a bill period with children for testing.
func createBillFixture(t *testing.T, db *gorm.DB, orgID, userID uint, year int, month time.Month, children []models.GovernmentFundingBillChild) *models.GovernmentFundingBillPeriod {
	t.Helper()
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)

	total := 0
	for _, child := range children {
		for _, p := range child.Payments {
			total += p.Amount
		}
	}

	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:  orgID,
		Period:          models.Period{From: from, To: &to},
		FileName:        fmt.Sprintf("bill_%d_%02d.xlsx", year, month),
		FileSha256:      fmt.Sprintf("hash_%d_%02d_%d", year, month, orgID),
		FacilityName:    "Test Kita",
		FacilityTotal:   total,
		ContractBooking: total,
		CreatedBy:       &userID,
		Children:        children,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("failed to create bill fixture: %v", err)
	}
	return period
}

// createChildWithVoucher creates a child with a contract that has a voucher number.
func createChildWithVoucher(t *testing.T, db *gorm.DB, firstName, lastName string, orgID, sectionID uint, voucher string, birthdate time.Time, from time.Time, to *time.Time, props models.ContractProperties) (*models.Child, *models.ChildContract) {
	t.Helper()
	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      firstName,
			LastName:       lastName,
			Birthdate:      birthdate,
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	// Create child_voucher entry
	if err := db.Create(&models.ChildVoucher{
		ChildID: child.ID, VoucherNumber: voucher, FirstSeen: from,
	}).Error; err != nil {
		t.Fatalf("failed to create child voucher: %v", err)
	}

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: from, To: to},
			SectionID:  sectionID,
			Properties: props,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create child contract: %v", err)
	}
	return child, contract
}

// ============================================================
// BuildComparisonSummary — pure function tests (no DB needed)
// ============================================================

func uintPtr(v uint) *uint { return &v }

// newBillSummaryService builds the service the billing-summary tests exercise.
func newBillSummaryService(db *gorm.DB) *GovernmentFundingBillService {
	return NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewChildVoucherStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
		store.NewTransactor(db),
	)
}

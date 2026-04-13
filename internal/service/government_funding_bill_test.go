package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestGovernmentFundingBillService_ListEmpty(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	ctx := context.Background()

	items, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestGovernmentFundingBillService_ListWithPeriods(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc1@example.com", "password")
	ctx := context.Background()

	// Create test periods directly in DB
	for i := range 3 {
		month := time.Month(i + 1)
		to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
		p := &models.GovernmentFundingBillPeriod{
			OrganizationID:    org.ID,
			Period:            models.Period{From: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC), To: &to},
			FileName:          "file.xlsx",
			FileSha256:        fmt.Sprintf("hash_%d", i),
			FacilityName:      "Kita",
			FacilityTotal:     100000 * (i + 1),
			ContractBooking:   90000 * (i + 1),
			CorrectionBooking: 10000 * (i + 1),
			CreatedBy:         user.ID,
		}
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("setup: Create() error = %v", err)
		}
	}

	items, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// Verify response fields are populated
	first := items[0]
	if first.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if first.FacilityName != "Kita" {
		t.Errorf("expected facility name 'Kita', got %q", first.FacilityName)
	}
	if first.From == "" {
		t.Error("expected non-empty From")
	}
}

func TestGovernmentFundingBillService_ListOrganizationIsolation(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billsvc2@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org1.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "org1.xlsx",
		FileSha256:     "hash1",
		FacilityName:   "Org1 Kita",
		CreatedBy:      user.ID,
	}).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	// Org2 should see nothing
	items, total, err := svc.List(ctx, org2.ID, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 0 {
		t.Errorf("org2 should see 0 bills, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("org2 should see 0 items, got %d", len(items))
	}
}

func TestGovernmentFundingBillService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc3@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          "detail.xlsx",
		FileSha256:        "detailhash",
		FacilityName:      "Kita Detail",
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
					{Key: "care_type", Value: "ganztag", Amount: 120000},
					{Key: "ndh", Value: "ndh", Amount: 8000},
					{Key: "qm/mss", Value: "qm/mss", Amount: 5000},
				},
			},
			{
				VoucherNumber: "GB-22222222222-01",
				ChildName:     "Kind, Zwei",
				BirthDate:     "08.20",
				District:      5,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "halbtag", Amount: 90000},
					{Key: "integration", Value: "integration", Amount: 12000},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Header fields
	if result.ID != period.ID {
		t.Errorf("expected ID %d, got %d", period.ID, result.ID)
	}
	if result.FacilityName != "Kita Detail" {
		t.Errorf("expected facility name 'Kita Detail', got %q", result.FacilityName)
	}
	if result.FacilityTotal != 300000 {
		t.Errorf("expected facility total 300000, got %d", result.FacilityTotal)
	}
	if result.ChildrenCount != 2 {
		t.Errorf("expected children count 2, got %d", result.ChildrenCount)
	}

	// All unmatched (no children in system)
	if result.MatchedCount != 0 {
		t.Errorf("expected matched count 0, got %d", result.MatchedCount)
	}
	if result.UnmatchedCount != 2 {
		t.Errorf("expected unmatched count 2, got %d", result.UnmatchedCount)
	}

	// Surcharges aggregated
	if len(result.Surcharges) != 3 {
		t.Fatalf("expected 3 surcharges, got %d", len(result.Surcharges))
	}
	surchargeMap := map[string]int{}
	for _, s := range result.Surcharges {
		surchargeMap[s.Key] = s.Amount
	}
	if surchargeMap["ndh"] != 8000 {
		t.Errorf("expected ndh surcharge 8000, got %d", surchargeMap["ndh"])
	}
	if surchargeMap["qm/mss"] != 5000 {
		t.Errorf("expected qm/mss surcharge 5000, got %d", surchargeMap["qm/mss"])
	}
	if surchargeMap["integration"] != 12000 {
		t.Errorf("expected integration surcharge 12000, got %d", surchargeMap["integration"])
	}

	// Children detail
	if len(result.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(result.Children))
	}
	child1 := result.Children[0]
	if child1.TotalAmount != 133000 { // 120000 + 8000 + 5000
		t.Errorf("expected child1 total 133000, got %d", child1.TotalAmount)
	}
	if child1.Matched {
		t.Error("expected child1 not matched")
	}
}

func TestGovernmentFundingBillService_GetByIDWithMatching(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc4@example.com", "password")
	section := getDefaultSection(t, db, org.ID)
	ctx := context.Background()

	// Create children with voucher numbers in the system
	child := createTestChild(t, db, "Max", "Mustermann", org.ID)
	voucher := "GB-99999999999-01"
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section.ID,
		},
		VoucherNumber: &voucher,
	}).Error; err != nil {
		t.Fatalf("setup: create contract error = %v", err)
	}

	// Create bill period with one matching and one non-matching child
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "match-test.xlsx",
		FileSha256:     "matchhash",
		FacilityName:   "Kita Match",
		FacilityTotal:  200000,
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: voucher, // matches
				ChildName:     "Mustermann, Max",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
				},
			},
			{
				VoucherNumber: "GB-00000000000-01", // no match
				ChildName:     "Unknown, Child",
				BirthDate:     "06.21",
				District:      3,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "halbtag", Amount: 80000},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if result.MatchedCount != 1 {
		t.Errorf("expected matched count 1, got %d", result.MatchedCount)
	}
	if result.UnmatchedCount != 1 {
		t.Errorf("expected unmatched count 1, got %d", result.UnmatchedCount)
	}

	// Find matched child in response
	var matchedChild *models.GovernmentFundingBillChildResponse
	var unmatchedChild *models.GovernmentFundingBillChildResponse
	for i := range result.Children {
		if result.Children[i].VoucherNumber == voucher {
			matchedChild = &result.Children[i]
		} else {
			unmatchedChild = &result.Children[i]
		}
	}

	if matchedChild == nil {
		t.Fatal("expected to find matched child in response")
	}
	if !matchedChild.Matched {
		t.Error("expected matched child to have Matched=true")
	}
	if matchedChild.ChildID == nil || *matchedChild.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %v", child.ID, matchedChild.ChildID)
	}
	if matchedChild.ContractID == nil {
		t.Error("expected contract_id to be set for matched child")
	}

	if unmatchedChild == nil {
		t.Fatal("expected to find unmatched child in response")
	}
	if unmatchedChild.Matched {
		t.Error("expected unmatched child to have Matched=false")
	}
	if unmatchedChild.ChildID != nil {
		t.Error("expected unmatched child_id to be nil")
	}
}

func TestGovernmentFundingBillService_GetByIDWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billsvc5@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org1.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "wrong-org.xlsx",
		FileSha256:     "wrongorghash",
		FacilityName:   "Kita",
		CreatedBy:      user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	// Try to access from org2 — should fail
	_, err := svc.GetByID(ctx, period.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGovernmentFundingBillService_GetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGovernmentFundingBillService_Delete(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc6@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "delete.xlsx",
		FileSha256:     "deletehash",
		FacilityName:   "Kita Delete",
		CreatedBy:      user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	deleted, err := svc.Delete(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.ID != period.ID {
		t.Errorf("expected deleted period ID %d, got %d", period.ID, deleted.ID)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, period.ID, org.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestGovernmentFundingBillService_DeleteWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billsvc7@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org1.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "protected.xlsx",
		FileSha256:     "protectedhash",
		FacilityName:   "Kita Protected",
		CreatedBy:      user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	// Try to delete from org2 — should fail
	_, err := svc.Delete(ctx, period.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org delete, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Verify it still exists
	_, err = svc.GetByID(ctx, period.ID, org1.ID)
	if err != nil {
		t.Errorf("period should still exist after failed delete: %v", err)
	}
}

func TestGovernmentFundingBillService_DeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	ctx := context.Background()

	_, err := svc.Delete(ctx, 99999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGovernmentFundingBillService_GetByIDSurchargesAggregation(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc8@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "surcharge-agg.xlsx",
		FileSha256:     "surchargehash",
		FacilityName:   "Kita Surcharge",
		FacilityTotal:  500000,
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-11111111111-01",
				ChildName:     "Kind A",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
					{Key: "ndh", Value: "ndh", Amount: 5000},
					{Key: "qm/mss", Value: "qm/mss", Amount: 3000},
				},
			},
			{
				VoucherNumber: "GB-22222222222-01",
				ChildName:     "Kind B",
				BirthDate:     "02.20",
				District:      2,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 110000},
					{Key: "ndh", Value: "ndh", Amount: 7000},
					{Key: "integration", Value: "integration", Amount: 15000},
				},
			},
			{
				VoucherNumber: "GB-33333333333-01",
				ChildName:     "Kind C",
				BirthDate:     "03.20",
				District:      3,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "halbtag", Amount: 80000},
					// No surcharges for this child
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Verify surcharges are summed across children
	surchargeMap := map[string]int{}
	for _, s := range result.Surcharges {
		surchargeMap[s.Key] = s.Amount
	}

	// ndh: 5000 + 7000 = 12000
	if surchargeMap["ndh"] != 12000 {
		t.Errorf("expected ndh=12000, got %d", surchargeMap["ndh"])
	}
	// qm/mss: 3000 (only from child A)
	if surchargeMap["qm/mss"] != 3000 {
		t.Errorf("expected qm/mss=3000, got %d", surchargeMap["qm/mss"])
	}
	// integration: 15000 (only from child B)
	if surchargeMap["integration"] != 15000 {
		t.Errorf("expected integration=15000, got %d", surchargeMap["integration"])
	}
}

func TestGovernmentFundingBillService_GetByIDNoSurcharges(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc9@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "no-surcharge.xlsx",
		FileSha256:     "nosurchargehash",
		FacilityName:   "Kita No Surcharge",
		FacilityTotal:  100000,
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-44444444444-01",
				ChildName:     "Kind, Plain",
				BirthDate:     "04.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Surcharges should exist but all be 0
	if len(result.Surcharges) != 3 {
		t.Fatalf("expected 3 surcharge entries, got %d", len(result.Surcharges))
	}
	for _, s := range result.Surcharges {
		if s.Amount != 0 {
			t.Errorf("expected surcharge %q to be 0, got %d", s.Key, s.Amount)
		}
	}
}

func TestGovernmentFundingBillService_GetByIDNoChildren(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc10@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "empty-children.xlsx",
		FileSha256:     "emptychildrenhash",
		FacilityName:   "Kita Empty",
		FacilityTotal:  0,
		CreatedBy:      user.ID,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if result.ChildrenCount != 0 {
		t.Errorf("expected children count 0, got %d", result.ChildrenCount)
	}
	if result.MatchedCount != 0 {
		t.Errorf("expected matched count 0, got %d", result.MatchedCount)
	}
	if result.UnmatchedCount != 0 {
		t.Errorf("expected unmatched count 0, got %d", result.UnmatchedCount)
	}
}

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

func TestGovernmentFundingBillService_GetByIDMatchingCrossTenancy(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billsvc11@example.com", "password")
	section2 := getDefaultSection(t, db, org2.ID)
	ctx := context.Background()

	// Create child with voucher in org2
	child := createTestChild(t, db, "Other", "OrgChild", org2.ID)
	voucher := "GB-55555555555-01"
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section2.ID,
		},
		VoucherNumber: &voucher,
	}).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	// Create bill period in org1 with same voucher
	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID: org1.ID,
		Period:         models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "cross-tenant.xlsx",
		FileSha256:     "crosshash",
		FacilityName:   "Org1 Kita",
		CreatedBy:      user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: voucher,
				ChildName:     "Cross, Tenant",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 100000},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org1.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Child in org2 should NOT match for org1's bill
	if result.MatchedCount != 0 {
		t.Errorf("expected 0 matches (cross-org should not match), got %d", result.MatchedCount)
	}
}

// ============================================================
// Compare tests
// ============================================================

// setupBillCompareService creates a service with all stores for Compare tests.
func setupBillCompareService(t *testing.T, db *gorm.DB) *GovernmentFundingBillService {
	t.Helper()
	return NewGovernmentFundingBillService(
		store.NewChildStore(db),
		store.NewGovernmentFundingBillPeriodStore(db),
		store.NewOrganizationStore(db),
		store.NewGovernmentFundingStore(db),
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
		CreatedBy:      userID,
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

	section := getDefaultSection(t, db, orgID)
	contract := &models.ChildContract{
		ChildID:       child.ID,
		VoucherNumber: &voucher,
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

func TestGovernmentFundingBillService_Compare(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare1@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Create two children with matching vouchers and properties that produce exact match
	// Child 1: age 4 (born 2021-05-01, bill date 2025-11-01 → age 4), care_type=ganztag → 120000
	child1 := createChildWithVoucherAndContract(t, db, "Max", "Mustermann", org.ID,
		"GB-11111111111-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})
	// Child 2: age 3, care_type=ganztag → 120000, ndh → 8000
	child2 := createChildWithVoucherAndContract(t, db, "Anna", "Schmidt", org.ID,
		"GB-22222222222-01", time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-11111111111-01",
			ChildName:     "Mustermann, Max",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
		{
			VoucherNumber: "GB-22222222222-01",
			ChildName:     "Schmidt, Anna",
			BirthDate:     "03.22",
			District:      2,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
				{Key: "ndh", Value: "ndh", Amount: 8000},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.BillID != period.ID {
		t.Errorf("expected bill_id %d, got %d", period.ID, result.BillID)
	}
	if result.ChildrenCount != 2 {
		t.Errorf("expected children_count 2, got %d", result.ChildrenCount)
	}
	if result.MatchCount != 2 {
		t.Errorf("expected match_count 2, got %d", result.MatchCount)
	}
	if result.DifferenceCount != 0 {
		t.Errorf("expected difference_count 0, got %d", result.DifferenceCount)
	}
	if result.BillTotal != 248000 { // 120000 + 128000
		t.Errorf("expected bill_total 248000, got %d", result.BillTotal)
	}
	if result.CalcTotal != 248000 {
		t.Errorf("expected calculated_total 248000, got %d", result.CalcTotal)
	}
	if result.Difference != 0 {
		t.Errorf("expected difference 0, got %d", result.Difference)
	}

	// Check individual children
	for _, child := range result.Children {
		if child.Status != "match" {
			t.Errorf("child %s: expected status 'match', got %q", child.VoucherNumber, child.Status)
		}
		if child.ChildID == nil {
			t.Errorf("child %s: expected child_id to be set", child.VoucherNumber)
		}
	}

	_ = child1
	_ = child2
}

func TestGovernmentFundingBillService_Compare_WithDifferences(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare2@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child: age 4, care_type=ganztag → calc 120000, but bill says 130000
	createChildWithVoucherAndContract(t, db, "Max", "Diff", org.ID,
		"GB-33333333333-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-33333333333-01",
			ChildName:     "Diff, Max",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 130000}, // bill differs from calc (120000)
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.DifferenceCount != 1 {
		t.Errorf("expected difference_count 1, got %d", result.DifferenceCount)
	}
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}

	child := result.Children[0]
	if child.Status != "difference" {
		t.Errorf("expected status 'difference', got %q", child.Status)
	}
	if child.Difference == nil || *child.Difference != 10000 { // 130000 - 120000
		t.Errorf("expected difference 10000, got %v", child.Difference)
	}

	// Verify property-level detail
	for _, prop := range child.Properties {
		if prop.Key == "care_type" && prop.Value == "ganztag" {
			if prop.BillAmount == nil || *prop.BillAmount != 130000 {
				t.Errorf("expected bill_amount 130000, got %v", prop.BillAmount)
			}
			if prop.CalcAmount == nil || *prop.CalcAmount != 120000 {
				t.Errorf("expected calc_amount 120000, got %v", prop.CalcAmount)
			}
			if prop.Difference != 10000 {
				t.Errorf("expected property difference 10000, got %d", prop.Difference)
			}
		}
	}
}

func TestGovernmentFundingBillService_Compare_BillOnlyChild(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare3@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Bill child with no matching voucher in system
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-99999999999-01",
			ChildName:     "Unknown, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.BillOnlyCount != 1 {
		t.Errorf("expected bill_only_count 1, got %d", result.BillOnlyCount)
	}
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}

	child := result.Children[0]
	if child.Status != "bill_only" {
		t.Errorf("expected status 'bill_only', got %q", child.Status)
	}
	if child.ChildID != nil {
		t.Error("expected child_id to be nil for bill_only")
	}
	if child.CalcTotal != nil {
		t.Error("expected calc_total to be nil for bill_only")
	}

	// bill_only children included in totals
	if result.BillTotal != 120000 {
		t.Errorf("expected bill_total 120000 (bill_only included), got %d", result.BillTotal)
	}
	if result.CalcTotal != 0 {
		t.Errorf("expected calc_total 0, got %d", result.CalcTotal)
	}
	if result.Difference != 120000 {
		t.Errorf("expected difference 120000, got %d", result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_CalcOnlyChild(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare4@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child with active contract not in bill
	createChildWithVoucherAndContract(t, db, "Missing", "FromBill", org.ID,
		"GB-44444444444-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Empty bill
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.CalcOnlyCount != 1 {
		t.Errorf("expected calc_only_count 1, got %d", result.CalcOnlyCount)
	}

	child := result.Children[0]
	if child.Status != "calc_only" {
		t.Errorf("expected status 'calc_only', got %q", child.Status)
	}
	if child.ChildID == nil {
		t.Error("expected child_id to be set for calc_only")
	}
	if child.CalcTotal == nil || *child.CalcTotal != 120000 { // age 4, ganztag
		t.Errorf("expected calc_total 120000, got %v", child.CalcTotal)
	}
	if child.BillTotal != 0 {
		t.Errorf("expected bill_total 0 for calc_only, got %d", child.BillTotal)
	}

	// calc_only children included in response-level totals
	if result.BillTotal != 0 {
		t.Errorf("expected response bill_total 0, got %d", result.BillTotal)
	}
	if result.CalcTotal != 120000 {
		t.Errorf("expected response calc_total 120000 (calc_only included), got %d", result.CalcTotal)
	}
	if result.Difference != -120000 {
		t.Errorf("expected response difference -120000, got %d", result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_MixedStatuses(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare5@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Matched child (exact match)
	createChildWithVoucherAndContract(t, db, "Match", "Child", org.ID,
		"GB-AAAAAAAAAA0-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Matched child (with difference)
	createChildWithVoucherAndContract(t, db, "Diff", "Child", org.ID,
		"GB-BBBBBBBBBB0-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Calc-only child (not in bill)
	createChildWithVoucherAndContract(t, db, "CalcOnly", "Child", org.ID,
		"GB-CCCCCCCCCC0-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-AAAAAAAAAA0-01",
			ChildName:     "Child, Match",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
		{
			VoucherNumber: "GB-BBBBBBBBBB0-01",
			ChildName:     "Child, Diff",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 125000}},
		},
		{
			VoucherNumber: "GB-DDDDDDDDDD0-01", // bill-only
			ChildName:     "BillOnly, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 110000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.MatchCount != 1 {
		t.Errorf("expected match_count 1, got %d", result.MatchCount)
	}
	if result.DifferenceCount != 1 {
		t.Errorf("expected difference_count 1, got %d", result.DifferenceCount)
	}
	if result.BillOnlyCount != 1 {
		t.Errorf("expected bill_only_count 1, got %d", result.BillOnlyCount)
	}
	if result.CalcOnlyCount != 1 {
		t.Errorf("expected calc_only_count 1, got %d", result.CalcOnlyCount)
	}
	if result.ChildrenCount != 4 {
		t.Errorf("expected children_count 4, got %d", result.ChildrenCount)
	}

	// Response-level totals include all children:
	// Match: bill=120000, calc=120000
	// Diff:  bill=125000, calc=120000
	// BillOnly: bill=110000, calc=0
	// CalcOnly: bill=0, calc=120000
	expectedBillTotal := 120000 + 125000 + 110000
	expectedCalcTotal := 120000 + 120000 + 120000
	if result.BillTotal != expectedBillTotal {
		t.Errorf("expected bill_total %d, got %d", expectedBillTotal, result.BillTotal)
	}
	if result.CalcTotal != expectedCalcTotal {
		t.Errorf("expected calc_total %d, got %d", expectedCalcTotal, result.CalcTotal)
	}
	if result.Difference != expectedBillTotal-expectedCalcTotal {
		t.Errorf("expected difference %d, got %d", expectedBillTotal-expectedCalcTotal, result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_NoFundingConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	ctx := context.Background()

	// Create org with a state that has no funding rates defined
	org := &models.Organization{Name: "No Funding Org", Active: true, State: "hamburg"}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}
	// Create default section for the org
	if err := db.Create(&models.Section{OrganizationID: org.ID, Name: "Default", IsDefault: true}).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}
	user := createTestUser(t, db, "User", "compare6@example.com", "password")

	// Create child with contract
	createChildWithVoucherAndContract(t, db, "Max", "NoFunding", org.ID,
		"GB-55555555555-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-55555555555-01",
			ChildName:     "NoFunding, Max",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// No funding config → calc amounts should be 0
	child := result.Children[0]
	if child.CalcTotal == nil || *child.CalcTotal != 0 {
		t.Errorf("expected calc_total 0 (no funding config), got %v", child.CalcTotal)
	}
	if child.Difference == nil || *child.Difference != 120000 {
		t.Errorf("expected difference 120000, got %v", child.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_NoFundingPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare7@example.com", "password")
	ctx := context.Background()

	// Create funding but with a period that doesn't cover the bill date
	funding := createTestGovernmentFunding(t, db, "Berlin Funding Old")
	periodTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fundingPeriod := createTestFundingPeriod(t, db, funding.ID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &periodTo, 39.0)
	createTestFundingProperty(t, db, fundingPeriod.ID, "care_type", "ganztag", 120000, 0, -1)

	createChildWithVoucherAndContract(t, db, "Max", "NoPeriod", org.ID,
		"GB-66666666666-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Bill date is 2025-11-01, but funding period ends 2024-12-31
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-66666666666-01",
			ChildName:     "NoPeriod, Max",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	child := result.Children[0]
	if child.CalcTotal == nil || *child.CalcTotal != 0 {
		t.Errorf("expected calc_total 0 (no covering period), got %v", child.CalcTotal)
	}
}

func TestGovernmentFundingBillService_Compare_AgeDependentRates(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare8@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child 1: age 2 (born 2023-06-01, bill date 2025-11-01 → age 2), U3 rate → 150000
	createChildWithVoucherAndContract(t, db, "Young", "Child", org.ID,
		"GB-77777777777-01", time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Child 2: age 4 (born 2021-05-01), Ü3 rate → 120000
	createChildWithVoucherAndContract(t, db, "Old", "Child", org.ID,
		"GB-88888888888-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-77777777777-01",
			ChildName:     "Child, Young",
			BirthDate:     "06.23",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 150000}},
		},
		{
			VoucherNumber: "GB-88888888888-01",
			ChildName:     "Child, Old",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.MatchCount != 2 {
		t.Errorf("expected match_count 2, got %d", result.MatchCount)
	}

	for _, child := range result.Children {
		switch child.VoucherNumber {
		case "GB-77777777777-01":
			if child.Age == nil || *child.Age != 2 {
				t.Errorf("young child: expected age 2, got %v", child.Age)
			}
			if child.CalcTotal == nil || *child.CalcTotal != 150000 {
				t.Errorf("young child: expected calc_total 150000 (U3), got %v", child.CalcTotal)
			}
		case "GB-88888888888-01":
			if child.Age == nil || *child.Age != 4 {
				t.Errorf("old child: expected age 4, got %v", child.Age)
			}
			if child.CalcTotal == nil || *child.CalcTotal != 120000 {
				t.Errorf("old child: expected calc_total 120000 (Ü3), got %v", child.CalcTotal)
			}
		}
	}
}

func TestGovernmentFundingBillService_Compare_PropertyLevelDetail(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare9@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child with multiple properties: care_type=ganztag, ndh, qm/mss, integration
	createChildWithVoucherAndContract(t, db, "Multi", "Props", org.ID,
		"GB-AABBCCDDEE0-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{
			"care_type":   "ganztag",
			"ndh":         "ndh",
			"qm/mss":      "qm/mss",
			"integration": "integration",
		})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-AABBCCDDEE0-01",
			ChildName:     "Props, Multi",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
				{Key: "ndh", Value: "ndh", Amount: 8000},
				{Key: "qm/mss", Value: "qm/mss", Amount: 5000},
				{Key: "integration", Value: "integration", Amount: 12000},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	child := result.Children[0]
	if len(child.Properties) != 4 {
		t.Fatalf("expected 4 properties, got %d", len(child.Properties))
	}

	propMap := make(map[string]models.FundingComparisonAmount)
	for _, p := range child.Properties {
		propMap[p.Key+":"+p.Value] = p
	}

	expected := map[string]int{
		"care_type:ganztag":       120000,
		"ndh:ndh":                 8000,
		"qm/mss:qm/mss":           5000,
		"integration:integration": 12000,
	}
	for kv, expectedAmt := range expected {
		prop, ok := propMap[kv]
		if !ok {
			t.Errorf("missing property %s", kv)
			continue
		}
		if prop.BillAmount == nil || *prop.BillAmount != expectedAmt {
			t.Errorf("property %s: expected bill_amount %d, got %v", kv, expectedAmt, prop.BillAmount)
		}
		if prop.CalcAmount == nil || *prop.CalcAmount != expectedAmt {
			t.Errorf("property %s: expected calc_amount %d, got %v", kv, expectedAmt, prop.CalcAmount)
		}
		if prop.Difference != 0 {
			t.Errorf("property %s: expected difference 0, got %d", kv, prop.Difference)
		}
	}

	// Total: 120000+8000+5000+12000 = 145000
	if child.CalcTotal == nil || *child.CalcTotal != 145000 {
		t.Errorf("expected calc_total 145000, got %v", child.CalcTotal)
	}
}

func TestGovernmentFundingBillService_Compare_BillOnlyProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare10@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child with only care_type in contract, but bill has parent/deduction items too
	createChildWithVoucherAndContract(t, db, "Extra", "Bill", org.ID,
		"GB-EEFFGGHHII0-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-EEFFGGHHII0-01",
			ChildName:     "Bill, Extra",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
				{Key: "parent_fee", Value: "parent_fee", Amount: -5000}, // no funding counterpart
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	child := result.Children[0]
	propMap := make(map[string]models.FundingComparisonAmount)
	for _, p := range child.Properties {
		propMap[p.Key+":"+p.Value] = p
	}

	// parent_fee should have bill_amount but no calc_amount
	parentFee, ok := propMap["parent_fee:parent_fee"]
	if !ok {
		t.Fatal("expected parent_fee property in comparison")
	}
	if parentFee.BillAmount == nil || *parentFee.BillAmount != -5000 {
		t.Errorf("expected parent_fee bill_amount -5000, got %v", parentFee.BillAmount)
	}
	if parentFee.CalcAmount != nil {
		t.Errorf("expected parent_fee calc_amount nil, got %v", parentFee.CalcAmount)
	}
}

func TestGovernmentFundingBillService_Compare_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "compare11@example.com", "password")
	ctx := context.Background()

	period := createBillPeriodForCompare(t, db, org1.ID, user.ID, nil)

	_, err := svc.Compare(ctx, period.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGovernmentFundingBillService_Compare_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	ctx := context.Background()

	_, err := svc.Compare(ctx, 99999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGovernmentFundingBillService_Compare_EmptyBill(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare12@example.com", "password")
	ctx := context.Background()

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.ChildrenCount != 0 {
		t.Errorf("expected children_count 0, got %d", result.ChildrenCount)
	}
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}
	if result.BillOnlyCount != 0 {
		t.Errorf("expected bill_only_count 0, got %d", result.BillOnlyCount)
	}
	if result.CalcOnlyCount != 0 {
		t.Errorf("expected calc_only_count 0, got %d", result.CalcOnlyCount)
	}
	if result.BillTotal != 0 {
		t.Errorf("expected bill_total 0, got %d", result.BillTotal)
	}
	if result.CalcTotal != 0 {
		t.Errorf("expected calc_total 0, got %d", result.CalcTotal)
	}
}

func TestGovernmentFundingBillService_Compare_CrossTenancy(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "compare13@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Create child with voucher in org2
	voucher := "GB-XXXXXXXXXX0-01"
	createChildWithVoucherAndContract(t, db, "Cross", "Tenant", org2.ID,
		voucher, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Create bill in org1 with same voucher
	period := createBillPeriodForCompare(t, db, org1.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: voucher,
			ChildName:     "Tenant, Cross",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org1.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Voucher in org2 should NOT match for org1's bill
	if result.BillOnlyCount != 1 {
		t.Errorf("expected bill_only_count 1 (cross-org voucher should not match), got %d", result.BillOnlyCount)
	}
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}
}

func TestGovernmentFundingBillService_Compare_DifferencesCancelOut(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare14@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child 1: bill overcharges by 10000 (bill=130000, calc=120000)
	createChildWithVoucherAndContract(t, db, "Over", "Charge", org.ID,
		"GB-CANCEL00001-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Child 2: bill undercharges by 10000 (bill=110000, calc=120000)
	createChildWithVoucherAndContract(t, db, "Under", "Charge", org.ID,
		"GB-CANCEL00002-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-CANCEL00001-01",
			ChildName:     "Charge, Over",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 130000}},
		},
		{
			VoucherNumber: "GB-CANCEL00002-01",
			ChildName:     "Charge, Under",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 110000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Response-level difference is 0 because +10000 and -10000 cancel out
	if result.Difference != 0 {
		t.Errorf("expected difference 0 (cancels out), got %d", result.Difference)
	}
	// But both children have individual differences
	if result.DifferenceCount != 2 {
		t.Errorf("expected difference_count 2, got %d", result.DifferenceCount)
	}
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}

	// Verify individual child differences
	for _, child := range result.Children {
		if child.Status != "difference" {
			t.Errorf("child %s: expected status 'difference', got %q", child.VoucherNumber, child.Status)
		}
		if child.Difference == nil {
			t.Errorf("child %s: expected non-nil difference", child.VoucherNumber)
		}
	}
}

func TestGovernmentFundingBillService_Compare_MultipleBillOnlyTotals(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare15@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Two bill-only children with different amounts
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-BILLONLY0001-01",
			ChildName:     "BillOnly, One",
			BirthDate:     "01.20",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
				{Key: "ndh", Value: "ndh", Amount: 8000},
			},
		},
		{
			VoucherNumber: "GB-BILLONLY0002-01",
			ChildName:     "BillOnly, Two",
			BirthDate:     "03.21",
			District:      2,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "halbtag", Amount: 80000},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.BillOnlyCount != 2 {
		t.Errorf("expected bill_only_count 2, got %d", result.BillOnlyCount)
	}
	// Response-level BillTotal should include all bill-only amounts
	expectedBillTotal := 120000 + 8000 + 80000 // 208000
	if result.BillTotal != expectedBillTotal {
		t.Errorf("expected bill_total %d, got %d", expectedBillTotal, result.BillTotal)
	}
	if result.CalcTotal != 0 {
		t.Errorf("expected calc_total 0, got %d", result.CalcTotal)
	}
	if result.Difference != expectedBillTotal {
		t.Errorf("expected difference %d, got %d", expectedBillTotal, result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_MultipleCalcOnlyTotals(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare16@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Two calc-only children
	createChildWithVoucherAndContract(t, db, "CalcA", "Child", org.ID,
		"GB-CALCONLY001-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"})

	createChildWithVoucherAndContract(t, db, "CalcB", "Child", org.ID,
		"GB-CALCONLY002-01", time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Empty bill — both children are calc-only
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.CalcOnlyCount != 2 {
		t.Errorf("expected calc_only_count 2, got %d", result.CalcOnlyCount)
	}
	// CalcA: age 4, ganztag=120000 + ndh=8000 = 128000
	// CalcB: age 2 (U3), ganztag=150000
	expectedCalcTotal := 128000 + 150000
	if result.CalcTotal != expectedCalcTotal {
		t.Errorf("expected calc_total %d, got %d", expectedCalcTotal, result.CalcTotal)
	}
	if result.BillTotal != 0 {
		t.Errorf("expected bill_total 0, got %d", result.BillTotal)
	}
	if result.Difference != -expectedCalcTotal {
		t.Errorf("expected difference %d, got %d", -expectedCalcTotal, result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_MatchedAndBillOnlyTotals(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare17@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Matched child: exact match, age 4, ganztag=120000
	createChildWithVoucherAndContract(t, db, "Matched", "Child", org.ID,
		"GB-MIXED00001-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-MIXED00001-01",
			ChildName:     "Child, Matched",
			BirthDate:     "05.21",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
		{
			VoucherNumber: "GB-MIXED00002-01", // bill-only
			ChildName:     "Unknown, Child",
			BirthDate:     "01.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 150000}},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.MatchCount != 1 {
		t.Errorf("expected match_count 1, got %d", result.MatchCount)
	}
	if result.BillOnlyCount != 1 {
		t.Errorf("expected bill_only_count 1, got %d", result.BillOnlyCount)
	}
	// BillTotal: matched(120000) + bill_only(150000) = 270000
	if result.BillTotal != 270000 {
		t.Errorf("expected bill_total 270000, got %d", result.BillTotal)
	}
	// CalcTotal: matched(120000) only
	if result.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %d", result.CalcTotal)
	}
	if result.Difference != 150000 {
		t.Errorf("expected difference 150000, got %d", result.Difference)
	}
}

func TestGovernmentFundingBillService_Compare_MultiRowCorrection(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare18@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child with age 4 (born 2021-05-01, bill date 2025-11-01), care_type=ganztag → calc 120000
	createChildWithVoucherAndContract(t, db, "Multi", "Row", org.ID,
		"GB-MULTIROW01-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Two payment rows for the same child: original + correction
	// Row 1: care_type=ganztag 150000 (original, wrong amount)
	// Row 2: care_type=ganztag -30000 (correction)
	// Net: 120000 → should match calculated
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-MULTIROW01-01",
			ChildName:     "Row, Multi",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 150000, RowIndex: 0},
				{Key: "care_type", Value: "ganztag", Amount: -30000, RowIndex: 1},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.MatchCount != 1 {
		t.Errorf("expected match_count 1, got %d", result.MatchCount)
	}
	if result.DifferenceCount != 0 {
		t.Errorf("expected difference_count 0, got %d", result.DifferenceCount)
	}

	child := result.Children[0]
	if child.Status != "match" {
		t.Errorf("expected status 'match', got %q", child.Status)
	}
	if child.BillTotal != 120000 {
		t.Errorf("expected bill_total 120000, got %d", child.BillTotal)
	}
	if *child.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %d", *child.CalcTotal)
	}
	if *child.Difference != 0 {
		t.Errorf("expected difference 0, got %d", *child.Difference)
	}

	// Property detail should show net bill amount
	if len(child.Properties) == 0 {
		t.Fatal("expected properties to be populated")
	}
	for _, prop := range child.Properties {
		if prop.Key == "care_type" && prop.Value == "ganztag" {
			if prop.BillAmount == nil || *prop.BillAmount != 120000 {
				t.Errorf("expected care_type bill_amount 120000, got %v", prop.BillAmount)
			}
			if prop.CalcAmount == nil || *prop.CalcAmount != 120000 {
				t.Errorf("expected care_type calc_amount 120000, got %v", prop.CalcAmount)
			}
			if prop.Difference != 0 {
				t.Errorf("expected care_type difference 0, got %d", prop.Difference)
			}
		}
	}
}

func TestGovernmentFundingBillService_GetByID_MultiRowGrouping(t *testing.T) {
	db := setupTestDB(t)
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	svc := NewGovernmentFundingBillService(childStore, billPeriodStore, store.NewOrganizationStore(db), store.NewGovernmentFundingStore(db))
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billsvc_multirow@example.com", "password")
	ctx := context.Background()

	to := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          "multirow.xlsx",
		FileSha256:        "multirowsha",
		FacilityName:      "Kita Multirow",
		FacilityTotal:     200000,
		ContractBooking:   180000,
		CorrectionBooking: 20000,
		CreatedBy:         user.ID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: "GB-MULTIROW01-01",
				ChildName:     "Kind, Multi",
				BirthDate:     "05.19",
				District:      2,
				Payments: []models.GovernmentFundingBillPayment{
					// Row 0: original billing
					{Key: "care_type", Value: "ganztag", Amount: 120000, RowIndex: 0},
					{Key: "ndh", Value: "ndh", Amount: 8000, RowIndex: 0},
					// Row 1: correction
					{Key: "care_type", Value: "ganztag", Amount: -20000, RowIndex: 1},
					{Key: "ndh", Value: "ndh", Amount: 0, RowIndex: 1},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup error: %v", err)
	}

	result, err := svc.GetByID(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	child := result.Children[0]
	// Total should sum all payments across rows
	if child.TotalAmount != 108000 { // 120000 + 8000 + (-20000) + 0
		t.Errorf("expected total 108000, got %d", child.TotalAmount)
	}

	// Should have 2 rows
	if len(child.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(child.Rows))
	}

	// Row 0
	if child.Rows[0].TotalRowAmount != 128000 { // 120000 + 8000
		t.Errorf("expected row 0 total 128000, got %d", child.Rows[0].TotalRowAmount)
	}
	if len(child.Rows[0].Amounts) != 2 {
		t.Errorf("expected row 0 to have 2 amounts, got %d", len(child.Rows[0].Amounts))
	}

	// Row 1 (correction)
	if child.Rows[1].TotalRowAmount != -20000 { // -20000 + 0
		t.Errorf("expected row 1 total -20000, got %d", child.Rows[1].TotalRowAmount)
	}
	if len(child.Rows[1].Amounts) != 2 {
		t.Errorf("expected row 1 to have 2 amounts, got %d", len(child.Rows[1].Amounts))
	}
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
		CreatedBy:      user.ID,
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
		CreatedBy:      user.ID,
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
		CreatedBy:      user.ID,
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

func TestProcessISBJ(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "process_isbj@example.com", "password")
	ctx := context.Background()

	// Open the test ISBJ Excel fixture.
	f, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open test fixture: %v", err)
	}
	defer f.Close()

	resp, err := svc.ProcessISBJ(ctx, org.ID, f, "test.xlsx", "testhash", user.ID)
	if err != nil {
		t.Fatalf("ProcessISBJ() error = %v", err)
	}

	if resp.FacilityName == "" {
		t.Error("expected non-empty FacilityName")
	}
	if resp.FacilityTotal == 0 {
		t.Error("expected non-zero FacilityTotal")
	}
	if resp.ChildrenCount == 0 {
		t.Error("expected at least 1 child")
	}
	if len(resp.Children) != resp.ChildrenCount {
		t.Errorf("len(Children) = %d, want %d", len(resp.Children), resp.ChildrenCount)
	}

	// All children should be unmatched (no children in DB with matching vouchers).
	if resp.MatchedCount != 0 {
		t.Errorf("MatchedCount = %d, want 0 (no vouchers in DB)", resp.MatchedCount)
	}
	if resp.UnmatchedCount != resp.ChildrenCount {
		t.Errorf("UnmatchedCount = %d, want %d", resp.UnmatchedCount, resp.ChildrenCount)
	}

	// Verify a bill period was persisted.
	var count int64
	db.Model(&models.GovernmentFundingBillPeriod{}).Where("organization_id = ?", org.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 persisted bill period, got %d", count)
	}
}

// TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_MixedChildren tests enrichment
// when multiple calc_only children coexist with different states: one with voucher + bill history,
// one with voucher + no history, one without voucher. Verifies no state leakage between children
// and that each child gets independently correct enrichment.
func TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_MixedChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_mixed_calc@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child A: has voucher, appeared in an older bill
	voucherA := "GB-MIXCALC-AAA-01"
	createChildWithVoucherAndContract(t, db, "ChildA", "WithHistory", org.ID,
		voucherA, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	toOct := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
	olderBill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: &toOct},
		FileName:       "oct.xlsx", FileSha256: "mixhash1", FacilityName: "Kita Oktober",
		CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: voucherA, ChildName: "WithHistory, ChildA", BirthDate: "05.21", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
		},
	}
	if err := db.Create(olderBill).Error; err != nil {
		t.Fatalf("setup: create older bill error = %v", err)
	}

	// Child B: has voucher, never appeared in any bill, contract with end date
	childB := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "ChildB",
			LastName:       "NoHistory",
			Birthdate:      time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC), // age 2 (U3)
		},
	}
	if err := db.Create(childB).Error; err != nil {
		t.Fatalf("setup: create childB error = %v", err)
	}
	section := getDefaultSection(t, db, org.ID)
	voucherB := "GB-MIXCALC-BBB-01"
	contractBTo := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	contractB := &models.ChildContract{
		ChildID:       childB.ID,
		VoucherNumber: &voucherB,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), To: &contractBTo},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	}
	if err := db.Create(contractB).Error; err != nil {
		t.Fatalf("setup: create contractB error = %v", err)
	}

	// Child C: no voucher number at all
	childC := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "ChildC",
			LastName:       "NoVoucher",
			Birthdate:      time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(childC).Error; err != nil {
		t.Fatalf("setup: create childC error = %v", err)
	}
	contractC := &models.ChildContract{
		ChildID:       childC.ID,
		VoucherNumber: nil,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "halbtag"},
		},
	}
	if err := db.Create(contractC).Error; err != nil {
		t.Fatalf("setup: create contractC error = %v", err)
	}

	// Empty bill — all three children are calc_only
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.CalcOnlyCount != 3 {
		t.Fatalf("expected 3 calc_only children, got %d", result.CalcOnlyCount)
	}

	// Build map by child name for stable assertions
	byName := make(map[string]models.FundingComparisonChild)
	for _, c := range result.Children {
		byName[c.ChildName] = c
	}

	// Child A: has history, open-ended contract
	a := byName["WithHistory, ChildA"]
	if a.Status != "calc_only" {
		t.Errorf("ChildA: expected status calc_only, got %q", a.Status)
	}
	if len(a.BillAppearances) != 1 {
		t.Errorf("ChildA: expected 1 bill appearance, got %d", len(a.BillAppearances))
	} else if a.BillAppearances[0].FacilityName != "Kita Oktober" {
		t.Errorf("ChildA: expected facility 'Kita Oktober', got %q", a.BillAppearances[0].FacilityName)
	}
	if a.ContractFrom == nil || *a.ContractFrom != "2024-01-01" {
		t.Errorf("ChildA: expected contract_from '2024-01-01', got %v", a.ContractFrom)
	}
	if a.ContractTo != nil {
		t.Errorf("ChildA: expected nil contract_to (open-ended), got %v", a.ContractTo)
	}

	// Child B: no history, contract with end date, U3 age
	b := byName["NoHistory, ChildB"]
	if len(b.BillAppearances) != 0 {
		t.Errorf("ChildB: expected 0 bill appearances, got %d", len(b.BillAppearances))
	}
	if b.ContractFrom == nil || *b.ContractFrom != "2025-04-01" {
		t.Errorf("ChildB: expected contract_from '2025-04-01', got %v", b.ContractFrom)
	}
	if b.ContractTo == nil || *b.ContractTo != "2026-03-31" {
		t.Errorf("ChildB: expected contract_to '2026-03-31', got %v", b.ContractTo)
	}
	// U3 ganztag = 150000
	if b.CalcTotal == nil || *b.CalcTotal != 150000 {
		t.Errorf("ChildB: expected calc_total 150000 (U3), got %v", b.CalcTotal)
	}

	// Child C: no voucher, appearances should be nil (skipped), contract dates still set
	c := byName["NoVoucher, ChildC"]
	if c.BillAppearances != nil {
		t.Errorf("ChildC: expected nil bill_appearances (no voucher), got %v", c.BillAppearances)
	}
	if c.ContractFrom == nil || *c.ContractFrom != "2024-06-01" {
		t.Errorf("ChildC: expected contract_from '2024-06-01', got %v", c.ContractFrom)
	}
	if c.ContractTo != nil {
		t.Errorf("ChildC: expected nil contract_to (open-ended), got %v", c.ContractTo)
	}
	// halbtag all ages = 80000
	if c.CalcTotal == nil || *c.CalcTotal != 80000 {
		t.Errorf("ChildC: expected calc_total 80000 (halbtag), got %v", c.CalcTotal)
	}
}

// TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_CrossTenancy verifies that
// bill appearances are org-scoped: when the same voucher exists in bills for two different orgs,
// only the current org's bills appear in the history.
func TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_CrossTenancy(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	orgA := createTestOrganization(t, db, "Org A")
	orgB := createTestOrganization(t, db, "Org B")
	user := createTestUser(t, db, "User", "compare_cross_tenant@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	voucher := "GB-CROSSTENANT-01"

	// Same voucher in both orgs
	createChildWithVoucherAndContract(t, db, "Shared", "Voucher", orgA.ID,
		voucher, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	createChildWithVoucherAndContract(t, db, "Shared", "Voucher", orgB.ID,
		voucher, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Create older bills with same voucher in BOTH orgs
	toOct := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
	billOrgA := &models.GovernmentFundingBillPeriod{
		OrganizationID: orgA.ID,
		Period:         models.Period{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: &toOct},
		FileName:       "orgA_oct.xlsx", FileSha256: "hashA", FacilityName: "Kita A",
		CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: voucher, ChildName: "Voucher, Shared", BirthDate: "05.21", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
		},
	}
	billOrgB := &models.GovernmentFundingBillPeriod{
		OrganizationID: orgB.ID,
		Period:         models.Period{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: &toOct},
		FileName:       "orgB_oct.xlsx", FileSha256: "hashB", FacilityName: "Kita B",
		CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: voucher, ChildName: "Voucher, Shared", BirthDate: "05.21", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
		},
	}
	for _, p := range []*models.GovernmentFundingBillPeriod{billOrgA, billOrgB} {
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("setup: create bill error = %v", err)
		}
	}

	// Current empty bill for orgA — child is calc_only
	currentBill := createBillPeriodForCompare(t, db, orgA.ID, user.ID, nil)

	result, err := svc.Compare(ctx, currentBill.ID, orgA.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.CalcOnlyCount != 1 {
		t.Fatalf("expected 1 calc_only, got %d", result.CalcOnlyCount)
	}

	child := result.Children[0]
	// Should only see orgA's bill, NOT orgB's
	if len(child.BillAppearances) != 1 {
		t.Fatalf("expected 1 bill appearance (orgA only), got %d", len(child.BillAppearances))
	}
	if child.BillAppearances[0].FacilityName != "Kita A" {
		t.Errorf("expected facility 'Kita A' (orgA), got %q", child.BillAppearances[0].FacilityName)
	}
}

// TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_ChildWasMatchedBefore tests the
// real-world scenario: a child was matched (present) in previous bills but dropped from the
// current bill. The child should be calc_only with bill history showing the previous appearances.
func TestGovernmentFundingBillService_Compare_CalcOnlyEnrichment_ChildWasMatchedBefore(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_was_matched@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	voucher := "GB-WASMATCHED-01"
	createChildWithVoucherAndContract(t, db, "Was", "Matched", org.ID,
		voucher, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag", "ndh": "ndh"})

	// Create 3 consecutive monthly bills where this child was present
	months := []struct {
		from     time.Time
		to       time.Time
		facility string
	}{
		{time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC), "Kita Aug"},
		{time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC), "Kita Sep"},
		{time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC), "Kita Oct"},
	}
	for _, m := range months {
		to := m.to
		bill := &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: m.from, To: &to},
			FileName:       m.facility + ".xlsx", FileSha256: "hash_" + m.facility, FacilityName: m.facility,
			CreatedBy: user.ID,
			Children: []models.GovernmentFundingBillChild{
				{VoucherNumber: voucher, ChildName: "Matched, Was", BirthDate: "05.21", District: 1,
					Payments: []models.GovernmentFundingBillPayment{
						{Key: "care_type", Value: "ganztag", Amount: 120000},
						{Key: "ndh", Value: "ndh", Amount: 8000},
					}},
			},
		}
		if err := db.Create(bill).Error; err != nil {
			t.Fatalf("setup: create bill error = %v", err)
		}
	}

	// Current bill (November) — child was DROPPED (empty bill)
	currentBill := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, currentBill.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.CalcOnlyCount != 1 {
		t.Fatalf("expected 1 calc_only child, got %d", result.CalcOnlyCount)
	}

	child := result.Children[0]
	if child.Status != "calc_only" {
		t.Fatalf("expected status 'calc_only', got %q", child.Status)
	}

	// Bill appearances should show all 3 previous months, not the current one
	if len(child.BillAppearances) != 3 {
		t.Fatalf("expected 3 bill appearances, got %d", len(child.BillAppearances))
	}
	expectedDates := []string{"2025-08-01", "2025-09-01", "2025-10-01"}
	expectedFacilities := []string{"Kita Aug", "Kita Sep", "Kita Oct"}
	for i, a := range child.BillAppearances {
		if a.BillFrom != expectedDates[i] {
			t.Errorf("appearance[%d]: expected date %q, got %q", i, expectedDates[i], a.BillFrom)
		}
		if a.FacilityName != expectedFacilities[i] {
			t.Errorf("appearance[%d]: expected facility %q, got %q", i, expectedFacilities[i], a.FacilityName)
		}
	}

	// Calc total: ganztag (120000) + ndh (8000) = 128000
	if child.CalcTotal == nil || *child.CalcTotal != 128000 {
		t.Errorf("expected calc_total 128000, got %v", child.CalcTotal)
	}
	if child.ContractFrom == nil || *child.ContractFrom != "2024-01-01" {
		t.Errorf("expected contract_from '2024-01-01', got %v", child.ContractFrom)
	}
}

// TestGovernmentFundingBillService_Compare_FullMixedEnrichmentScope verifies that enrichment
// fields (bill_appearances, contract_from, contract_to) are ONLY set on calc_only children
// and NOT on match, difference, or bill_only children — even when those other children
// have extensive bill history.
func TestGovernmentFundingBillService_Compare_FullMixedEnrichmentScope(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_full_mixed@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Matched child — in bill + system
	createChildWithVoucherAndContract(t, db, "Matched", "Child", org.ID,
		"GB-FMIX-MATCH-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Difference child — in bill with wrong amount
	createChildWithVoucherAndContract(t, db, "Diff", "Child", org.ID,
		"GB-FMIX-DIFF-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Calc-only child — in system only, with voucher and bill history
	voucherCalc := "GB-FMIX-CALC-01"
	createChildWithVoucherAndContract(t, db, "CalcOnly", "Child", org.ID,
		voucherCalc, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Create older bill with the calc_only child AND the matched child (to ensure
	// the matched child having history doesn't bleed into its enrichment fields)
	toOct := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
	olderBill := &models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: &toOct},
		FileName:       "older.xlsx", FileSha256: "olderhash", FacilityName: "Kita Older",
		CreatedBy: user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: voucherCalc, ChildName: "Child, CalcOnly", BirthDate: "05.21", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
			{VoucherNumber: "GB-FMIX-MATCH-01", ChildName: "Child, Matched", BirthDate: "05.21", District: 1,
				Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
		},
	}
	if err := db.Create(olderBill).Error; err != nil {
		t.Fatalf("setup: create older bill error = %v", err)
	}

	// Current bill: matched + difference + bill_only (no calc_only child)
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-FMIX-MATCH-01", ChildName: "Child, Matched", BirthDate: "05.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}}},
		{VoucherNumber: "GB-FMIX-DIFF-01", ChildName: "Child, Diff", BirthDate: "05.21", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 999999}}},
		{VoucherNumber: "GB-FMIX-BILLONLY-01", ChildName: "BillOnly, Child", BirthDate: "01.20", District: 1,
			Payments: []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 110000}}},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.ChildrenCount != 4 {
		t.Fatalf("expected 4 children, got %d", result.ChildrenCount)
	}

	byStatus := make(map[string][]models.FundingComparisonChild)
	for _, c := range result.Children {
		byStatus[c.Status] = append(byStatus[c.Status], c)
	}

	// Matched child: NO enrichment
	matched := byStatus["match"]
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].BillAppearances != nil {
		t.Errorf("matched child should have nil bill_appearances, got %v", matched[0].BillAppearances)
	}
	if matched[0].ContractFrom != nil {
		t.Errorf("matched child should have nil contract_from, got %v", matched[0].ContractFrom)
	}

	// Difference child: NO enrichment
	diff := byStatus["difference"]
	if len(diff) != 1 {
		t.Fatalf("expected 1 difference, got %d", len(diff))
	}
	if diff[0].BillAppearances != nil {
		t.Errorf("difference child should have nil bill_appearances, got %v", diff[0].BillAppearances)
	}
	if diff[0].ContractFrom != nil {
		t.Errorf("difference child should have nil contract_from, got %v", diff[0].ContractFrom)
	}

	// Bill-only child: NO enrichment
	billOnly := byStatus["bill_only"]
	if len(billOnly) != 1 {
		t.Fatalf("expected 1 bill_only, got %d", len(billOnly))
	}
	if billOnly[0].BillAppearances != nil {
		t.Errorf("bill_only child should have nil bill_appearances, got %v", billOnly[0].BillAppearances)
	}
	if billOnly[0].ContractFrom != nil {
		t.Errorf("bill_only child should have nil contract_from, got %v", billOnly[0].ContractFrom)
	}

	// Calc-only child: HAS enrichment
	calcOnly := byStatus["calc_only"]
	if len(calcOnly) != 1 {
		t.Fatalf("expected 1 calc_only, got %d", len(calcOnly))
	}
	if len(calcOnly[0].BillAppearances) != 1 {
		t.Errorf("calc_only child should have 1 bill appearance, got %d", len(calcOnly[0].BillAppearances))
	}
	if calcOnly[0].ContractFrom == nil {
		t.Error("calc_only child should have contract_from set")
	}
}

func TestProcessISBJ_DuplicateHash(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "dup_hash@example.com", "password")
	ctx := context.Background()

	// First upload succeeds.
	f1, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f1.Close()

	_, err = svc.ProcessISBJ(ctx, org.ID, f1, "test.xlsx", "samehash123", user.ID)
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	// Second upload with same hash must be rejected.
	f2, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f2.Close()

	_, err = svc.ProcessISBJ(ctx, org.ID, f2, "test2.xlsx", "samehash123", user.ID)
	if err == nil {
		t.Fatal("expected duplicate hash error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Code != 409 {
		t.Errorf("expected HTTP 409, got %d", appErr.Code)
	}
	if appErr.ErrorCode != apperror.CodeDuplicateBillHash {
		t.Errorf("expected error code %q, got %q", apperror.CodeDuplicateBillHash, appErr.ErrorCode)
	}

	// Verify only one bill was persisted.
	var count int64
	db.Model(&models.GovernmentFundingBillPeriod{}).Where("organization_id = ?", org.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 bill period, got %d", count)
	}
}

func TestProcessISBJ_DuplicateMonth(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "dup_month@example.com", "password")
	ctx := context.Background()

	// First upload succeeds.
	f1, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f1.Close()

	_, err = svc.ProcessISBJ(ctx, org.ID, f1, "test.xlsx", "hash_a", user.ID)
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	// Second upload with different hash but same billing month must be rejected.
	f2, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f2.Close()

	_, err = svc.ProcessISBJ(ctx, org.ID, f2, "test2.xlsx", "hash_b", user.ID)
	if err == nil {
		t.Fatal("expected duplicate month error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Code != 409 {
		t.Errorf("expected HTTP 409, got %d", appErr.Code)
	}
	if appErr.ErrorCode != apperror.CodeDuplicateBillMonth {
		t.Errorf("expected error code %q, got %q", apperror.CodeDuplicateBillMonth, appErr.ErrorCode)
	}

	// Verify only one bill was persisted.
	var count int64
	db.Model(&models.GovernmentFundingBillPeriod{}).Where("organization_id = ?", org.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 bill period, got %d", count)
	}
}

func TestProcessISBJ_DuplicateHash_DifferentOrg_Allowed(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "dup_hash_difforg@example.com", "password")
	ctx := context.Background()

	// Upload to org1.
	f1, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f1.Close()

	_, err = svc.ProcessISBJ(ctx, org1.ID, f1, "test.xlsx", "sharedhash", user.ID)
	if err != nil {
		t.Fatalf("org1 upload: %v", err)
	}

	// Upload same hash to org2 must succeed (duplicate check is per-org).
	f2, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f2.Close()

	_, err = svc.ProcessISBJ(ctx, org2.ID, f2, "test.xlsx", "sharedhash", user.ID)
	if err != nil {
		t.Fatalf("org2 upload should succeed but got: %v", err)
	}
}

func TestProcessISBJ_ReuploadAfterDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	billStore := store.NewGovernmentFundingBillPeriodStore(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "reupload@example.com", "password")
	ctx := context.Background()

	// First upload succeeds.
	f1, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f1.Close()

	resp, err := svc.ProcessISBJ(ctx, org.ID, f1, "test.xlsx", "deletehash", user.ID)
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	// Delete the bill.
	if err := billStore.Delete(ctx, resp.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Re-upload with same hash must now succeed.
	f2, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f2.Close()

	_, err = svc.ProcessISBJ(ctx, org.ID, f2, "test.xlsx", "deletehash", user.ID)
	if err != nil {
		t.Fatalf("re-upload after delete should succeed but got: %v", err)
	}
}

func TestProcessISBJ_InvalidExcel(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "process_isbj_invalid@example.com", "password")
	ctx := context.Background()

	// Pass invalid data that isn't an Excel file.
	reader := bytes.NewReader([]byte("not an excel file"))

	_, err := svc.ProcessISBJ(ctx, org.ID, reader, "bad.xlsx", "badhash", user.ID)
	if err == nil {
		t.Fatal("expected error for invalid Excel data, got nil")
	}
}

// TestGovernmentFundingBillService_Compare_ChildWithNoVoucher verifies that a child with an
// active contract but no voucher number appears as a calc_only entry (has calculated funding
// but is not in the bill).
func TestGovernmentFundingBillService_Compare_ChildWithNoVoucher(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_novoucher@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Create a child with an active contract but NO voucher number.
	// The contract has care_type=ganztag, so funding calculation should yield 120000 (age 4).
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "NoVoucher",
			LastName:       "Child",
			Birthdate:      time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("setup: create child error = %v", err)
	}
	section := getDefaultSection(t, db, org.ID)
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
		// VoucherNumber intentionally nil
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("setup: create contract error = %v", err)
	}

	// Empty bill — no children in bill at all
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, nil)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// The child should appear as calc_only
	if result.CalcOnlyCount != 1 {
		t.Errorf("expected calc_only_count 1, got %d", result.CalcOnlyCount)
	}
	if result.ChildrenCount != 1 {
		t.Errorf("expected children_count 1, got %d", result.ChildrenCount)
	}

	compChild := result.Children[0]
	if compChild.Status != "calc_only" {
		t.Errorf("expected status 'calc_only', got %q", compChild.Status)
	}
	if compChild.ChildID == nil || *compChild.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %v", child.ID, compChild.ChildID)
	}
	if compChild.VoucherNumber != "" {
		t.Errorf("expected empty voucher_number for child without voucher, got %q", compChild.VoucherNumber)
	}
	// age 4 (born 2021-05-01, bill date 2025-11-01), care_type=ganztag → 120000
	if compChild.CalcTotal == nil || *compChild.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %v", compChild.CalcTotal)
	}
	if compChild.BillTotal != 0 {
		t.Errorf("expected bill_total 0, got %d", compChild.BillTotal)
	}

	// Response-level totals
	if result.BillTotal != 0 {
		t.Errorf("expected response bill_total 0, got %d", result.BillTotal)
	}
	if result.CalcTotal != 120000 {
		t.Errorf("expected response calc_total 120000, got %d", result.CalcTotal)
	}
	if result.Difference != -120000 {
		t.Errorf("expected response difference -120000, got %d", result.Difference)
	}
}

// TestGovernmentFundingBillService_Compare_NegativeBillAmount verifies that when a bill entry
// has a negative amount (a correction), the totals are computed correctly. The negative amount
// should reduce the child's bill total and the overall response totals.
func TestGovernmentFundingBillService_Compare_NegativeBillAmount(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_negative@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Child: age 4, care_type=ganztag → calc 120000
	createChildWithVoucherAndContract(t, db, "Neg", "Amount", org.ID,
		"GB-NEGATIVEAM-01", time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		models.ContractProperties{"care_type": "ganztag"})

	// Bill has a negative total amount (e.g., a correction refund).
	// care_type=ganztag: -50000 (correction/refund)
	period := createBillPeriodForCompare(t, db, org.ID, user.ID, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-NEGATIVEAM-01",
			ChildName:     "Amount, Neg",
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: -50000},
			},
		},
	})

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Child should be matched with a difference
	if result.MatchCount != 0 {
		t.Errorf("expected match_count 0, got %d", result.MatchCount)
	}
	if result.DifferenceCount != 1 {
		t.Errorf("expected difference_count 1, got %d", result.DifferenceCount)
	}

	child := result.Children[0]
	if child.Status != "difference" {
		t.Errorf("expected status 'difference', got %q", child.Status)
	}
	if child.BillTotal != -50000 {
		t.Errorf("expected bill_total -50000, got %d", child.BillTotal)
	}
	if child.CalcTotal == nil || *child.CalcTotal != 120000 {
		t.Errorf("expected calc_total 120000, got %v", child.CalcTotal)
	}
	// difference = bill - calc = -50000 - 120000 = -170000
	if child.Difference == nil || *child.Difference != -170000 {
		t.Errorf("expected difference -170000, got %v", child.Difference)
	}

	// Response-level totals
	if result.BillTotal != -50000 {
		t.Errorf("expected response bill_total -50000, got %d", result.BillTotal)
	}
	if result.CalcTotal != 120000 {
		t.Errorf("expected response calc_total 120000, got %d", result.CalcTotal)
	}
	if result.Difference != -170000 {
		t.Errorf("expected response difference -170000, got %d", result.Difference)
	}

	// Property-level detail
	if len(child.Properties) == 0 {
		t.Fatal("expected properties to be populated")
	}
	for _, prop := range child.Properties {
		if prop.Key == "care_type" && prop.Value == "ganztag" {
			if prop.BillAmount == nil || *prop.BillAmount != -50000 {
				t.Errorf("expected care_type bill_amount -50000, got %v", prop.BillAmount)
			}
			if prop.CalcAmount == nil || *prop.CalcAmount != 120000 {
				t.Errorf("expected care_type calc_amount 120000, got %v", prop.CalcAmount)
			}
			if prop.Difference != -170000 {
				t.Errorf("expected care_type difference -170000, got %d", prop.Difference)
			}
		}
	}
}

// TestGovernmentFundingBillService_Compare_LargeChildCount verifies that Compare handles 50
// children correctly with no truncation or off-by-one errors.
func TestGovernmentFundingBillService_Compare_LargeChildCount(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "compare_large@example.com", "password")
	ctx := context.Background()

	setupFundingRates(t, db)

	// Create 50 children, each with a unique voucher and care_type=ganztag.
	// All are age 4 (born 2021-05-01, bill date 2025-11-01) → calc 120000 each.
	const childCount = 50
	billChildren := make([]models.GovernmentFundingBillChild, childCount)

	for i := range childCount {
		voucher := fmt.Sprintf("GB-LARGE%05d-01", i)
		firstName := fmt.Sprintf("Child%d", i)
		lastName := fmt.Sprintf("Large%d", i)

		createChildWithVoucherAndContract(t, db, firstName, lastName, org.ID,
			voucher, time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
			models.ContractProperties{"care_type": "ganztag"})

		billChildren[i] = models.GovernmentFundingBillChild{
			VoucherNumber: voucher,
			ChildName:     fmt.Sprintf("%s, %s", lastName, firstName),
			BirthDate:     "05.21",
			District:      1,
			Payments: []models.GovernmentFundingBillPayment{
				{Key: "care_type", Value: "ganztag", Amount: 120000},
			},
		}
	}

	period := createBillPeriodForCompare(t, db, org.ID, user.ID, billChildren)

	result, err := svc.Compare(ctx, period.ID, org.ID)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// All 50 should be matched exactly
	if result.ChildrenCount != childCount {
		t.Errorf("expected children_count %d, got %d", childCount, result.ChildrenCount)
	}
	if result.MatchCount != childCount {
		t.Errorf("expected match_count %d, got %d", childCount, result.MatchCount)
	}
	if result.DifferenceCount != 0 {
		t.Errorf("expected difference_count 0, got %d", result.DifferenceCount)
	}
	if result.BillOnlyCount != 0 {
		t.Errorf("expected bill_only_count 0, got %d", result.BillOnlyCount)
	}
	if result.CalcOnlyCount != 0 {
		t.Errorf("expected calc_only_count 0, got %d", result.CalcOnlyCount)
	}

	// Verify totals: 50 * 120000 = 6000000
	expectedTotal := childCount * 120000
	if result.BillTotal != expectedTotal {
		t.Errorf("expected bill_total %d, got %d", expectedTotal, result.BillTotal)
	}
	if result.CalcTotal != expectedTotal {
		t.Errorf("expected calc_total %d, got %d", expectedTotal, result.CalcTotal)
	}
	if result.Difference != 0 {
		t.Errorf("expected difference 0, got %d", result.Difference)
	}

	// Verify each child is matched
	matchCount := 0
	for _, child := range result.Children {
		if child.Status == "match" {
			matchCount++
		}
		if child.ChildID == nil {
			t.Errorf("child %s: expected child_id to be set", child.VoucherNumber)
		}
	}
	if matchCount != childCount {
		t.Errorf("expected %d matched children, got %d", childCount, matchCount)
	}
}

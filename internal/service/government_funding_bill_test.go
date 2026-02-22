package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
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
	for i := 0; i < 3; i++ {
		month := time.Month(i + 1)
		to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
		p := &models.GovernmentFundingBillPeriod{
			OrganizationID:    org.ID,
			Period:            models.Period{From: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC), To: &to},
			FileName:          "file.xlsx",
			FileSha256:        "hash",
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
			"care_type": "ganztag",
			"ndh":       "ndh",
			"qm/mss":    "qm/mss",
			"integration":       "integration",
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
		"care_type:ganztag": 120000,
		"ndh:ndh":           8000,
		"qm/mss:qm/mss":     5000,
		"integration:integration":           12000,
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

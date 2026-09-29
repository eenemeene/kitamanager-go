package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

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

// TestProcessISBJ_PersistsBillingMonth walks the real anonymized fixture end
// to end and asserts the ISBJ "Monat/ Typ" month reaches the database. It was
// parsed and unit-tested long before this, then dropped in Convert, so every
// correction counted as money for the month its bill arrived in.
//
// The fixture is a plain November bill: 40 regular rows, all about November.
// That is the case where attribution and arrival agree, which is exactly why
// it is worth pinning -- it proves the column is populated rather than left
// NULL, without which the COALESCE fallback would hide the whole feature.
func TestProcessISBJ_PersistsBillingMonth(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Billing Month Org")
	user := createTestUser(t, db, "User", "billing_month@example.com", "password")
	ctx := context.Background()

	f, err := os.Open("../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx")
	if err != nil {
		t.Fatalf("open test fixture: %v", err)
	}
	defer f.Close()

	if _, err := svc.ProcessISBJ(ctx, org.ID, f, "test.xlsx", "billingmonthhash", user.ID); err != nil {
		t.Fatalf("ProcessISBJ() error = %v", err)
	}

	var payments []models.GovernmentFundingBillPayment
	if err := db.
		Joins("JOIN government_funding_bill_children c ON c.id = government_funding_bill_payments.child_id").
		Joins("JOIN government_funding_bill_periods p ON p.id = c.period_id").
		Where("p.organization_id = ?", org.ID).
		Find(&payments).Error; err != nil {
		t.Fatalf("loading payments: %v", err)
	}
	if len(payments) == 0 {
		t.Fatal("expected persisted payments")
	}

	want := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range payments {
		if p.BillingMonth == nil {
			t.Fatalf("payment %d has NULL billing_month; the Monat column was dropped again", p.ID)
		}
		if got := p.BillingMonth.UTC(); !got.Equal(want) {
			t.Fatalf("payment %d billing_month = %s, want %s",
				p.ID, got.Format("2006-01-02"), want.Format("2006-01-02"))
		}
	}
}

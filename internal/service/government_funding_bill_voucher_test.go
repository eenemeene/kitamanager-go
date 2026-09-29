package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/isbj"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestAutoDiscoverVouchers(t *testing.T) {
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

	// Create child with known voucher
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Max",
			LastName:       "Mustermann",
			Birthdate:      time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})
	// Known voucher -02
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: "GB-12345678901-02", FirstSeen: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)})

	// Call autoDiscoverVouchers with a ConvertedSettlement containing a NEW voucher
	converted := &isbj.ConvertedSettlement{
		Children: []isbj.ConvertedChild{
			{
				VoucherNumber: "GB-12345678901-08", // unknown voucher
				ChildName:     "Mustermann,Max",
				BirthDate:     "06.20",
			},
		},
	}
	billDate := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	svc.autoDiscoverVouchers(ctx, org.ID, billDate, converted)

	// Verify the child_voucher entry was created
	vouchers, err := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if err != nil {
		t.Fatalf("FindVouchersByChildID error: %v", err)
	}
	if len(vouchers) != 2 {
		t.Fatalf("expected 2 vouchers (old -02 + new -08), got %d", len(vouchers))
	}
	// Check the new voucher
	found := false
	for _, v := range vouchers {
		if v.VoucherNumber == "GB-12345678901-08" {
			found = true
			if !v.FirstSeen.Equal(billDate) {
				t.Errorf("expected first_seen %v, got %v", billDate, v.FirstSeen)
			}
		}
	}
	if !found {
		t.Error("new voucher GB-12345678901-08 not found in child_vouchers")
	}
}

func TestAutoDiscoverVouchers_NoMatch(t *testing.T) {
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

	// No children in the system — auto-discovery should do nothing
	converted := &isbj.ConvertedSettlement{
		Children: []isbj.ConvertedChild{
			{VoucherNumber: "GB-99999999999-01", ChildName: "Unknown,Child", BirthDate: "01.20"},
		},
	}
	svc.autoDiscoverVouchers(ctx, org.ID, time.Now(), converted)

	// No voucher entries should have been created
	vouchers, _ := store.NewChildVoucherStore(db).FindVouchersByOrganization(ctx, org.ID)
	if len(vouchers) != 0 {
		t.Errorf("expected 0 vouchers, got %d", len(vouchers))
	}
}

func TestAutoDiscoverVouchers_CaseInsensitive(t *testing.T) {
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

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Wilda",
			LastName:       "Beetz",
			Birthdate:      time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)

	// Bill has "WIlda" (capital I) — should still match via LOWER()
	converted := &isbj.ConvertedSettlement{
		Children: []isbj.ConvertedChild{
			{VoucherNumber: "GB-11111111111-01", ChildName: "Beetz,WIlda", BirthDate: "06.23"},
		},
	}
	svc.autoDiscoverVouchers(ctx, org.ID, time.Now(), converted)

	vouchers, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if len(vouchers) != 1 {
		t.Errorf("expected 1 voucher (case-insensitive match), got %d", len(vouchers))
	}
}

// ============================================================
// ChildrenWithoutVouchers tests
// ============================================================

func TestChildrenWithoutVouchers_Empty(t *testing.T) {
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

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestChildrenWithoutVouchers_ChildWithVoucher(t *testing.T) {
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

	// Child WITH voucher — should NOT appear
	createChildWithVoucher(t, db, "Has", "Voucher", org.ID, section.ID, "GB-11111111111-01",
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"},
	)

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 (child has voucher), got %d", len(result))
	}
}

func TestChildrenWithoutVouchers_ChildWithoutVoucher(t *testing.T) {
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

	// Child WITHOUT voucher but WITH active contract — should appear
	child := createTestChildWithContract(t, db, "No", "Voucher", org.ID, section.ID)

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].ID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, result[0].ID)
	}
}

func TestChildrenWithoutVouchers_ExpiredContract(t *testing.T) {
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

	// Child without voucher but contract EXPIRED — should NOT appear
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Old",
			LastName:       "Child",
			Birthdate:      time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	contractEnd := time.Date(2020, 7, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC), To: &contractEnd},
			SectionID: section.ID,
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 (contract expired), got %d", len(result))
	}
}

func TestChildrenWithoutVouchers_FuzzySuggestion(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy1@example.com", "password")
	ctx := context.Background()

	// Child WITHOUT voucher: "Anna Berger", born Aug 1, 2020
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	// Active contract (no voucher)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill with similar name: "Berger, Anna Lena" (extra middle name), same birth month/year
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-FUZZY00001-01",
			ChildName:     "Berger, Anna Lena",
			BirthDate:     "08.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result))
	}
	if len(result[0].Suggestions) == 0 {
		t.Fatal("expected at least 1 suggestion")
	}
	s := result[0].Suggestions[0]
	if s.VoucherNumber != "GB-FUZZY00001-01" {
		t.Errorf("expected voucher GB-FUZZY00001-01, got %s", s.VoucherNumber)
	}
	if s.BillFirstName != "Anna Lena" {
		t.Errorf("expected bill first name 'Anna Lena', got %q", s.BillFirstName)
	}
	if s.BillLastName != "Berger" {
		t.Errorf("expected bill last name 'Berger', got %q", s.BillLastName)
	}
	if s.Similarity < 0.65 {
		t.Errorf("expected similarity >= 0.65, got %f", s.Similarity)
	}
}

// A bill date inside the ±2 month tolerance window is now surfaced as
// a suggestion even with a perfect-name match — the prior strict-
// equality filter dropped these, leaving children silently without a
// match candidate to accept. Score is penalised so only confident-name
// matches survive.
func TestChildrenWithoutVouchers_BirthDateNearMissOneMonth(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy-near-1mo@example.com", "password")
	ctx := context.Background()

	// Child born August 2020.
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill child: identical name, born September 2020 — one month off.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-NEAR00001-01",
			ChildName:     "Berger, Anna",
			BirthDate:     "09.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 || len(result[0].Suggestions) != 1 {
		t.Fatalf("expected 1 child with 1 suggestion (1-month near-miss), got %d children / %d suggestions",
			len(result), len(result[0].Suggestions))
	}
	s := result[0].Suggestions[0]
	if s.VoucherNumber != "GB-NEAR00001-01" {
		t.Errorf("expected voucher GB-NEAR00001-01, got %s", s.VoucherNumber)
	}
	// Perfect name match (1.0) penalised by 0.3 — score must be <1 but ≥ threshold.
	if s.Similarity >= 1.0 {
		t.Errorf("expected score <1.0 due to penalty, got %f", s.Similarity)
	}
	if s.Similarity < 0.65 {
		t.Errorf("expected score ≥0.65 (passes threshold post-penalty), got %f", s.Similarity)
	}
}

// ±2 months is the inclusive boundary — still surfaced.
func TestChildrenWithoutVouchers_BirthDateNearMissTwoMonthsBoundary(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy-near-2mo@example.com", "password")
	ctx := context.Background()

	child := &models.Child{
		Person: models.Person{
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill child: identical name, born June 2020 — two months earlier (boundary).
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-NEAR00002-01",
			ChildName:     "Berger, Anna",
			BirthDate:     "06.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 || len(result[0].Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion at the ±2 month boundary, got %d", len(result[0].Suggestions))
	}
}

// >2 months: the suggestion is dropped — too divergent to plausibly be
// the same child even with an identical name.
func TestChildrenWithoutVouchers_NoSuggestionWhenBirthDateDeltaTooLarge(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy-far@example.com", "password")
	ctx := context.Background()

	child := &models.Child{
		Person: models.Person{
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill child: identical name, born December 2020 — four months off.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-FAR00001-01",
			ChildName:     "Berger, Anna",
			BirthDate:     "12.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result))
	}
	if len(result[0].Suggestions) != 0 {
		t.Errorf("expected 0 suggestions (>2 month delta), got %d", len(result[0].Suggestions))
	}
}

// A weak-name fuzzy match (~0.75) combined with a date mismatch falls
// below threshold after the 0.3 penalty — strong-name-only matches
// survive a date discrepancy, weak ones don't pollute the suggestion list.
func TestChildrenWithoutVouchers_WeakNameWithDateMismatchDropped(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy-weak@example.com", "password")
	ctx := context.Background()

	child := &models.Child{
		Person: models.Person{
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Different last name (fuzzy ~0.7) AND date 1 month off — penalty
	// pushes combined score below 0.65 → no suggestion.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-WEAK00001-01",
			ChildName:     "Müller, Anna",
			BirthDate:     "09.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result))
	}
	if len(result[0].Suggestions) != 0 {
		t.Errorf("expected 0 suggestions (weak name + date mismatch combine to fail threshold), got %d",
			len(result[0].Suggestions))
	}
}

func TestChildrenWithoutVouchers_NoSuggestionWhenNamesTooFar(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy3@example.com", "password")
	ctx := context.Background()

	// Child: "Felix Weber"
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Felix",
			LastName:       "Weber",
			Gender:         "male",
			Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill child: completely different name, same birth month
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-FUZZY00003-01",
			ChildName:     "Schmidt, Maria",
			BirthDate:     "03.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result))
	}
	if len(result[0].Suggestions) != 0 {
		t.Errorf("expected 0 suggestions (names too different), got %d", len(result[0].Suggestions))
	}
}

func TestChildrenWithoutVouchers_NoBills(t *testing.T) {
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

	// Child without voucher, but NO bills exist
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Felix",
			LastName:       "Weber",
			Gender:         "male",
			Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result))
	}
	if len(result[0].Suggestions) != 0 {
		t.Errorf("expected 0 suggestions (no bills), got %d", len(result[0].Suggestions))
	}
}

func TestChildrenWithoutVouchers_SkipsAlreadyMatchedVouchers(t *testing.T) {
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
	user := createTestUser(t, db, "User", "fuzzy5@example.com", "password")
	ctx := context.Background()

	// Child 1: HAS a voucher (should not appear in results at all)
	createChildWithVoucher(t, db, "Tom", "Klein", org.ID, section.ID, "GB-MATCHED0001-01",
		time.Date(2020, 5, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil,
		models.ContractProperties{"care_type": "ganztag"})

	// Child 2: no voucher, similar name to a bill child
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Tom",
			LastName:       "Klein",
			Gender:         "male",
			Birthdate:      time.Date(2021, 7, 20, 0, 0, 0, 0, time.UTC),
			OrganizationID: org.ID,
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Bill with two children: one matched, one not
	createBillFixture(t, db, org.ID, user.ID, 2025, time.November, []models.GovernmentFundingBillChild{
		{
			VoucherNumber: "GB-MATCHED0001-01", // already linked to child 1
			ChildName:     "Klein, Tom",
			BirthDate:     "05.20",
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
		{
			VoucherNumber: "GB-UNMATCHED001-01", // not linked to anyone
			ChildName:     "Klein, Tom Rio",
			BirthDate:     "07.21", // matches child 2's birthdate
			District:      1,
			Payments:      []models.GovernmentFundingBillPayment{{Key: "care_type", Value: "ganztag", Amount: 120000}},
		},
	})

	result, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	// Only child 2 should appear (child 1 has a voucher)
	if len(result) != 1 {
		t.Fatalf("expected 1 child without voucher, got %d", len(result))
	}
	if result[0].ID != child.ID {
		t.Errorf("expected child ID %d, got %d", child.ID, result[0].ID)
	}
	// Should have suggestion from the unmatched bill child
	if len(result[0].Suggestions) == 0 {
		t.Fatal("expected suggestion from unmatched bill child")
	}
	if result[0].Suggestions[0].VoucherNumber != "GB-UNMATCHED001-01" {
		t.Errorf("expected voucher GB-UNMATCHED001-01, got %s", result[0].Suggestions[0].VoucherNumber)
	}
}

// ============================================================
// AssignVoucher tests
// ============================================================

func TestAssignVoucher_Success(t *testing.T) {
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

	child := createTestChildWithContract(t, db, "No", "Voucher", org.ID, section.ID)

	voucher, err := svc.AssignVoucher(ctx, child.ID, org.ID, "GB-12345678901-01")
	if err != nil {
		t.Fatalf("AssignVoucher() error = %v", err)
	}
	if voucher == nil || voucher.ID == 0 {
		t.Fatalf("AssignVoucher() returned voucher with zero id: %+v", voucher)
	}
	if voucher.VoucherNumber != "GB-12345678901-01" {
		t.Errorf("returned voucher number = %q, want GB-12345678901-01", voucher.VoucherNumber)
	}

	// Verify voucher was created
	vouchers, err := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if err != nil {
		t.Fatalf("FindVouchersByChildID() error = %v", err)
	}
	if len(vouchers) != 1 {
		t.Fatalf("expected 1 voucher, got %d", len(vouchers))
	}
	if vouchers[0].VoucherNumber != "GB-12345678901-01" {
		t.Errorf("expected voucher GB-12345678901-01, got %s", vouchers[0].VoucherNumber)
	}
}

func TestAssignVoucher_Idempotent(t *testing.T) {
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

	child := createTestChildWithContract(t, db, "Test", "Child", org.ID, section.ID)

	// Assign twice
	first, err := svc.AssignVoucher(ctx, child.ID, org.ID, "GB-12345678901-01")
	if err != nil {
		t.Fatalf("first AssignVoucher() error = %v", err)
	}
	second, err := svc.AssignVoucher(ctx, child.ID, org.ID, "GB-12345678901-01")
	if err != nil {
		t.Fatalf("second AssignVoucher() error = %v", err)
	}
	// Idempotent assign must return the SAME voucher row both times so
	// the audit log records a stable ResourceID.
	if first.ID != second.ID {
		t.Errorf("idempotent assign returned different voucher ids: first=%d second=%d", first.ID, second.ID)
	}

	// Only one voucher entry
	vouchers, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if len(vouchers) != 1 {
		t.Errorf("expected 1 voucher after duplicate assign, got %d", len(vouchers))
	}
}

func TestAssignVoucher_ChildNotFound(t *testing.T) {
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

	_, err := svc.AssignVoucher(ctx, 99999, org.ID, "GB-12345678901-01")
	if err == nil {
		t.Fatal("expected error for non-existent child")
	}
	if apperror.HTTPStatus(err) != 404 {
		t.Errorf("expected HTTP 404, got %d", apperror.HTTPStatus(err))
	}
}

func TestAssignVoucher_WrongOrg(t *testing.T) {
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
	section := getDefaultSection(t, db, org1.ID)
	ctx := context.Background()

	child := createTestChildWithContract(t, db, "Test", "Child", org1.ID, section.ID)

	// Try to assign via wrong org
	_, err := svc.AssignVoucher(ctx, child.ID, org2.ID, "GB-12345678901-01")
	if err == nil {
		t.Fatal("expected error for wrong org")
	}
	if apperror.HTTPStatus(err) != 404 {
		t.Errorf("expected HTTP 404, got %d", apperror.HTTPStatus(err))
	}

	// Verify no voucher was created
	vouchers, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if len(vouchers) != 0 {
		t.Errorf("expected 0 vouchers after wrong-org attempt, got %d", len(vouchers))
	}
}

func TestAssignVoucher_RemovesFromWithoutVouchersList(t *testing.T) {
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

	child := createTestChildWithContract(t, db, "No", "Voucher", org.ID, section.ID)

	// Before: child appears in without-vouchers list
	before, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("ChildrenWithoutVouchers() error = %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("expected 1 child without voucher, got %d", len(before))
	}

	// Assign voucher
	if _, err := svc.AssignVoucher(ctx, child.ID, org.ID, "GB-12345678901-01"); err != nil {
		t.Fatalf("AssignVoucher() error = %v", err)
	}

	// After: child should no longer appear
	after, err := svc.ChildrenWithoutVouchers(ctx, org.ID)
	if err != nil {
		t.Fatalf("ChildrenWithoutVouchers() after error = %v", err)
	}
	if len(after) != 0 {
		t.Errorf("expected 0 children without voucher after assign, got %d", len(after))
	}
}

// ============================================================
// AssignVoucher — review finding M1
// ============================================================
//
// Pre-fix behaviour: ChildVoucherStore.CreateVoucher used
// `clause.OnConflict{DoNothing: true}`. So if the user tried to assign
// voucher V to childB while V was already on childA, the INSERT silently
// succeeded with 0 rows affected. The handler returned 200 to the user,
// but childB had no voucher and childA kept it. Effectively a silent
// data-loss bug.
//
// Post-fix: AssignVoucher runs in a transaction, uses CreateVoucherStrict
// (no ON CONFLICT clause), and on duplicate-key looks up the existing row
// to decide between idempotent re-assign (same child → success) and a
// real cross-child conflict (different child → 409).

func TestAssignVoucher_ConflictWhenVoucherOnDifferentChild(t *testing.T) {
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

	childA := createTestChildWithContract(t, db, "Anna", "A", org.ID, section.ID)
	childB := createTestChildWithContract(t, db, "Bea", "B", org.ID, section.ID)

	// Assign V to childA — succeeds.
	if _, err := svc.AssignVoucher(ctx, childA.ID, org.ID, "GB-12345678901-01"); err != nil {
		t.Fatalf("first assign: %v", err)
	}

	// Try to assign the SAME voucher to childB — must fail with 409,
	// NOT silently no-op (pre-fix bug).
	_, err := svc.AssignVoucher(ctx, childB.ID, org.ID, "GB-12345678901-01")
	if err == nil {
		t.Fatal("expected conflict when reassigning voucher to a different child — pre-fix would silently succeed")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
	if apperror.HTTPStatus(err) != 409 {
		t.Errorf("expected HTTP 409, got %d", apperror.HTTPStatus(err))
	}

	// Critical: childA still owns the voucher; childB has none.
	vouchersA, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, childA.ID)
	if len(vouchersA) != 1 || vouchersA[0].VoucherNumber != "GB-12345678901-01" {
		t.Errorf("childA should still own the voucher, got %+v", vouchersA)
	}
	vouchersB, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, childB.ID)
	if len(vouchersB) != 0 {
		t.Errorf("childB should have no vouchers (the conflict should have prevented insertion), got %+v", vouchersB)
	}
}

func TestAssignVoucher_ConflictAcrossOrgs(t *testing.T) {
	// Voucher uniqueness is GLOBAL (migration 000006), not per-org. So a
	// voucher assigned in org1 must conflict with an assign attempt in
	// org2, even though the two orgs are otherwise isolated. This is
	// less ergonomic than per-org uniqueness but matches the actual
	// schema and protects against the case where a voucher number is
	// accidentally re-used across two Kitas (which would corrupt the
	// ISBJ matching logic).
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
	child1 := createTestChildWithContract(t, db, "C1", "Org1", org1.ID, getDefaultSection(t, db, org1.ID).ID)
	child2 := createTestChildWithContract(t, db, "C2", "Org2", org2.ID, getDefaultSection(t, db, org2.ID).ID)
	ctx := context.Background()

	if _, err := svc.AssignVoucher(ctx, child1.ID, org1.ID, "GB-12345678901-01"); err != nil {
		t.Fatalf("org1 assign: %v", err)
	}
	_, err := svc.AssignVoucher(ctx, child2.ID, org2.ID, "GB-12345678901-01")
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected ErrConflict for cross-org voucher reuse, got %v", err)
	}
}

func TestAssignVoucher_ConcurrentAssignsToSameChildAreIdempotent(t *testing.T) {
	// N goroutines all assign voucher V to childA. Exactly one INSERT
	// wins; the others see store.ErrDuplicateKey, look up the existing
	// row, see ChildID==childA.ID, and return nil. End state: one row,
	// no errors.
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
	child := createTestChildWithContract(t, db, "C", "X", org.ID, section.ID)
	ctx := context.Background()

	const concurrency = 6
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.AssignVoucher(ctx, child.ID, org.ID, "GB-12345678901-01")
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d returned %v — concurrent self-assign should be idempotent", i, err)
		}
	}
	vouchers, _ := store.NewChildVoucherStore(db).FindVouchersByChildID(ctx, child.ID)
	if len(vouchers) != 1 {
		t.Errorf("expected 1 voucher row after %d concurrent self-assigns, got %d", concurrency, len(vouchers))
	}
}

func TestAssignVoucher_ConcurrentAssignsToDifferentChildrenSerializeOnUniqueIndex(t *testing.T) {
	// N goroutines try to assign voucher V to N different children.
	// Exactly one wins. The others MUST get ErrConflict (HTTP 409) —
	// pre-fix they all returned 200 with the silent no-op behaviour.
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

	const concurrency = 5
	children := make([]*models.Child, concurrency)
	for i := range concurrency {
		children[i] = createTestChildWithContract(t, db, "Child", string(rune('A'+i)), org.ID, section.ID)
	}

	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.AssignVoucher(ctx, children[idx].ID, org.ID, "GB-12345678901-01")
		}(i)
	}
	wg.Wait()

	successes := 0
	conflicts := 0
	for i, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, apperror.ErrConflict):
			conflicts++
		default:
			t.Errorf("worker %d returned unexpected error: %v", i, err)
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d (conflicts=%d)", successes, conflicts)
	}
	if conflicts != concurrency-1 {
		t.Errorf("expected %d conflicts, got %d", concurrency-1, conflicts)
	}

	// Exactly one child ends up owning the voucher.
	all, _ := store.NewChildVoucherStore(db).FindVouchersByOrganization(ctx, org.ID)
	if len(all) != 1 {
		t.Errorf("expected 1 voucher row org-wide, got %d", len(all))
	}
}

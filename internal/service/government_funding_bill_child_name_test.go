package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// TestComparePeriod_ChildNameIsConsistentAcrossBothSides pins the name a
// comparison row shows for a child KitaManager knows.
//
// Rows for one child come from both sides of the comparison: a month whose
// bill carries the child is built from the bill row, a month whose bill does
// not is built from the contract. The two sides spell a name differently --
// ISBJ writes "Nachname,Vorname" with no space, KitaManager composes
// "Nachname, Vorname" with one -- so a child with an issue in each kind of
// month appeared in the issue list twice under two spellings and read as two
// different children.
//
// The grouping was never wrong: the issue list keys on child_id, and these two
// entries are genuinely different problems in different months. Only the
// displayed name made them look unrelated.
func TestComparePeriod_ChildNameIsConsistentAcrossBothSides(t *testing.T) {
	db := setupTestDB(t)
	svc := setupBillCompareService(t, db)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	section := getDefaultSection(t, db, org.ID)
	user := createTestUser(t, db, "User", "childname1@example.com", "password")
	ctx := context.Background()
	setupFundingRates(t, db)

	const voucher = "GB-10000000003-01"
	const otherVoucher = "GB-10000000004-01"

	// KitaManager renders this child as "Beispiel, Lena".
	createChildWithVoucher(t, db, "Lena", "Beispiel", org.ID, section.ID,
		voucher, time.Date(2021, 4, 10, 0, 0, 0, 0, time.UTC),
		firstOf(2025, time.May), nil,
		models.ContractProperties{"care_type": "ganztag"})

	may := firstOf(2025, time.May)
	jun := firstOf(2025, time.June)

	// May: the bill carries the child, spelled the way ISBJ spells it.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.May, []models.GovernmentFundingBillChild{{
		VoucherNumber: voucher, ChildName: "Beispiel,Lena", BirthDate: "04.21", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, RowIndex: 0, BillingMonth: &may},
		},
	}})

	// June: the bill carries somebody else, so our child is reported from the
	// contract side instead.
	createBillFixture(t, db, org.ID, user.ID, 2025, time.June, []models.GovernmentFundingBillChild{{
		VoucherNumber: otherVoucher, ChildName: "Andere,Person", BirthDate: "01.20", District: 1,
		Payments: []models.GovernmentFundingBillPayment{
			{Key: "care_type", Value: "ganztag", Amount: 120000, RowType: models.RowTypeRegular, RowIndex: 0, BillingMonth: &jun},
		},
	}})

	comps, err := svc.CompareRange(ctx, org.ID, may, jun)
	if err != nil {
		t.Fatalf("CompareRange() error = %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("len(comparisons) = %d, want 2", len(comps))
	}

	// Collect this child's row from each month, plus the bill-only row.
	namesByMonth := map[string]string{}
	statusByMonth := map[string]string{}
	billOnlyName := ""
	for _, comp := range comps {
		month := comp.BillFrom
		for _, ch := range comp.Children {
			switch {
			case ch.ChildID != nil:
				namesByMonth[month] = ch.ChildName
				statusByMonth[month] = ch.Status
			case ch.Status == "bill_only":
				billOnlyName = ch.ChildName
			}
		}
	}

	if len(namesByMonth) != 2 {
		t.Fatalf("expected the child in both months, got %d: %v", len(namesByMonth), namesByMonth)
	}

	// The premise of the test: one month from each side of the comparison.
	mayMonth := may.Format(models.DateFormat)
	junMonth := jun.Format(models.DateFormat)
	if statusByMonth[mayMonth] == "calc_only" {
		t.Fatalf("May was expected to be built from the bill row, got status %q", statusByMonth[mayMonth])
	}
	if statusByMonth[junMonth] != "calc_only" {
		t.Fatalf("June was expected to be built from the contract, got status %q", statusByMonth[junMonth])
	}

	const want = "Beispiel, Lena"
	for month, got := range namesByMonth {
		if got != want {
			t.Errorf("%s: child_name = %q, want %q (KitaManager's spelling, whichever side the row came from)",
				month, got, want)
		}
	}
	if namesByMonth[mayMonth] != namesByMonth[junMonth] {
		t.Errorf("the same child is named %q in one month and %q in the other; the issue list then shows two entries that look like two children",
			namesByMonth[mayMonth], namesByMonth[junMonth])
	}

	// A child the bill carries and KitaManager does not keeps the Senate's
	// spelling: there is no KitaManager child whose name could be preferred.
	if billOnlyName != "Andere,Person" {
		t.Errorf("bill_only child_name = %q, want the bill's own spelling %q", billOnlyName, "Andere,Person")
	}
}

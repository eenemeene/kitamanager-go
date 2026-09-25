package seed

import (
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// billFundingPeriod is a minimal Berlin-shaped funding period: one care type
// and one surcharge, no age bands, which is all buildBillPeriod needs to emit
// payment rows.
func billFundingPeriod() *models.GovernmentFundingPeriod {
	return &models.GovernmentFundingPeriod{
		Properties: []models.GovernmentFundingProperty{
			{Key: "care_type", Value: "ganztag", Payment: 104332},
			{Key: "ndh", Value: "ndh", Payment: 12345},
		},
	}
}

// billSeededChildren builds enough children for buildBillPeriod to reach the
// correction branch, which keys off children[10].
//
// Every voucher is distinct. An earlier version built them with i%10, which
// gave children[10] the same Gutschein as children[0] -- so the correction
// attached to the wrong child's row and the attach branch passed for the wrong
// reason. Two children never share a voucher in real data; a test that lets
// them tests something the importer cannot produce.
func billSeededChildren(t *testing.T, billDate time.Time) []seededChild {
	t.Helper()
	from := billDate.AddDate(-1, 0, 0)
	children := make([]seededChild, 0, 12)
	for i := range 12 {
		children = append(children, seededChild{
			child: &models.Child{
				Person: models.Person{
					FirstName: "Kind",
					LastName:  "Sonnenschein",
					Birthdate: billDate.AddDate(-4, 0, 0),
				},
			},
			voucherNum: fmt.Sprintf("GB-%011d-01", 12345678900+i),
			contracts: []seededContract{{
				from:       from,
				properties: models.ContractProperties{"care_type": "ganztag"},
			}},
		})
	}
	return children
}

// TestBuildBillPeriod_CorrectionRowIsTypedAndAttributed guards a defect the
// demo data carried silently: the correction rows were created without a
// RowType, so BeforeCreate defaulted them to "regular". The bill header
// advertised a CorrectionBooking while no payment row under it was a
// correction, which left the Korrektur column at 0 EUR across the whole app
// and meant no seeded bill ever exercised the correction path.
func TestBuildBillPeriod_CorrectionRowIsTypedAndAttributed(t *testing.T) {
	billDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	children := billSeededChildren(t, billDate)

	period := buildBillPeriod(1, billDate, billFundingPeriod(), children, 1)
	if period == nil {
		t.Fatal("buildBillPeriod returned nil")
	}

	var corrections, regulars int
	var correctionMonths []time.Time
	for _, c := range period.Children {
		for _, p := range c.Payments {
			switch p.RowType {
			case models.RowTypeCorrection:
				corrections++
				if p.BillingMonth == nil {
					t.Fatal("correction row has no billing month")
				}
				correctionMonths = append(correctionMonths, p.BillingMonth.UTC())
			default:
				regulars++
				if p.BillingMonth == nil {
					t.Fatal("regular row has no billing month")
				}
				if got := p.BillingMonth.UTC(); !got.Equal(billDate) {
					t.Errorf("regular row billing month = %s, want %s",
						got.Format("2006-01"), billDate.Format("2006-01"))
				}
			}
		}
	}

	if corrections != 1 {
		t.Fatalf("expected exactly 1 correction row, got %d", corrections)
	}
	if regulars == 0 {
		t.Fatal("expected regular rows alongside the correction")
	}

	// One record per voucher, as Convert would build it. The seeder used to
	// append a second GovernmentFundingBillChild for a voucher that already
	// had one, and comparePeriod computes the calculated side once per
	// record, so the duplicate was awarded a second full month of funding.
	seen := make(map[string]int, len(period.Children))
	for _, c := range period.Children {
		seen[c.VoucherNumber]++
	}
	for voucher, n := range seen {
		if n > 1 {
			t.Errorf("voucher %s has %d bill-child records, want 1", voucher, n)
		}
	}
	// And it is the corrected child's own record it landed on.
	corrVoucher := children[10].voucherNum
	for _, c := range period.Children {
		if c.VoucherNumber != corrVoucher {
			continue
		}
		var hasRegular, hasCorrection bool
		for _, p := range c.Payments {
			switch p.RowType {
			case models.RowTypeCorrection:
				hasCorrection = true
			default:
				hasRegular = true
			}
		}
		if !hasRegular || !hasCorrection {
			t.Errorf("corrected child %s: regular=%v correction=%v, want both",
				corrVoucher, hasRegular, hasCorrection)
		}
	}

	// The correction is about the month BEFORE the bill. If it were about the
	// bill's own month the two keyings would agree by accident and the demo
	// data would demonstrate nothing.
	wantMonth := billDate.AddDate(0, -1, 0)
	if !correctionMonths[0].Equal(wantMonth) {
		t.Errorf("correction billing month = %s, want %s",
			correctionMonths[0].Format("2006-01"), wantMonth.Format("2006-01"))
	}

	// A header that claims a correction booking must be backed by rows.
	if period.CorrectionBooking == 0 {
		t.Error("CorrectionBooking is 0 but a correction row was emitted")
	}
}

// TestBuildBillPeriod_OlderBillsCarryNoCorrection pins the other side: only
// the two most recent bills get a correction, so an older month's rows are all
// regular and all about their own month.
func TestBuildBillPeriod_OlderBillsCarryNoCorrection(t *testing.T) {
	billDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	children := billSeededChildren(t, billDate)

	period := buildBillPeriod(1, billDate, billFundingPeriod(), children, 5)
	if period == nil {
		t.Fatal("buildBillPeriod returned nil")
	}

	for _, c := range period.Children {
		for _, p := range c.Payments {
			if p.RowType == models.RowTypeCorrection {
				t.Errorf("bill 5 months ago should carry no correction, got %+v", p)
			}
		}
	}
	if period.CorrectionBooking != 0 {
		t.Errorf("CorrectionBooking = %d, want 0", period.CorrectionBooking)
	}
}

// TestBuildBillPeriod_CorrectionForChildWithoutRegularRow covers the other
// branch of the attach: a correction for a child who has no regular row in
// this bill, which is what a correction for someone who has since left looks
// like. It gets a bill-child record of its own, carrying the correction alone.
func TestBuildBillPeriod_CorrectionForChildWithoutRegularRow(t *testing.T) {
	billDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	children := billSeededChildren(t, billDate)

	// The corrected child's contract ended before the bill month, so
	// buildBillPeriod emits no regular row for it.
	ended := billDate.AddDate(0, -2, 0)
	children[10].contracts[0].to = &ended

	period := buildBillPeriod(1, billDate, billFundingPeriod(), children, 1)
	if period == nil {
		t.Fatal("buildBillPeriod returned nil")
	}

	corrVoucher := children[10].voucherNum
	var found *models.GovernmentFundingBillChild
	for i := range period.Children {
		if period.Children[i].VoucherNumber == corrVoucher {
			found = &period.Children[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no bill-child record for the corrected voucher %s", corrVoucher)
	}
	if len(found.Payments) != 1 {
		t.Fatalf("correction-only record has %d payments, want 1", len(found.Payments))
	}
	p := found.Payments[0]
	if p.RowType != models.RowTypeCorrection {
		t.Errorf("row type = %q, want %q", p.RowType, models.RowTypeCorrection)
	}
	if p.BillingMonth == nil {
		t.Fatal("correction row has no billing month")
	}
	if want := billDate.AddDate(0, -1, 0); !p.BillingMonth.UTC().Equal(want) {
		t.Errorf("billing month = %s, want %s",
			p.BillingMonth.UTC().Format("2006-01"), want.Format("2006-01"))
	}
}

// TestBuildBillPeriod_HeaderBookingsAddUp pins the invariant a real
// Senatsabrechnung satisfies: Vertragsbuchung + Korrekturbuchung equals the
// Einrichtungssumme. The seeder subtracted the corrections from the contract
// booking instead of leaving it alone, so the header was short by twice them.
func TestBuildBillPeriod_HeaderBookingsAddUp(t *testing.T) {
	billDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	for _, monthsAgo := range []int{1, 2, 5} {
		period := buildBillPeriod(1, billDate, billFundingPeriod(),
			billSeededChildren(t, billDate), monthsAgo)
		if period == nil {
			t.Fatalf("monthsAgo %d: buildBillPeriod returned nil", monthsAgo)
		}
		if got := period.ContractBooking + period.CorrectionBooking; got != period.FacilityTotal {
			t.Errorf("monthsAgo %d: contract %d + correction %d = %d, want facility total %d",
				monthsAgo, period.ContractBooking, period.CorrectionBooking,
				got, period.FacilityTotal)
		}
	}
}

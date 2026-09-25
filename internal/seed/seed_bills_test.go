package seed

import (
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
			voucherNum: "GB-1234567890" + string(rune('0'+i%10)) + "-01",
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

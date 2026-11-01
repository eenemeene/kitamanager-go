package isbj

import (
	"fmt"
	"strings"
	"time"
)

// SettlementAmount represents a single financial line item in a settlement.
// Mirrors GovernmentFundingProperty's (Key, Value, Payment) structure.
// Every item has a Key, Value, and Amount — no special cases.
// Items whose Key+Value match a GovernmentFundingProperty can be compared
// directly against the expected payment.
type SettlementAmount struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Amount int    `json:"amount"`
}

// ConvertedChildRow represents one Excel row (one billing line item).
type ConvertedChildRow struct {
	IsCorrection   bool               `json:"is_correction"` // true if this row is a correction (Typ="K"), false for regular billing (Typ="A")
	TotalRowAmount int                `json:"total_row_amount"`
	Amounts        []SettlementAmount `json:"amounts"`
	// BillingMonth is the month this row is ABOUT, which for a
	// correction is not the month the bill arrived in: a single bill
	// carries one "A" row for its own month plus any number of "K"
	// rows each correcting an earlier one. Both come out of the same
	// merged "Monat/ Typ" header cell, so this is populated exactly
	// as often as IsCorrection is meaningful.
	//
	// Zero when the source file has no "Monat/ Typ" column or the
	// cell does not parse. Zero means UNKNOWN, never "the bill's own
	// month" -- the read path decides what to fall back to.
	BillingMonth time.Time `json:"billing_month"`
}

// ConvertedChild represents a child grouped by voucher number.
type ConvertedChild struct {
	VoucherNumber string              `json:"voucher_number"`
	ChildName     string              `json:"child_name"`
	BirthDate     string              `json:"birth_date"`
	District      int64               `json:"district"`
	TotalAmount   int                 `json:"total_amount"`
	Rows          []ConvertedChildRow `json:"rows"`
}

// ConvertedSettlement represents the full converted settlement.
type ConvertedSettlement struct {
	FacilityName      string `json:"facility_name"`
	FacilityTotal     int    `json:"facility_total"`
	ContractBooking   int    `json:"contract_booking"`
	CorrectionBooking int    `json:"correction_booking"`
	// The months CorrectionBooking settles, and why, from the
	// Korrekturübersicht sheet. Zero and empty when the bill carries no manual
	// booking, or when the sheet's rows did not reconcile with the stated
	// Korrektursumme -- in which case the amount above still stands and only
	// its attribution is unknown.
	//
	// A span rather than the rows themselves: nobody decides anything per row,
	// and the question the amount raises is which months it is about. A
	// booking that arrives in November 2025 and settles months in 2023/24 is
	// real money that must not be credited to 2025/26.
	CorrectionBookingFromMonth time.Time          `json:"correction_booking_from_month,omitempty"`
	CorrectionBookingToMonth   time.Time          `json:"correction_booking_to_month,omitempty"`
	CorrectionBookingReason    string             `json:"correction_booking_reason,omitempty"`
	ChildrenCount              int                `json:"children_count"`
	Surcharges                 []SettlementAmount `json:"surcharges"`
	Children                   []ConvertedChild   `json:"children"`
}

// SurchargeKeys are the ISBJ payment keys that represent surcharges (Zuschläge).
// These match the keys in the government funding configuration (berlin.yaml).
var SurchargeKeys = []string{"ndh", "qm/mss", "integration"}

var careScopeMap = map[string]string{
	"ganztags":  "ganztag",
	"erweitert": "ganztag erweitert",
	"teilzeit":  "teilzeit",
	"halbtag":   "halbtag",
}

func isFlagActive(flagName, value string) bool {
	switch flagName {
	case "QM", "MSS":
		return strings.EqualFold(value, "ja")
	case "HS":
		return value != "D" && value != ""
	default:
		return false
	}
}

// integrationFlagToValue maps the Excel "SpH" flag column values to
// government funding property values. The column is labeled "SpH" but
// actually contains the integration status: A=integration a, B=integration b, N=none.
func integrationFlagToValue(flag string) string {
	switch strings.ToUpper(strings.TrimSpace(flag)) {
	case "A":
		return "integration a"
	case "B":
		return "integration b"
	default:
		return ""
	}
}

// Convert translates raw SenatsabrechnungOutput into a ConvertedSettlement
// with normalized key/value/amount triples.
func Convert(output *SenatsabrechnungOutput) (*ConvertedSettlement, error) {
	korrektur := SummarizeKorrektur(output.Korrekturen)
	result := &ConvertedSettlement{
		FacilityName:               output.Einrichtung.Name,
		FacilityTotal:              output.Einrichtung.Summe,
		ContractBooking:            output.Abrechnung.VertragsBuchung,
		CorrectionBooking:          output.Abrechnung.KorrekturBuchung,
		CorrectionBookingFromMonth: korrektur.FromMonth,
		CorrectionBookingToMonth:   korrektur.ToMonth,
		CorrectionBookingReason:    korrektur.Reason,
		Surcharges: []SettlementAmount{
			{Key: "ndh", Value: "ndh", Amount: output.Einrichtung.ZuschlagNDH},
			{Key: "qm/mss", Value: "qm/mss", Amount: output.Einrichtung.ZuschlagQM + output.Einrichtung.ZuschlagMSS},
			{Key: "integration", Value: "integration", Amount: output.Einrichtung.ZuschlagIntegration},
		},
	}

	// Group children by voucher number.
	indexByVoucher := make(map[string]int) // voucher → index in result.Children
	for _, kind := range output.Vertrag.Kinder {
		row, meta, err := convertChild(&kind)
		if err != nil {
			return nil, err
		}

		if idx, ok := indexByVoucher[meta.VoucherNumber]; ok {
			result.Children[idx].Rows = append(result.Children[idx].Rows, *row)
			result.Children[idx].TotalAmount += row.TotalRowAmount
		} else {
			indexByVoucher[meta.VoucherNumber] = len(result.Children)
			result.Children = append(result.Children, ConvertedChild{
				VoucherNumber: meta.VoucherNumber,
				ChildName:     meta.ChildName,
				BirthDate:     meta.BirthDate,
				District:      meta.District,
				TotalAmount:   row.TotalRowAmount,
				Rows:          []ConvertedChildRow{*row},
			})
		}
	}
	result.ChildrenCount = len(result.Children)
	return result, nil
}

// maxMealShareCents bounds the parent meal share that reconcileRow is willing
// to infer for a BuT row. See the reasoning at the comparison itself.
const maxMealShareCents = 10000 // 100,00 EUR

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// reconcileRow makes the per-key amounts add up to the row total the sheet
// states, and refuses the row when it cannot explain the difference.
//
// Nothing persists Kind.Summe: there is no total column on
// government_funding_bill_payments, so once a bill is imported every total is
// rebuilt by summing these amounts. Storing a set that disagrees with the
// sheet's own figure therefore does not produce a visible inconsistency, it
// produces a wrong number with nothing to compare it against. Measured on the
// production instance before this check existed: the payment rows exceeded the
// bills' own facility totals by 828,00 EUR across 11 of 57 billed months.
//
// The documented formula is
//
//	Anteil Bezirk    = Basisentgeld + AoM - (Elternanteil Betreuung + Essen)
//	Abrechnungssumme = Anteil Bezirk + QM + NdH + SpH
//
// (Dokumentation Abrechnungsübersicht, Einrichtungsübersicht). That document
// predates BuT, and BuT is the one case where the sheet's total cannot be
// rebuilt from its own columns: where a child holds a berlinpass-BuT the Senate
// waives the parent's meal share and pays a subsidy instead, and it nets the
// waived share into the total without ever showing it in the Essen column. On
// all 1393 sample rows, 1380 fit the documented formula exactly and the 13 BuT
// rows fit it plus `BuT - 23.00`, with no exceptions either way.
//
// So a BuT row's residual is attributed to parent:meals -- which is what it is,
// and which lines the bill up with the `parent:meals` entitlement the funding
// configuration already applies to every contract. The amount is taken from the
// sheet rather than from a 23.00 constant, because the share is "derzeit 23
// Euro" and will not stay there.
//
// Any other residual is an error. It means a column was renamed, moved out of
// reach, or added -- and reading the wrong cell is worse than refusing the file.
func reconcileRow(kind *Kind, amounts []SettlementAmount) error {
	sum := 0
	for _, a := range amounts {
		sum += a.Amount
	}
	residual := kind.Summe - sum
	if residual == 0 {
		return nil
	}

	// The share adjustment mirrors the subsidy: where BuT is paid the share is
	// waived (residual negative), and where BuT is clawed back the waiver is
	// reversed (residual positive). So the two must have opposite signs.
	//
	// The ceiling is deliberately a round number rather than 23,00: the share
	// is "derzeit 23 Euro" and will move, while the figure this is defending
	// against does not move at all -- a mis-read Basisentgeld or Anteil-Bezirk
	// column produces a residual in the hundreds or thousands of euros, not the
	// tens. It applies only to rows that carry a BuT amount, 13 of 1393 in the
	// sample, so it is a narrow door rather than a general tolerance.
	oppositeSigns := (residual < 0) == (kind.BuT > 0)
	if kind.BuT != 0 && oppositeSigns && abs(residual) <= maxMealShareCents {
		for i := range amounts {
			if amounts[i].Key == "parent" && amounts[i].Value == "meals" {
				amounts[i].Amount += residual
				return nil
			}
		}
		// parent:meals is built unconditionally above, so this is unreachable
		// unless that changes. Say so rather than silently dropping the
		// residual.
		return fmt.Errorf("row for month %s: no parent:meals entry to carry the BuT meal share of %d cents",
			kind.Abrechnungsmonat.Format("2006-01"), residual)
	}

	// Deliberately no name, birthdate or Gutschein number: this text reaches
	// logs. The month and the figures are enough to find the row in the sheet.
	return fmt.Errorf(
		"row for month %s: the per-key amounts total %d cents but the sheet's "+
			"Abrechn.-summe for the row is %d cents (difference %+d). Components read: "+
			"Basisentgeld=%d AbzugOM=%d ElternBetreuung=%d ElternEssen=%d BuT=%d "+
			"ZuschlagQM=%d ZuschlagMSS=%d ZuschlagNDH=%d ZuschlagIntegration=%d. "+
			"Either a column was renamed or moved so that a figure was read as zero, "+
			"or the Senate added one this importer does not know about",
		kind.Abrechnungsmonat.Format("2006-01"), sum, kind.Summe, residual,
		kind.Basisentgeld, kind.AbzugOM, kind.ElternBetreuung, kind.ElternEssen, kind.BuT,
		kind.ZuschlagQM, kind.ZuschlagMSS, kind.ZuschlagNDH, kind.ZuschlagIntegration)
}

// convertChildMeta holds the per-row metadata used for grouping.
type convertChildMeta struct {
	VoucherNumber string
	ChildName     string
	BirthDate     string
	District      int64
}

func convertChild(kind *Kind) (*ConvertedChildRow, *convertChildMeta, error) {
	careType, ok := careScopeMap[kind.Betreuungsumfang]
	if !ok {
		return nil, nil, fmt.Errorf("child %s: unknown Betreuungsumfang %q", kind.Name, kind.Betreuungsumfang)
	}

	qmActive := isFlagActive("QM", kind.QM)
	mssActive := isFlagActive("MSS", kind.MSS)
	ndhActive := isFlagActive("HS", kind.HS)

	combinedQMMSS := kind.ZuschlagQM + kind.ZuschlagMSS

	// Derive flag value from amount when flag is inactive but amount is non-zero.
	// This happens in government data (e.g., retroactive adjustments).
	flagValue := func(active bool, value string, amount int) string {
		if active || amount != 0 {
			return value
		}
		return ""
	}

	// Map integration flag (A/B/N) to property value.
	// If flag says no integration but there's still an amount, use generic "integration".
	integrationValue := integrationFlagToValue(kind.Integration)
	if integrationValue == "" && kind.ZuschlagIntegration != 0 {
		integrationValue = "integration"
	}

	allAmounts := []SettlementAmount{
		{Key: "care_type", Value: careType, Amount: kind.Basisentgeld},
		// "Abzug Halbtagsbetreuung ohne Mittagessen". Part of the Senate's
		// formula -- "Basisentgelt + AoM - (Elternanteil …)" -- and held in the
		// sheet as a negative value, so it is added rather than subtracted:
		// "Der Elternanteil für Essen bleibt auf Null und der Abzug
		// Halbtagsbetreuung ohne Mittagessen enthält den entsprechend negativen
		// Wert" (Dokumentation Abrechnungsübersicht, p. 4).
		//
		// It was omitted here until the row reconciliation below started
		// demanding that the parts add up, at which point a fixture with a
		// non-zero AoM failed. Zero on all 1393 sample rows, so the omission
		// had never produced a wrong figure -- but a single half-day contract
		// without lunch would have, and after the reconciliation it would have
		// had its bill rejected instead. No funding property corresponds to it
		// (berlin.yaml has `halbtag` but no without-lunch variant), so like
		// `but` it is a bill-side figure the calculated side cannot yet predict.
		//
		// Valued only when non-zero, the same convention as `but` below, so
		// that the filter underneath drops it. It is zero on all 1393 sample
		// rows; emitting it unconditionally would add a meaningless 0 payment
		// row to every child of every bill and a 0 line to every property
		// breakdown.
		{Key: "abzug", Value: flagValue(false, "ohne_mittagessen", kind.AbzugOM), Amount: kind.AbzugOM},
		{Key: "ndh", Value: flagValue(ndhActive, "ndh", kind.ZuschlagNDH), Amount: kind.ZuschlagNDH},
		{Key: "qm/mss", Value: flagValue(qmActive || mssActive, "qm/mss", combinedQMMSS), Amount: combinedQMMSS},
		{Key: "integration", Value: integrationValue, Amount: kind.ZuschlagIntegration},
		{Key: "but", Value: flagValue(false, "but", kind.BuT), Amount: kind.BuT},
		{Key: "parent", Value: "care", Amount: -kind.ElternBetreuung},
		{Key: "parent", Value: "meals", Amount: -kind.ElternEssen},
	}

	// Filter out entries with no value and zero amount (e.g. inactive integration/ndh/qm).
	amounts := make([]SettlementAmount, 0, len(allAmounts))
	for _, a := range allAmounts {
		if a.Value == "" && a.Amount == 0 {
			continue
		}
		amounts = append(amounts, a)
	}

	if err := reconcileRow(kind, amounts); err != nil {
		return nil, nil, err
	}

	return &ConvertedChildRow{
		IsCorrection:   kind.Typ == "K",
		TotalRowAmount: kind.Summe,
		Amounts:        amounts,
		BillingMonth:   kind.Abrechnungsmonat,
	}, &convertChildMeta{
		VoucherNumber: kind.Gutscheinnummer,
		ChildName:     kind.Name,
		BirthDate:     kind.Geburtsdatum,
		District:      kind.Bezirk,
	}, nil
}

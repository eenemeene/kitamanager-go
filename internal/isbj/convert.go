package isbj

import (
	"fmt"
	"strings"
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
	TotalRowAmount int                `json:"total_row_amount"`
	Amounts        []SettlementAmount `json:"amounts"`
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
	FacilityName      string             `json:"facility_name"`
	FacilityTotal     int                `json:"facility_total"`
	ContractBooking   int                `json:"contract_booking"`
	CorrectionBooking int                `json:"correction_booking"`
	ChildrenCount     int                `json:"children_count"`
	Surcharges        []SettlementAmount `json:"surcharges"`
	Children          []ConvertedChild   `json:"children"`
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
	result := &ConvertedSettlement{
		FacilityName:      output.Einrichtung.Name,
		FacilityTotal:     output.Einrichtung.Summe,
		ContractBooking:   output.Abrechnung.VertragsBuchung,
		CorrectionBooking: output.Abrechnung.KorrekturBuchung,
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
		{Key: "ndh", Value: flagValue(ndhActive, "ndh", kind.ZuschlagNDH), Amount: kind.ZuschlagNDH},
		{Key: "qm/mss", Value: flagValue(qmActive || mssActive, "qm/mss", combinedQMMSS), Amount: combinedQMMSS},
		{Key: "integration", Value: integrationValue, Amount: kind.ZuschlagIntegration},
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

	return &ConvertedChildRow{
			TotalRowAmount: kind.Summe,
			Amounts:        amounts,
		}, &convertChildMeta{
			VoucherNumber: kind.Gutscheinnummer,
			ChildName:     kind.Name,
			BirthDate:     kind.Geburtsdatum,
			District:      kind.Bezirk,
		}, nil
}

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

// ConvertedChild represents a single child's settlement data.
type ConvertedChild struct {
	VoucherNumber string             `json:"voucher_number"`
	ChildName     string             `json:"child_name"`
	BirthDate     string             `json:"birth_date"`
	District      int64              `json:"district"`
	TotalAmount   int                `json:"total_amount"`
	Amounts       []SettlementAmount `json:"amounts"`
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
// These are fixed by the Berlin billing format.
var SurchargeKeys = []string{"ndh", "qm/mss", "sph"}

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
	case "SPH":
		return value != "N" && value != ""
	default:
		return false
	}
}

func validateFlagAmount(childName, flagName string, flagActive bool, amount int) error {
	if !flagActive && amount != 0 {
		return fmt.Errorf("child %s: flag %s is inactive but amount is %d", childName, flagName, amount)
	}
	return nil
}

// Convert translates raw SenatsabrechnungOutput into a ConvertedSettlement
// with normalized key/value/amount triples.
func Convert(output *SenatsabrechnungOutput) (*ConvertedSettlement, error) {
	result := &ConvertedSettlement{
		FacilityName:      output.Einrichtung.Name,
		FacilityTotal:     output.Einrichtung.Summe,
		ContractBooking:   output.Abrechnung.VertragsBuchung,
		CorrectionBooking: output.Abrechnung.KorrekturBuchung,
		ChildrenCount:     len(output.Vertrag.Kinder),
		Surcharges: []SettlementAmount{
			{Key: "ndh", Value: "ndh", Amount: output.Einrichtung.ZuschlagNDH},
			{Key: "qm/mss", Value: "qm/mss", Amount: output.Einrichtung.ZuschlagQM + output.Einrichtung.ZuschlagMSS},
			{Key: "sph", Value: "sph", Amount: output.Einrichtung.ZuschlagSPH},
		},
	}

	for _, kind := range output.Vertrag.Kinder {
		child, err := convertChild(&kind)
		if err != nil {
			return nil, err
		}
		result.Children = append(result.Children, *child)
	}
	return result, nil
}

func convertChild(kind *Kind) (*ConvertedChild, error) {
	careType, ok := careScopeMap[kind.Betreuungsumfang]
	if !ok {
		return nil, fmt.Errorf("child %s: unknown Betreuungsumfang %q", kind.Name, kind.Betreuungsumfang)
	}

	qmActive := isFlagActive("QM", kind.QM)
	mssActive := isFlagActive("MSS", kind.MSS)
	ndhActive := isFlagActive("HS", kind.HS)
	sphActive := isFlagActive("SPH", kind.SPH)

	combinedQMMSS := kind.ZuschlagQM + kind.ZuschlagMSS
	if err := validateFlagAmount(kind.Name, "QM/MSS", qmActive || mssActive, combinedQMMSS); err != nil {
		return nil, err
	}
	if err := validateFlagAmount(kind.Name, "ndH", ndhActive, kind.ZuschlagNDH); err != nil {
		return nil, err
	}
	if err := validateFlagAmount(kind.Name, "SpH", sphActive, kind.ZuschlagSPH); err != nil {
		return nil, err
	}

	flagValue := func(active bool, value string) string {
		if active {
			return value
		}
		return ""
	}

	amounts := []SettlementAmount{
		{Key: "care_type", Value: careType, Amount: kind.Basisentgeld},
		{Key: "ndh", Value: flagValue(ndhActive, "ndh"), Amount: kind.ZuschlagNDH},
		{Key: "qm/mss", Value: flagValue(qmActive || mssActive, "qm/mss"), Amount: combinedQMMSS},
		{Key: "sph", Value: flagValue(sphActive, "sph"), Amount: kind.ZuschlagSPH},
		{Key: "deduction", Value: "om", Amount: kind.AbzugOM},
		{Key: "parent", Value: "care", Amount: kind.ElternBetreuung},
		{Key: "parent", Value: "meals", Amount: kind.ElternEssen},
		{Key: "but", Value: "but", Amount: kind.BuT},
		{Key: "district", Value: "share", Amount: kind.AnteilBezirk},
	}

	return &ConvertedChild{
		VoucherNumber: kind.Gutscheinnummer,
		ChildName:     kind.Name,
		BirthDate:     kind.Geburtsdatum,
		District:      kind.Bezirk,
		TotalAmount:   kind.Summe,
		Amounts:       amounts,
	}, nil
}

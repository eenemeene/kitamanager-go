package isbj

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestOutput() *SenatsabrechnungOutput {
	return &SenatsabrechnungOutput{
		Einrichtung: &Einrichtung{
			Name:                "Kita Sonnenschein",
			ZuschlagQM:          10000,
			ZuschlagMSS:         5000,
			ZuschlagNDH:         20000,
			ZuschlagIntegration: 3000,
			Summe:               500000,
		},
		Abrechnung: &Abrechnung{
			VertragsBuchung:  400000,
			KorrekturBuchung: 100000,
		},
		Vertrag: &Vertrag{
			Kinder: []Kind{
				{
					Gutscheinnummer:     "GB-12345678901-01",
					Name:                "Musterkind, Max",
					Geburtsdatum:        "01.20",
					QM:                  "ja",
					MSS:                 "nein",
					HS:                  "D",
					Integration:         "N",
					Betreuungsumfang:    "ganztags",
					Bezirk:              1,
					Basisentgeld:        89000,
					AbzugOM:             -500,
					ElternBetreuung:     5000,
					ElternEssen:         2300,
					BuT:                 0,
					AnteilBezirk:        45000,
					ZuschlagQM:          5531,
					ZuschlagMSS:         0,
					ZuschlagNDH:         0,
					ZuschlagIntegration: 0,
					Summe:               141331,
				},
			},
		},
	}
}

func TestConvert_HappyPath(t *testing.T) {
	output := makeTestOutput()

	result, err := Convert(output)
	require.NoError(t, err)

	assert.Equal(t, "Kita Sonnenschein", result.FacilityName)
	assert.Equal(t, 500000, result.FacilityTotal)
	assert.Equal(t, 400000, result.ContractBooking)
	assert.Equal(t, 100000, result.CorrectionBooking)
	assert.Equal(t, 1, result.ChildrenCount)
	require.Len(t, result.Children, 1)

	child := result.Children[0]
	assert.Equal(t, "GB-12345678901-01", child.VoucherNumber)
	assert.Equal(t, "Musterkind, Max", child.ChildName)
	assert.Equal(t, "01.20", child.BirthDate)
	assert.Equal(t, int64(1), child.District)
	assert.Equal(t, 141331, child.TotalAmount)

	require.Len(t, child.Amounts, 6)
	assert.Equal(t, SettlementAmount{Key: "care_type", Value: "ganztag", Amount: 89000}, child.Amounts[0])
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "", Amount: 0}, child.Amounts[1])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 5531}, child.Amounts[2])
	assert.Equal(t, SettlementAmount{Key: "integration", Value: "", Amount: 0}, child.Amounts[3])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: 5000}, child.Amounts[4])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: 2300}, child.Amounts[5])
}

func TestConvert_CareTypeTranslation(t *testing.T) {
	tests := []struct {
		betreuungsumfang string
		expectedValue    string
	}{
		{"ganztags", "ganztag"},
		{"erweitert", "ganztag erweitert"},
		{"teilzeit", "teilzeit"},
		{"halbtag", "halbtag"},
	}

	for _, tt := range tests {
		t.Run(tt.betreuungsumfang, func(t *testing.T) {
			output := makeTestOutput()
			output.Vertrag.Kinder[0].Betreuungsumfang = tt.betreuungsumfang

			result, err := Convert(output)
			require.NoError(t, err)

			careAmount := result.Children[0].Amounts[0]
			assert.Equal(t, "care_type", careAmount.Key)
			assert.Equal(t, tt.expectedValue, careAmount.Value)
		})
	}
}

func TestConvert_UnknownBetreuungsumfang(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder[0].Betreuungsumfang = "unknown"

	_, err := Convert(output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Betreuungsumfang")
	assert.Contains(t, err.Error(), `"unknown"`)
}

func TestConvert_FlagToValue(t *testing.T) {
	t.Run("QM ja sets qm/mss value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "ja"
		output.Vertrag.Kinder[0].MSS = "nein"
		output.Vertrag.Kinder[0].ZuschlagQM = 5000
		output.Vertrag.Kinder[0].ZuschlagMSS = 0

		result, err := Convert(output)
		require.NoError(t, err)

		qmAmount := result.Children[0].Amounts[2]
		assert.Equal(t, "qm/mss", qmAmount.Key)
		assert.Equal(t, "qm/mss", qmAmount.Value)
		assert.Equal(t, 5000, qmAmount.Amount)
	})

	t.Run("MSS ja sets qm/mss value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "nein"
		output.Vertrag.Kinder[0].MSS = "ja"
		output.Vertrag.Kinder[0].ZuschlagQM = 0
		output.Vertrag.Kinder[0].ZuschlagMSS = 3000

		result, err := Convert(output)
		require.NoError(t, err)

		qmAmount := result.Children[0].Amounts[2]
		assert.Equal(t, "qm/mss", qmAmount.Value)
		assert.Equal(t, 3000, qmAmount.Amount)
	})

	t.Run("both QM and MSS inactive clears value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "nein"
		output.Vertrag.Kinder[0].MSS = "nein"
		output.Vertrag.Kinder[0].ZuschlagQM = 0
		output.Vertrag.Kinder[0].ZuschlagMSS = 0

		result, err := Convert(output)
		require.NoError(t, err)

		qmAmount := result.Children[0].Amounts[2]
		assert.Equal(t, "qm/mss", qmAmount.Key)
		assert.Equal(t, "", qmAmount.Value)
		assert.Equal(t, 0, qmAmount.Amount)
	})

	t.Run("HS=D means ndh inactive", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "D"
		output.Vertrag.Kinder[0].ZuschlagNDH = 0

		result, err := Convert(output)
		require.NoError(t, err)

		ndhAmount := result.Children[0].Amounts[1]
		assert.Equal(t, "ndh", ndhAmount.Key)
		assert.Equal(t, "", ndhAmount.Value)
		assert.Equal(t, 0, ndhAmount.Amount)
	})

	t.Run("HS=ND means ndh active", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "ND"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10116

		result, err := Convert(output)
		require.NoError(t, err)

		ndhAmount := result.Children[0].Amounts[1]
		assert.Equal(t, "ndh", ndhAmount.Key)
		assert.Equal(t, "ndh", ndhAmount.Value)
		assert.Equal(t, 10116, ndhAmount.Amount)
	})

	t.Run("Integration=N means no integration", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		intAmount := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intAmount.Key)
		assert.Equal(t, "", intAmount.Value)
		assert.Equal(t, 0, intAmount.Amount)
	})

	t.Run("Integration=A means integration a", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "A"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 165680

		result, err := Convert(output)
		require.NoError(t, err)

		intAmount := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intAmount.Key)
		assert.Equal(t, "integration a", intAmount.Value)
		assert.Equal(t, 165680, intAmount.Amount)
	})

	t.Run("Integration=B means integration b", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "B"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 330641

		result, err := Convert(output)
		require.NoError(t, err)

		intAmount := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intAmount.Key)
		assert.Equal(t, "integration b", intAmount.Value)
		assert.Equal(t, 330641, intAmount.Amount)
	})
}

func TestConvert_FlagActiveAmountZeroAllowed(t *testing.T) {
	t.Run("QM active but amount 0 is allowed", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "ja"
		output.Vertrag.Kinder[0].ZuschlagQM = 0
		output.Vertrag.Kinder[0].ZuschlagMSS = 0

		result, err := Convert(output)
		require.NoError(t, err)

		qm := result.Children[0].Amounts[2]
		assert.Equal(t, "qm/mss", qm.Key)
		assert.Equal(t, "qm/mss", qm.Value)
		assert.Equal(t, 0, qm.Amount)
	})

	t.Run("ndH active but amount 0 is allowed", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "ND"
		output.Vertrag.Kinder[0].ZuschlagNDH = 0

		result, err := Convert(output)
		require.NoError(t, err)

		ndh := result.Children[0].Amounts[1]
		assert.Equal(t, "ndh", ndh.Key)
		assert.Equal(t, "ndh", ndh.Value)
		assert.Equal(t, 0, ndh.Amount)
	})

	t.Run("Integration A but amount 0 is allowed", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "A"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		intg := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intg.Key)
		assert.Equal(t, "integration a", intg.Value)
		assert.Equal(t, 0, intg.Amount)
	})
}

func TestConvert_InactiveFlagWithNonZeroAmount(t *testing.T) {
	t.Run("QM/MSS inactive but amount non-zero passes through", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "nein"
		output.Vertrag.Kinder[0].MSS = "nein"
		output.Vertrag.Kinder[0].ZuschlagQM = 5000

		result, err := Convert(output)
		require.NoError(t, err)

		qmmss := result.Children[0].Amounts[2]
		assert.Equal(t, "qm/mss", qmmss.Key)
		assert.Equal(t, "qm/mss", qmmss.Value, "value should be set when amount is non-zero")
		assert.Equal(t, 5000, qmmss.Amount)
	})

	t.Run("ndH inactive but amount non-zero passes through", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "D"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10000

		result, err := Convert(output)
		require.NoError(t, err)

		ndh := result.Children[0].Amounts[1]
		assert.Equal(t, "ndh", ndh.Key)
		assert.Equal(t, "ndh", ndh.Value, "value should be set when amount is non-zero")
		assert.Equal(t, 10000, ndh.Amount)
	})

	t.Run("Integration N but amount non-zero uses generic value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 165680

		result, err := Convert(output)
		require.NoError(t, err)

		intg := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intg.Key)
		assert.Equal(t, "integration", intg.Value, "generic value when flag is N but amount is non-zero")
		assert.Equal(t, 165680, intg.Amount)
	})

	t.Run("Integration N with zero amount has empty value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		intg := result.Children[0].Amounts[3]
		assert.Equal(t, "integration", intg.Key)
		assert.Equal(t, "", intg.Value, "value should be empty when flag N and amount is zero")
		assert.Equal(t, 0, intg.Amount)
	})
}

func TestConvert_FacilitySurcharges(t *testing.T) {
	output := makeTestOutput()

	result, err := Convert(output)
	require.NoError(t, err)

	require.Len(t, result.Surcharges, 3)
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 20000}, result.Surcharges[0])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 15000}, result.Surcharges[1])
	assert.Equal(t, SettlementAmount{Key: "integration", Value: "integration", Amount: 3000}, result.Surcharges[2])
}

func TestConvert_OtherLineItems(t *testing.T) {
	output := makeTestOutput()

	result, err := Convert(output)
	require.NoError(t, err)

	child := result.Children[0]

	// parent care
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: 5000}, child.Amounts[4])
	// parent meals
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: 2300}, child.Amounts[5])
}

func TestConvert_MultipleChildren(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:     "GB-98765432109-02",
		Name:                "Testkind, Anna",
		Geburtsdatum:        "06.19",
		QM:                  "nein",
		MSS:                 "ja",
		HS:                  "ND",
		Integration:         "A",
		Betreuungsumfang:    "teilzeit",
		Bezirk:              5,
		Basisentgeld:        65000,
		AbzugOM:             0,
		ElternBetreuung:     3000,
		ElternEssen:         1500,
		BuT:                 800,
		AnteilBezirk:        30000,
		ZuschlagQM:          0,
		ZuschlagMSS:         4000,
		ZuschlagNDH:         10116,
		ZuschlagIntegration: 165680,
		Summe:               114416,
	})

	result, err := Convert(output)
	require.NoError(t, err)

	assert.Equal(t, 2, result.ChildrenCount)
	require.Len(t, result.Children, 2)

	second := result.Children[1]
	assert.Equal(t, "GB-98765432109-02", second.VoucherNumber)
	assert.Equal(t, "Testkind, Anna", second.ChildName)
	assert.Equal(t, int64(5), second.District)

	// care_type = teilzeit
	assert.Equal(t, SettlementAmount{Key: "care_type", Value: "teilzeit", Amount: 65000}, second.Amounts[0])
	// ndh active (HS=ND)
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 10116}, second.Amounts[1])
	// qm/mss active (MSS=ja)
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 4000}, second.Amounts[2])
	// integration a (Integration=A)
	assert.Equal(t, SettlementAmount{Key: "integration", Value: "integration a", Amount: 165680}, second.Amounts[3])
}

func TestIsFlagActive(t *testing.T) {
	tests := []struct {
		flagName string
		value    string
		expected bool
	}{
		{"QM", "ja", true},
		{"QM", "Ja", true},
		{"QM", "JA", true},
		{"QM", "nein", false},
		{"QM", "", false},
		{"MSS", "ja", true},
		{"MSS", "nein", false},
		{"HS", "D", false},
		{"HS", "", false},
		{"HS", "ND", true},
		{"HS", "T", true},
		{"UNKNOWN", "ja", false},
	}

	for _, tt := range tests {
		t.Run(tt.flagName+"="+tt.value, func(t *testing.T) {
			assert.Equal(t, tt.expected, isFlagActive(tt.flagName, tt.value))
		})
	}
}

func TestIntegrationFlagToValue(t *testing.T) {
	tests := []struct {
		flag     string
		expected string
	}{
		{"A", "integration a"},
		{"a", "integration a"},
		{"B", "integration b"},
		{"b", "integration b"},
		{"N", ""},
		{"", ""},
		{"X", ""},
	}

	for _, tt := range tests {
		t.Run("flag="+tt.flag, func(t *testing.T) {
			assert.Equal(t, tt.expected, integrationFlagToValue(tt.flag))
		})
	}
}

func TestConvert_EmptyChildrenList(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder = []Kind{}

	result, err := Convert(output)
	require.NoError(t, err)

	assert.Equal(t, 0, result.ChildrenCount)
	assert.Empty(t, result.Children)
	// Facility-level data should still be present.
	assert.Equal(t, "Kita Sonnenschein", result.FacilityName)
	assert.Len(t, result.Surcharges, 3)
}

func TestConvert_BothQMAndMSSActive(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder[0].QM = "ja"
	output.Vertrag.Kinder[0].MSS = "ja"
	output.Vertrag.Kinder[0].ZuschlagQM = 3000
	output.Vertrag.Kinder[0].ZuschlagMSS = 2000

	result, err := Convert(output)
	require.NoError(t, err)

	qm := result.Children[0].Amounts[2]
	assert.Equal(t, "qm/mss", qm.Key)
	assert.Equal(t, "qm/mss", qm.Value)
	assert.Equal(t, 5000, qm.Amount) // combined
}

func TestConvert_AllFlagsActive(t *testing.T) {
	output := makeTestOutput()
	k := &output.Vertrag.Kinder[0]
	k.QM = "ja"
	k.MSS = "ja"
	k.HS = "ND"
	k.Integration = "B"
	k.ZuschlagQM = 3000
	k.ZuschlagMSS = 2000
	k.ZuschlagNDH = 10116
	k.ZuschlagIntegration = 330641

	result, err := Convert(output)
	require.NoError(t, err)

	child := result.Children[0]
	assert.Equal(t, "ndh", child.Amounts[1].Value)
	assert.Equal(t, 10116, child.Amounts[1].Amount)
	assert.Equal(t, "qm/mss", child.Amounts[2].Value)
	assert.Equal(t, 5000, child.Amounts[2].Amount)
	assert.Equal(t, "integration b", child.Amounts[3].Value)
	assert.Equal(t, 330641, child.Amounts[3].Amount)
}

func TestConvert_AllFlagsInactive(t *testing.T) {
	output := makeTestOutput()
	k := &output.Vertrag.Kinder[0]
	k.QM = "nein"
	k.MSS = "nein"
	k.HS = "D"
	k.Integration = "N"
	k.ZuschlagQM = 0
	k.ZuschlagMSS = 0
	k.ZuschlagNDH = 0
	k.ZuschlagIntegration = 0

	result, err := Convert(output)
	require.NoError(t, err)

	child := result.Children[0]
	assert.Equal(t, "", child.Amounts[1].Value)
	assert.Equal(t, 0, child.Amounts[1].Amount)
	assert.Equal(t, "", child.Amounts[2].Value)
	assert.Equal(t, 0, child.Amounts[2].Amount)
	assert.Equal(t, "", child.Amounts[3].Value)
	assert.Equal(t, 0, child.Amounts[3].Amount)
}

func TestConvert_QMCaseInsensitiveThroughConvert(t *testing.T) {
	// isFlagActive handles case-insensitivity; verify it works end-to-end.
	for _, qmValue := range []string{"ja", "Ja", "JA"} {
		t.Run("QM="+qmValue, func(t *testing.T) {
			output := makeTestOutput()
			output.Vertrag.Kinder[0].QM = qmValue
			output.Vertrag.Kinder[0].MSS = "nein"
			output.Vertrag.Kinder[0].ZuschlagQM = 4000
			output.Vertrag.Kinder[0].ZuschlagMSS = 0

			result, err := Convert(output)
			require.NoError(t, err)

			qm := result.Children[0].Amounts[2]
			assert.Equal(t, "qm/mss", qm.Value)
			assert.Equal(t, 4000, qm.Amount)
		})
	}
}

func TestConvert_ZeroFacilitySurcharges(t *testing.T) {
	output := makeTestOutput()
	output.Einrichtung.ZuschlagQM = 0
	output.Einrichtung.ZuschlagMSS = 0
	output.Einrichtung.ZuschlagNDH = 0
	output.Einrichtung.ZuschlagIntegration = 0

	result, err := Convert(output)
	require.NoError(t, err)

	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 0}, result.Surcharges[0])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 0}, result.Surcharges[1])
	assert.Equal(t, SettlementAmount{Key: "integration", Value: "integration", Amount: 0}, result.Surcharges[2])
}

func TestConvert_EmptyBetreuungsumfang(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder[0].Betreuungsumfang = ""

	_, err := Convert(output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Betreuungsumfang")
}

func TestConvert_SecondChildInactiveFlagWithAmount(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:     "GB-11111111111-01",
		Name:                "Testkind, Lisa",
		Geburtsdatum:        "03.21",
		QM:                  "nein",
		MSS:                 "nein",
		HS:                  "D",
		Integration:         "N",
		Betreuungsumfang:    "ganztags",
		Bezirk:              3,
		Basisentgeld:        50000,
		ZuschlagQM:          5000, // QM/MSS inactive but amount non-zero → passes through
		ZuschlagMSS:         0,
		ZuschlagNDH:         0,
		ZuschlagIntegration: 0,
	})

	result, err := Convert(output)
	require.NoError(t, err)
	require.Len(t, result.Children, 2)

	qmmss := result.Children[1].Amounts[2]
	assert.Equal(t, "qm/mss", qmmss.Value)
	assert.Equal(t, 5000, qmmss.Amount)
}

func TestConvert_QMActiveAmountOnlyInMSSSurcharge(t *testing.T) {
	// QM="ja", MSS="nein", but the amount is in ZuschlagMSS not ZuschlagQM.
	// Since validation checks the combined amount and QM is active, this should pass.
	output := makeTestOutput()
	output.Vertrag.Kinder[0].QM = "ja"
	output.Vertrag.Kinder[0].MSS = "nein"
	output.Vertrag.Kinder[0].ZuschlagQM = 0
	output.Vertrag.Kinder[0].ZuschlagMSS = 5000

	result, err := Convert(output)
	require.NoError(t, err)

	qm := result.Children[0].Amounts[2]
	assert.Equal(t, "qm/mss", qm.Value)
	assert.Equal(t, 5000, qm.Amount)
}

func TestConvert_MultipleInactiveFlagsWithAmountsPassThrough(t *testing.T) {
	output := makeTestOutput()
	k := &output.Vertrag.Kinder[0]
	k.QM = "nein"
	k.MSS = "nein"
	k.HS = "D"
	k.ZuschlagQM = 5000
	k.ZuschlagMSS = 0
	k.ZuschlagNDH = 3000

	result, err := Convert(output)
	require.NoError(t, err)

	ndh := result.Children[0].Amounts[1]
	assert.Equal(t, "ndh", ndh.Value)
	assert.Equal(t, 3000, ndh.Amount)

	qmmss := result.Children[0].Amounts[2]
	assert.Equal(t, "qm/mss", qmmss.Value)
	assert.Equal(t, 5000, qmmss.Amount)
}

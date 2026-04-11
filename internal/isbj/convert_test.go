package isbj

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findAmount returns the first SettlementAmount matching the given key.
func findAmount(t *testing.T, amounts []SettlementAmount, key string) SettlementAmount {
	t.Helper()
	for _, a := range amounts {
		if a.Key == key {
			return a
		}
	}
	t.Fatalf("no amount with key %q found", key)
	return SettlementAmount{}
}

// assertNoAmount asserts no SettlementAmount exists with the given key.
func assertNoAmount(t *testing.T, amounts []SettlementAmount, key string) {
	t.Helper()
	for _, a := range amounts {
		if a.Key == key {
			t.Errorf("expected no amount with key %q, but found %+v", key, a)
		}
	}
}

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

	// ndh (HS=D, amount=0) and integration (N, amount=0) are filtered out
	require.Len(t, child.Rows[0].Amounts, 4)
	assert.Equal(t, SettlementAmount{Key: "care_type", Value: "ganztag", Amount: 89000}, child.Rows[0].Amounts[0])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 5531}, child.Rows[0].Amounts[1])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: -5000}, child.Rows[0].Amounts[2])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: -2300}, child.Rows[0].Amounts[3])
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

			careAmount := result.Children[0].Rows[0].Amounts[0]
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

		qmAmount := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
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

		qmAmount := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
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

		// Both inactive and zero → filtered out
		assertNoAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
	})

	t.Run("HS=D means ndh inactive", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "D"
		output.Vertrag.Kinder[0].ZuschlagNDH = 0

		result, err := Convert(output)
		require.NoError(t, err)

		// Inactive and zero → filtered out
		assertNoAmount(t, result.Children[0].Rows[0].Amounts, "ndh")
	})

	t.Run("HS=ND means ndh active", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "ND"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10116

		result, err := Convert(output)
		require.NoError(t, err)

		ndhAmount := findAmount(t, result.Children[0].Rows[0].Amounts, "ndh")
		assert.Equal(t, "ndh", ndhAmount.Value)
		assert.Equal(t, 10116, ndhAmount.Amount)
	})

	t.Run("Integration=N means no integration", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		// Inactive and zero → filtered out
		assertNoAmount(t, result.Children[0].Rows[0].Amounts, "integration")
	})

	t.Run("Integration=A means integration a", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "A"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 165680

		result, err := Convert(output)
		require.NoError(t, err)

		intAmount := findAmount(t, result.Children[0].Rows[0].Amounts, "integration")
		assert.Equal(t, "integration a", intAmount.Value)
		assert.Equal(t, 165680, intAmount.Amount)
	})

	t.Run("Integration=B means integration b", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "B"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 330641

		result, err := Convert(output)
		require.NoError(t, err)

		intAmount := findAmount(t, result.Children[0].Rows[0].Amounts, "integration")
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

		qm := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
		assert.Equal(t, "qm/mss", qm.Value)
		assert.Equal(t, 0, qm.Amount)
	})

	t.Run("ndH active but amount 0 is allowed", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "ND"
		output.Vertrag.Kinder[0].ZuschlagNDH = 0

		result, err := Convert(output)
		require.NoError(t, err)

		ndh := findAmount(t, result.Children[0].Rows[0].Amounts, "ndh")
		assert.Equal(t, "ndh", ndh.Value)
		assert.Equal(t, 0, ndh.Amount)
	})

	t.Run("Integration A but amount 0 is allowed", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "A"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		intg := findAmount(t, result.Children[0].Rows[0].Amounts, "integration")
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

		qmmss := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
		assert.Equal(t, "qm/mss", qmmss.Value, "value should be set when amount is non-zero")
		assert.Equal(t, 5000, qmmss.Amount)
	})

	t.Run("ndH inactive but amount non-zero passes through", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "D"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10000

		result, err := Convert(output)
		require.NoError(t, err)

		ndh := findAmount(t, result.Children[0].Rows[0].Amounts, "ndh")
		assert.Equal(t, "ndh", ndh.Value, "value should be set when amount is non-zero")
		assert.Equal(t, 10000, ndh.Amount)
	})

	t.Run("Integration N but amount non-zero uses generic value", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 165680

		result, err := Convert(output)
		require.NoError(t, err)

		intg := findAmount(t, result.Children[0].Rows[0].Amounts, "integration")
		assert.Equal(t, "integration", intg.Value, "generic value when flag is N but amount is non-zero")
		assert.Equal(t, 165680, intg.Amount)
	})

	t.Run("Integration N with zero amount is filtered out", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].Integration = "N"
		output.Vertrag.Kinder[0].ZuschlagIntegration = 0

		result, err := Convert(output)
		require.NoError(t, err)

		assertNoAmount(t, result.Children[0].Rows[0].Amounts, "integration")
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
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: -5000}, child.Rows[0].Amounts[2])
	// parent meals
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: -2300}, child.Rows[0].Amounts[3])
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
	assert.Equal(t, SettlementAmount{Key: "care_type", Value: "teilzeit", Amount: 65000}, second.Rows[0].Amounts[0])
	// ndh active (HS=ND)
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 10116}, second.Rows[0].Amounts[1])
	// qm/mss active (MSS=ja)
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 4000}, second.Rows[0].Amounts[2])
	// integration a (Integration=A)
	assert.Equal(t, SettlementAmount{Key: "integration", Value: "integration a", Amount: 165680}, second.Rows[0].Amounts[3])
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

	qm := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
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
	ndh := findAmount(t, child.Rows[0].Amounts, "ndh")
	assert.Equal(t, "ndh", ndh.Value)
	assert.Equal(t, 10116, ndh.Amount)
	qmmss := findAmount(t, child.Rows[0].Amounts, "qm/mss")
	assert.Equal(t, "qm/mss", qmmss.Value)
	assert.Equal(t, 5000, qmmss.Amount)
	intg := findAmount(t, child.Rows[0].Amounts, "integration")
	assert.Equal(t, "integration b", intg.Value)
	assert.Equal(t, 330641, intg.Amount)
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
	assertNoAmount(t, child.Rows[0].Amounts, "ndh")
	assertNoAmount(t, child.Rows[0].Amounts, "qm/mss")
	assertNoAmount(t, child.Rows[0].Amounts, "integration")
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

			qm := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
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

	qmmss := findAmount(t, result.Children[1].Rows[0].Amounts, "qm/mss")
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

	qm := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
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

	ndh := findAmount(t, result.Children[0].Rows[0].Amounts, "ndh")
	assert.Equal(t, "ndh", ndh.Value)
	assert.Equal(t, 3000, ndh.Amount)

	qmmss := findAmount(t, result.Children[0].Rows[0].Amounts, "qm/mss")
	assert.Equal(t, "qm/mss", qmmss.Value)
	assert.Equal(t, 5000, qmmss.Amount)
}

func TestConvert_GroupsByVoucherNumber(t *testing.T) {
	output := makeTestOutput()
	// Add a second row with the same voucher number (e.g. a correction row).
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:  "GB-12345678901-01", // same voucher
		Name:             "Musterkind, Max",
		Geburtsdatum:     "01.20",
		QM:               "nein",
		MSS:              "nein",
		HS:               "D",
		Integration:      "N",
		Betreuungsumfang: "ganztags",
		Bezirk:           1,
		Basisentgeld:     -89000,
		ElternBetreuung:  -5000,
		ElternEssen:      -2300,
		Summe:            -96300,
	})

	result, err := Convert(output)
	require.NoError(t, err)

	// Two Excel rows with the same voucher → one ConvertedChild with 2 rows.
	assert.Equal(t, 1, result.ChildrenCount)
	require.Len(t, result.Children, 1)

	child := result.Children[0]
	assert.Equal(t, "GB-12345678901-01", child.VoucherNumber)
	require.Len(t, child.Rows, 2)

	// TotalAmount = sum of both row totals
	assert.Equal(t, 141331+(-96300), child.TotalAmount)
	assert.Equal(t, 141331, child.Rows[0].TotalRowAmount)
	assert.Equal(t, -96300, child.Rows[1].TotalRowAmount)
}

func TestConvert_GroupsPreservesDifferentVouchers(t *testing.T) {
	output := makeTestOutput()
	// Add a child with a different voucher number.
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:  "GB-98765432109-02",
		Name:             "Testkind, Anna",
		Geburtsdatum:     "06.19",
		QM:               "nein",
		MSS:              "nein",
		HS:               "D",
		Integration:      "N",
		Betreuungsumfang: "halbtag",
		Bezirk:           5,
		Basisentgeld:     50000,
		ElternBetreuung:  2000,
		ElternEssen:      1000,
		Summe:            53000,
	})

	result, err := Convert(output)
	require.NoError(t, err)

	// Two different vouchers → two ConvertedChild entries.
	assert.Equal(t, 2, result.ChildrenCount)
	require.Len(t, result.Children, 2)
	assert.Equal(t, "GB-12345678901-01", result.Children[0].VoucherNumber)
	assert.Equal(t, "GB-98765432109-02", result.Children[1].VoucherNumber)
	require.Len(t, result.Children[0].Rows, 1)
	require.Len(t, result.Children[1].Rows, 1)
}

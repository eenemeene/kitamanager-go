package isbj

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestOutput() *SenatsabrechnungOutput {
	return &SenatsabrechnungOutput{
		Einrichtung: &Einrichtung{
			Name:        "Kita Sonnenschein",
			ZuschlagQM:  10000,
			ZuschlagMSS: 5000,
			ZuschlagNDH: 20000,
			ZuschlagSPH: 3000,
			Summe:       500000,
		},
		Abrechnung: &Abrechnung{
			VertragsBuchung:  400000,
			KorrekturBuchung: 100000,
		},
		Vertrag: &Vertrag{
			Kinder: []Kind{
				{
					Gutscheinnummer:  "GB-12345678901-01",
					Name:             "Musterkind, Max",
					Geburtsdatum:     "01.20",
					QM:               "ja",
					MSS:              "nein",
					HS:               "D",
					SPH:              "N",
					Betreuungsumfang: "ganztags",
					Bezirk:           1,
					Basisentgeld:     89000,
					AbzugOM:          -500,
					ElternBetreuung:  5000,
					ElternEssen:      2300,
					BuT:              0,
					AnteilBezirk:     45000,
					ZuschlagQM:       5531,
					ZuschlagMSS:      0,
					ZuschlagNDH:      0,
					ZuschlagSPH:      0,
					Summe:            141331,
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

	require.Len(t, child.Amounts, 9)
	assert.Equal(t, SettlementAmount{Key: "care_type", Value: "ganztag", Amount: 89000}, child.Amounts[0])
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "", Amount: 0}, child.Amounts[1])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 5531}, child.Amounts[2])
	assert.Equal(t, SettlementAmount{Key: "sph", Value: "", Amount: 0}, child.Amounts[3])
	assert.Equal(t, SettlementAmount{Key: "deduction", Value: "om", Amount: -500}, child.Amounts[4])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: 5000}, child.Amounts[5])
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: 2300}, child.Amounts[6])
	assert.Equal(t, SettlementAmount{Key: "but", Value: "but", Amount: 0}, child.Amounts[7])
	assert.Equal(t, SettlementAmount{Key: "district", Value: "share", Amount: 45000}, child.Amounts[8])
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

	t.Run("HS=A means ndh active", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "A"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10116

		result, err := Convert(output)
		require.NoError(t, err)

		ndhAmount := result.Children[0].Amounts[1]
		assert.Equal(t, "ndh", ndhAmount.Key)
		assert.Equal(t, "ndh", ndhAmount.Value)
		assert.Equal(t, 10116, ndhAmount.Amount)
	})

	t.Run("SPH=N means sph inactive", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].SPH = "N"
		output.Vertrag.Kinder[0].ZuschlagSPH = 0

		result, err := Convert(output)
		require.NoError(t, err)

		sphAmount := result.Children[0].Amounts[3]
		assert.Equal(t, "sph", sphAmount.Key)
		assert.Equal(t, "", sphAmount.Value)
		assert.Equal(t, 0, sphAmount.Amount)
	})

	t.Run("SPH=J means sph active", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].SPH = "J"
		output.Vertrag.Kinder[0].ZuschlagSPH = 7500

		result, err := Convert(output)
		require.NoError(t, err)

		sphAmount := result.Children[0].Amounts[3]
		assert.Equal(t, "sph", sphAmount.Key)
		assert.Equal(t, "sph", sphAmount.Value)
		assert.Equal(t, 7500, sphAmount.Amount)
	})
}

func TestConvert_FlagValidationErrors(t *testing.T) {
	t.Run("QM active but amount 0", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "ja"
		output.Vertrag.Kinder[0].ZuschlagQM = 0
		output.Vertrag.Kinder[0].ZuschlagMSS = 0

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "QM/MSS")
		assert.Contains(t, err.Error(), "active but amount is 0")
	})

	t.Run("QM/MSS inactive but amount non-zero", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].QM = "nein"
		output.Vertrag.Kinder[0].MSS = "nein"
		output.Vertrag.Kinder[0].ZuschlagQM = 5000
		output.Vertrag.Kinder[0].ZuschlagMSS = 0

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "QM/MSS")
		assert.Contains(t, err.Error(), "inactive but amount is")
	})

	t.Run("ndH active but amount 0", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "A"
		output.Vertrag.Kinder[0].ZuschlagNDH = 0

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ndH")
		assert.Contains(t, err.Error(), "active but amount is 0")
	})

	t.Run("ndH inactive but amount non-zero", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].HS = "D"
		output.Vertrag.Kinder[0].ZuschlagNDH = 10000

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ndH")
		assert.Contains(t, err.Error(), "inactive but amount is")
	})

	t.Run("SpH active but amount 0", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].SPH = "J"
		output.Vertrag.Kinder[0].ZuschlagSPH = 0

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SpH")
		assert.Contains(t, err.Error(), "active but amount is 0")
	})

	t.Run("SpH inactive but amount non-zero", func(t *testing.T) {
		output := makeTestOutput()
		output.Vertrag.Kinder[0].SPH = "N"
		output.Vertrag.Kinder[0].ZuschlagSPH = 5000

		_, err := Convert(output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SpH")
		assert.Contains(t, err.Error(), "inactive but amount is")
	})
}

func TestConvert_FacilitySurcharges(t *testing.T) {
	output := makeTestOutput()

	result, err := Convert(output)
	require.NoError(t, err)

	require.Len(t, result.Surcharges, 3)
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 20000}, result.Surcharges[0])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 15000}, result.Surcharges[1])
	assert.Equal(t, SettlementAmount{Key: "sph", Value: "sph", Amount: 3000}, result.Surcharges[2])
}

func TestConvert_OtherLineItems(t *testing.T) {
	output := makeTestOutput()

	result, err := Convert(output)
	require.NoError(t, err)

	child := result.Children[0]

	// deduction
	assert.Equal(t, SettlementAmount{Key: "deduction", Value: "om", Amount: -500}, child.Amounts[4])
	// parent care
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "care", Amount: 5000}, child.Amounts[5])
	// parent meals
	assert.Equal(t, SettlementAmount{Key: "parent", Value: "meals", Amount: 2300}, child.Amounts[6])
	// BuT
	assert.Equal(t, SettlementAmount{Key: "but", Value: "but", Amount: 0}, child.Amounts[7])
	// district share
	assert.Equal(t, SettlementAmount{Key: "district", Value: "share", Amount: 45000}, child.Amounts[8])
}

func TestConvert_MultipleChildren(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:  "GB-98765432109-02",
		Name:             "Testkind, Anna",
		Geburtsdatum:     "06.19",
		QM:               "nein",
		MSS:              "ja",
		HS:               "A",
		SPH:              "N",
		Betreuungsumfang: "teilzeit",
		Bezirk:           5,
		Basisentgeld:     65000,
		AbzugOM:          0,
		ElternBetreuung:  3000,
		ElternEssen:      1500,
		BuT:              800,
		AnteilBezirk:     30000,
		ZuschlagQM:       0,
		ZuschlagMSS:      4000,
		ZuschlagNDH:      10116,
		ZuschlagSPH:      0,
		Summe:            114416,
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
	// ndh active (HS=A)
	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 10116}, second.Amounts[1])
	// qm/mss active (MSS=ja)
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 4000}, second.Amounts[2])
	// sph inactive (SPH=N)
	assert.Equal(t, SettlementAmount{Key: "sph", Value: "", Amount: 0}, second.Amounts[3])
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
		{"HS", "A", true},
		{"HS", "T", true},
		{"SPH", "N", false},
		{"SPH", "", false},
		{"SPH", "J", true},
		{"SPH", "X", true},
		{"UNKNOWN", "ja", false},
	}

	for _, tt := range tests {
		t.Run(tt.flagName+"="+tt.value, func(t *testing.T) {
			assert.Equal(t, tt.expected, isFlagActive(tt.flagName, tt.value))
		})
	}
}

func TestValidateFlagAmount(t *testing.T) {
	assert.NoError(t, validateFlagAmount("child", "QM", true, 5000))
	assert.NoError(t, validateFlagAmount("child", "QM", false, 0))
	assert.Error(t, validateFlagAmount("child", "QM", true, 0))
	assert.Error(t, validateFlagAmount("child", "QM", false, 5000))
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
	k.HS = "A"
	k.SPH = "J"
	k.ZuschlagQM = 3000
	k.ZuschlagMSS = 2000
	k.ZuschlagNDH = 10116
	k.ZuschlagSPH = 7500

	result, err := Convert(output)
	require.NoError(t, err)

	child := result.Children[0]
	assert.Equal(t, "ndh", child.Amounts[1].Value)
	assert.Equal(t, 10116, child.Amounts[1].Amount)
	assert.Equal(t, "qm/mss", child.Amounts[2].Value)
	assert.Equal(t, 5000, child.Amounts[2].Amount)
	assert.Equal(t, "sph", child.Amounts[3].Value)
	assert.Equal(t, 7500, child.Amounts[3].Amount)
}

func TestConvert_AllFlagsInactive(t *testing.T) {
	output := makeTestOutput()
	k := &output.Vertrag.Kinder[0]
	k.QM = "nein"
	k.MSS = "nein"
	k.HS = "D"
	k.SPH = "N"
	k.ZuschlagQM = 0
	k.ZuschlagMSS = 0
	k.ZuschlagNDH = 0
	k.ZuschlagSPH = 0

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
	output.Einrichtung.ZuschlagSPH = 0

	result, err := Convert(output)
	require.NoError(t, err)

	assert.Equal(t, SettlementAmount{Key: "ndh", Value: "ndh", Amount: 0}, result.Surcharges[0])
	assert.Equal(t, SettlementAmount{Key: "qm/mss", Value: "qm/mss", Amount: 0}, result.Surcharges[1])
	assert.Equal(t, SettlementAmount{Key: "sph", Value: "sph", Amount: 0}, result.Surcharges[2])
}

func TestConvert_EmptyBetreuungsumfang(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder[0].Betreuungsumfang = ""

	_, err := Convert(output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Betreuungsumfang")
}

func TestConvert_ErrorOnSecondChildIncludesName(t *testing.T) {
	output := makeTestOutput()
	output.Vertrag.Kinder = append(output.Vertrag.Kinder, Kind{
		Gutscheinnummer:  "GB-11111111111-01",
		Name:             "Fehlerkind, Lisa",
		Geburtsdatum:     "03.21",
		QM:               "ja",
		MSS:              "nein",
		HS:               "D",
		SPH:              "N",
		Betreuungsumfang: "ganztags",
		Bezirk:           3,
		Basisentgeld:     50000,
		ZuschlagQM:       0, // QM active but amount 0 → error
		ZuschlagMSS:      0,
		ZuschlagNDH:      0,
		ZuschlagSPH:      0,
	})

	_, err := Convert(output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Fehlerkind, Lisa")
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

func TestValidateFlagAmount_ErrorMessages(t *testing.T) {
	err := validateFlagAmount("Musterkind, Max", "QM", true, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Musterkind, Max")
	assert.Contains(t, err.Error(), "QM")
	assert.Contains(t, err.Error(), "active but amount is 0")

	err = validateFlagAmount("Testkind, Anna", "ndH", false, 12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Testkind, Anna")
	assert.Contains(t, err.Error(), "ndH")
	assert.Contains(t, err.Error(), "12345")
}

func TestConvert_FirstValidationErrorStopsConversion(t *testing.T) {
	// Both ndH and SpH are invalid, but only ndH error should surface
	// because it's validated first.
	output := makeTestOutput()
	k := &output.Vertrag.Kinder[0]
	k.HS = "A"
	k.SPH = "J"
	k.ZuschlagNDH = 0 // ndH active but 0
	k.ZuschlagSPH = 0 // SpH active but 0

	_, err := Convert(output)
	require.Error(t, err)
	// QM/MSS is checked first (and passes), then ndH fails.
	assert.Contains(t, err.Error(), "ndH")
	assert.NotContains(t, err.Error(), "SpH")
}

package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupTestDB(t)
}

func setupImporter(t *testing.T) (*GovernmentFundingImporter, *store.GovernmentFundingStore, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	transactor := store.NewTransactor(db)
	svc := service.NewGovernmentFundingService(fundingStore, transactor)
	imp := NewGovernmentFundingImporter(svc, transactor)
	return imp, fundingStore, db
}

func createTestYAMLFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-government-funding.yaml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// ---------------------------------------------------------------------------
// Unit tests for helpers
// ---------------------------------------------------------------------------

func TestEuroToCents(t *testing.T) {
	tests := []struct {
		name     string
		eur      float64
		expected int
	}{
		{"zero", 0, 0},
		{"whole number", 100.00, 10000},
		{"with cents", 1668.47, 166847},
		{"small amount", 0.01, 1},
		{"rounding up", 100.005, 10001},
		{"rounding down", 100.004, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := euroToCents(tt.eur)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectYear  int
		expectMonth int
		expectDay   int
	}{
		{"valid date", "2023-03-01", false, 2023, 3, 1},
		{"valid date 2", "2022-01-01", false, 2022, 1, 1},
		{"invalid format", "01-03-2023", true, 0, 0, 0},
		{"empty string", "", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectYear, result.Year())
				assert.Equal(t, tt.expectMonth, int(result.Month()))
				assert.Equal(t, tt.expectDay, result.Day())
			}
		})
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple word", "ganztag", "Ganztag"},
		{"underscore separated", "care_type", "Care Type"},
		{"slash separated", "qm/mss", "Qm Mss"},
		{"hyphen separated", "full-time", "Full Time"},
		{"mixed separators", "care_type/full-time", "Care Type Full Time"},
		{"already capitalized", "Ganztag", "Ganztag"},
		{"multiple underscores", "a_b_c", "A B C"},
		{"single char words", "a_b", "A B"},
		{"empty string", "", ""},
		{"spaces preserved in words", "ganztag erweitert", "Ganztag erweitert"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyMatchKey(t *testing.T) {
	minAge, maxAge := 0, 2
	key1 := propertyMatchKey("care_type", "ganztag", &minAge, &maxAge)
	key2 := propertyMatchKey("care_type", "ganztag", &minAge, &maxAge)
	assert.Equal(t, key1, key2)

	// Different age range = different key
	maxAge2 := 5
	key3 := propertyMatchKey("care_type", "ganztag", &minAge, &maxAge2)
	assert.NotEqual(t, key1, key3)

	// Same age, different value = different key
	key4 := propertyMatchKey("care_type", "halbtag", &minAge, &maxAge)
	assert.NotEqual(t, key1, key4)

	// Nil ages
	key5 := propertyMatchKey("care_type", "ganztag", nil, nil)
	assert.NotEqual(t, key1, key5)
}

func TestTimePtrEqual(t *testing.T) {
	t1, _ := parseDate("2024-01-01")
	t2, _ := parseDate("2024-01-01")
	t3, _ := parseDate("2024-06-01")

	assert.True(t, timePtrEqual(nil, nil))
	assert.True(t, timePtrEqual(&t1, &t2))
	assert.False(t, timePtrEqual(&t1, nil))
	assert.False(t, timePtrEqual(nil, &t1))
	assert.False(t, timePtrEqual(&t1, &t3))
}

// ---------------------------------------------------------------------------
// Fresh import tests
// ---------------------------------------------------------------------------

const basicYAML = `---
-
  from: '2023-03-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 1668.47
          requirement: 0.261
        - key: care_type
          value: halbtag
          payment: 1066.64
          requirement: 0.14
-
  from: '2022-01-01'
  to: '2023-02-28'
  full_time_weekly_hours: 39
  comment: |
    Test period
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 1640.43
          requirement: 0.261
`

func TestImportGovernmentFundingFromFile_FreshImport(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	filePath := createTestYAMLFile(t, basicYAML)

	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.NotZero(t, result.FundingID)
	assert.Equal(t, 2, result.PeriodsCreated)
	assert.Equal(t, 3, result.PropertiesCreated) // 2 + 1

	// Verify data in DB
	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "Berlin Kita-Förderung", funding.Name)
	assert.Equal(t, "berlin", funding.State)
	assert.Len(t, funding.Periods, 2)

	// Periods are ordered by from_date DESC
	period1 := funding.Periods[0] // 2023-03-01 (ongoing)
	assert.Nil(t, period1.To)
	assert.Len(t, period1.Properties, 2)

	var ganztag *models.GovernmentFundingProperty
	for i := range period1.Properties {
		if period1.Properties[i].Value == "ganztag" {
			ganztag = &period1.Properties[i]
			break
		}
	}
	require.NotNil(t, ganztag)
	assert.Equal(t, 166847, ganztag.Payment)
	assert.Equal(t, 0.261, ganztag.Requirement)
	require.NotNil(t, ganztag.MinAge)
	require.NotNil(t, ganztag.MaxAge)
	assert.Equal(t, 0, *ganztag.MinAge)
	assert.Equal(t, 2, *ganztag.MaxAge)

	// Second period
	period2 := funding.Periods[1] // 2022-01-01 to 2023-02-28
	require.NotNil(t, period2.To)
	assert.Equal(t, 2023, period2.To.Year())
	assert.Contains(t, period2.Comment, "Test period")
}

func TestImportGovernmentFundingFromFile_FarFutureDateTreatedAsOngoing(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2023-03-01'
  to: '2060-01-01'
  full_time_weekly_hours: 39
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 1668.47
          requirement: 0.261
`
	filePath := createTestYAMLFile(t, yaml)

	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)
	require.Len(t, funding.Periods, 1)
	assert.Nil(t, funding.Periods[0].To)
}

func TestImportGovernmentFundingFromFile_FileNotFound(t *testing.T) {
	imp, _, _ := setupImporter(t)
	_, err := imp.ImportGovernmentFundingFromFile(context.Background(), "/nonexistent/file.yaml", "berlin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestImportGovernmentFundingFromFile_InvalidYAML(t *testing.T) {
	imp, _, _ := setupImporter(t)
	filePath := createTestYAMLFile(t, "invalid: yaml: content: [")
	_, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestImportGovernmentFundingFromFile_InvalidState(t *testing.T) {
	imp, _, _ := setupImporter(t)
	_, err := imp.ImportGovernmentFundingFromFile(context.Background(), "/any/path.yaml", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state")
}

func TestImportGovernmentFunding_LabelFromYAML(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          label: Ganztag (bis 9h)
          payment: 1668.47
          requirement: 0.261
`
	filePath := createTestYAMLFile(t, yaml)
	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "Ganztag (bis 9h)", funding.Periods[0].Properties[0].Label)
}

func TestImportGovernmentFunding_LabelAutoGenerated(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag_erweitert
          payment: 2000.00
          requirement: 0.3
        - key: integration
          value: integration a
          payment: 1500.00
          requirement: 0.25
`
	filePath := createTestYAMLFile(t, yaml)
	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)

	var erweitert, integration *models.GovernmentFundingProperty
	for i := range funding.Periods[0].Properties {
		switch funding.Periods[0].Properties[i].Value {
		case "ganztag_erweitert":
			erweitert = &funding.Periods[0].Properties[i]
		case "integration a":
			integration = &funding.Periods[0].Properties[i]
		}
	}

	require.NotNil(t, erweitert)
	assert.Equal(t, "Ganztag Erweitert", erweitert.Label)
	require.NotNil(t, integration)
	assert.Equal(t, "Integration a", integration.Label)
}

func TestImportGovernmentFunding_LabelTrimmed(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          label: "  Ganztag  "
          payment: 1668.47
          requirement: 0.261
`
	filePath := createTestYAMLFile(t, yaml)
	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "Ganztag", funding.Periods[0].Properties[0].Label)
}

func TestImportGovernmentFunding_ApplyToAllContracts(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,8]
      properties:
        - key: parent
          value: meals
          label: Parent Meals
          payment: 23.0
          requirement: 0
          apply_to_all_contracts: true
        - key: care_type
          value: ganztag
          label: Ganztag
          payment: 1668.47
          requirement: 0.261
`
	filePath := createTestYAMLFile(t, yaml)
	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)

	var meals, careType *models.GovernmentFundingProperty
	for i := range funding.Periods[0].Properties {
		p := &funding.Periods[0].Properties[i]
		if p.Key == "parent" && p.Value == "meals" {
			meals = p
		}
		if p.Key == "care_type" {
			careType = p
		}
	}

	require.NotNil(t, meals)
	assert.True(t, meals.ApplyToAllContracts)
	require.NotNil(t, careType)
	assert.False(t, careType.ApplyToAllContracts)
}

func TestImportGovernmentFunding_SingleAgeRange(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [2,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 1800.00
          requirement: 0.245
`
	filePath := createTestYAMLFile(t, yaml)
	result, err := imp.ImportGovernmentFundingFromFile(context.Background(), filePath, "berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(context.Background(), result.FundingID, 0, nil)
	require.NoError(t, err)
	require.Len(t, funding.Periods[0].Properties, 1)
	prop := funding.Periods[0].Properties[0]
	require.NotNil(t, prop.MinAge)
	require.NotNil(t, prop.MaxAge)
	assert.Equal(t, 2, *prop.MinAge)
	assert.Equal(t, 2, *prop.MaxAge)
}

// ---------------------------------------------------------------------------
// Incremental update tests
// ---------------------------------------------------------------------------

func TestIncrementalImport_AddNewPeriod(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	// Initial import with 1 period
	yaml1 := `---
-
  from: '2023-01-01'
  to: '2023-12-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result1, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)
	assert.True(t, result1.Created)
	assert.Equal(t, 1, result1.PeriodsCreated)

	// Re-import with 2 periods
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1700.00
          requirement: 0.27
-
  from: '2023-01-01'
  to: '2023-12-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.False(t, result2.Created)
	assert.Equal(t, result1.FundingID, result2.FundingID)
	assert.Equal(t, 1, result2.PeriodsCreated)
	assert.Equal(t, 0, result2.PeriodsUpdated)
	assert.Equal(t, 1, result2.PropertiesCreated)

	// Verify both periods exist
	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Len(t, funding.Periods, 2)
}

func TestIncrementalImport_UpdatePeriodValues(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  comment: old comment
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result1, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Update: change weekly hours, comment, and add end date
	yaml2 := `---
-
  from: '2024-01-01'
  to: '2024-12-31'
  full_time_weekly_hours: 40
  comment: new comment
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.False(t, result2.Created)
	assert.Equal(t, 1, result2.PeriodsUpdated)
	assert.Equal(t, 0, result2.PeriodsCreated)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result1.FundingID, 0, nil)
	require.NoError(t, err)
	require.Len(t, funding.Periods, 1)
	assert.Equal(t, 40.0, funding.Periods[0].FullTimeWeeklyHours)
	assert.Equal(t, "new comment", funding.Periods[0].Comment)
	require.NotNil(t, funding.Periods[0].To)
}

func TestIncrementalImport_UpdatePropertyPayment(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result1, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Update payment
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1700.50
          requirement: 0.28
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 1, result2.PropertiesUpdated)
	assert.Equal(t, 0, result2.PropertiesCreated)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result1.FundingID, 0, nil)
	require.NoError(t, err)
	prop := funding.Periods[0].Properties[0]
	assert.Equal(t, 170050, prop.Payment) // 1700.50 EUR = 170050 cents
	assert.Equal(t, 0.28, prop.Requirement)
}

func TestIncrementalImport_AddNewProperty(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	_, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Add halbtag property
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
        - key: care_type
          value: halbtag
          payment: 1000.00
          requirement: 0.14
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 1, result2.PropertiesCreated)
	assert.Equal(t, 0, result2.PropertiesUpdated)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Len(t, funding.Periods[0].Properties, 2)
}

func TestIncrementalImport_DeleteRemovedProperty(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
        - key: care_type
          value: halbtag
          payment: 1000.00
          requirement: 0.14
`
	_, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Remove halbtag
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 1, result2.PropertiesDeleted)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Len(t, funding.Periods[0].Properties, 1)
	assert.Equal(t, "ganztag", funding.Periods[0].Properties[0].Value)
}

func TestIncrementalImport_PreservesDBPeriodsNotInYAML(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	// Import 2 periods
	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1700.00
          requirement: 0.27
-
  from: '2023-01-01'
  to: '2023-12-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	_, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Re-import with only 1 period (the 2023 period is missing)
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1700.00
          requirement: 0.27
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 0, result2.PeriodsCreated)

	// Both periods should still exist
	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Len(t, funding.Periods, 2)
}

func TestIncrementalImport_Idempotent(t *testing.T) {
	imp, _, _ := setupImporter(t)
	ctx := context.Background()

	yaml := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1668.47
          requirement: 0.261
`
	result1, err := imp.ImportGovernmentFunding(ctx, []byte(yaml), "berlin")
	require.NoError(t, err)
	assert.True(t, result1.Created)

	// Second import: no changes
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml), "berlin")
	require.NoError(t, err)
	assert.False(t, result2.Created)
	assert.Equal(t, 0, result2.PeriodsCreated)
	assert.Equal(t, 0, result2.PeriodsUpdated)
	assert.Equal(t, 0, result2.PropertiesCreated)
	assert.Equal(t, 0, result2.PropertiesUpdated)
	assert.Equal(t, 0, result2.PropertiesDeleted)
}

func TestIncrementalImport_OverlapDetection(t *testing.T) {
	imp, _, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: '2024-12-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	_, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	// Try to add overlapping period
	yaml2 := `---
-
  from: '2024-06-01'
  to: '2025-05-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1700.00
          requirement: 0.27
-
  from: '2024-01-01'
  to: '2024-12-31'
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1600.00
          requirement: 0.261
`
	_, err = imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overlap")
}

func TestIncrementalImport_PropertyMatchByCompositeKey(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	// Two entries with same key+value but different age ranges
	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 2000.00
          requirement: 0.30
    - age: [3,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1200.00
          requirement: 0.12
`
	result1, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 2, result1.PropertiesCreated)

	// Update only the 0-2 age range payment
	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 2100.00
          requirement: 0.31
    - age: [3,6]
      properties:
        - key: care_type
          value: ganztag
          payment: 1200.00
          requirement: 0.12
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 1, result2.PropertiesUpdated) // only 0-2 changed
	assert.Equal(t, 0, result2.PropertiesCreated)
	assert.Equal(t, 0, result2.PropertiesDeleted)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	require.Len(t, funding.Periods[0].Properties, 2)

	for _, p := range funding.Periods[0].Properties {
		if *p.MinAge == 0 {
			assert.Equal(t, 210000, p.Payment) // updated
		} else {
			assert.Equal(t, 120000, p.Payment) // unchanged
		}
	}
}

func TestIncrementalImport_UpdatePropertyLabel(t *testing.T) {
	imp, fundingStore, _ := setupImporter(t)
	ctx := context.Background()

	yaml1 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          label: Old Label
          payment: 1600.00
          requirement: 0.261
`
	_, err := imp.ImportGovernmentFunding(ctx, []byte(yaml1), "berlin")
	require.NoError(t, err)

	yaml2 := `---
-
  from: '2024-01-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,6]
      properties:
        - key: care_type
          value: ganztag
          label: New Label
          payment: 1600.00
          requirement: 0.261
`
	result2, err := imp.ImportGovernmentFunding(ctx, []byte(yaml2), "berlin")
	require.NoError(t, err)
	assert.Equal(t, 1, result2.PropertiesUpdated)

	funding, err := fundingStore.FindByIDWithDetails(ctx, result2.FundingID, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "New Label", funding.Periods[0].Properties[0].Label)
}

package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.GovernmentFunding{},
		&models.GovernmentFundingPeriod{},
		&models.GovernmentFundingProperty{},
	)
	require.NoError(t, err)

	return db
}

func createTestYAMLFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-government-funding.yaml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

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

func TestImportGovernmentFundingFromFile(t *testing.T) {
	yamlContent := `---
-
  from: '2023-03-01'
  to: ''
  entries:
    - age: [0,2]
      properties:
        'ganztag':
          'payment': 1668.47
          'requirement': 0.261
        'halbtag':
          'payment': 1066.64
          'requirement': 0.14
-
  from: '2022-01-01'
  to: '2023-03-01'
  comment: |
    Test period
  entries:
    - age: [0,2]
      properties:
        'ganztag':
          'payment': 1640.43
          'requirement': 0.261
`

	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	importer := NewGovernmentFundingImporter(db, fundingStore)

	filePath := createTestYAMLFile(t, yamlContent)

	// First import should succeed
	fundingID, err := importer.ImportGovernmentFundingFromFile(filePath, "Test Government Funding")
	require.NoError(t, err)
	assert.NotZero(t, fundingID)

	// Verify government funding was created
	funding, err := fundingStore.FindByIDWithDetails(fundingID, 0)
	require.NoError(t, err)
	assert.Equal(t, "Test Government Funding", funding.Name)
	assert.Len(t, funding.Periods, 2)

	// Periods are ordered by from_date DESC, so the latest (2023-03-01) is first
	period1 := funding.Periods[0] // 2023-03-01 onwards (ongoing - no end date)
	assert.Nil(t, period1.To)
	assert.Len(t, period1.Properties, 2)

	// Check ganztag property has correct cents conversion and age range
	var ganztag *models.GovernmentFundingProperty
	for i := range period1.Properties {
		if period1.Properties[i].Name == "ganztag" {
			ganztag = &period1.Properties[i]
			break
		}
	}
	require.NotNil(t, ganztag)
	assert.Equal(t, 166847, ganztag.Payment) // 1668.47 EUR = 166847 cents
	assert.Equal(t, 0.261, ganztag.Requirement)
	require.NotNil(t, ganztag.MinAge)
	require.NotNil(t, ganztag.MaxAge)
	assert.Equal(t, 0, *ganztag.MinAge)
	// Both MinAge and MaxAge are inclusive: [0,2] means ages 0, 1, and 2
	assert.Equal(t, 2, *ganztag.MaxAge)

	// Check second period has end date and comment
	period2 := funding.Periods[1] // 2022-01-01 to 2023-03-01
	require.NotNil(t, period2.To)
	assert.Equal(t, 2023, period2.To.Year())
	assert.Contains(t, period2.Comment, "Test period")
}

func TestImportGovernmentFundingFromFile_Idempotency(t *testing.T) {
	yamlContent := `---
-
  from: '2023-03-01'
  to: ''
  entries:
    - age: [0,2]
      properties:
        'ganztag':
          'payment': 1668.47
          'requirement': 0.261
`

	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	importer := NewGovernmentFundingImporter(db, fundingStore)

	filePath := createTestYAMLFile(t, yamlContent)

	// First import
	fundingID1, err := importer.ImportGovernmentFundingFromFile(filePath, "Berlin")
	require.NoError(t, err)

	// Second import should return ErrGovernmentFundingExists
	fundingID2, err := importer.ImportGovernmentFundingFromFile(filePath, "Berlin")
	assert.ErrorIs(t, err, ErrGovernmentFundingExists)
	assert.Equal(t, fundingID1, fundingID2)

	// Verify only one government funding exists
	fundings, total, err := fundingStore.FindAll(100, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, fundings, 1)
}

func TestImportGovernmentFundingFromFile_FarFutureDateTreatedAsOngoing(t *testing.T) {
	yamlContent := `---
-
  from: '2023-03-01'
  to: '2060-01-01'
  entries:
    - age: [0,2]
      properties:
        'ganztag':
          'payment': 1668.47
          'requirement': 0.261
`

	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	importer := NewGovernmentFundingImporter(db, fundingStore)

	filePath := createTestYAMLFile(t, yamlContent)

	fundingID, err := importer.ImportGovernmentFundingFromFile(filePath, "Berlin")
	require.NoError(t, err)

	funding, err := fundingStore.FindByIDWithDetails(fundingID, 0)
	require.NoError(t, err)
	require.Len(t, funding.Periods, 1)

	// 2060-01-01 should be treated as nil (ongoing)
	assert.Nil(t, funding.Periods[0].To)
}

func TestImportGovernmentFundingFromFile_FileNotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	importer := NewGovernmentFundingImporter(db, fundingStore)

	_, err := importer.ImportGovernmentFundingFromFile("/nonexistent/file.yaml", "Test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestImportGovernmentFundingFromFile_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	importer := NewGovernmentFundingImporter(db, fundingStore)

	filePath := createTestYAMLFile(t, "invalid: yaml: content: [")

	_, err := importer.ImportGovernmentFundingFromFile(filePath, "Test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

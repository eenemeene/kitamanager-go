package export

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func strP(s string) *string { return &s }

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func dateP(y, m, d int) *time.Time {
	t := date(y, m, d)
	return &t
}

func TestWriteEmployeesExcel(t *testing.T) {
	employees := []models.EmployeeResponse{
		{
			ID:        1,
			FirstName: "Anna",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: date(1990, 5, 15),
			Contracts: []models.EmployeeContractResponse{
				{
					ID:            1,
					From:          date(2025, 1, 1),
					To:            dateP(2025, 12, 31),
					SectionName:   strP("Krippe"),
					StaffCategory: "qualified",
					Grade:         "S8a",
					Step:          3,
					WeeklyHours:   39.0,
				},
			},
		},
		{
			ID:        2,
			FirstName: "Max",
			LastName:  "Müller",
			Gender:    "male",
			Birthdate: date(1985, 3, 20),
		},
	}

	var buf bytes.Buffer
	err := WriteEmployeesExcel(&buf, employees)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	// Parse back and verify
	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	sheets := f.GetSheetList()
	require.Contains(t, sheets, "Mitarbeiter")

	// Verify headers
	h1, _ := f.GetCellValue("Mitarbeiter", "A1")
	assert.Equal(t, "Vorname", h1)
	h2, _ := f.GetCellValue("Mitarbeiter", "B1")
	assert.Equal(t, "Nachname", h2)
	h3, _ := f.GetCellValue("Mitarbeiter", "C1")
	assert.Equal(t, "Geschlecht", h3)

	// Verify first employee data
	a2, _ := f.GetCellValue("Mitarbeiter", "A2")
	assert.Equal(t, "Anna", a2)
	b2, _ := f.GetCellValue("Mitarbeiter", "B2")
	assert.Equal(t, "Schmidt", b2)
	c2, _ := f.GetCellValue("Mitarbeiter", "C2")
	assert.Equal(t, "weiblich", c2)
	e2, _ := f.GetCellValue("Mitarbeiter", "E2")
	assert.Equal(t, "Krippe", e2)
	f2, _ := f.GetCellValue("Mitarbeiter", "F2")
	assert.Equal(t, "qualified", f2)
	g2, _ := f.GetCellValue("Mitarbeiter", "G2")
	assert.Equal(t, "3", g2)
	h2Val, _ := f.GetCellValue("Mitarbeiter", "H2")
	assert.Equal(t, "S8a", h2Val)
	i2, _ := f.GetCellValue("Mitarbeiter", "I2")
	assert.Equal(t, "39.00", i2)

	// Verify second employee (no contract) has name but empty contract fields
	a3, _ := f.GetCellValue("Mitarbeiter", "A3")
	assert.Equal(t, "Max", a3)
	c3, _ := f.GetCellValue("Mitarbeiter", "C3")
	assert.Equal(t, "männlich", c3)
	e3, _ := f.GetCellValue("Mitarbeiter", "E3")
	assert.Equal(t, "", e3)

	// Verify no fourth row (only header + 2 data rows)
	a4, _ := f.GetCellValue("Mitarbeiter", "A4")
	assert.Equal(t, "", a4)
}

func TestWriteChildrenExcel(t *testing.T) {
	children := []models.ChildResponse{
		{
			ID:        1,
			FirstName: "Emma",
			LastName:  "Weber",
			Gender:    "female",
			Birthdate: date(2020, 3, 10),
			Contracts: []models.ChildContractResponse{
				{
					ID:          1,
					From:        date(2025, 1, 1),
					To:          dateP(2025, 12, 31),
					SectionName: strP("Elementar"),
					Properties: models.ContractProperties{
						"care_type":   "ganztag",
						"supplements": []any{"ndh", "mss"},
					},
				},
			},
		},
		{
			ID:        2,
			FirstName: "Liam",
			LastName:  "Fischer",
			Gender:    "diverse",
			Birthdate: date(2021, 7, 5),
		},
	}

	var buf bytes.Buffer
	err := WriteChildrenExcel(&buf, children)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	// Parse back and verify
	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	sheets := f.GetSheetList()
	require.Contains(t, sheets, "Kinder")

	// Verify headers
	h1, _ := f.GetCellValue("Kinder", "A1")
	assert.Equal(t, "Vorname", h1)
	h8, _ := f.GetCellValue("Kinder", "H1")
	assert.Equal(t, "Betreuungsumfang", h8)
	h9, _ := f.GetCellValue("Kinder", "I1")
	assert.Equal(t, "Zuschläge", h9)

	// Verify first child data
	a2, _ := f.GetCellValue("Kinder", "A2")
	assert.Equal(t, "Emma", a2)
	c2, _ := f.GetCellValue("Kinder", "C2")
	assert.Equal(t, "weiblich", c2)
	e2, _ := f.GetCellValue("Kinder", "E2")
	assert.Equal(t, "Elementar", e2)
	h2, _ := f.GetCellValue("Kinder", "H2")
	assert.Equal(t, "ganztag", h2)
	i2, _ := f.GetCellValue("Kinder", "I2")
	assert.Equal(t, "ndh, mss", i2)

	// Verify second child (no contract)
	a3, _ := f.GetCellValue("Kinder", "A3")
	assert.Equal(t, "Liam", a3)
	c3, _ := f.GetCellValue("Kinder", "C3")
	assert.Equal(t, "divers", c3)
	h3, _ := f.GetCellValue("Kinder", "H3")
	assert.Equal(t, "", h3)
}

func TestWriteEmployeesExcel_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteEmployeesExcel(&buf, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	// Should still have headers
	h1, _ := f.GetCellValue("Mitarbeiter", "A1")
	assert.Equal(t, "Vorname", h1)
}

func TestWriteChildrenExcel_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteChildrenExcel(&buf, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	h1, _ := f.GetCellValue("Kinder", "A1")
	assert.Equal(t, "Vorname", h1)
}

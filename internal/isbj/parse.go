package isbj

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type Einrichtung struct {
	Name                string `json:"name"`
	ElternBetreuung     int    `json:"eltern_betreuung"`
	ElternEssen         int    `json:"eltern_essen"`
	ZuschlagQM          int    `json:"zuschlag_qm"`
	ZuschlagMSS         int    `json:"zuschlag_mss"`
	ZuschlagNDH         int    `json:"zuschlag_ndh"`
	ZuschlagIntegration int    `json:"zuschlag_integration"`
	Summe               int    `json:"summe"`
}

func (e Einrichtung) String() string {
	return fmt.Sprintf("Einrichtung: %s: ∑ %.2f € (QM: %.2f €, MSS: %.2f €, NDH: %.2f €, Integration: %.2f €)",
		e.Name,
		float64(e.Summe)/100,
		float64(e.ZuschlagQM)/100,
		float64(e.ZuschlagMSS)/100,
		float64(e.ZuschlagNDH)/100,
		float64(e.ZuschlagIntegration)/100,
	)
}

type Abrechnung struct {
	VertragsBuchung  int `json:"vertrags_buchung"`
	KorrekturBuchung int `json:"korrektur_buchung"`
}

func (a Abrechnung) String() string {
	return fmt.Sprintf("Abrechnung: %.2f € (Vertrag), %.2f € (Korrektur)",
		float64(a.VertragsBuchung)/100,
		float64(a.KorrekturBuchung)/100,
	)
}

type Kind struct {
	Gutscheinnummer     string `json:"gutscheinnummer"`
	Name                string `json:"name"`
	Geburtsdatum        string `json:"geburtsdatum"`
	QM                  string `json:"qm"`
	MSS                 string `json:"mss"`
	HS                  string `json:"hs"`
	Integration         string `json:"integration"`
	Registrierdatum     string `json:"registrierdatum"`
	Betreuungsumfang    string `json:"betreuungsumfang"`
	Bezirk              int64  `json:"bezirk"`
	Basisentgeld        int    `json:"basisentgeld"`
	AbzugOM             int    `json:"abzug_om"`
	ElternBetreuung     int    `json:"eltern_betreuung"`
	ElternEssen         int    `json:"eltern_essen"`
	BuT                 int    `json:"but"`
	AnteilBezirk        int    `json:"anteil_bezirk"`
	ZuschlagQM          int    `json:"zuschlag_qm"`
	ZuschlagMSS         int    `json:"zuschlag_mss"`
	ZuschlagNDH         int    `json:"zuschlag_ndh"`
	ZuschlagIntegration int    `json:"zuschlag_integration"`
	Summe               int    `json:"summe"`
}

func (k Kind) String() string {
	return fmt.Sprintf("%s (%s, geb. %s): %.2f €", k.Name, k.Gutscheinnummer, k.Geburtsdatum, float64(k.Summe)/100)
}

type Vertrag struct {
	Kinder []Kind `json:"kinder"`
}

func (v Vertrag) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Vertrag mit %d Kindern:\n", len(v.Kinder)))
	for i, k := range v.Kinder {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, k.String()))
	}
	return sb.String()
}

type SenatsabrechnungOutput struct {
	Einrichtung  *Einrichtung `json:"einrichtung"`
	Abrechnung   *Abrechnung  `json:"abrechnung"`
	Vertrag      *Vertrag     `json:"vertrag"`
	BillingMonth time.Time    `json:"billing_month"`
}

func cellAsString(f *excelize.File, sheet, cell string) string {
	val, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}

func cellAsInt64(f *excelize.File, sheet, cell string) (int64, error) {
	val := cellAsString(f, sheet, cell)
	if val == "" {
		return 0, nil
	}
	return strconv.ParseInt(val, 10, 64)
}

// normalizeDecimal converts a number string from either US format (1,234.56)
// or German format (1.234,56) to a plain float string (1234.56).
func normalizeDecimal(val string) string {
	hasDot := strings.Contains(val, ".")
	hasComma := strings.Contains(val, ",")

	switch {
	case hasDot && hasComma:
		// Determine which is the decimal separator (last one wins)
		lastDot := strings.LastIndex(val, ".")
		lastComma := strings.LastIndex(val, ",")
		if lastComma > lastDot {
			// German: 1.234,56 → strip dots, replace comma with dot
			val = strings.ReplaceAll(val, ".", "")
			val = strings.Replace(val, ",", ".", 1)
		} else {
			// US: 1,234.56 → strip commas
			val = strings.ReplaceAll(val, ",", "")
		}
	case hasComma && !hasDot:
		// Could be German decimal (899,87) or US thousands (1,234)
		// Check if comma separates exactly 2 decimal digits at the end
		lastComma := strings.LastIndex(val, ",")
		afterComma := val[lastComma+1:]
		if len(afterComma) <= 2 {
			// German decimal: 899,87 → 899.87
			val = strings.Replace(val, ",", ".", 1)
		} else {
			// US thousands: 1,234 → 1234
			val = strings.ReplaceAll(val, ",", "")
		}
	}
	// If only dot → already US format, nothing to do
	return val
}

func cellAsCents(f *excelize.File, sheet, cell string) (int, error) {
	val := cellAsString(f, sheet, cell)
	if val == "" {
		return 0, nil
	}
	val = normalizeDecimal(val)
	f64, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("sheet %q cell %s: %w", sheet, cell, err)
	}
	return int(math.Round(f64 * 100)), nil
}

func cellAsCentsRequired(f *excelize.File, sheetName string, cell string) (int, error) {
	cellVal, err := f.GetCellValue(sheetName, cell)
	if err != nil {
		return 0, fmt.Errorf("sheet %q cell %s: %w", sheetName, cell, err)
	}
	cellVal = strings.TrimSpace(cellVal)
	cellVal = normalizeDecimal(cellVal)
	f64, err := strconv.ParseFloat(cellVal, 64)
	if err != nil {
		return 0, fmt.Errorf("sheet %q cell %s: %w", sheetName, cell, err)
	}
	return int(math.Round(f64 * 100)), nil
}

func OpenSenatsabrechnung(dateiname string) (*excelize.File, error) {
	f, err := excelize.OpenFile(dateiname)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ParseFromReader parses a Senatsabrechnung from an io.Reader (e.g., HTTP upload body).
func ParseFromReader(r io.Reader) (*SenatsabrechnungOutput, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("opening excel from reader: %w", err)
	}
	defer func() { _ = f.Close() }()

	eu, err := ParseEinrichtung(f)
	if err != nil {
		return nil, fmt.Errorf("parsing Einrichtung: %w", err)
	}

	ar, err := ParseAbrechnung(f)
	if err != nil {
		return nil, fmt.Errorf("parsing Abrechnung: %w", err)
	}

	v, err := ParseVertrag(f)
	if err != nil {
		return nil, fmt.Errorf("parsing Vertrag: %w", err)
	}

	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		return nil, fmt.Errorf("parsing billing month: %w", err)
	}

	return &SenatsabrechnungOutput{
		Einrichtung:  eu,
		Abrechnung:   ar,
		Vertrag:      v,
		BillingMonth: billingMonth,
	}, nil
}

// ParseBillingMonth extracts the billing month from the Einrichtungsübersicht sheet.
// It finds the "Trägernummer" label, locates the carrier number value in the same
// row (scanning rightward), and reads the billing month (MM/YY) from the cell
// directly below the carrier number value.
func ParseBillingMonth(f *excelize.File) (time.Time, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "Trägernummer")
	if err != nil {
		return time.Time{}, fmt.Errorf("finding Trägernummer header: %w", err)
	}

	colNum, err := excelize.ColumnNameToNumber(col)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing column name %q: %w", col, err)
	}

	// Find the carrier number value cell (same row, to the right of the label).
	// Skip merged cells that repeat the label text.
	labelText := cellAsString(f, SheetEinrichtung, fmt.Sprintf("%s%d", col, row))
	valueCol := ""
	for offset := 1; offset <= 10; offset++ {
		cn, err := excelize.ColumnNumberToName(colNum + offset)
		if err != nil {
			continue
		}
		v := cellAsString(f, SheetEinrichtung, fmt.Sprintf("%s%d", cn, row))
		if v != "" && v != labelText {
			valueCol = cn
			break
		}
	}
	if valueCol == "" {
		return time.Time{}, fmt.Errorf("trägernummer value not found in row %d", row)
	}

	// Billing month is directly below the carrier number value
	dateStr := cellAsString(f, SheetEinrichtung, fmt.Sprintf("%s%d", valueCol, row+1))
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("billing month cell below Trägernummer value is empty")
	}

	t, err := time.Parse("01/06", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing billing month %q: expected MM/YY format: %w", dateStr, err)
	}

	return t, nil
}

func getValueByHeaderName(f *excelize.File, sheetName string, header string) (string, int, error) {
	result, err := f.SearchSheet(sheetName, regexp.QuoteMeta(header), true)
	if err != nil {
		return "", 0, err
	}
	if len(result) != 1 {
		return "", 0, fmt.Errorf("sheet %q: header %q found %d times, expected 1", sheetName, header, len(result))
	}
	col, row, err := excelize.SplitCellName(result[0])
	if err != nil {
		return "", 0, err
	}
	return col, row, nil
}

func getEinrichtungName(f *excelize.File) (string, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "Einrichtungsname")
	if err != nil {
		return "", err
	}
	val, err := f.GetCellValue(SheetEinrichtung, fmt.Sprintf("%s%d", col, row+2))
	if err != nil {
		return "", err
	}
	return val, nil
}

func getEinrichtungZuschlagQM(f *excelize.File) (int, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "QM")
	if err != nil {
		return 0, err
	}
	coords := fmt.Sprintf("%s%d", col, row+1)
	return cellAsCentsRequired(f, SheetEinrichtung, coords)
}

func getEinrichtungZuschlagMSS(f *excelize.File) (int, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "MSS")
	if err != nil {
		return 0, err
	}
	coords := fmt.Sprintf("%s%d", col, row+1)
	return cellAsCentsRequired(f, SheetEinrichtung, coords)
}

func getEinrichtungZuschlagNDH(f *excelize.File) (int, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "ndH")
	if err != nil {
		// Old format (2020-Jun 2025) uses "NdHs" instead of "ndH"
		col, row, err = getValueByHeaderName(f, SheetEinrichtung, "NdHs")
		if err != nil {
			return 0, fmt.Errorf("ndH/NdHs header not found in Einrichtung sheet")
		}
	}
	coords := fmt.Sprintf("%s%d", col, row+1)
	return cellAsCentsRequired(f, SheetEinrichtung, coords)
}

// getEinrichtungZuschlagIntegration reads the facility-level integration surcharge.
// The Excel header is labeled "SpH" but it actually contains the integration surcharge.
func getEinrichtungZuschlagIntegration(f *excelize.File) (int, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "SpH")
	if err != nil {
		return 0, err
	}
	coords := fmt.Sprintf("%s%d", col, row+1)
	return cellAsCentsRequired(f, SheetEinrichtung, coords)
}

func getEinrichtungSumme(f *excelize.File) (int, error) {
	col, row, err := getValueByHeaderName(f, SheetEinrichtung, "Abrechn.")
	if err != nil {
		return 0, err
	}
	coords := fmt.Sprintf("%s%d", col, row+2)
	return cellAsCentsRequired(f, SheetEinrichtung, coords)
}

func ParseEinrichtung(f *excelize.File) (*Einrichtung, error) {
	eu := &Einrichtung{}
	name, err := getEinrichtungName(f)
	if err != nil {
		return nil, err
	}
	eu.Name = name

	zuschlagQM, err := getEinrichtungZuschlagQM(f)
	if err != nil {
		return nil, err
	}
	eu.ZuschlagQM = zuschlagQM

	zuschlagMSS, err := getEinrichtungZuschlagMSS(f)
	if err != nil {
		return nil, err
	}
	eu.ZuschlagMSS = zuschlagMSS

	zuschlagNDH, err := getEinrichtungZuschlagNDH(f)
	if err != nil {
		return nil, err
	}
	eu.ZuschlagNDH = zuschlagNDH

	zuschlagIntegration, err := getEinrichtungZuschlagIntegration(f)
	if err != nil {
		return nil, err
	}
	eu.ZuschlagIntegration = zuschlagIntegration

	summe, err := getEinrichtungSumme(f)
	if err != nil {
		return nil, err
	}
	eu.Summe = summe

	return eu, nil
}

func ParseAbrechnung(f *excelize.File) (*Abrechnung, error) {
	ar := &Abrechnung{}
	// Get the row where the sum is
	_, rowSum, err := getValueByHeaderName(f, SheetAbrechnung, "Summe:")
	if err != nil {
		return nil, err
	}

	// Get Vertragsbuchung
	colVB, rowVB, err := getValueByHeaderName(f, SheetAbrechnung, "Vertragsbuchung")
	if err != nil {
		return nil, err
	}
	ar.VertragsBuchung, err = readOrSumColumn(f, SheetAbrechnung, colVB, rowVB+1, rowSum)
	if err != nil {
		return nil, fmt.Errorf("vertragsbuchung: %w", err)
	}

	// Get Korrekturbuchung
	colKB, rowKB, err := getValueByHeaderName(f, SheetAbrechnung, "Korrekturbuchung")
	if err != nil {
		return nil, err
	}
	ar.KorrekturBuchung, err = readOrSumColumn(f, SheetAbrechnung, colKB, rowKB+1, rowSum)
	if err != nil {
		return nil, fmt.Errorf("korrekturbuchung: %w", err)
	}
	return ar, nil
}

// readOrSumColumn reads the value at (col, sumRow). If the cell is empty
// (e.g., an uncached SUM formula), it manually sums all cells from
// (col, dataStartRow) to (col, sumRow-1).
func readOrSumColumn(f *excelize.File, sheet, col string, dataStartRow, sumRow int) (int, error) {
	coords := fmt.Sprintf("%s%d", col, sumRow)
	val := cellAsString(f, sheet, coords)
	if val != "" {
		return cellAsCentsRequired(f, sheet, coords)
	}

	// Empty Summe cell — formula not cached; sum the data rows manually.
	total := 0
	for r := dataStartRow; r < sumRow; r++ {
		c := fmt.Sprintf("%s%d", col, r)
		v, err := cellAsCents(f, sheet, c)
		if err != nil {
			return 0, fmt.Errorf("sheet %q cell %s: %w", sheet, c, err)
		}
		total += v
	}
	return total, nil
}

// voucherPattern matches voucher numbers like GB-XXXXXXXXXXX-XX
var voucherPattern = regexp.MustCompile(`^GB-\d{11}-\d{2}$`)

const (
	SheetVertrag     = "Vertragsübersicht"
	SheetAbrechnung  = "Abrechnungsübersicht"
	SheetEinrichtung = "Einrichtungsübersicht"
)

// vertragColumns holds the dynamically discovered column letters for the Vertragsübersicht sheet.
type vertragColumns struct {
	gutscheinNr         string
	nameDerKinder       string
	gebDatum            string
	qm                  string
	mss                 string
	hs                  string
	integration         string
	registrAb           string
	betreuUmfang        string
	bezirk              string
	basisEntgelt        string
	abzugOM             string
	elternBetreuung     string
	elternEssen         string
	but                 string
	anteilBezirk        string
	zuschlagQM          string
	zuschlagMSS         string
	zuschlagNDH         string
	zuschlagIntegration string
	abrechnSumme        string
	headerRow           int
	dataStartRow        int
}

// normalizeHeaderCell trims whitespace and replaces embedded newlines with spaces.
// Some Excel cells contain literal "\n" (e.g., "Name\ndes Kindes") which would
// break header matching.
func normalizeHeaderCell(cell string) string {
	s := strings.ReplaceAll(cell, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// discoverVertragColumns finds the header row by locating "GutscheinNr" and maps
// all column headers to their column letters. This handles different file layouts.
func discoverVertragColumns(f *excelize.File) (*vertragColumns, error) {
	col, row, err := getValueByHeaderName(f, SheetVertrag, "GutscheinNr")
	if err != nil {
		return nil, fmt.Errorf("finding GutscheinNr header: %w", err)
	}

	vc := &vertragColumns{
		gutscheinNr: col,
		headerRow:   row,
	}

	// Read header row cells into a map: column letter → header text
	rows, err := f.GetRows(SheetVertrag)
	if err != nil {
		return nil, fmt.Errorf("reading rows: %w", err)
	}
	if row-1 >= len(rows) {
		return nil, fmt.Errorf("header row %d out of range", row)
	}
	headerCells := rows[row-1]

	// Also read the sub-header row (one below) for split headers like "Anteil Eltern" → "Betreuung"/"Essen"
	// and "Zuschläge" → "QM"/"MSS"/"ndH"/"SpH"
	var subHeaderCells []string
	if row < len(rows) {
		subHeaderCells = rows[row]
	}

	colMap := map[string]string{} // normalized header → column letter
	for j, cell := range headerCells {
		trimmed := normalizeHeaderCell(cell)
		if trimmed != "" {
			cn, _ := excelize.ColumnNumberToName(j + 1)
			colMap[trimmed] = cn
		}
	}
	subColMap := map[string]string{}
	for j, cell := range subHeaderCells {
		trimmed := normalizeHeaderCell(cell)
		if trimmed != "" {
			cn, _ := excelize.ColumnNumberToName(j + 1)
			subColMap[trimmed] = cn
		}
	}

	// Map header names to struct fields
	vc.nameDerKinder = colMap["Name des Kindes"]
	vc.gebDatum = colMap["Geb.-datum"]
	vc.registrAb = colMap["Registr. ab"]
	vc.betreuUmfang = colMap["Betreu.-umfang"]
	vc.bezirk = colMap["Be-zirk"]
	vc.basisEntgelt = colMap["Basis-entgelt"]
	vc.abzugOM = colMap["Abzug o.M."]
	vc.abrechnSumme = colMap["Abrechn.-summe"]

	// BuT appears twice (header row and sub-header); use the header row one
	// that's near GutscheinNr (before Finanzierung columns)
	if v, ok := colMap["BuT"]; ok {
		vc.but = v
	}

	// "Anteil Bezirk" in header row
	vc.anteilBezirk = colMap["Anteil Bezirk"]

	// QM/MSS/Hs appear in the main header row (before Finanzierung).
	// The Excel column labeled "SpH" actually contains the integration flag (A/B/N).
	vc.qm = colMap["QM"]
	vc.mss = colMap["MSS"]
	vc.hs = colMap["Hs"]
	vc.integration = colMap["SpH"]

	// Sub-header row has "Betreuung"/"Betr." and "Essen" under "Anteil Eltern"
	if v, ok := subColMap["Betreuung"]; ok {
		vc.elternBetreuung = v
	} else if v, ok := subColMap["Betr."]; ok {
		vc.elternBetreuung = v
	}
	vc.elternEssen = subColMap["Essen"]

	// Sub-header has Zuschlag columns: QM, MSS, ndH/NdHs, SpH (= integration surcharge).
	// These shadow the main header QM/MSS etc, so we need to distinguish:
	// the sub-header "QM" is the Zuschlag QM (further right than the header QM)
	if v, ok := subColMap["QM"]; ok {
		vc.zuschlagQM = v
	}
	if v, ok := subColMap["MSS"]; ok {
		vc.zuschlagMSS = v
	}
	if v, ok := subColMap["ndH"]; ok {
		vc.zuschlagNDH = v
	} else if v, ok := subColMap["NdHs"]; ok {
		vc.zuschlagNDH = v
	}
	// The Excel sub-header "SpH" is actually the integration surcharge amount.
	if v, ok := subColMap["SpH"]; ok {
		vc.zuschlagIntegration = v
	}

	// BuT also appears in sub-header row (near Zuschläge area); the one we want
	// for the BuT financial column is in the header row near "Anteil Bezirk"
	// Re-scan: find the BuT in header row that comes AFTER "Anteil Eltern"
	// For simplicity, if there are two BuT columns in the header, pick the second one
	butCols := []string{}
	for j, cell := range headerCells {
		if strings.TrimSpace(cell) == "BuT" {
			cn, _ := excelize.ColumnNumberToName(j + 1)
			butCols = append(butCols, cn)
		}
	}
	if len(butCols) >= 2 {
		// First BuT is the flag column, second is the financial BuT column
		vc.but = butCols[1]
	} else if len(butCols) == 1 {
		vc.but = butCols[0]
	}

	// Data starts 2 rows after header (header row + sub-header row + section header = +3,
	// but section header is variable). Find first voucher row.
	for i := row; i < len(rows); i++ {
		for _, cell := range rows[i] {
			if voucherPattern.MatchString(strings.TrimSpace(cell)) {
				vc.dataStartRow = i + 1 // 1-indexed
				return vc, nil
			}
		}
	}

	return nil, fmt.Errorf("no voucher data rows found after header row %d", row)
}

func parseKindWithColumns(f *excelize.File, row int, vc *vertragColumns) (*Kind, error) {
	kind := &Kind{}

	kind.Gutscheinnummer = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.gutscheinNr, row))
	kind.Name = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.nameDerKinder, row))
	kind.Geburtsdatum = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.gebDatum, row))
	kind.QM = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.qm, row))
	kind.MSS = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.mss, row))
	kind.HS = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.hs, row))
	kind.Integration = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.integration, row))
	kind.Registrierdatum = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.registrAb, row))
	kind.Betreuungsumfang = cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.betreuUmfang, row))

	bezirk, err := cellAsInt64(f, SheetVertrag, fmt.Sprintf("%s%d", vc.bezirk, row))
	if err != nil {
		return nil, fmt.Errorf("parsing Bezirk at row %d: %w", row, err)
	}
	if bezirk < 1 || bezirk > 12 {
		return nil, fmt.Errorf("row %d: Bezirk %d is not a valid Berlin district (1-12)", row, bezirk)
	}
	kind.Bezirk = bezirk

	switch kind.Betreuungsumfang {
	case "teilzeit", "ganztags", "erweitert", "halbtag":
		// valid
	default:
		return nil, fmt.Errorf("row %d: Betreuungsumfang %q is not valid (expected \"teilzeit\", \"ganztags\", \"erweitert\", or \"halbtag\")", row, kind.Betreuungsumfang)
	}

	kind.Basisentgeld, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.basisEntgelt, row))
	if err != nil {
		return nil, fmt.Errorf("parsing Basisentgeld at row %d: %w", row, err)
	}
	kind.AbzugOM, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.abzugOM, row))
	if err != nil {
		return nil, fmt.Errorf("parsing AbzugOM at row %d: %w", row, err)
	}
	kind.ElternBetreuung, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.elternBetreuung, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ElternBetreuung at row %d: %w", row, err)
	}
	kind.ElternEssen, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.elternEssen, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ElternEssen at row %d: %w", row, err)
	}
	kind.BuT, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.but, row))
	if err != nil {
		return nil, fmt.Errorf("parsing BuT at row %d: %w", row, err)
	}
	kind.AnteilBezirk, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.anteilBezirk, row))
	if err != nil {
		return nil, fmt.Errorf("parsing AnteilBezirk at row %d: %w", row, err)
	}
	kind.ZuschlagQM, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.zuschlagQM, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ZuschlagQM at row %d: %w", row, err)
	}
	kind.ZuschlagMSS, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.zuschlagMSS, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ZuschlagMSS at row %d: %w", row, err)
	}
	kind.ZuschlagNDH, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.zuschlagNDH, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ZuschlagNDH at row %d: %w", row, err)
	}
	kind.ZuschlagIntegration, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.zuschlagIntegration, row))
	if err != nil {
		return nil, fmt.Errorf("parsing ZuschlagIntegration at row %d: %w", row, err)
	}
	kind.Summe, err = cellAsCents(f, SheetVertrag, fmt.Sprintf("%s%d", vc.abrechnSumme, row))
	if err != nil {
		return nil, fmt.Errorf("parsing Summe at row %d: %w", row, err)
	}

	return kind, nil
}

// ParseKind parses a single child row from the Vertragsübersicht sheet.
// It discovers the column layout automatically.
func ParseKind(f *excelize.File, row int) (*Kind, error) {
	vc, err := discoverVertragColumns(f)
	if err != nil {
		return nil, fmt.Errorf("discovering column layout: %w", err)
	}
	return parseKindWithColumns(f, row, vc)
}

func ParseVertrag(f *excelize.File) (*Vertrag, error) {
	vc, err := discoverVertragColumns(f)
	if err != nil {
		// If the sheet is missing or has no recognizable headers, return empty vertrag
		return &Vertrag{Kinder: []Kind{}}, nil
	}

	vertrag := &Vertrag{
		Kinder: []Kind{},
	}

	row := vc.dataStartRow
	for {
		voucher := cellAsString(f, SheetVertrag, fmt.Sprintf("%s%d", vc.gutscheinNr, row))

		// Stop if empty or contains "Abrechnungssumme"
		if voucher == "" || strings.Contains(voucher, "Abrechnungssumme") {
			break
		}

		// Validate voucher pattern
		if !voucherPattern.MatchString(voucher) {
			break
		}

		kind, err := parseKindWithColumns(f, row, vc)
		if err != nil {
			return nil, fmt.Errorf("parsing kind at row %d: %w", row, err)
		}

		vertrag.Kinder = append(vertrag.Kinder, *kind)
		row++
	}

	return vertrag, nil
}

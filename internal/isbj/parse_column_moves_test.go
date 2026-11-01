package isbj

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Column-layout tests.
//
// The Senate moves columns. Among 21 sample bills there are three distinct
// Vertragsübersicht layouts: the older "e_" generation, the current one, and
// two months in between whose whole sheet sits two columns to the left. The
// Korrekturübersicht's amount column moved between F and C over the same
// period. So the importer finds every column by header *name*, and these tests
// pin both halves of that bargain:
//
//   - a column that MOVES must change nothing, at any offset;
//   - a header that cannot be found must fail the import, loudly and by name,
//     because the alternative is reading the wrong cell.
//
// The second half is not hypothetical. Before these checks existed, renaming
// "Basis-entgelt" on a real bill imported all 41 children with a base
// entitlement of 0,00 EUR and reported success; renaming "GutscheinNr."
// imported the file with zero children while the header still claimed a
// Vertragsbuchung of 58.033,99 EUR. An unresolved column is the empty string,
// cellAsString swallows excelize's invalid-reference error, and cellAsCents("")
// is a clean zero -- so nothing anywhere raised a hand.

// workbookSpec describes a synthetic bill. The three offsets shift each sheet's
// columns independently, which is what makes these tests about layout rather
// than about one hard-coded shape.
type workbookSpec struct {
	einrichtungOffset int
	abrechnungOffset  int
	vertragOffset     int
	// rename maps a header's exact text to what to write instead, simulating
	// the Senate relabelling it. Keyed per sheet.
	renameEinrichtung map[string]string
	renameAbrechnung  map[string]string
	renameVertrag     map[string]string
	// extraVertragRows are appended after the ordinary data row, used for the
	// hole-in-the-table and junk-cell cases.
	extraVertragRows []vertragRowSpec
	// omitTerminator leaves out the "Abrechnungssumme" row.
	omitTerminator bool
	// vertragsbuchungOverride replaces the header total with a figure that
	// disagrees with the rows.
	vertragsbuchungOverride *float64
}

type vertragRowSpec struct {
	voucher string // written verbatim; "" leaves the cell empty
	basis   float64
	essen   float64
	summe   float64
}

// The one ordinary contract row every spec gets. Arithmetically what the
// Senate's formula produces: Anteil Bezirk = 1000,00 - 23,00 = 977,00, and with
// no surcharges the Abrechn.-summe is the same.
var baseRow = vertragRowSpec{
	voucher: "GB-12345678901-01",
	basis:   1000.00,
	essen:   23.00,
	summe:   977.00,
}

func col(n int) string {
	name, err := excelize.ColumnNumberToName(n)
	if err != nil {
		panic(err)
	}
	return name
}

// buildWorkbook writes a bill the parser accepts, subject to spec.
func buildWorkbook(t *testing.T, spec workbookSpec) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	for _, s := range []string{SheetEinrichtung, SheetAbrechnung, SheetVertrag} {
		if _, err := f.NewSheet(s); err != nil {
			t.Fatalf("NewSheet(%s): %v", s, err)
		}
	}

	// set writes a cell, applying the sheet's rename map to header text.
	set := func(sheet string, c int, r int, v any, renames map[string]string) {
		t.Helper()
		if s, ok := v.(string); ok {
			if replacement, found := renames[s]; found {
				v = replacement
			}
		}
		ref := col(c) + strconv.Itoa(r)
		if err := f.SetCellValue(sheet, ref, v); err != nil {
			t.Fatalf("SetCellValue(%s!%s): %v", sheet, ref, err)
		}
	}

	// --- Einrichtungsübersicht -------------------------------------------
	// Trägernummer label, its value one column right, the billing month
	// directly below that value. Then the surcharge headers with their values
	// one row below, and "Abrechn." with its value two rows below.
	eo := spec.einrichtungOffset
	er := spec.renameEinrichtung
	set(SheetEinrichtung, 1+eo, 1, "Trägernummer", er)
	set(SheetEinrichtung, 2+eo, 1, "0770", er)
	set(SheetEinrichtung, 2+eo, 2, "09/26", er)
	set(SheetEinrichtung, 1+eo, 4, "Einrichtungsname", er)
	set(SheetEinrichtung, 1+eo, 6, "Kita Sonnenschein", er)
	set(SheetEinrichtung, 3+eo, 4, "QM", er)
	set(SheetEinrichtung, 3+eo, 5, 0.0, er)
	set(SheetEinrichtung, 4+eo, 4, "MSS", er)
	set(SheetEinrichtung, 4+eo, 5, 0.0, er)
	set(SheetEinrichtung, 5+eo, 4, "ndH", er)
	set(SheetEinrichtung, 5+eo, 5, 0.0, er)
	set(SheetEinrichtung, 6+eo, 4, "SpH", er)
	set(SheetEinrichtung, 6+eo, 5, 0.0, er)
	set(SheetEinrichtung, 7+eo, 4, "Abrechn.", er)
	set(SheetEinrichtung, 7+eo, 6, 977.00, er)

	// --- Abrechnungsübersicht --------------------------------------------
	// Headers on row 1, the "Summe:" row carrying the totals on row 3.
	ao := spec.abrechnungOffset
	ar := spec.renameAbrechnung
	vertragsbuchung := 977.00
	if spec.vertragsbuchungOverride != nil {
		vertragsbuchung = *spec.vertragsbuchungOverride
	}
	set(SheetAbrechnung, 1+ao, 1, "Vertragsbuchung", ar)
	set(SheetAbrechnung, 2+ao, 1, "Korrekturbuchung", ar)
	set(SheetAbrechnung, 1+ao, 3, vertragsbuchung, ar)
	set(SheetAbrechnung, 2+ao, 3, 0.0, ar)
	set(SheetAbrechnung, 4+ao, 3, "Summe:", ar)

	// --- Vertragsübersicht -----------------------------------------------
	// Header row 6, sub-header row 7, data from row 8 -- the shape of the real
	// sheet, including the two-row split headers.
	vo := spec.vertragOffset
	vr := spec.renameVertrag
	const hdr = 6
	cols := map[string]int{
		"Monat/ Typ":      1 + vo, // merged over two columns; typ is the next one
		"GutscheinNr.":    3 + vo,
		"Name des Kindes": 4 + vo,
		"Geb.-datum":      5 + vo,
		"BuT flag":        6 + vo,
		"QM":              7 + vo,
		"MSS":             8 + vo,
		"Hs":              9 + vo,
		"SpH":             10 + vo,
		"Registr. ab":     11 + vo,
		"Betreu.-umfang":  12 + vo,
		"Be-zirk":         13 + vo,
		"Basis-entgelt":   14 + vo,
		"Abzug o.M.":      15 + vo,
		"Anteil Eltern":   16 + vo, // sub: Betreuung here, Essen next
		"BuT":             18 + vo,
		"Anteil Bezirk":   19 + vo,
		"Zuschläge":       20 + vo, // sub: QM, MSS, ndH, SpH over four columns
		"Abrechn.-summe":  24 + vo,
	}
	set(SheetVertrag, cols["Monat/ Typ"], hdr, "Monat/ Typ", vr)
	set(SheetVertrag, cols["GutscheinNr."], hdr, "GutscheinNr.", vr)
	set(SheetVertrag, cols["Name des Kindes"], hdr, "Name des Kindes", vr)
	set(SheetVertrag, cols["Geb.-datum"], hdr, "Geb.-datum", vr)
	set(SheetVertrag, cols["BuT flag"], hdr, "BuT", vr)
	set(SheetVertrag, cols["QM"], hdr, "QM", vr)
	set(SheetVertrag, cols["MSS"], hdr, "MSS", vr)
	set(SheetVertrag, cols["Hs"], hdr, "Hs", vr)
	set(SheetVertrag, cols["SpH"], hdr, "SpH", vr)
	set(SheetVertrag, cols["Registr. ab"], hdr, "Registr. ab", vr)
	set(SheetVertrag, cols["Betreu.-umfang"], hdr, "Betreu.-umfang", vr)
	set(SheetVertrag, cols["Be-zirk"], hdr, "Be-zirk", vr)
	set(SheetVertrag, cols["Basis-entgelt"], hdr, "Basis-entgelt", vr)
	set(SheetVertrag, cols["Abzug o.M."], hdr, "Abzug o.M.", vr)
	set(SheetVertrag, cols["Anteil Eltern"], hdr, "Anteil Eltern", vr)
	set(SheetVertrag, cols["BuT"], hdr, "BuT", vr)
	set(SheetVertrag, cols["Anteil Bezirk"], hdr, "Anteil Bezirk", vr)
	set(SheetVertrag, cols["Zuschläge"], hdr, "Zuschläge", vr)
	set(SheetVertrag, cols["Abrechn.-summe"], hdr, "Abrechn.-summe", vr)
	// Sub-headers.
	set(SheetVertrag, cols["Anteil Eltern"], hdr+1, "Betreuung", vr)
	set(SheetVertrag, cols["Anteil Eltern"]+1, hdr+1, "Essen", vr)
	set(SheetVertrag, cols["Zuschläge"], hdr+1, "QM", vr)
	set(SheetVertrag, cols["Zuschläge"]+1, hdr+1, "MSS", vr)
	set(SheetVertrag, cols["Zuschläge"]+2, hdr+1, "ndH", vr)
	set(SheetVertrag, cols["Zuschläge"]+3, hdr+1, "SpH", vr)

	rows := append([]vertragRowSpec{baseRow}, spec.extraVertragRows...)
	r := hdr + 2
	for _, row := range rows {
		if row.voucher != "" {
			set(SheetVertrag, cols["GutscheinNr."], r, row.voucher, nil)
		}
		set(SheetVertrag, cols["Monat/ Typ"], r, "09.26", nil)
		set(SheetVertrag, cols["Monat/ Typ"]+1, r, "A", nil)
		set(SheetVertrag, cols["Name des Kindes"], r, "Mustermann, Max", nil)
		set(SheetVertrag, cols["Geb.-datum"], r, "01.20", nil)
		set(SheetVertrag, cols["Hs"], r, "D", nil)
		set(SheetVertrag, cols["SpH"], r, "N", nil)
		set(SheetVertrag, cols["Betreu.-umfang"], r, "ganztags", nil)
		set(SheetVertrag, cols["Be-zirk"], r, 1, nil)
		set(SheetVertrag, cols["Basis-entgelt"], r, row.basis, nil)
		set(SheetVertrag, cols["Anteil Eltern"]+1, r, row.essen, nil)
		set(SheetVertrag, cols["Anteil Bezirk"], r, row.summe, nil)
		set(SheetVertrag, cols["Abrechn.-summe"], r, row.summe, nil)
		r++
	}
	if !spec.omitTerminator {
		set(SheetVertrag, cols["GutscheinNr."], r, "Abrechnungssumme", nil)
	}
	return f
}

// parseWorkbook runs the full parse over a built workbook.
func parseWorkbook(t *testing.T, spec workbookSpec) (*SenatsabrechnungOutput, error) {
	t.Helper()
	f := buildWorkbook(t, spec)
	defer func() { _ = f.Close() }()
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	return ParseFromReader(buf)
}

// --- a column that moves must change nothing --------------------------------

// TestColumnsMayMove_AllSheets shifts each sheet's columns independently and
// requires the same figures out. This is the behaviour two of the real bills
// depend on: Abrechnung_07-25 and Abrechnung_08-25 carry the whole
// Vertragsübersicht two columns to the left of where every other bill has it.
func TestColumnsMayMove_AllSheets(t *testing.T) {
	for _, tc := range []struct {
		name            string
		einr, abr, vert int
	}{
		{"no offset", 0, 0, 0},
		{"Vertragsübersicht shifted 2 right", 0, 0, 2},
		{"Vertragsübersicht shifted 7 right", 0, 0, 7},
		{"Einrichtungsübersicht shifted 3", 3, 0, 0},
		{"Abrechnungsübersicht shifted 5", 0, 5, 0},
		{"every sheet shifted differently", 2, 4, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parseWorkbook(t, workbookSpec{
				einrichtungOffset: tc.einr,
				abrechnungOffset:  tc.abr,
				vertragOffset:     tc.vert,
			})
			if err != nil {
				t.Fatalf("a layout that only moved columns was rejected: %v", err)
			}
			if got := len(out.Vertrag.Kinder); got != 1 {
				t.Fatalf("children = %d, want 1", got)
			}
			k := out.Vertrag.Kinder[0]
			if k.Basisentgeld != 100000 {
				t.Errorf("Basisentgeld = %d, want 100000", k.Basisentgeld)
			}
			if k.ElternEssen != 2300 {
				t.Errorf("ElternEssen = %d, want 2300", k.ElternEssen)
			}
			if k.Summe != 97700 {
				t.Errorf("Summe = %d, want 97700", k.Summe)
			}
			if out.Abrechnung.VertragsBuchung != 97700 {
				t.Errorf("VertragsBuchung = %d, want 97700", out.Abrechnung.VertragsBuchung)
			}
			if out.Einrichtung.Summe != 97700 {
				t.Errorf("Einrichtung.Summe = %d, want 97700", out.Einrichtung.Summe)
			}
			if out.BillingMonth.Format("2006-01") != "2026-09" {
				t.Errorf("BillingMonth = %s, want 2026-09", out.BillingMonth.Format("2006-01"))
			}
		})
	}
}

// --- a header that cannot be found must fail the import ---------------------

// TestRenamedVertragHeaderIsRejected renames one Vertragsübersicht header at a
// time. Each must fail, and the message must name the header, because the whole
// point is telling an operator which cell to go and look at.
func TestRenamedVertragHeaderIsRejected(t *testing.T) {
	// Every header the row parser reads, with the text validate() reports it
	// under. "Monat/ Typ" is excluded deliberately: it is optional, since
	// older files predate it, and its absence is warned about rather than
	// rejected -- covered by TestMissingMonatTypColumnIsAcceptedWithWarning.
	for header, reported := range map[string]string{
		"Name des Kindes": "Name des Kindes",
		"Geb.-datum":      "Geb.-datum",
		"Registr. ab":     "Registr. ab",
		"Betreu.-umfang":  "Betreu.-umfang",
		"Be-zirk":         "Be-zirk",
		"Basis-entgelt":   "Basis-entgelt",
		"Abzug o.M.":      "Abzug o.M.",
		"Anteil Bezirk":   "Anteil Bezirk",
		"Abrechn.-summe":  "Abrechn.-summe",
	} {
		t.Run(header, func(t *testing.T) {
			_, err := parseWorkbook(t, workbookSpec{
				renameVertrag: map[string]string{header: "Umbenannt"},
			})
			if err == nil {
				t.Fatalf("renaming %q was accepted; the figure it holds would have been read as zero", header)
			}
			if !strings.Contains(err.Error(), reported) {
				t.Errorf("error does not name %q, so an operator cannot tell which column to look at: %v", reported, err)
			}
		})
	}
}

// TestRenamedVertragSubHeaderIsRejected covers the split headers under
// "Anteil Eltern" and "Zuschläge", which live on their own row and were read
// through a second map with no checks at all.
func TestRenamedVertragSubHeaderIsRejected(t *testing.T) {
	for _, tc := range []struct{ header, reported string }{
		{"Betreuung", "Anteil Eltern / Betreuung"},
		{"Essen", "Anteil Eltern / Essen"},
		{"ndH", "Zuschläge / ndH"},
	} {
		t.Run(tc.header, func(t *testing.T) {
			_, err := parseWorkbook(t, workbookSpec{
				renameVertrag: map[string]string{tc.header: "Umbenannt"},
			})
			if err == nil {
				t.Fatalf("renaming the %q sub-header was accepted", tc.header)
			}
			if !strings.Contains(err.Error(), tc.reported) {
				t.Errorf("error does not name %q: %v", tc.reported, err)
			}
		})
	}
}

// TestRenamedAnchorHeaderIsRejected covers the headers each sheet's discovery
// hangs on. These already failed before, through getValueByHeaderName, which
// also refuses a header it finds more than once -- an ambiguous match is as
// unusable as a missing one.
func TestRenamedAnchorHeaderIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec workbookSpec
		want string
	}{
		{
			"Vertragsübersicht GutscheinNr.",
			workbookSpec{renameVertrag: map[string]string{"GutscheinNr.": "Voucher-Nr."}},
			"GutscheinNr",
		},
		{
			"Abrechnungsübersicht Vertragsbuchung",
			workbookSpec{renameAbrechnung: map[string]string{"Vertragsbuchung": "Vertragssumme"}},
			"Vertragsbuchung",
		},
		{
			"Abrechnungsübersicht Korrekturbuchung",
			workbookSpec{renameAbrechnung: map[string]string{"Korrekturbuchung": "Korrektursumme"}},
			"Korrekturbuchung",
		},
		{
			"Abrechnungsübersicht Summe:",
			workbookSpec{renameAbrechnung: map[string]string{"Summe:": "Gesamt:"}},
			"Summe:",
		},
		{
			"Einrichtungsübersicht Einrichtungsname",
			workbookSpec{renameEinrichtung: map[string]string{"Einrichtungsname": "Name der Einrichtung"}},
			"Einrichtungsname",
		},
		{
			"Einrichtungsübersicht Abrechn.",
			workbookSpec{renameEinrichtung: map[string]string{"Abrechn.": "Gesamt"}},
			"Abrechn.",
		},
		{
			"Einrichtungsübersicht Trägernummer",
			workbookSpec{renameEinrichtung: map[string]string{"Trägernummer": "Tr.-Nr."}},
			"Trägernummer",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWorkbook(t, tc.spec)
			if err == nil {
				t.Fatal("a renamed anchor header was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error does not mention %q: %v", tc.want, err)
			}
		})
	}
}

// TestMissingMonatTypColumnIsAcceptedWithWarning pins the one column that is
// deliberately optional. It arrived with a later file generation; a bill
// without it still imports, and correction rows then fall back to the bill's
// own month in the aggregation SQL. Rejecting those files to guard against a
// rename would be the wrong trade, so it is warned about instead.
func TestMissingMonatTypColumnIsAcceptedWithWarning(t *testing.T) {
	out, err := parseWorkbook(t, workbookSpec{
		renameVertrag: map[string]string{"Monat/ Typ": "Zeitraum"},
	})
	if err != nil {
		t.Fatalf("a bill without the Monat/ Typ column must still import: %v", err)
	}
	if got := len(out.Vertrag.Kinder); got != 1 {
		t.Fatalf("children = %d, want 1", got)
	}
	if !out.Vertrag.Kinder[0].Abrechnungsmonat.IsZero() {
		t.Errorf("Abrechnungsmonat = %v, want the zero time so the month reads as unknown rather than guessed",
			out.Vertrag.Kinder[0].Abrechnungsmonat)
	}
}

// --- the table itself -------------------------------------------------------

// TestHoleInContractTableIsRejected covers a blank Gutschein cell with rows
// still below it. The loop used to stop at the blank and return what it had:
// on a real bill, blanking one cell imported 5 children of 41 and reported
// success.
func TestHoleInContractTableIsRejected(t *testing.T) {
	_, err := parseWorkbook(t, workbookSpec{
		extraVertragRows: []vertragRowSpec{
			{voucher: "", basis: 1000, essen: 23, summe: 977},
			{voucher: "GB-12345678901-02", basis: 1000, essen: 23, summe: 977},
		},
	})
	if err == nil {
		t.Fatal("a hole in the contract table was accepted, silently dropping the rows below it")
	}
	if !strings.Contains(err.Error(), "further contract row") {
		t.Errorf("error should say that rows follow the gap: %v", err)
	}
}

// TestJunkInVoucherColumnIsRejected covers content that is neither a Gutschein
// number nor the terminator -- a subtotal label, say. That also used to break
// the loop and discard everything below.
func TestJunkInVoucherColumnIsRejected(t *testing.T) {
	_, err := parseWorkbook(t, workbookSpec{
		extraVertragRows: []vertragRowSpec{
			{voucher: "Zwischensumme", basis: 0, essen: 0, summe: 0},
			{voucher: "GB-12345678901-02", basis: 1000, essen: 23, summe: 977},
		},
	})
	if err == nil {
		t.Fatal("junk in the GutscheinNr. column was accepted")
	}
	if !strings.Contains(err.Error(), "Zwischensumme") {
		t.Errorf("error should quote the offending cell: %v", err)
	}
}

// TestTableEndingWithoutTerminatorIsAccepted is the other side of the hole
// check: a blank cell with nothing below it really is the end of the table, and
// must not be mistaken for a gap.
func TestTableEndingWithoutTerminatorIsAccepted(t *testing.T) {
	out, err := parseWorkbook(t, workbookSpec{omitTerminator: true})
	if err != nil {
		t.Fatalf("a table that simply ends was rejected: %v", err)
	}
	if got := len(out.Vertrag.Kinder); got != 1 {
		t.Errorf("children = %d, want 1", got)
	}
}

// TestRowsDisagreeingWithHeaderTotalAreRejected pins the bill-level
// reconciliation. The Senate defines the Vertragsbuchung as the sum of the
// contract rows, and the two are read through entirely different code -- the
// header through the strict getValueByHeaderName, the rows through the
// forgiving column map -- so their disagreement is the broadest single signal
// that something was misread.
func TestRowsDisagreeingWithHeaderTotalAreRejected(t *testing.T) {
	wrong := 5000.00
	_, err := parseWorkbook(t, workbookSpec{vertragsbuchungOverride: &wrong})
	if err == nil {
		t.Fatal("rows totalling 977,00 EUR against a stated Vertragsbuchung of 5.000,00 EUR were accepted")
	}
	for _, want := range []string{"97700", "500000", "Vertragsbuchung"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should contain %q so the gap is readable: %v", want, err)
		}
	}
}

// --- the Korrekturübersicht stays non-fatal ---------------------------------

// TestKorrekturSheetDriftDoesNotFailTheImport pins the one sheet that is
// deliberately allowed to drift. It carries detail behind a figure the
// Abrechnungsübersicht already states correctly, its layout has already moved
// once, and a bill must not become un-importable over it. Absent or
// unreadable, it costs the month attribution of the manual correction booking
// and nothing else.
func TestKorrekturSheetDriftDoesNotFailTheImport(t *testing.T) {
	// No Korrekturübersicht sheet at all, which is the ordinary case: 18 of
	// the 21 sample bills carry no manual booking.
	out, err := parseWorkbook(t, workbookSpec{})
	if err != nil {
		t.Fatalf("a bill with no Korrekturübersicht must import: %v", err)
	}
	if out.Korrekturen != nil {
		t.Errorf("Korrekturen = %v, want nil when the sheet is absent", out.Korrekturen)
	}

	// And with the sheet present but its anchor header renamed.
	f := buildWorkbook(t, workbookSpec{})
	if _, err := f.NewSheet(SheetKorrektur); err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	if err := f.SetCellValue(SheetKorrektur, "B1", "Grund der Korrektur"); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	_ = f.Close()
	if _, err := ParseFromReader(buf); err != nil {
		t.Fatalf("a renamed Korrekturübersicht header must not fail the import: %v", err)
	}
}

// --- the per-row reconciliation --------------------------------------------

// TestRowComponentsMustSumToRowTotal is the check that catches what the
// bill-level one cannot. Kind.Summe is read from "Abrechn.-summe" rather than
// computed, so a renamed component column leaves the row total right while a
// figure inside it silently becomes zero -- and nothing persists the row total,
// so after import every figure is rebuilt from the components alone.
func TestRowComponentsMustSumToRowTotal(t *testing.T) {
	out, err := parseWorkbook(t, workbookSpec{})
	if err != nil {
		t.Fatalf("baseline parse: %v", err)
	}
	// Move money into a column the converter does not emit, leaving the row
	// total untouched. This is what a renamed component column looks like from
	// Convert's point of view.
	out.Vertrag.Kinder[0].Basisentgeld -= 5000
	if _, err := Convert(out); err == nil {
		t.Fatal("a row whose components no longer sum to its own total was accepted")
	} else if !strings.Contains(err.Error(), "Abrechn.-summe") {
		t.Errorf("error should point at the row total it disagrees with: %v", err)
	}
}

// TestBuTRowReconcilesToTheMealShare pins the one residual the importer is
// allowed to explain. Where a child holds a berlinpass-BuT the Senate waives
// the 23,00 EUR parent meal share and pays a subsidy instead, and it nets the
// waived share into the row total without ever showing it in the Essen column.
// All 13 BuT rows in the sample bills have exactly this shape.
func TestBuTRowReconcilesToTheMealShare(t *testing.T) {
	out, err := parseWorkbook(t, workbookSpec{})
	if err != nil {
		t.Fatalf("baseline parse: %v", err)
	}
	k := &out.Vertrag.Kinder[0]
	// A BuT row as the Senate writes one: the share is not in the Essen
	// column, the subsidy is, and the total already nets both.
	k.ElternEssen = 0
	k.BuT = 2365
	k.Summe = k.Basisentgeld + k.BuT - 2300
	k.AnteilBezirk = k.Summe
	out.Abrechnung.VertragsBuchung = k.Summe

	conv, err := Convert(out)
	if err != nil {
		t.Fatalf("a BuT row was rejected: %v", err)
	}
	row := conv.Children[0].Rows[0]
	sum := 0
	var meals, but int
	for _, a := range row.Amounts {
		sum += a.Amount
		if a.Key == "parent" && a.Value == "meals" {
			meals = a.Amount
		}
		if a.Key == "but" {
			but = a.Amount
		}
	}
	if sum != row.TotalRowAmount {
		t.Errorf("amounts sum to %d, row total is %d", sum, row.TotalRowAmount)
	}
	if meals != -2300 {
		t.Errorf("parent:meals = %d, want -2300: the waived share belongs on the key that holds it, "+
			"so the bill lines up with the parent:meals entitlement the funding config applies", meals)
	}
	if but != 2365 {
		t.Errorf("but = %d, want 2365", but)
	}
}

// TestUnexplainedResidualIsRejectedEvenWithBuT guards the narrow door the BuT
// case opens. A residual is only attributed to the meal share when it mirrors
// the subsidy's sign and stays within a meal-share-sized bound; a structural
// misread is neither.
func TestUnexplainedResidualIsRejectedEvenWithBuT(t *testing.T) {
	for _, tc := range []struct {
		name          string
		but, residual int
	}{
		{"residual far too large to be a meal share", 2365, -50000},
		{"residual has the same sign as the subsidy", 2365, 2300},
		{"subsidy clawed back but share also deducted", -2365, -2300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parseWorkbook(t, workbookSpec{})
			if err != nil {
				t.Fatalf("baseline parse: %v", err)
			}
			k := &out.Vertrag.Kinder[0]
			k.ElternEssen = 0
			k.BuT = tc.but
			k.Summe = k.Basisentgeld + k.BuT + tc.residual
			out.Abrechnung.VertragsBuchung = k.Summe
			if _, err := Convert(out); err == nil {
				t.Fatalf("residual %d with BuT %d was accepted as a meal share", tc.residual, tc.but)
			}
		})
	}
}

// TestValidateNamesEveryMissingColumnAtOnce: an operator fixing a layout change
// should see the whole list, not discover them one upload at a time.
func TestValidateNamesEveryMissingColumnAtOnce(t *testing.T) {
	_, err := parseWorkbook(t, workbookSpec{
		renameVertrag: map[string]string{
			"Basis-entgelt":  "Grundentgelt",
			"Abzug o.M.":     "Abzug",
			"Abrechn.-summe": "Gesamt",
		},
	})
	if err == nil {
		t.Fatal("three renamed headers were accepted")
	}
	for _, want := range []string{"Basis-entgelt", "Abzug o.M.", "Abrechn.-summe"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error omits %q; all missing columns should be reported together: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d of", 3)) {
		t.Errorf("error should count the missing columns: %v", err)
	}
}

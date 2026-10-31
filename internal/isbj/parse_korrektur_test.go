package isbj

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// korrRow is one row to write into a synthetic Korrekturübersicht.
type korrRow struct {
	grund  string
	betrag string
}

// newKorrekturFile builds a Korrekturübersicht in the shape a real one has.
// betragCol is given explicitly because it moved between file generations --
// "F" in the current layout, "C" in the older "e_" one -- and the parser is
// supposed to find it either way.
func newKorrekturFile(t *testing.T, headerRow int, betragCol string, rows []korrRow, korrektursumme string) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if _, err := f.NewSheet(SheetKorrektur); err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	set := func(cell string, v any) {
		t.Helper()
		if err := f.SetCellValue(SheetKorrektur, cell, v); err != nil {
			t.Fatalf("SetCellValue(%s): %v", cell, err)
		}
	}
	set("A"+strconv.Itoa(headerRow), "Bezirk")
	set("B"+strconv.Itoa(headerRow), "Korrekturgrund")
	set(betragCol+strconv.Itoa(headerRow), "Betrag / € (Vorschuss bzw. Erstattung)")
	r := headerRow + 1
	for _, row := range rows {
		set("A"+strconv.Itoa(r), "11 Lichtenberg")
		set("B"+strconv.Itoa(r), row.grund)
		set(betragCol+strconv.Itoa(r), row.betrag)
		r++
	}
	if korrektursumme != "" {
		// The label sits in a column of its own, and which one differs between
		// generations: column C in the current layout where the amount is in
		// F, column B in the older one where the amount is in C. Never the
		// amount column itself. The parser searches the whole row for it, so
		// B serves for both here.
		set("B"+strconv.Itoa(r), "Korrektursumme")
		set(betragCol+strconv.Itoa(r), korrektursumme)
	}
	return f
}

func mustMonth(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01", s)
	if err != nil {
		t.Fatalf("bad month %q: %v", s, err)
	}
	return v
}

func TestParseKorrektur_MissingSheet(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if posten != nil {
		t.Errorf("got %d rows, want nil for a workbook with no Korrekturübersicht", len(posten))
	}
}

// The ordinary case: the sheet exists, carries its header and a Korrektursumme
// of zero, and no rows. Nineteen of twenty-two real bills look like this.
func TestParseKorrektur_NoDataRows(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", nil, "0,00")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if posten != nil {
		t.Errorf("got %d rows, want nil", len(posten))
	}
}

func TestParseKorrektur_SingleMonth(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung des I-Status 12/24", betrag: "3.124,40"},
	}, "3.124,40")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 1 {
		t.Fatalf("got %d rows, want 1", len(posten))
	}
	p := posten[0]
	if p.Summe != 312440 {
		t.Errorf("summe = %d, want 312440", p.Summe)
	}
	if want := mustMonth(t, "2024-12"); !p.FromMonth.Equal(want) || !p.ToMonth.Equal(want) {
		t.Errorf("span = %s..%s, want %s..%s",
			p.FromMonth.Format("2006-01"), p.ToMonth.Format("2006-01"),
			want.Format("2006-01"), want.Format("2006-01"))
	}
	if p.Reason != KorrekturReasonNachfinanzierungIStatus {
		t.Errorf("reason = %q, want %q", p.Reason, KorrekturReasonNachfinanzierungIStatus)
	}
}

func TestParseKorrektur_MonthRange(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 02/24-09/24", betrag: "11.984,88"},
	}, "11.984,88")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 1 {
		t.Fatalf("got %d rows, want 1", len(posten))
	}
	if got, want := posten[0].FromMonth, mustMonth(t, "2024-02"); !got.Equal(want) {
		t.Errorf("from = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
	if got, want := posten[0].ToMonth, mustMonth(t, "2024-09"); !got.Equal(want) {
		t.Errorf("to = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
}

// The older "e_" generation puts the amount in column C. No file in the sample
// set carries both that layout AND data rows, which is exactly why the column
// is discovered from the header rather than assumed.
func TestParseKorrektur_OlderLayoutAmountInColumnC(t *testing.T) {
	f := newKorrekturFile(t, 6, "C", []korrRow{
		{grund: "Nachfinanzierung I-Status 01/24", betrag: "1.948,11"},
	}, "1.948,11")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 1 {
		t.Fatalf("got %d rows, want 1", len(posten))
	}
	if posten[0].Summe != 194811 {
		t.Errorf("summe = %d, want 194811", posten[0].Summe)
	}
}

// A birthdate in the free text must not be read as a month. This is the whole
// reason the pattern is slash-only: a dotted pattern turns "geb. 11.19.." into
// November 2019 and attributes the money to a month nobody named.
func TestParseKorrektur_DottedDateIsNotAMonth(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 01/24, Kind geb. 11.19.2019", betrag: "100,00"},
	}, "100,00")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 1 {
		t.Fatalf("got %d rows, want 1", len(posten))
	}
	jan := mustMonth(t, "2024-01")
	if !posten[0].FromMonth.Equal(jan) || !posten[0].ToMonth.Equal(jan) {
		t.Errorf("span = %s..%s, want 2024-01 only — the dotted date is not a month",
			posten[0].FromMonth.Format("2006-01"), posten[0].ToMonth.Format("2006-01"))
	}
}

// Nothing derived from the Korrekturgrund may carry the text itself: it holds
// children's names and birthdates, which is the highest protection level there
// is. The parsed row must be inspectable without exposing any of it.
func TestParseKorrektur_KeepsNoFreeText(t *testing.T) {
	const name = "Mustermann"
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 01/24 " + name + ", Max geb. 01.01.2020", betrag: "100,00"},
	}, "100,00")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 1 {
		t.Fatalf("got %d rows, want 1", len(posten))
	}
	p := posten[0]
	for field, v := range map[string]string{"reason": p.Reason} {
		if strings.Contains(v, name) || strings.Contains(v, "geb.") {
			t.Errorf("%s = %q carries the Korrekturgrund text", field, v)
		}
	}
	if p.Reason != KorrekturReasonNachfinanzierungIStatus {
		t.Errorf("reason = %q, want the classified code", p.Reason)
	}
}

// Rows that do not add up to the stated Korrektursumme are refused outright.
// The caller keeps the aggregate, which is never wrong; half-parsed rows would
// attribute real money to months at random.
func TestParseKorrektur_RefusesRowsThatDoNotReconcile(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 01/24", betrag: "100,00"},
		{grund: "Nachfinanzierung I-Status 02/24", betrag: "100,00"},
	}, "999,00")
	defer func() { _ = f.Close() }()

	if _, err := ParseKorrektur(f); err == nil {
		t.Fatal("ParseKorrektur() returned no error for rows that do not match the Korrektursumme")
	}
}

func TestParseKorrektur_RefusesRowsWithNoKorrektursumme(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 01/24", betrag: "100,00"},
	}, "")
	defer func() { _ = f.Close() }()

	if _, err := ParseKorrektur(f); err == nil {
		t.Fatal("ParseKorrektur() returned no error for rows with nothing to check them against")
	}
}

// The shape of the real November bill: three rows, spanning two Kita years
// before the one the bill arrived in.
func TestSummarizeKorrektur_SpansEveryRow(t *testing.T) {
	f := newKorrekturFile(t, 4, "F", []korrRow{
		{grund: "Nachfinanzierung I-Status 10/23-12/23", betrag: "4.297,86"},
		{grund: "Nachfinanzierung I-Status 02/24-09/24", betrag: "11.984,88"},
		{grund: "Nachfinanzierung I-Status 01/24", betrag: "1.948,11"},
	}, "18.230,85")
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 3 {
		t.Fatalf("got %d rows, want 3", len(posten))
	}

	s := SummarizeKorrektur(posten)
	if got, want := s.FromMonth, mustMonth(t, "2023-10"); !got.Equal(want) {
		t.Errorf("from = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
	if got, want := s.ToMonth, mustMonth(t, "2024-09"); !got.Equal(want) {
		t.Errorf("to = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
	if s.Reason != KorrekturReasonNachfinanzierungIStatus {
		t.Errorf("reason = %q, want %q", s.Reason, KorrekturReasonNachfinanzierungIStatus)
	}
}

func TestSummarizeKorrektur_DisagreeingReasonsAreMixed(t *testing.T) {
	s := SummarizeKorrektur([]Korrekturposten{
		{Reason: KorrekturReasonNachfinanzierung, FromMonth: mustMonth(t, "2024-01"), ToMonth: mustMonth(t, "2024-01")},
		{Reason: KorrekturReasonVorschuss, FromMonth: mustMonth(t, "2024-03"), ToMonth: mustMonth(t, "2024-03")},
	})
	if s.Reason != KorrekturReasonMixed {
		t.Errorf("reason = %q, want %q", s.Reason, KorrekturReasonMixed)
	}
}

// A row nobody can place must not widen the span to cover it.
func TestSummarizeKorrektur_IgnoresUnattributedRowsForTheSpan(t *testing.T) {
	s := SummarizeKorrektur([]Korrekturposten{
		{Reason: KorrekturReasonVorschuss},
		{Reason: KorrekturReasonVorschuss, FromMonth: mustMonth(t, "2024-05"), ToMonth: mustMonth(t, "2024-05")},
	})
	if got, want := s.FromMonth, mustMonth(t, "2024-05"); !got.Equal(want) {
		t.Errorf("from = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
	if got, want := s.ToMonth, mustMonth(t, "2024-05"); !got.Equal(want) {
		t.Errorf("to = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
}

// The committed anonymized November bill, which carries three real
// Korrekturübersicht rows. The strongest check available: the layout, the
// German wording, the number formatting and the reconciliation all come from a
// file the Senate produced rather than from a fixture built to pass.
//
// The span runs to September 2024 while the bill itself arrived in November
// 2025 — two Kita years later. That is the whole reason this sheet is parsed:
// the amount alone invites crediting 18.230,85 EUR to the year it was paid in.
func TestParseKorrektur_AnonymizedRealBill(t *testing.T) {
	f, err := excelize.OpenFile(testFile)
	if err != nil {
		t.Fatalf("opening %s: %v", testFile, err)
	}
	defer func() { _ = f.Close() }()

	posten, err := ParseKorrektur(f)
	if err != nil {
		t.Fatalf("ParseKorrektur() error = %v", err)
	}
	if len(posten) != 3 {
		t.Fatalf("got %d rows, want 3", len(posten))
	}

	sum := 0
	for _, p := range posten {
		sum += p.Summe
		if p.Reason != KorrekturReasonNachfinanzierungIStatus {
			t.Errorf("reason = %q, want %q", p.Reason, KorrekturReasonNachfinanzierungIStatus)
		}
		if p.FromMonth.IsZero() {
			t.Error("a row named no month; every row in this bill does")
		}
	}

	// Reconciles with the Abrechnungsübersicht, not just with the sheet's own
	// Korrektursumme row.
	ar, err := ParseAbrechnung(f)
	if err != nil {
		t.Fatalf("ParseAbrechnung() error = %v", err)
	}
	if sum != ar.KorrekturBuchung {
		t.Errorf("rows total %d but Korrekturbuchung is %d", sum, ar.KorrekturBuchung)
	}
	if sum != 1823085 {
		t.Errorf("rows total %d, want 1823085", sum)
	}

	s := SummarizeKorrektur(posten)
	if got, want := s.FromMonth, mustMonth(t, "2023-10"); !got.Equal(want) {
		t.Errorf("from = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}
	if got, want := s.ToMonth, mustMonth(t, "2024-09"); !got.Equal(want) {
		t.Errorf("to = %s, want %s", got.Format("2006-01"), want.Format("2006-01"))
	}

	// The bill's own month is far outside the span it settles.
	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		t.Fatalf("ParseBillingMonth() error = %v", err)
	}
	if !s.ToMonth.Before(billingMonth) {
		t.Errorf("span ends %s, bill is %s: the span must precede the bill that paid it",
			s.ToMonth.Format("2006-01"), billingMonth.Format("2006-01"))
	}
}

package isbj

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// SheetKorrektur holds the per-row detail behind the Abrechnungsübersicht's
// single "Korrekturbuchung manuell" figure: which months the Senate settled by
// hand, and why.
const SheetKorrektur = "Korrekturübersicht"

// Korrekturposten is one row of the Korrekturübersicht.
//
// The sheet's Korrekturgrund column is free German text, and it carries
// children's names and birthdates. None of it survives parsing: only the
// months, the amount, and a classified reason. That is also why Reason is a
// code from a fixed set rather than the text it was derived from -- there must
// be no path by which the original wording reaches a database, a log line or an
// API response.
type Korrekturposten struct {
	// Reason classifies the Korrekturgrund, or is empty when it matched none
	// of the known phrasings. Empty is information, not failure: the amount
	// and the months are still right.
	Reason string `json:"reason"`
	// FromMonth and ToMonth bound the months this row settles, inclusive, as
	// first-of-month dates. Equal for a single month. Both zero when the text
	// named no month in MM/YY form, which leaves the row unattributed rather
	// than attributed to a guess.
	FromMonth time.Time `json:"from_month"`
	ToMonth   time.Time `json:"to_month"`
	Summe     int       `json:"summe"`
}

// Reason codes. Deliberately few: the point is to say what kind of booking it
// was without keeping the sentence it came from.
const (
	KorrekturReasonNachfinanzierungIStatus = "nachfinanzierung_i_status"
	KorrekturReasonNachfinanzierung        = "nachfinanzierung"
	KorrekturReasonVorschuss               = "vorschuss"
	KorrekturReasonErstattung              = "erstattung"
	// KorrekturReasonMixed is used by KorrekturSummary when a bill's rows do
	// not agree on one reason.
	KorrekturReasonMixed = "mixed"
)

// korrekturMonthPattern matches a month in the slash form the Korrekturgrund
// writes them in: "Nachfinanzierung I-Status 10/23-12/23".
//
// Slash only, deliberately. The same text carries birthdates, and a dotted
// pattern reads "geb. 11.19.." as November 2019 -- which is how an earlier pass
// over these files produced month tokens that were really pieces of a date.
// Every month token in the sample bills uses a slash; every dotted token in
// them belonged to a date.
var korrekturMonthPattern = regexp.MustCompile(`\b(0[1-9]|1[0-2])/(\d{2})\b`)

// korrekturReasons maps a phrase in the Korrekturgrund to the code stored for
// it. Checked in order, first match wins, so the more specific phrasing comes
// first.
var korrekturReasons = []struct {
	match  *regexp.Regexp
	reason string
}{
	{regexp.MustCompile(`(?i)nachfinanzierung.*i-?status`), KorrekturReasonNachfinanzierungIStatus},
	{regexp.MustCompile(`(?i)nachfinanzierung`), KorrekturReasonNachfinanzierung},
	{regexp.MustCompile(`(?i)vorschuss`), KorrekturReasonVorschuss},
	{regexp.MustCompile(`(?i)erstattung|rückforderung`), KorrekturReasonErstattung},
}

func classifyKorrekturgrund(text string) string {
	for _, r := range korrekturReasons {
		if r.match.MatchString(text) {
			return r.reason
		}
	}
	return ""
}

// ParseKorrektur reads the Korrekturübersicht sheet.
//
// It returns nil, nil when the workbook has no such sheet, when the sheet has
// no header, or when it has no data rows -- which is the ordinary case. Across
// a full Kita year of real Abrechnungen only three of twenty-two bills carried
// any rows at all.
//
// The sheet's own Korrektursumme row states the total. ParseKorrektur returns
// the rows only when they add up to it, and errors when they do not: a
// half-parsed table would attribute real money to months at random, and the
// caller can always fall back to the aggregate figure in the
// Abrechnungsübersicht, which is never wrong.
func ParseKorrektur(f *excelize.File) ([]Korrekturposten, error) {
	if idx, err := f.GetSheetIndex(SheetKorrektur); err != nil || idx < 0 {
		return nil, nil
	}

	// The header is located by its one unambiguous label, and the amount
	// column is read from that row rather than fixed, because it moved between
	// file generations: column F in the current layout, C in the older "e_"
	// one.
	grundCol, headerRow, err := getValueByHeaderName(f, SheetKorrektur, "Korrekturgrund")
	if err != nil {
		// No header means no per-row detail to read, which is not a broken
		// file: the aggregate in the Abrechnungsübersicht still stands.
		return nil, nil
	}

	rows, err := f.GetRows(SheetKorrektur)
	if err != nil {
		return nil, fmt.Errorf("reading sheet %q: %w", SheetKorrektur, err)
	}
	if headerRow-1 >= len(rows) {
		return nil, nil
	}
	betragCol := ""
	for j, cell := range rows[headerRow-1] {
		if strings.Contains(normalizeHeaderCell(cell), "Betrag") {
			betragCol, _ = excelize.ColumnNumberToName(j + 1)
			break
		}
	}
	if betragCol == "" {
		return nil, nil
	}

	var posten []Korrekturposten
	sum := 0
	for r := headerRow + 1; r <= len(rows); r++ {
		// The Korrektursumme row closes the table. Its label sits in a column
		// that moved as well, so the whole row is searched for it.
		isTotal := false
		for _, cell := range rows[r-1] {
			if strings.Contains(cell, "Korrektursumme") {
				isTotal = true
				break
			}
		}
		amount, err := cellAsCents(f, SheetKorrektur, fmt.Sprintf("%s%d", betragCol, r))
		if err != nil {
			return nil, fmt.Errorf("sheet %q row %d: %w", SheetKorrektur, r, err)
		}
		if isTotal {
			if len(posten) == 0 {
				return nil, nil
			}
			if sum != amount {
				return nil, fmt.Errorf(
					"sheet %q: rows total %d cents but Korrektursumme states %d",
					SheetKorrektur, sum, amount)
			}
			return posten, nil
		}
		grund := cellAsString(f, SheetKorrektur, fmt.Sprintf("%s%d", grundCol, r))
		if grund == "" && amount == 0 {
			continue
		}
		from, to := korrekturMonthRange(grund)
		posten = append(posten, Korrekturposten{
			Reason:    classifyKorrekturgrund(grund),
			FromMonth: from,
			ToMonth:   to,
			Summe:     amount,
		})
		sum += amount
	}

	// Ran off the end without meeting a Korrektursumme row. Nothing checked
	// the rows, so nothing is returned.
	if len(posten) > 0 {
		return nil, fmt.Errorf("sheet %q: %d rows but no Korrektursumme to check them against",
			SheetKorrektur, len(posten))
	}
	return nil, nil
}

// korrekturMonthRange pulls the corrected month span out of a Korrekturgrund.
// One token means a single month; two or more are taken as the earliest and
// latest, so a written range and a list of months behave alike. Zero tokens
// leave both zero.
func korrekturMonthRange(text string) (time.Time, time.Time) {
	var from, to time.Time
	for _, m := range korrekturMonthPattern.FindAllStringSubmatch(text, -1) {
		month, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		year, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		// Two-digit years, as the sheet writes them. The ISBJ procedure has no
		// twentieth-century months and these settle the recent past, so 2000+
		// is the only reading that is ever right.
		t := time.Date(2000+year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		if from.IsZero() || t.Before(from) {
			from = t
		}
		if to.IsZero() || t.After(to) {
			to = t
		}
	}
	return from, to
}

// KorrekturSummary reduces a bill's Korrekturübersicht rows to what a reader
// acts on: the span of months the manual booking settles, and why.
//
// A span rather than the rows themselves, because nobody decides anything per
// row. The one thing the figure needs is context -- a Korrekturbuchung that
// arrived in November 2025 and settles months in 2023/24 is real money that
// must not be credited to 2025/26, and the span is what says so.
type KorrekturSummary struct {
	FromMonth time.Time
	ToMonth   time.Time
	Reason    string
}

// SummarizeKorrektur folds the rows into one span. Rows that named no month are
// skipped for the span but still counted for the reason: the amount is known,
// the months are not, and pretending otherwise would widen the span to cover a
// row nobody can place.
func SummarizeKorrektur(posten []Korrekturposten) KorrekturSummary {
	var out KorrekturSummary
	reasons := make(map[string]bool)
	for _, p := range posten {
		if p.Reason != "" {
			reasons[p.Reason] = true
		}
		if p.FromMonth.IsZero() {
			continue
		}
		if out.FromMonth.IsZero() || p.FromMonth.Before(out.FromMonth) {
			out.FromMonth = p.FromMonth
		}
		if out.ToMonth.IsZero() || p.ToMonth.After(out.ToMonth) {
			out.ToMonth = p.ToMonth
		}
	}
	switch len(reasons) {
	case 0:
	case 1:
		for r := range reasons {
			out.Reason = r
		}
	default:
		out.Reason = KorrekturReasonMixed
	}
	return out
}

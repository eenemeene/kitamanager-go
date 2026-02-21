package isbj

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

const testFile = "testdata/Abrechnung_11-25_0770_anonymized.xlsx"

func TestOpenSenatsabrechnung(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	if f == nil {
		t.Fatal("OpenSenatsabrechnung() returned nil file")
	}
}

func TestOpenSenatsabrechnungNonExistent(t *testing.T) {
	_, err := OpenSenatsabrechnung("nonexistent.xlsx")
	if err == nil {
		t.Error("OpenSenatsabrechnung() expected error for non-existent file")
	}
}

func TestParseEinrichtung(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	eu, err := ParseEinrichtung(f)
	if err != nil {
		t.Fatalf("ParseEinrichtung() error = %v", err)
	}

	if eu == nil {
		t.Fatal("ParseEinrichtung() returned nil")
	}

	// Check name
	if eu.Name != "Kindertagesstätte Sternenstaub" {
		t.Errorf("ParseEinrichtung() Name = %q, expected %q", eu.Name, "Kindertagesstätte Sternenstaub")
	}

	// Check summe is positive
	if eu.Summe <= 0 {
		t.Errorf("ParseEinrichtung() Summe = %d, expected positive value", eu.Summe)
	}

	// Check integration surcharge (1656.80 EUR = 165680 cents)
	// The Excel header says "SpH" but this is actually the integration surcharge.
	expectedIntegration := 165680
	if eu.ZuschlagIntegration != expectedIntegration {
		t.Errorf("ParseEinrichtung() ZuschlagIntegration = %d, expected %d", eu.ZuschlagIntegration, expectedIntegration)
	}

	// Check Summe (49406.53 EUR = 4940653 cents)
	expectedSumme := 4940653
	if eu.Summe != expectedSumme {
		t.Errorf("ParseEinrichtung() Summe = %d, expected %d", eu.Summe, expectedSumme)
	}
}

func TestParseAbrechnung(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	ar, err := ParseAbrechnung(f)
	if err != nil {
		t.Fatalf("ParseAbrechnung() error = %v", err)
	}

	if ar == nil {
		t.Fatal("ParseAbrechnung() returned nil")
	}

	// VertragsBuchung: 49406.53 EUR = 4940653 cents
	if ar.VertragsBuchung != 4940653 {
		t.Errorf("ParseAbrechnung() VertragsBuchung = %d, expected %d", ar.VertragsBuchung, 4940653)
	}

	// KorrekturBuchung: 18230.85 EUR = 1823085 cents
	if ar.KorrekturBuchung != 1823085 {
		t.Errorf("ParseAbrechnung() KorrekturBuchung = %d, expected %d", ar.KorrekturBuchung, 1823085)
	}
}

func TestParseVertrag(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	if v == nil {
		t.Fatal("ParseVertrag() returned nil")
	}

	// Check we have exactly 40 children
	expectedCount := 40
	if len(v.Kinder) != expectedCount {
		t.Errorf("ParseVertrag() got %d children, expected %d", len(v.Kinder), expectedCount)
	}
}

func TestParseVertragKinderDetails(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	if len(v.Kinder) == 0 {
		t.Fatal("ParseVertrag() returned no children")
	}

	// Check first child has required fields
	first := v.Kinder[0]

	if first.Gutscheinnummer == "" {
		t.Error("First child has empty Gutscheinnummer")
	}

	if !voucherPattern.MatchString(first.Gutscheinnummer) {
		t.Errorf("First child Gutscheinnummer %q doesn't match pattern GB-XXXXXXXXXXX-XX", first.Gutscheinnummer)
	}

	if first.Name == "" {
		t.Error("First child has empty Name")
	}

	if first.Geburtsdatum == "" {
		t.Error("First child has empty Geburtsdatum")
	}

	if first.Betreuungsumfang == "" {
		t.Error("First child has empty Betreuungsumfang")
	}

	if first.Bezirk == 0 {
		t.Error("First child has zero Bezirk")
	}

	// Check QM/MSS are valid values
	validQMMSS := map[string]bool{"ja": true, "nein": true, "": true}
	if !validQMMSS[first.QM] {
		t.Errorf("First child QM = %q, expected 'ja', 'nein', or empty", first.QM)
	}
	if !validQMMSS[first.MSS] {
		t.Errorf("First child MSS = %q, expected 'ja', 'nein', or empty", first.MSS)
	}

	// Check Betreuungsumfang is valid
	validScope := map[string]bool{"teilzeit": true, "ganztags": true, "erweitert": true}
	if !validScope[first.Betreuungsumfang] {
		t.Errorf("First child Betreuungsumfang = %q, expected 'teilzeit', 'ganztags', or 'erweitert'", first.Betreuungsumfang)
	}

	// Check first child Summe (899.87 EUR = 89987 cents)
	if first.Summe != 89987 {
		t.Errorf("First child Summe = %d, expected %d", first.Summe, 89987)
	}
}

func TestParseVertragAllKinderHaveVoucher(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	for i, kind := range v.Kinder {
		if !voucherPattern.MatchString(kind.Gutscheinnummer) {
			t.Errorf("Child %d: Gutscheinnummer %q doesn't match pattern", i, kind.Gutscheinnummer)
		}
	}
}

func TestParseVertragAllKinderHavePositiveSumme(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	for i, kind := range v.Kinder {
		if kind.Summe <= 0 {
			t.Errorf("Child %d (%s): Summe = %d, expected positive value", i, kind.Name, kind.Summe)
		}
	}
}

func TestParseKind(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Row 9 should be the first child
	kind, err := ParseKind(f, 9)
	if err != nil {
		t.Fatalf("ParseKind() error = %v", err)
	}

	if kind == nil {
		t.Fatal("ParseKind() returned nil")
	}

	if kind.Gutscheinnummer == "" {
		t.Error("ParseKind() Gutscheinnummer is empty")
	}

	if kind.Name == "" {
		t.Error("ParseKind() Name is empty")
	}
}

func TestEinrichtungString(t *testing.T) {
	eu := Einrichtung{
		Name:       "Test Kita",
		Summe:      100050, // 1000.50 EUR
		ZuschlagQM: 10000,  // 100.00 EUR
	}

	s := eu.String()

	if s == "" {
		t.Error("Einrichtung.String() returned empty string")
	}

	if !strings.Contains(s, "Test Kita") {
		t.Errorf("Einrichtung.String() = %q, expected to contain 'Test Kita'", s)
	}

	if !strings.Contains(s, "1000.50") {
		t.Errorf("Einrichtung.String() = %q, expected to contain '1000.50'", s)
	}
}

func TestAbrechnungString(t *testing.T) {
	ar := Abrechnung{
		VertragsBuchung:  500000, // 5000.00 EUR
		KorrekturBuchung: 100000, // 1000.00 EUR
	}

	s := ar.String()

	if s == "" {
		t.Error("Abrechnung.String() returned empty string")
	}

	if !strings.Contains(s, "5000.00") {
		t.Errorf("Abrechnung.String() = %q, expected to contain '5000.00'", s)
	}
	if !strings.Contains(s, "1000.00") {
		t.Errorf("Abrechnung.String() = %q, expected to contain '1000.00'", s)
	}
}

func TestKindString(t *testing.T) {
	kind := Kind{
		Gutscheinnummer: "GB-12345678901-02",
		Name:            "Löwe, Tiger",
		Geburtsdatum:    "05.20",
		Summe:           89987, // 899.87 EUR
	}

	s := kind.String()

	if s == "" {
		t.Error("Kind.String() returned empty string")
	}

	if !strings.Contains(s, "Löwe, Tiger") {
		t.Errorf("Kind.String() = %q, expected to contain 'Löwe, Tiger'", s)
	}

	if !strings.Contains(s, "GB-12345678901-02") {
		t.Errorf("Kind.String() = %q, expected to contain voucher number", s)
	}

	if !strings.Contains(s, "05.20") {
		t.Errorf("Kind.String() = %q, expected to contain birth date", s)
	}

	if !strings.Contains(s, "899.87") {
		t.Errorf("Kind.String() = %q, expected to contain '899.87'", s)
	}
}

func TestVertragString(t *testing.T) {
	v := Vertrag{
		Kinder: []Kind{
			{Name: "Kind 1", Gutscheinnummer: "GB-11111111111-01", Geburtsdatum: "01.20", Summe: 10000},
			{Name: "Kind 2", Gutscheinnummer: "GB-22222222222-02", Geburtsdatum: "02.21", Summe: 20000},
		},
	}

	s := v.String()

	if s == "" {
		t.Error("Vertrag.String() returned empty string")
	}

	if !strings.Contains(s, "2 Kindern") {
		t.Errorf("Vertrag.String() = %q, expected to contain '2 Kindern'", s)
	}

	if !strings.Contains(s, "Kind 1") {
		t.Errorf("Vertrag.String() = %q, expected to contain 'Kind 1'", s)
	}
	if !strings.Contains(s, "Kind 2") {
		t.Errorf("Vertrag.String() = %q, expected to contain 'Kind 2'", s)
	}
}

func TestCellAsString(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Test reading a known cell (E9 should be a voucher number)
	val := cellAsString(f, SheetVertrag, "E9")
	if val == "" {
		t.Error("cellAsString() returned empty for E9")
	}

	if !voucherPattern.MatchString(val) {
		t.Errorf("cellAsString() E9 = %q, expected voucher pattern", val)
	}
}

func TestCellAsInt64(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Test reading Bezirk (O9)
	val, err := cellAsInt64(f, SheetVertrag, "O9")
	if err != nil {
		t.Fatalf("cellAsInt64() error = %v", err)
	}

	// Bezirk should be a valid Berlin district (1-12)
	if val < 1 || val > 12 {
		t.Errorf("cellAsInt64() O9 = %d, expected Berlin district 1-12", val)
	}
}

func TestCellAsInt64Empty(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Test reading an empty cell (should return 0, no error)
	val, err := cellAsInt64(f, SheetVertrag, "ZZ999")
	if err != nil {
		t.Fatalf("cellAsInt64() error for empty cell = %v", err)
	}

	if val != 0 {
		t.Errorf("cellAsInt64() for empty cell = %d, expected 0", val)
	}
}

func TestCellAsCents(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Test reading Summe (Z9)
	val, err := cellAsCents(f, SheetVertrag, "Z9")
	if err != nil {
		t.Fatalf("cellAsCents() error = %v", err)
	}

	// Should be positive
	if val <= 0 {
		t.Errorf("cellAsCents() Z9 = %d, expected positive value", val)
	}
}

func TestCellAsCentsEmpty(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	// Test reading an empty cell
	val, err := cellAsCents(f, SheetVertrag, "ZZ999")
	if err != nil {
		t.Fatalf("cellAsCents() error = %v", err)
	}

	// Should return zero
	if val != 0 {
		t.Errorf("cellAsCents() for empty cell = %d, expected 0", val)
	}
}

func TestVoucherPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"GB-12345678901-02", true},
		{"GB-00000000000-00", true},
		{"GB-99999999999-99", true},
		{"", false},
		{"GB-1234567890-02", false},   // too few digits
		{"GB-123456789012-02", false}, // too many digits
		{"GB-12345678901-2", false},   // suffix too short
		{"GB-12345678901-123", false}, // suffix too long
		{"AB-12345678901-02", false},  // wrong prefix
		{"GB12345678901-02", false},   // missing dash
		{"Abrechnungssumme", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := voucherPattern.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("voucherPattern.MatchString(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSenatsabrechnungOutputStruct(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	eu, err := ParseEinrichtung(f)
	if err != nil {
		t.Fatalf("ParseEinrichtung() error = %v", err)
	}

	ar, err := ParseAbrechnung(f)
	if err != nil {
		t.Fatalf("ParseAbrechnung() error = %v", err)
	}

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	output := SenatsabrechnungOutput{
		Einrichtung: eu,
		Abrechnung:  ar,
		Vertrag:     v,
	}

	if output.Einrichtung == nil {
		t.Error("SenatsabrechnungOutput.Einrichtung is nil")
	}
	if output.Abrechnung == nil {
		t.Error("SenatsabrechnungOutput.Abrechnung is nil")
	}
	if output.Vertrag == nil {
		t.Error("SenatsabrechnungOutput.Vertrag is nil")
	}
}

// --- Malformed-input tests using in-memory Excel files ---

func TestParseVertragMissingSheet(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	_, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() unexpected error for missing sheet: %v", err)
	}
}

func TestParseVertragEmptySheet(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetVertrag)

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}
	if len(v.Kinder) != 0 {
		t.Errorf("ParseVertrag() got %d children, expected 0 for empty sheet", len(v.Kinder))
	}
}

func TestCellAsCentsRequiredInvalidValue(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "not-a-number")

	_, err := cellAsCentsRequired(f, sheet, "A1")
	if err == nil {
		t.Fatal("cellAsCentsRequired() expected error for non-numeric value")
	}
	if !strings.Contains(err.Error(), "Sheet1") {
		t.Errorf("error %q should contain sheet name", err.Error())
	}
	if !strings.Contains(err.Error(), "A1") {
		t.Errorf("error %q should contain cell reference", err.Error())
	}
}

func TestCellAsCentsInvalidValue(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "B2", "abc")

	_, err := cellAsCents(f, sheet, "B2")
	if err == nil {
		t.Fatal("cellAsCents() expected error for non-numeric value")
	}
	if !strings.Contains(err.Error(), "B2") {
		t.Errorf("error %q should contain cell reference", err.Error())
	}
}

func TestParseEinrichtungMissingHeader(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	_, err := ParseEinrichtung(f)
	if err == nil {
		t.Fatal("ParseEinrichtung() expected error for missing headers")
	}
	if !strings.Contains(err.Error(), SheetEinrichtung) {
		t.Errorf("error %q should contain sheet name", err.Error())
	}
	if !strings.Contains(err.Error(), "Einrichtungsname") {
		t.Errorf("error %q should name the missing header", err.Error())
	}
}

func TestParseAbrechnungMissingHeader(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetAbrechnung)

	_, err := ParseAbrechnung(f)
	if err == nil {
		t.Fatal("ParseAbrechnung() expected error for missing headers")
	}
	if !strings.Contains(err.Error(), SheetAbrechnung) {
		t.Errorf("error %q should contain sheet name", err.Error())
	}
}

func TestParseFromReader(t *testing.T) {
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	output, err := ParseFromReader(file)
	if err != nil {
		t.Fatalf("ParseFromReader() error = %v", err)
	}

	if output == nil {
		t.Fatal("ParseFromReader() returned nil")
	}

	if output.Einrichtung == nil {
		t.Error("ParseFromReader() Einrichtung is nil")
	}
	if output.Abrechnung == nil {
		t.Error("ParseFromReader() Abrechnung is nil")
	}
	if output.Vertrag == nil {
		t.Error("ParseFromReader() Vertrag is nil")
	}

	// Verify same values as file-based parsing
	if output.Einrichtung.Name != "Kindertagesstätte Sternenstaub" {
		t.Errorf("ParseFromReader() Einrichtung.Name = %q, expected %q", output.Einrichtung.Name, "Kindertagesstätte Sternenstaub")
	}
	if len(output.Vertrag.Kinder) != 40 {
		t.Errorf("ParseFromReader() got %d children, expected 40", len(output.Vertrag.Kinder))
	}
	if output.Einrichtung.ZuschlagIntegration != 165680 {
		t.Errorf("ParseFromReader() ZuschlagIntegration = %d, expected %d", output.Einrichtung.ZuschlagIntegration, 165680)
	}

	// Verify billing month
	expectedMonth := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	if !output.BillingMonth.Equal(expectedMonth) {
		t.Errorf("ParseFromReader() BillingMonth = %v, expected %v", output.BillingMonth, expectedMonth)
	}
}

func TestParseBillingMonth(t *testing.T) {
	f, err := OpenSenatsabrechnung(testFile)
	if err != nil {
		t.Fatalf("OpenSenatsabrechnung() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		t.Fatalf("ParseBillingMonth() error = %v", err)
	}

	// Expected: November 2025
	expected := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	if !billingMonth.Equal(expected) {
		t.Errorf("ParseBillingMonth() = %v, expected %v", billingMonth, expected)
	}
}

func TestParseBillingMonthMissingHeader(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	_, err := ParseBillingMonth(f)
	if err == nil {
		t.Fatal("ParseBillingMonth() expected error for missing header")
	}
}

func TestParseBillingMonthInvalidFormat(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	// Trägernummer label at A1, value at C1, invalid date at C2
	_ = f.SetCellValue(SheetEinrichtung, "A1", "Trägernummer")
	_ = f.SetCellValue(SheetEinrichtung, "C1", "0770")
	_ = f.SetCellValue(SheetEinrichtung, "C2", "not-a-date")

	_, err := ParseBillingMonth(f)
	if err == nil {
		t.Fatal("ParseBillingMonth() expected error for invalid date format")
	}
}

func TestParseBillingMonthValueNextColumn(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	// Trägernummer label at A1, value at B1, date at B2
	_ = f.SetCellValue(SheetEinrichtung, "A1", "Trägernummer")
	_ = f.SetCellValue(SheetEinrichtung, "B1", "0770")
	_ = f.SetCellValue(SheetEinrichtung, "B2", "03/25")

	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		t.Fatalf("ParseBillingMonth() error = %v", err)
	}
	expected := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	if !billingMonth.Equal(expected) {
		t.Errorf("ParseBillingMonth() = %v, expected %v", billingMonth, expected)
	}
}

func TestParseBillingMonthValueSeveralColumnsRight(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	// Matches real file layout: label at K1, value at O1, date at O2
	_ = f.SetCellValue(SheetEinrichtung, "K1", "Trägernummer:")
	_ = f.SetCellValue(SheetEinrichtung, "O1", "0770")
	_ = f.SetCellValue(SheetEinrichtung, "O2", "07/25")

	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		t.Fatalf("ParseBillingMonth() error = %v", err)
	}
	expected := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if !billingMonth.Equal(expected) {
		t.Errorf("ParseBillingMonth() = %v, expected %v", billingMonth, expected)
	}
}

// --- normalizeDecimal tests ---

func TestNormalizeDecimal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// US format (dot decimal, comma thousands)
		{"US simple", "899.87", "899.87"},
		{"US with thousands", "1,234.56", "1234.56"},
		{"US large", "66,747.00", "66747.00"},
		{"US integer", "100.00", "100.00"},
		{"US zero", "0.00", "0.00"},

		// German format (comma decimal, dot thousands)
		{"German simple", "899,87", "899.87"},
		{"German with thousands", "1.234,56", "1234.56"},
		{"German large", "66.747,00", "66747.00"},
		{"German zero", "0,00", "0.00"},
		{"German single decimal", "5,5", "5.5"},

		// No separators
		{"integer", "100", "100"},
		{"zero", "0", "0"},

		// US thousands only (3 digits after comma = thousands)
		{"US thousands no decimal", "1,234", "1234"},

		// Ambiguous but handled: comma with 2 digits after → German decimal
		{"German 10,00", "10,00", "10.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDecimal(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeDecimal(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Column discovery and layout tests ---

// buildVertragSheet creates an in-memory Vertragsübersicht sheet with the given layout.
// headerRow is the 1-indexed row where main headers go (sub-headers go headerRow+1).
// It creates one child data row starting at dataRow.
func buildVertragSheet(t *testing.T, f *excelize.File, layout string) {
	t.Helper()
	_, _ = f.NewSheet(SheetVertrag)

	switch layout {
	case "old":
		// Old layout: headers at row 6, sub-headers at row 7, section header at row 8, data at row 9
		// GutscheinNr at E6, Name at F6, etc.
		_ = f.SetCellValue(SheetVertrag, "E6", "GutscheinNr.")
		_ = f.SetCellValue(SheetVertrag, "F6", "Name des Kindes")
		_ = f.SetCellValue(SheetVertrag, "G6", "Geb.-datum")
		_ = f.SetCellValue(SheetVertrag, "I6", "QM")
		_ = f.SetCellValue(SheetVertrag, "J6", "MSS")
		_ = f.SetCellValue(SheetVertrag, "K6", "Hs")
		_ = f.SetCellValue(SheetVertrag, "L6", "SpH")
		_ = f.SetCellValue(SheetVertrag, "M6", "Registr. ab")
		_ = f.SetCellValue(SheetVertrag, "N6", "Betreu.-umfang")
		_ = f.SetCellValue(SheetVertrag, "O6", "Be-zirk")
		_ = f.SetCellValue(SheetVertrag, "P6", "Basis-entgelt")
		_ = f.SetCellValue(SheetVertrag, "Q6", "Abzug o.M.")
		_ = f.SetCellValue(SheetVertrag, "R6", "Anteil Eltern")
		_ = f.SetCellValue(SheetVertrag, "H6", "BuT")
		_ = f.SetCellValue(SheetVertrag, "T6", "BuT")
		_ = f.SetCellValue(SheetVertrag, "U6", "Anteil Bezirk")
		_ = f.SetCellValue(SheetVertrag, "V6", "Zuschläge")
		_ = f.SetCellValue(SheetVertrag, "Z6", "Abrechn.-summe")
		// Sub-headers at row 7
		_ = f.SetCellValue(SheetVertrag, "R7", "Betr.")
		_ = f.SetCellValue(SheetVertrag, "S7", "Essen")
		_ = f.SetCellValue(SheetVertrag, "V7", "QM")
		_ = f.SetCellValue(SheetVertrag, "W7", "MSS")
		_ = f.SetCellValue(SheetVertrag, "X7", "NdHs")
		_ = f.SetCellValue(SheetVertrag, "Y7", "SpH")
		// Section header at row 8
		_ = f.SetCellValue(SheetVertrag, "A8", "11170130-EKT Eene meene mopel")
		// Data at row 9
		_ = f.SetCellValue(SheetVertrag, "E9", "GB-12345678901-02")
		_ = f.SetCellValue(SheetVertrag, "F9", "Mustermann,Max")
		_ = f.SetCellValue(SheetVertrag, "G9", "05.20")
		_ = f.SetCellValue(SheetVertrag, "I9", "nein")
		_ = f.SetCellValue(SheetVertrag, "J9", "nein")
		_ = f.SetCellValue(SheetVertrag, "K9", "D")
		_ = f.SetCellValue(SheetVertrag, "L9", "N")
		_ = f.SetCellValue(SheetVertrag, "M9", "10.22")
		_ = f.SetCellValue(SheetVertrag, "N9", "teilzeit")
		_ = f.SetCellValue(SheetVertrag, "O9", "11")
		_ = f.SetCellValue(SheetVertrag, "P9", "922.87")
		_ = f.SetCellValue(SheetVertrag, "Q9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "R9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "S9", "23.00")
		_ = f.SetCellValue(SheetVertrag, "T9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "U9", "899.87")
		_ = f.SetCellValue(SheetVertrag, "V9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "W9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "X9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "Y9", "0.00")
		_ = f.SetCellValue(SheetVertrag, "Z9", "899.87")

	case "new":
		// New layout (July 2025+): headers at row 5, sub-headers at row 6, section at row 7, data at row 8
		// GutscheinNr at C5, Name at D5, columns shifted left by 2
		_ = f.SetCellValue(SheetVertrag, "C5", "GutscheinNr.")
		_ = f.SetCellValue(SheetVertrag, "D5", "Name des Kindes")
		_ = f.SetCellValue(SheetVertrag, "E5", "Geb.-datum")
		_ = f.SetCellValue(SheetVertrag, "F5", "BuT")
		_ = f.SetCellValue(SheetVertrag, "G5", "QM")
		_ = f.SetCellValue(SheetVertrag, "H5", "MSS")
		_ = f.SetCellValue(SheetVertrag, "I5", "Hs")
		_ = f.SetCellValue(SheetVertrag, "J5", "SpH")
		_ = f.SetCellValue(SheetVertrag, "K5", "Registr. ab")
		_ = f.SetCellValue(SheetVertrag, "L5", "Betreu.-umfang")
		_ = f.SetCellValue(SheetVertrag, "M5", "Be-zirk")
		_ = f.SetCellValue(SheetVertrag, "N5", "Basis-entgelt")
		_ = f.SetCellValue(SheetVertrag, "O5", "Abzug o.M.")
		_ = f.SetCellValue(SheetVertrag, "Q5", "Anteil Eltern")
		_ = f.SetCellValue(SheetVertrag, "S5", "BuT")
		_ = f.SetCellValue(SheetVertrag, "T5", "Anteil Bezirk")
		_ = f.SetCellValue(SheetVertrag, "V5", "Zuschläge")
		_ = f.SetCellValue(SheetVertrag, "AA5", "Abrechn.-summe")
		// Sub-headers at row 6
		_ = f.SetCellValue(SheetVertrag, "Q6", "Betreuung")
		_ = f.SetCellValue(SheetVertrag, "R6", "Essen")
		_ = f.SetCellValue(SheetVertrag, "V6", "QM")
		_ = f.SetCellValue(SheetVertrag, "W6", "MSS")
		_ = f.SetCellValue(SheetVertrag, "X6", "ndH")
		_ = f.SetCellValue(SheetVertrag, "Z6", "SpH")
		// Section header at row 7
		_ = f.SetCellValue(SheetVertrag, "A7", "11170130-EKT Eene meene mopel")
		// Data at row 8 — using German number format (comma decimal)
		_ = f.SetCellValue(SheetVertrag, "C8", "GB-98765432101-03")
		_ = f.SetCellValue(SheetVertrag, "D8", "Schmidt,Anna")
		_ = f.SetCellValue(SheetVertrag, "E8", "07.18")
		_ = f.SetCellValue(SheetVertrag, "G8", "nein")
		_ = f.SetCellValue(SheetVertrag, "H8", "nein")
		_ = f.SetCellValue(SheetVertrag, "I8", "D")
		_ = f.SetCellValue(SheetVertrag, "J8", "N")
		_ = f.SetCellValue(SheetVertrag, "K8", "08.24")
		_ = f.SetCellValue(SheetVertrag, "L8", "ganztags")
		_ = f.SetCellValue(SheetVertrag, "M8", "11")
		_ = f.SetCellValue(SheetVertrag, "N8", "1.037,61")
		_ = f.SetCellValue(SheetVertrag, "O8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "Q8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "R8", "23,00")
		_ = f.SetCellValue(SheetVertrag, "S8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "T8", "1.014,61")
		_ = f.SetCellValue(SheetVertrag, "V8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "W8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "X8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "Z8", "0,00")
		_ = f.SetCellValue(SheetVertrag, "AA8", "1.014,61")

	default:
		t.Fatalf("unknown layout %q", layout)
	}
}

func TestDiscoverVertragColumnsOldLayout(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	buildVertragSheet(t, f, "old")

	vc, err := discoverVertragColumns(f)
	if err != nil {
		t.Fatalf("discoverVertragColumns() error = %v", err)
	}

	if vc.gutscheinNr != "E" {
		t.Errorf("gutscheinNr = %q, expected E", vc.gutscheinNr)
	}
	if vc.nameDerKinder != "F" {
		t.Errorf("nameDerKinder = %q, expected F", vc.nameDerKinder)
	}
	if vc.bezirk != "O" {
		t.Errorf("bezirk = %q, expected O", vc.bezirk)
	}
	if vc.abrechnSumme != "Z" {
		t.Errorf("abrechnSumme = %q, expected Z", vc.abrechnSumme)
	}
	if vc.headerRow != 6 {
		t.Errorf("headerRow = %d, expected 6", vc.headerRow)
	}
	if vc.dataStartRow != 9 {
		t.Errorf("dataStartRow = %d, expected 9", vc.dataStartRow)
	}
}

func TestDiscoverVertragColumnsNewLayout(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	buildVertragSheet(t, f, "new")

	vc, err := discoverVertragColumns(f)
	if err != nil {
		t.Fatalf("discoverVertragColumns() error = %v", err)
	}

	if vc.gutscheinNr != "C" {
		t.Errorf("gutscheinNr = %q, expected C", vc.gutscheinNr)
	}
	if vc.nameDerKinder != "D" {
		t.Errorf("nameDerKinder = %q, expected D", vc.nameDerKinder)
	}
	if vc.bezirk != "M" {
		t.Errorf("bezirk = %q, expected M", vc.bezirk)
	}
	if vc.abrechnSumme != "AA" {
		t.Errorf("abrechnSumme = %q, expected AA", vc.abrechnSumme)
	}
	if vc.headerRow != 5 {
		t.Errorf("headerRow = %d, expected 5", vc.headerRow)
	}
	if vc.dataStartRow != 8 {
		t.Errorf("dataStartRow = %d, expected 8", vc.dataStartRow)
	}
}

func TestParseVertragOldLayout(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	buildVertragSheet(t, f, "old")

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	if len(v.Kinder) != 1 {
		t.Fatalf("ParseVertrag() got %d children, expected 1", len(v.Kinder))
	}

	k := v.Kinder[0]
	if k.Gutscheinnummer != "GB-12345678901-02" {
		t.Errorf("Gutscheinnummer = %q, expected GB-12345678901-02", k.Gutscheinnummer)
	}
	if k.Name != "Mustermann,Max" {
		t.Errorf("Name = %q, expected Mustermann,Max", k.Name)
	}
	if k.Betreuungsumfang != "teilzeit" {
		t.Errorf("Betreuungsumfang = %q, expected teilzeit", k.Betreuungsumfang)
	}
	if k.Bezirk != 11 {
		t.Errorf("Bezirk = %d, expected 11", k.Bezirk)
	}
	// 899.87 EUR = 89987 cents (US format)
	if k.Summe != 89987 {
		t.Errorf("Summe = %d, expected 89987", k.Summe)
	}
	if k.Basisentgeld != 92287 {
		t.Errorf("Basisentgeld = %d, expected 92287", k.Basisentgeld)
	}
	if k.ElternEssen != 2300 {
		t.Errorf("ElternEssen = %d, expected 2300", k.ElternEssen)
	}
	if k.AnteilBezirk != 89987 {
		t.Errorf("AnteilBezirk = %d, expected 89987", k.AnteilBezirk)
	}
}

func TestParseVertragNewLayout(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	buildVertragSheet(t, f, "new")

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	if len(v.Kinder) != 1 {
		t.Fatalf("ParseVertrag() got %d children, expected 1", len(v.Kinder))
	}

	k := v.Kinder[0]
	if k.Gutscheinnummer != "GB-98765432101-03" {
		t.Errorf("Gutscheinnummer = %q, expected GB-98765432101-03", k.Gutscheinnummer)
	}
	if k.Name != "Schmidt,Anna" {
		t.Errorf("Name = %q, expected Schmidt,Anna", k.Name)
	}
	if k.Betreuungsumfang != "ganztags" {
		t.Errorf("Betreuungsumfang = %q, expected ganztags", k.Betreuungsumfang)
	}
	if k.Bezirk != 11 {
		t.Errorf("Bezirk = %d, expected 11", k.Bezirk)
	}
	// 1.014,61 EUR (German) = 101461 cents
	if k.Summe != 101461 {
		t.Errorf("Summe = %d, expected 101461", k.Summe)
	}
	// 1.037,61 EUR = 103761 cents
	if k.Basisentgeld != 103761 {
		t.Errorf("Basisentgeld = %d, expected 103761", k.Basisentgeld)
	}
	if k.ElternEssen != 2300 {
		t.Errorf("ElternEssen = %d, expected 2300", k.ElternEssen)
	}
	// 1.014,61 EUR = 101461 cents
	if k.AnteilBezirk != 101461 {
		t.Errorf("AnteilBezirk = %d, expected 101461", k.AnteilBezirk)
	}
}

func TestParseVertragNewLayoutSubHeaderBetr(t *testing.T) {
	// Verify that "Betr." (abbreviated) sub-header works the same as "Betreuung"
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetVertrag)

	// Minimal old-style layout with "Betr." abbreviation
	_ = f.SetCellValue(SheetVertrag, "E6", "GutscheinNr.")
	_ = f.SetCellValue(SheetVertrag, "F6", "Name des Kindes")
	_ = f.SetCellValue(SheetVertrag, "G6", "Geb.-datum")
	_ = f.SetCellValue(SheetVertrag, "I6", "QM")
	_ = f.SetCellValue(SheetVertrag, "J6", "MSS")
	_ = f.SetCellValue(SheetVertrag, "K6", "Hs")
	_ = f.SetCellValue(SheetVertrag, "L6", "SpH")
	_ = f.SetCellValue(SheetVertrag, "M6", "Registr. ab")
	_ = f.SetCellValue(SheetVertrag, "N6", "Betreu.-umfang")
	_ = f.SetCellValue(SheetVertrag, "O6", "Be-zirk")
	_ = f.SetCellValue(SheetVertrag, "P6", "Basis-entgelt")
	_ = f.SetCellValue(SheetVertrag, "Q6", "Abzug o.M.")
	_ = f.SetCellValue(SheetVertrag, "R6", "Anteil Eltern")
	_ = f.SetCellValue(SheetVertrag, "H6", "BuT")
	_ = f.SetCellValue(SheetVertrag, "T6", "BuT")
	_ = f.SetCellValue(SheetVertrag, "U6", "Anteil Bezirk")
	_ = f.SetCellValue(SheetVertrag, "V6", "Zuschläge")
	_ = f.SetCellValue(SheetVertrag, "Z6", "Abrechn.-summe")
	// "Betr." instead of "Betreuung"
	_ = f.SetCellValue(SheetVertrag, "R7", "Betr.")
	_ = f.SetCellValue(SheetVertrag, "S7", "Essen")
	_ = f.SetCellValue(SheetVertrag, "V7", "QM")
	_ = f.SetCellValue(SheetVertrag, "W7", "MSS")
	_ = f.SetCellValue(SheetVertrag, "X7", "NdHs")
	_ = f.SetCellValue(SheetVertrag, "Y7", "SpH")

	vc, err := discoverVertragColumns(f)
	if err == nil {
		// No data rows, so discoverVertragColumns should fail
		// But we can still check the column mapping if we had data
		t.Logf("vc = %+v", vc)
	}

	// The important check: "Betr." maps to elternBetreuung column R
	// We need to add a data row for full discovery
	_ = f.SetCellValue(SheetVertrag, "E9", "GB-11111111111-01")
	vc, err = discoverVertragColumns(f)
	if err != nil {
		t.Fatalf("discoverVertragColumns() error = %v", err)
	}
	if vc.elternBetreuung != "R" {
		t.Errorf("elternBetreuung = %q, expected R (from 'Betr.' sub-header)", vc.elternBetreuung)
	}
}

func TestParseVertragMultipleChildren(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	buildVertragSheet(t, f, "old")

	// Add a second child at row 10
	_ = f.SetCellValue(SheetVertrag, "E10", "GB-99999999999-01")
	_ = f.SetCellValue(SheetVertrag, "F10", "Schneider,Lisa")
	_ = f.SetCellValue(SheetVertrag, "G10", "03.19")
	_ = f.SetCellValue(SheetVertrag, "I10", "ja")
	_ = f.SetCellValue(SheetVertrag, "J10", "nein")
	_ = f.SetCellValue(SheetVertrag, "K10", "D")
	_ = f.SetCellValue(SheetVertrag, "L10", "N")
	_ = f.SetCellValue(SheetVertrag, "M10", "01.23")
	_ = f.SetCellValue(SheetVertrag, "N10", "ganztags")
	_ = f.SetCellValue(SheetVertrag, "O10", "3")
	_ = f.SetCellValue(SheetVertrag, "P10", "1037.61")
	_ = f.SetCellValue(SheetVertrag, "Q10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "R10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "S10", "23.00")
	_ = f.SetCellValue(SheetVertrag, "T10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "U10", "1014.61")
	_ = f.SetCellValue(SheetVertrag, "V10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "W10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "X10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "Y10", "0.00")
	_ = f.SetCellValue(SheetVertrag, "Z10", "1014.61")

	v, err := ParseVertrag(f)
	if err != nil {
		t.Fatalf("ParseVertrag() error = %v", err)
	}

	if len(v.Kinder) != 2 {
		t.Fatalf("ParseVertrag() got %d children, expected 2", len(v.Kinder))
	}

	if v.Kinder[0].Name != "Mustermann,Max" {
		t.Errorf("first child Name = %q, expected Mustermann,Max", v.Kinder[0].Name)
	}
	if v.Kinder[1].Name != "Schneider,Lisa" {
		t.Errorf("second child Name = %q, expected Schneider,Lisa", v.Kinder[1].Name)
	}
	if v.Kinder[1].QM != "ja" {
		t.Errorf("second child QM = %q, expected ja", v.Kinder[1].QM)
	}
	if v.Kinder[1].Bezirk != 3 {
		t.Errorf("second child Bezirk = %d, expected 3", v.Kinder[1].Bezirk)
	}
}

func TestParseBillingMonthMergedLabelCells(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_, _ = f.NewSheet(SheetEinrichtung)

	// Merged label cells K1:N1, value at O1, date at O2 (matches real file layout)
	_ = f.SetCellValue(SheetEinrichtung, "K1", "Trägernummer:")
	_ = f.MergeCell(SheetEinrichtung, "K1", "N1")
	_ = f.SetCellValue(SheetEinrichtung, "O1", "0770")
	_ = f.SetCellValue(SheetEinrichtung, "O2", "07/25")

	billingMonth, err := ParseBillingMonth(f)
	if err != nil {
		t.Fatalf("ParseBillingMonth() error = %v", err)
	}
	expected := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if !billingMonth.Equal(expected) {
		t.Errorf("ParseBillingMonth() = %v, expected %v", billingMonth, expected)
	}
}

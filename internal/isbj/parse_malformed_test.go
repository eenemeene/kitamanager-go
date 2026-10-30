package isbj

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// GO-2026-6452 was a panic in excelize on a cell whose shared-string index is
// negative: File.getFromStringItem tested only the upper bound, so a <v> of -1
// indexed f.sharedStringItem out of range (rows.go:362). The path is taken only
// once the shared-string table is large enough for excelize to stream it to a
// temp file, governed by UnzipXMLSizeLimit — and xlsxOpenOptions sets that to
// 25 MB, so every ISBJ upload was one crafted workbook away from it.
//
// Upstream fixed it after v2.11.0, and this module now pins the commit that
// carries the fix. The fixture stays as the regression guard, with the
// assertions inverted: a workbook that once panicked must now parse far enough
// to fail on its own structure. recoverMalformedWorkbook stays in parse.go too
// — not for this bug, which is fixed, but as the net for the next one.
//
// If either test starts reporting a "malformed spreadsheet" error, a panic was
// recovered and upstream has regressed. That is the signal these tests exist to
// give; they are deliberately written to fail loudly on it rather than to
// tolerate either outcome.
//
// The workbook is built here rather than committed as a fixture: it is 30 MB
// uncompressed, and it carries no data worth storing.
func buildNegativeSharedStringIndexWorkbook(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name, body string) {
		t.Helper()
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	add("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/></Types>`)
	add("_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)
	add("xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	add("xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/></Relationships>`)

	// Push sharedStrings.xml past UnzipXMLSizeLimit so excelize streams it.
	var sst strings.Builder
	sst.WriteString(`<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	filler := strings.Repeat("A", 1000)
	for range 30000 {
		fmt.Fprintf(&sst, "<si><t>%s</t></si>", filler)
	}
	sst.WriteString(`</sst>`)
	if int64(sst.Len()) <= MaxXLSXUnzipXMLSize {
		t.Fatalf("shared strings are %d bytes, not over the %d limit — the streaming path this test needs would not be taken",
			sst.Len(), MaxXLSXUnzipXMLSize)
	}
	add("xl/sharedStrings.xml", sst.String())

	add("xl/worksheets/sheet1.xml", `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>-1</v></c></row></sheetData></worksheet>`)

	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip: %v", err)
	}
	return buf.Bytes()
}

func TestParseFromReaderSurvivesNegativeSharedStringIndex(t *testing.T) {
	data := buildNegativeSharedStringIndexWorkbook(t)

	// A panic here fails the test by taking the goroutine down, which is the
	// first half of the guard. The second half is the shape of the error: the
	// fixture carries one sheet named "Sheet1", so a parser that gets as far as
	// looking for the ISBJ sheets must fail on the missing one.
	out, err := ParseFromReader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected an error: the fixture carries no Einrichtungsübersicht sheet")
	}
	if out != nil {
		t.Errorf("expected no output alongside the error, got %+v", out)
	}
	if strings.Contains(err.Error(), "malformed spreadsheet") {
		t.Fatalf("error = %q — a panic was recovered, so excelize has regressed on GO-2026-6452", err)
	}
	t.Logf("parsed past the shared-string table and failed on structure, as it should: %v", err)
}

func TestOpenSenatsabrechnungSurvivesNegativeSharedStringIndex(t *testing.T) {
	path := t.TempDir() + "/malformed.xlsx"
	if err := writeFile(t, path, buildNegativeSharedStringIndexWorkbook(t)); err != nil {
		t.Fatal(err)
	}

	// OpenSenatsabrechnung only opens the workbook and enforces the size limits,
	// neither of which this fixture breaches, so with the panic fixed upstream it
	// now succeeds where it used to return a recovered error.
	f, err := OpenSenatsabrechnung(path)
	if err != nil {
		if strings.Contains(err.Error(), "malformed spreadsheet") {
			t.Fatalf("error = %q — a panic was recovered, so excelize has regressed on GO-2026-6452", err)
		}
		t.Fatalf("OpenSenatsabrechnung() error = %v, want it to open cleanly", err)
	}
	if f == nil {
		t.Fatal("OpenSenatsabrechnung() returned no error and no workbook handle")
	}
	t.Cleanup(func() { _ = f.Close() })
}

func writeFile(t *testing.T, path string, data []byte) error {
	t.Helper()
	return os.WriteFile(path, data, 0o600)
}

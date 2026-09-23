package isbj

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// GO-2026-6452 is a panic in excelize on a cell whose shared-string index is
// negative. v2.11.0 fixed the bounds check in xlsxC.getValueFrom but not the
// one in File.getFromStringItem, which still tests only the upper bound:
//
//	if len(f.sharedStringItem) <= index {   // rows.go:359
//	    return strconv.Itoa(index)
//	}
//	offsetRange := f.sharedStringItem[index]  // rows.go:362 — panics on -1
//
// That path is taken only once the shared-string table is large enough for
// excelize to stream it to a temp file, which is governed by
// UnzipXMLSizeLimit — and xlsxOpenOptions sets that to 25 MB, so every ISBJ
// upload is one crafted workbook away from it. There is no released excelize
// with a fix, so the parser catches the panic instead.
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

	// Without recoverMalformedWorkbook this panics inside GetRows and takes
	// the calling goroutine with it.
	out, err := ParseFromReader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected an error for a workbook with a negative shared-string index, got nil")
	}
	if out != nil {
		t.Errorf("expected no output alongside the error, got %+v", out)
	}
	if !strings.Contains(err.Error(), "malformed spreadsheet") {
		t.Errorf("error = %q, want it to name the malformed spreadsheet", err)
	}
	t.Logf("recovered as: %v", err)
}

func TestOpenSenatsabrechnungSurvivesNegativeSharedStringIndex(t *testing.T) {
	path := t.TempDir() + "/malformed.xlsx"
	if err := writeFile(t, path, buildNegativeSharedStringIndexWorkbook(t)); err != nil {
		t.Fatal(err)
	}

	f, err := OpenSenatsabrechnung(path)
	if err == nil {
		t.Fatal("expected an error for a workbook with a negative shared-string index, got nil")
	}
	if f != nil {
		t.Error("expected no workbook handle alongside the error")
	}
	if !strings.Contains(err.Error(), "malformed spreadsheet") {
		t.Errorf("error = %q, want it to name the malformed spreadsheet", err)
	}
}

func writeFile(t *testing.T, path string, data []byte) error {
	t.Helper()
	return os.WriteFile(path, data, 0o600)
}

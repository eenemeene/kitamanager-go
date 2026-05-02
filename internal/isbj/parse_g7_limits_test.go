package isbj

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// G7 / I-M-1 — workbooks with more sheets than MaxXLSXSheetCount must
// be rejected before any parser runs. Without this guard a malicious
// upload could allocate arbitrarily many shared-string and styles
// caches inside excelize, even at modest unzip totals.
func TestEnforceWorkbookLimits_TooManySheets(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// excelize.NewFile() opens with one default sheet ("Sheet1").
	// Add sheets until we exceed the cap by 1.
	for i := range MaxXLSXSheetCount + 1 {
		name := fmt.Sprintf("Sheet%d", i+2)
		if _, err := f.NewSheet(name); err != nil {
			t.Fatalf("NewSheet: %v", err)
		}
	}

	err := enforceWorkbookLimits(f)
	if err == nil {
		t.Fatal("expected error for over-sized sheet count")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected sheet-count error, got: %v", err)
	}
}

// G7 / I-M-1 — sheets with more rows than MaxXLSXRowsPerSheet are
// rejected. We pick a row count just past the cap so the fixture
// builds in seconds; the production cap is far above any legitimate
// Senatsabrechnung.
func TestEnforceWorkbookLimits_TooManyRows(t *testing.T) {
	if testing.Short() {
		t.Skip("row-bomb fixture takes a few seconds to build")
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)

	// One scalar per row — keeps the fixture lean while still tripping
	// the row counter.
	for i := range MaxXLSXRowsPerSheet + 5 {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetCellValue(sheet, cell, "x"); err != nil {
			t.Fatalf("SetCellValue: %v", err)
		}
	}

	err := enforceWorkbookLimits(f)
	if err == nil {
		t.Fatal("expected error for over-sized row count")
	}
	if !strings.Contains(err.Error(), "rows") {
		t.Errorf("expected row-count error, got: %v", err)
	}
}

// G7 / I-M-1 — happy path: a small workbook within all caps must pass
// the guard. This documents that the cap is *not* zero — i.e. nothing
// in the limits silently rejects legitimate inputs.
func TestEnforceWorkbookLimits_Happy(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "hello")

	if err := enforceWorkbookLimits(f); err != nil {
		t.Fatalf("expected nil for small workbook, got: %v", err)
	}
}

// G7 / I-M-1 — `xlsxOpenOptions` actually wires the unzip caps into
// excelize; a regression that drops one of them would silently revert
// to the 16 GB default. Asserting on the returned struct keeps the
// hardening contract visible.
func TestXLSXOpenOptions_PinsUnzipCaps(t *testing.T) {
	opts := xlsxOpenOptions()
	if opts.UnzipSizeLimit != MaxXLSXUnzipSize {
		t.Errorf("UnzipSizeLimit = %d, want %d", opts.UnzipSizeLimit, MaxXLSXUnzipSize)
	}
	if opts.UnzipXMLSizeLimit != MaxXLSXUnzipXMLSize {
		t.Errorf("UnzipXMLSizeLimit = %d, want %d", opts.UnzipXMLSizeLimit, MaxXLSXUnzipXMLSize)
	}
}

// G7 / I-M-1 — ParseFromReader rejects oversized workbooks end-to-end
// (not just the lower-level enforceWorkbookLimits helper). Build a
// workbook with too many sheets, marshal to xlsx bytes, feed through
// ParseFromReader, and assert it fails before reaching cell parsing.
func TestParseFromReader_RejectsOversizedSheets(t *testing.T) {
	f := excelize.NewFile()
	for i := range MaxXLSXSheetCount + 1 {
		name := fmt.Sprintf("Sheet%d", i+2)
		if _, err := f.NewSheet(name); err != nil {
			t.Fatalf("NewSheet: %v", err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	_ = f.Close()

	_, err := ParseFromReader(&buf)
	if err == nil {
		t.Fatal("expected error from ParseFromReader on too-many-sheets")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("unexpected error from ParseFromReader: %v", err)
	}
}

package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

// openXLSX parses an XLSX response body into an excelize.File for content verification.
func openXLSX(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to open XLSX from response: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestExportHandler_ExportEmployees_Success(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Test Org")

	emp := testutil.CreateTestEmployee(t, db, "Jane", "Doe", org.ID)
	createActiveEmployeeContract(t, db, emp.ID)

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/employees/export/excel", handler.ExportEmployees)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees/export/excel", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		t.Errorf("expected XLSX content type, got %q", ct)
	}

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "mitarbeiter.xlsx") {
		t.Errorf("expected Content-Disposition with mitarbeiter.xlsx, got %q", cd)
	}

	// Verify the XLSX contains the employee data
	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Mitarbeiter")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	// Row 0 = headers, Row 1 = Jane Doe
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows (header + 1 employee), got %d", len(rows))
	}
	if rows[1][0] != "Jane" {
		t.Errorf("expected first name 'Jane', got %q", rows[1][0])
	}
	if rows[1][1] != "Doe" {
		t.Errorf("expected last name 'Doe', got %q", rows[1][1])
	}
}

func TestExportHandler_ExportEmployees_Empty(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Empty Org")

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/employees/export/excel", handler.ExportEmployees)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees/export/excel", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		t.Errorf("expected XLSX content type, got %q", ct)
	}

	// Verify the XLSX has only the header row
	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Mitarbeiter")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row (headers only), got %d", len(rows))
	}
}

func TestExportHandler_ExportEmployees_WithSectionFilter(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Test Org")

	// Create an employee with a contract in the default section
	emp := testutil.CreateTestEmployee(t, db, "Alice", "Smith", org.ID)
	createActiveEmployeeContract(t, db, emp.ID)

	// Create a second section and an employee in it
	section2 := createTestSectionWithOrg(t, db, "Section B", org.ID)
	emp2 := testutil.CreateTestEmployee(t, db, "Bob", "Jones", org.ID)
	payPlanID := ensureTestPayPlan(t, db, org.ID)
	if err := db.Create(&models.EmployeeContract{
		EmployeeID:    emp2.ID,
		BaseContract:  models.BaseContract{Period: models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}, SectionID: section2.ID},
		StaffCategory: "qualified",
		WeeklyHours:   40,
		PayPlanID:     payPlanID,
	}).Error; err != nil {
		t.Fatalf("failed to create employee contract: %v", err)
	}

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/employees/export/excel", handler.ExportEmployees)

	// Filter by section2 — should only include Bob, not Alice
	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/employees/export/excel?section_id=%d", org.ID, section2.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Mitarbeiter")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	// Header + 1 employee (Bob)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (header + Bob), got %d", len(rows))
	}
	if rows[1][0] != "Bob" {
		t.Errorf("expected first name 'Bob' in section filter result, got %q", rows[1][0])
	}
}

func TestExportHandler_ExportEmployees_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/employees/export/excel", handler.ExportEmployees)

	w := performRequest(r, "GET", "/api/v1/organizations/abc/employees/export/excel", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportHandler_ExportChildren_Success(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Test Org")

	child := testutil.CreateTestChild(t, db, "Max", "Mueller", org.ID)
	createActiveChildContract(t, db, child.ID)

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/children/export/excel", handler.ExportChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/children/export/excel", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		t.Errorf("expected XLSX content type, got %q", ct)
	}

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "kinder.xlsx") {
		t.Errorf("expected Content-Disposition with kinder.xlsx, got %q", cd)
	}

	// Verify the XLSX contains the child data
	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Kinder")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows (header + 1 child), got %d", len(rows))
	}
	if rows[1][0] != "Max" {
		t.Errorf("expected first name 'Max', got %q", rows[1][0])
	}
	if rows[1][1] != "Mueller" {
		t.Errorf("expected last name 'Mueller', got %q", rows[1][1])
	}
}

func TestExportHandler_ExportChildren_Empty(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Empty Org")

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/children/export/excel", handler.ExportChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/children/export/excel", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		t.Errorf("expected XLSX content type, got %q", ct)
	}

	// Verify the XLSX has only the header row
	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Kinder")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row (headers only), got %d", len(rows))
	}
}

func TestExportHandler_ExportChildren_WithContractAfter(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Test Org")

	child := testutil.CreateTestChild(t, db, "Lisa", "Schmidt", org.ID)
	createActiveChildContract(t, db, child.ID) // contract from 2020-01-01

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/children/export/excel", handler.ExportChildren)

	// contract_after=2019-01-01 should include the child whose contract starts 2020-01-01
	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/children/export/excel?contract_after=2019-01-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Kinder")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows (header + Lisa), got %d", len(rows))
	}
	if rows[1][0] != "Lisa" {
		t.Errorf("expected first name 'Lisa', got %q", rows[1][0])
	}
}

func TestExportHandler_ExportChildren_DefaultActiveOnToday(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestSuperAdmin(t, db)
	r := setupTestRouterWithUser(admin.ID)
	org := createTestOrganization(t, db, "Test Org")

	// Create a child with an active contract (from 2020-01-01, no end date -- active today)
	child := testutil.CreateTestChild(t, db, "Anna", "Weber", org.ID)
	createActiveChildContract(t, db, child.ID)

	// Create a child with an ended contract (should NOT appear in the default export)
	child2 := testutil.CreateTestChild(t, db, "Old", "Child", org.ID)
	sectionID := ensureTestSection(t, db, org.ID)
	endDate := time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ChildContract{
		ChildID: child2.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC), To: &endDate},
			SectionID: sectionID,
		},
	}).Error; err != nil {
		t.Fatalf("failed to create ended child contract: %v", err)
	}

	employeeSvc := createEmployeeService(db)
	childSvc := createChildService(db)
	handler := NewExportHandler(employeeSvc, childSvc, createAuditService(db))

	r.GET("/api/v1/organizations/:orgId/children/export/excel", handler.ExportChildren)

	// No active_on or contract_after params -- should default active_on to today
	w := performRequest(r, "GET", fmt.Sprintf("/api/v1/organizations/%d/children/export/excel", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		t.Errorf("expected XLSX content type, got %q", ct)
	}

	// Verify only the active child appears, not the one with the ended contract
	f := openXLSX(t, w.Body.Bytes())
	rows, err := f.GetRows("Kinder")
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}
	// Header + 1 active child (Anna). Old Child's contract ended in 2019, should be excluded.
	if len(rows) != 2 {
		t.Errorf("expected 2 rows (header + Anna), got %d -- children with ended contracts should be excluded when defaulting active_on to today", len(rows))
	}
	if len(rows) >= 2 && rows[1][0] != "Anna" {
		t.Errorf("expected first name 'Anna', got %q", rows[1][0])
	}
}

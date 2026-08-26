package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

// Coverage for the values an audit row records, as opposed to the fact that it
// exists at all.
//
// Every resource here is money-shaped: a pay plan entry is a salary, a budget
// item entry is a euro amount, a funding property is a Fördersatz worth tens to
// hundreds of euros per child per month. Before these paths carried a diff, the
// audit row for a salary change said `{"resource_name":"period=3"}` — enough to
// know an edit happened, not enough to know what it was, and the previous value
// was gone from the database by the time anyone asked.
//
// The delete cases matter for a different reason: an update can be reconstructed
// from the surviving row plus the diff, a delete cannot. Once the row is gone the
// snapshot is the only record of what was removed.

// auditDetails parses an audit row's Details JSON.
func auditDetails(t *testing.T, row *models.AuditLog) map[string]any {
	t.Helper()
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details is not JSON (%q): %v", row.Details, err)
	}
	return details
}

// auditChangeEntry pulls one {old,new} pair out of a row's `changes` map.
func auditChangeEntry(t *testing.T, row *models.AuditLog, field string) (oldVal, newVal any) {
	t.Helper()
	details := auditDetails(t, row)
	changes, ok := details["changes"].(map[string]any)
	if !ok {
		t.Fatalf("audit row carries no changes map: %s", row.Details)
	}
	entry, ok := changes[field].(map[string]any)
	if !ok {
		t.Fatalf("changes has no entry for %q: %s", field, row.Details)
	}
	return entry["old"], entry["new"]
}

func auditSnapshotMap(t *testing.T, row *models.AuditLog) map[string]any {
	t.Helper()
	details := auditDetails(t, row)
	snapshot, ok := details["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("audit row carries no snapshot: %s", row.Details)
	}
	return snapshot
}

// assertNoChanges is the guard against a diff that fires on every update
// regardless of content — a derived or bookkeeping field flickering between the
// read and the write would make the `changes` map meaningless.
func assertNoChanges(t *testing.T, row *models.AuditLog) {
	t.Helper()
	details := auditDetails(t, row)
	if changes, present := details["changes"]; present {
		t.Errorf("expected no changes map for a no-op update, got %v", changes)
	}
}

func newPayPlanTestHandler(db *gorm.DB) *PayPlanHandler {
	return NewPayPlanHandler(service.NewPayPlanService(store.NewPayPlanStore(db), store.NewTransactor(db)), createAuditService(db))
}

// --- pay plan entries: the salary itself ---

func TestAudit_PayPlanEntryUpdate_RecordsTheSalaryChange(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := newPayPlanTestHandler(db)

	r := setupTestRouter()
	base := "/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries"
	r.POST(base, handler.CreateEntry)
	r.PUT(base+"/:entryId", handler.UpdateEntry)
	r.DELETE(base+"/:entryId", handler.DeleteEntry)

	planID, periodID := seedPayPlanPeriod(t, db, org.ID)

	created := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries", org.ID, planID, periodID),
		models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 350000})
	if created.Code != http.StatusCreated {
		t.Fatalf("create entry: %d %s", created.Code, created.Body.String())
	}
	var entry models.PayPlanEntryResponse
	parseResponse(t, created, &entry)

	path := fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, planID, periodID, entry.ID)
	w := performRequest(r, "PUT", path, models.PayPlanEntryUpdateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 400000})
	if w.Code != http.StatusOK {
		t.Fatalf("update entry: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("pay_plan_entry_update"),
		ResourceType: "pay_plan_entry",
		ResourceID:   entry.ID,
	})
	oldVal, newVal := auditChangeEntry(t, row, "monthly_amount")
	if fmt.Sprint(oldVal) != "350000" || fmt.Sprint(newVal) != "400000" {
		t.Errorf("expected the salary recorded as 350000 -> 400000, got %v -> %v", oldVal, newVal)
	}
}

func TestAudit_PayPlanEntryUpdate_NoOpRecordsNoChanges(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := newPayPlanTestHandler(db)

	r := setupTestRouter()
	base := "/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries"
	r.POST(base, handler.CreateEntry)
	r.PUT(base+"/:entryId", handler.UpdateEntry)

	planID, periodID := seedPayPlanPeriod(t, db, org.ID)
	created := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries", org.ID, planID, periodID),
		models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 350000})
	var entry models.PayPlanEntryResponse
	parseResponse(t, created, &entry)

	path := fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, planID, periodID, entry.ID)
	w := performRequest(r, "PUT", path, models.PayPlanEntryUpdateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 350000})
	if w.Code != http.StatusOK {
		t.Fatalf("update entry: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("pay_plan_entry_update"),
		ResourceType: "pay_plan_entry",
		ResourceID:   entry.ID,
	})
	assertNoChanges(t, row)
}

func TestAudit_PayPlanEntryDelete_RecordsSnapshot(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := newPayPlanTestHandler(db)

	r := setupTestRouter()
	base := "/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries"
	r.POST(base, handler.CreateEntry)
	r.DELETE(base+"/:entryId", handler.DeleteEntry)

	planID, periodID := seedPayPlanPeriod(t, db, org.ID)
	created := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries", org.ID, planID, periodID),
		models.PayPlanEntryCreateRequest{Grade: "S8a", Step: 3, MonthlyAmount: 350000})
	var entry models.PayPlanEntryResponse
	parseResponse(t, created, &entry)

	path := fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, planID, periodID, entry.ID)
	if w := performRequest(r, "DELETE", path, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete entry: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("pay_plan_entry_delete"),
		ResourceType: "pay_plan_entry",
		ResourceID:   entry.ID,
	})
	snapshot := auditSnapshotMap(t, row)
	if fmt.Sprint(snapshot["monthly_amount"]) != "350000" {
		t.Errorf("expected the deleted salary in the snapshot, got %v", snapshot["monthly_amount"])
	}
	if snapshot["grade"] != "S8a" {
		t.Errorf("expected the grade in the snapshot, got %v", snapshot["grade"])
	}
	if _, present := snapshot["created_at"]; present {
		t.Error("bookkeeping timestamps must not reach the snapshot")
	}
}

// --- budget item entries ---

func TestAudit_BudgetItemEntryUpdate_RecordsTheAmountChange(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := NewBudgetItemHandler(createBudgetItemService(db), createAuditService(db))

	r := setupTestRouter()
	r.POST("/organizations/:orgId/budget-items", handler.Create)
	r.POST("/organizations/:orgId/budget-items/:budgetItemId/entries", handler.CreateEntry)
	r.PUT("/organizations/:orgId/budget-items/:budgetItemId/entries/:entryId", handler.UpdateEntry)
	r.DELETE("/organizations/:orgId/budget-items/:budgetItemId/entries/:entryId", handler.DeleteEntry)

	itemResp := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/budget-items", org.ID),
		models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income", PerChild: true})
	if itemResp.Code != http.StatusCreated {
		t.Fatalf("create item: %d %s", itemResp.Code, itemResp.Body.String())
	}
	var item models.BudgetItemResponse
	parseResponse(t, itemResp, &item)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entryResp := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/budget-items/%d/entries", org.ID, item.ID),
		models.BudgetItemEntryCreateRequest{From: from, AmountCents: 50000})
	if entryResp.Code != http.StatusCreated {
		t.Fatalf("create entry: %d %s", entryResp.Code, entryResp.Body.String())
	}
	var entry models.BudgetItemEntryResponse
	parseResponse(t, entryResp, &entry)

	path := fmt.Sprintf("/organizations/%d/budget-items/%d/entries/%d", org.ID, item.ID, entry.ID)
	w := performRequest(r, "PUT", path, models.BudgetItemEntryUpdateRequest{From: from, AmountCents: 75000})
	if w.Code != http.StatusOK {
		t.Fatalf("update entry: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("budget_item_entry_update"),
		ResourceType: "budget_item_entry",
		ResourceID:   entry.ID,
	})
	oldVal, newVal := auditChangeEntry(t, row, "amount_cents")
	if fmt.Sprint(oldVal) != "50000" || fmt.Sprint(newVal) != "75000" {
		t.Errorf("expected 50000 -> 75000 cents, got %v -> %v", oldVal, newVal)
	}

	if w := performRequest(r, "DELETE", path, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete entry: %d %s", w.Code, w.Body.String())
	}
	delRow := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("budget_item_entry_delete"),
		ResourceType: "budget_item_entry",
		ResourceID:   entry.ID,
	})
	if got := auditSnapshotMap(t, delRow)["amount_cents"]; fmt.Sprint(got) != "75000" {
		t.Errorf("expected the deleted amount in the snapshot, got %v", got)
	}
}

// --- government funding properties: the Fördersätze ---

func TestAudit_FundingPropertyUpdate_RecordsThePaymentChange(t *testing.T) {
	db := setupTestDB(t)
	svc := service.NewGovernmentFundingService(store.NewGovernmentFundingStore(db), store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db), importer.NewGovernmentFundingImporter(svc, store.NewTransactor(db)))

	r := setupTestRouter()
	r.POST("/fundings", handler.Create)
	r.POST("/fundings/:fundingId/periods", handler.CreatePeriod)
	r.POST("/fundings/:fundingId/periods/:periodId/properties", handler.CreateProperty)
	r.PUT("/fundings/:fundingId/periods/:periodId/properties/:propertyId", handler.UpdateProperty)
	r.DELETE("/fundings/:fundingId/periods/:periodId/properties/:propertyId", handler.DeleteProperty)

	var funding models.GovernmentFundingResponse
	parseResponse(t, performRequest(r, "POST", "/fundings",
		models.GovernmentFundingCreateRequest{Name: "Berlin", State: "berlin"}), &funding)

	var period models.GovernmentFundingPeriodResponse
	periodResp := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods", funding.ID),
		models.GovernmentFundingPeriodCreateRequest{
			From:                time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			FullTimeWeeklyHours: 39,
		})
	if periodResp.Code != http.StatusCreated {
		t.Fatalf("create period: %d %s", periodResp.Code, periodResp.Body.String())
	}
	parseResponse(t, periodResp, &period)

	propResp := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/%d/properties", funding.ID, period.ID),
		models.GovernmentFundingPropertyCreateRequest{
			Key: "care_type", Value: "ganztag", Label: "Ganztag", Payment: 166847, Requirement: 0.261,
		})
	if propResp.Code != http.StatusCreated {
		t.Fatalf("create property: %d %s", propResp.Code, propResp.Body.String())
	}
	var prop models.GovernmentFundingPropertyResponse
	parseResponse(t, propResp, &prop)

	path := fmt.Sprintf("/fundings/%d/periods/%d/properties/%d", funding.ID, period.ID, prop.ID)
	newPayment := 180000
	w := performRequest(r, "PUT", path, models.GovernmentFundingPropertyUpdateRequest{Payment: &newPayment})
	if w.Code != http.StatusOK {
		t.Fatalf("update property: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("gov_funding_property_update"),
		ResourceType: "gov_funding_property",
		ResourceID:   prop.ID,
	})
	oldVal, newVal := auditChangeEntry(t, row, "payment")
	if fmt.Sprint(oldVal) != "166847" || fmt.Sprint(newVal) != "180000" {
		t.Errorf("expected the Fördersatz recorded as 166847 -> 180000, got %v -> %v", oldVal, newVal)
	}

	if w := performRequest(r, "DELETE", path, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete property: %d %s", w.Code, w.Body.String())
	}
	delRow := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("gov_funding_property_delete"),
		ResourceType: "gov_funding_property",
		ResourceID:   prop.ID,
	})
	snapshot := auditSnapshotMap(t, delRow)
	if fmt.Sprint(snapshot["payment"]) != "180000" {
		t.Errorf("expected the deleted payment in the snapshot, got %v", snapshot["payment"])
	}
	if snapshot["key"] != "care_type" || snapshot["value"] != "ganztag" {
		t.Errorf("expected the property identity in the snapshot, got %v", snapshot)
	}
}

// --- sections: the org-scoped CRUD helper path ---

func TestAudit_SectionUpdateAndDelete_RecordValues(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := NewSectionHandler(createSectionService(db), createAuditService(db))

	r := setupTestRouter()
	r.POST("/organizations/:orgId/sections", handler.Create)
	r.PUT("/organizations/:orgId/sections/:sectionId", handler.Update)
	r.DELETE("/organizations/:orgId/sections/:sectionId", handler.Delete)

	createResp := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/sections", org.ID),
		models.SectionCreateRequest{Name: "Krippe"})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create section: %d %s", createResp.Code, createResp.Body.String())
	}
	var section models.SectionResponse
	parseResponse(t, createResp, &section)

	newName := "Elementarbereich"
	path := fmt.Sprintf("/organizations/%d/sections/%d", org.ID, section.ID)
	if w := performRequest(r, "PUT", path, models.SectionUpdateRequest{Name: &newName}); w.Code != http.StatusOK {
		t.Fatalf("update section: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("section_update"),
		ResourceType: "section",
		ResourceID:   section.ID,
	})
	oldVal, newVal := auditChangeEntry(t, row, "name")
	if oldVal != "Krippe" || newVal != "Elementarbereich" {
		t.Errorf("expected Krippe -> Elementarbereich, got %v -> %v", oldVal, newVal)
	}

	if w := performRequest(r, "DELETE", path, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete section: %d %s", w.Code, w.Body.String())
	}
	delRow := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("section_delete"),
		ResourceType: "section",
		ResourceID:   section.ID,
	})
	if got := auditSnapshotMap(t, delRow)["name"]; got != "Elementarbereich" {
		t.Errorf("expected the deleted section name in the snapshot, got %v", got)
	}
}

// seedPayPlanPeriod creates a pay plan with one period and returns both ids.
func seedPayPlanPeriod(t *testing.T, db *gorm.DB, orgID uint) (planID, periodID uint) {
	t.Helper()
	plan := &models.PayPlan{OrganizationID: orgID, Name: "TVöD-SuE"}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("seed pay plan: %v", err)
	}
	period := &models.PayPlanPeriod{
		PayPlanID:                plan.ID,
		Period:                   models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours:              39,
		EmployerContributionRate: 2200,
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("seed pay plan period: %v", err)
	}
	return plan.ID, period.ID
}

// --- users: deactivation and address changes ---

// The finding this closes: PUT /users/:userId recorded only
// `{"resource_name": "<new email>"}`. The two edits the endpoint exists to make
// — changing an account's address and switching it off — both landed as
// "somebody updated this user", with the new email as the only evidence and no
// way to tell which of the two had happened, or what the address had been.
//
// The user here has no org memberships on purpose, so exactly one
// identity-level row is written and the assertion can be exact.
func TestAudit_UserUpdate_RecordsDeactivationAndEmailChange(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	t.Cleanup(auditService.Shutdown)
	handler := NewUserHandler(createUserService(db), createUserOrganizationService(db), auditService, nil)

	user := createTestUser(t, db, "Test User", "before@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	active := false
	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d", user.ID),
		models.UserUpdateRequest{Name: "Test User", Email: "after@example.com", Active: &active})
	if w.Code != http.StatusOK {
		t.Fatalf("update user: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("user_update"),
		ResourceType: "user",
		ResourceID:   user.ID,
	})

	oldEmail, newEmail := auditChangeEntry(t, row, "email")
	if oldEmail != "before@example.com" || newEmail != "after@example.com" {
		t.Errorf("expected the address change recorded, got %v -> %v", oldEmail, newEmail)
	}
	oldActive, newActive := auditChangeEntry(t, row, "active")
	if oldActive != true || newActive != false {
		t.Errorf("expected the deactivation recorded as true -> false, got %v -> %v", oldActive, newActive)
	}
}

// A rename must not drag the account's status into the diff, or "was this
// account switched off, and when?" stops being answerable by searching for it.
func TestAudit_UserUpdate_RenameDoesNotReportUnchangedFields(t *testing.T) {
	db := setupTestDB(t)
	auditService := createAuditService(db)
	t.Cleanup(auditService.Shutdown)
	handler := NewUserHandler(createUserService(db), createUserOrganizationService(db), auditService, nil)

	user := createTestUser(t, db, "Old Name", "stable@example.com", "password")

	r := setupTestRouter()
	r.PUT("/users/:userId", handler.Update)

	w := performRequest(r, "PUT", fmt.Sprintf("/users/%d", user.ID),
		models.UserUpdateRequest{Name: "New Name"})
	if w.Code != http.StatusOK {
		t.Fatalf("update user: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("user_update"),
		ResourceType: "user",
		ResourceID:   user.ID,
	})
	details := auditDetails(t, row)
	changes, ok := details["changes"].(map[string]any)
	if !ok {
		t.Fatalf("expected a changes map for a rename: %s", row.Details)
	}
	if len(changes) != 1 {
		t.Errorf("expected only `name` to be reported, got %v", changes)
	}
	oldName, newName := auditChangeEntry(t, row, "name")
	if oldName != "Old Name" || newName != "New Name" {
		t.Errorf("expected Old Name -> New Name, got %v -> %v", oldName, newName)
	}
}

// --- bulk reads and file imports ---

// The pay plan YAML export hands over every grade and step in one file. The
// child and employee exports already emitted an audit row for a bulk read of
// personal data; this one, the bulk read of the whole salary table, did not.
func TestAudit_PayPlanExport_IsRecorded(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	handler := newPayPlanTestHandler(db)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans/:payPlanId/export", handler.Export)

	planID, _ := seedPayPlanPeriod(t, db, org.ID)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/export", org.ID, planID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("export: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("pay_plan_export"),
		ResourceType: "pay_plan",
	})
	details := auditDetails(t, row)
	if fmt.Sprint(details["record_count"]) != "1" {
		t.Errorf("expected the exported period count, got %v", details["record_count"])
	}
	if fmt.Sprint(details["organization_id"]) != fmt.Sprint(org.ID) {
		t.Errorf("expected the org on the export row, got %v", details["organization_id"])
	}
}

// The funding import wrote a bare government_funding_create or _update row: it
// recorded that a rate had been created or updated and nothing else — not which
// file it came from, and not that the import had rewritten forty Fördersätze.
//
// properties_deleted is the number worth having. A YAML import is the only path
// that removes funding properties without anyone pressing delete, and each one
// is a rate worth tens to hundreds of euros per child per month.
func TestAudit_GovernmentFundingImport_RecordsFilenameAndCounts(t *testing.T) {
	db := setupTestDB(t)
	svc := service.NewGovernmentFundingService(store.NewGovernmentFundingStore(db), store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db), importer.NewGovernmentFundingImporter(svc, store.NewTransactor(db)))

	r := setupTestRouter()
	r.POST("/fundings/import", handler.Import)

	const fundingYAML = `---
-
  from: '2023-03-01'
  to: ''
  full_time_weekly_hours: 39
  entries:
    - age: [0,2]
      properties:
        - key: care_type
          value: ganztag
          payment: 1668.47
          requirement: 0.261
        - key: care_type
          value: halbtag
          payment: 1066.64
          requirement: 0.14
`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "berlin-2024.yaml")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, strings.NewReader(fundingYAML)); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	writer.Close()

	req, _ := http.NewRequest("POST", "/fundings/import?state=berlin", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: %d %s", w.Code, w.Body.String())
	}

	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("government_funding_import"),
		ResourceType: "government_funding",
	})
	details := auditDetails(t, row)

	if details["filename"] != "berlin-2024.yaml" {
		t.Errorf("expected the uploaded filename, got %v", details["filename"])
	}
	if details["state"] != "berlin" {
		t.Errorf("expected the state, got %v", details["state"])
	}
	if details["created"] != true {
		t.Errorf("expected created=true for a fresh import, got %v", details["created"])
	}
	if fmt.Sprint(details["periods_created"]) != "1" {
		t.Errorf("expected 1 period created, got %v", details["periods_created"])
	}
	if fmt.Sprint(details["properties_created"]) != "2" {
		t.Errorf("expected 2 properties created, got %v", details["properties_created"])
	}
	if fmt.Sprint(details["record_count"]) != "3" {
		t.Errorf("expected 3 rows touched in total, got %v", details["record_count"])
	}

	// Global resource: organization_id must stay absent rather than be written
	// as a zero, which the foreign key would reject.
	if _, present := details["organization_id"]; present {
		t.Errorf("expected no organization_id on a global import, got %v", details["organization_id"])
	}
	if row.OrganizationID != nil {
		t.Errorf("expected organization_id NULL on the row, got %v", *row.OrganizationID)
	}
}

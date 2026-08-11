package handlers

// Audit diffs for contract updates.
//
// Contract update rows used to record only "contract N was updated, by whom,
// when". That is why the timeline boundary drag could strip care_type and every
// funding supplement off two contracts without leaving a trace anywhere: an
// admin asking "why did this child's funding drop in March?" had nothing to read.
// These tests pin the diff that now goes into the row's Details JSON.
//
// Pattern follows TestChildHandler_Update in child_test.go: perform the request,
// fetch the row with testutil.AssertAuditLog, unmarshal Details, assert `changes`.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

// contractAuditChangesFor pulls the `changes` map out of the audit row for a
// contract. Returns nil when the row carries no diff at all.
func contractAuditChangesFor(t *testing.T, db *gorm.DB, contractID uint, resourceType, action string) map[string]any {
	t.Helper()
	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction(action),
		ResourceType: resourceType,
		ResourceID:   contractID,
	})
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("Details JSON parse: %v (raw=%q)", err, row.Details)
	}
	changes, _ := details["changes"].(map[string]any)
	return changes
}

// oldNew reads a {old, new} pair for a field, failing if it is absent.
func oldNew(t *testing.T, changes map[string]any, field string) (any, any) {
	t.Helper()
	pair, ok := changes[field].(map[string]any)
	if !ok {
		t.Fatalf("expected a diff for %q, got changes=%+v", field, changes)
	}
	return pair["old"], pair["new"]
}

// childContractAuditFixture sets up a child with one ongoing contract carrying
// funding properties, and a router with the contract update + batch routes.
func childContractAuditFixture(t *testing.T, db *gorm.DB, orgName string) (
	*models.Organization, *models.Child, *models.ChildContract, *gin.Engine,
) {
	t.Helper()
	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, orgName)
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Audit", LastName: "Child", Gender: "female",
		Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)

	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: models.Today().AddDate(0, 0, 1), To: nil},
			SectionID:  sectionID,
			Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)
	r.PUT("/organizations/:orgId/children/:childId/contracts/batch", handler.BatchUpdateContracts)
	return org, child, contract, r
}

// A dates-only batch entry — the timeline boundary drag — must record the date
// move and must NOT report a properties change. This is the canary that was
// missing when the drag was silently wiping them.
func TestContractAudit_BatchDatesOnly_NoPropertiesDiff(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := childContractAuditFixture(t, db, "Audit DatesOnly")

	newTo := models.Today().AddDate(0, 6, 0)
	body := models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: contract.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{To: &newTo}},
		},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/batch", org.ID, child.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("batch update: status %d: %s", w.Code, w.Body.String())
	}

	changes := contractAuditChangesFor(t, db, contract.ID, "child_contract", "child_contract_update")
	if changes == nil {
		t.Fatal("expected a changes map for a dates-only batch update")
	}
	if _, present := changes["properties"]; present {
		t.Errorf("dates-only edit must not report a properties change, got %+v", changes["properties"])
	}
	old, nw := oldNew(t, changes, "to")
	if old != nil {
		t.Errorf("to.old = %v, want null (contract was ongoing)", old)
	}
	if nw == nil {
		t.Error("to.new should carry the new date")
	}
}

// When properties really do change, both sides must be recorded — this is the
// row that would have exposed the wipe.
func TestContractAudit_PropertiesChange_RecordsOldAndNew(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := childContractAuditFixture(t, db, "Audit Props")

	body := models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: contract.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				Properties: models.ContractProperties{"care_type": "ganztag"},
			}},
		},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/batch", org.ID, child.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("batch update: status %d: %s", w.Code, w.Body.String())
	}

	changes := contractAuditChangesFor(t, db, contract.ID, "child_contract", "child_contract_update")
	old, nw := oldNew(t, changes, "properties")
	oldMap, _ := old.(map[string]any)
	newMap, _ := nw.(map[string]any)
	if oldMap["care_type"] != "halbtag" {
		t.Errorf("properties.old.care_type = %v, want halbtag", oldMap["care_type"])
	}
	if oldMap["ndh"] != "ndh" {
		t.Errorf("properties.old should still show the supplement that existed, got %+v", oldMap)
	}
	if newMap["care_type"] != "ganztag" {
		t.Errorf("properties.new.care_type = %v, want ganztag", newMap["care_type"])
	}
}

// A no-op update carries no diff at all — same contract as
// TestChildHandler_Update's negative case, so the log stays readable.
func TestContractAudit_NoOpUpdate_NoChangesMap(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := childContractAuditFixture(t, db, "Audit NoOp")

	// Re-send exactly the current state.
	body := models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: contract.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				From:       &contract.From,
				Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
			}},
		},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/batch", org.ID, child.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("batch update: status %d: %s", w.Code, w.Body.String())
	}

	if changes := contractAuditChangesFor(t, db, contract.ID, "child_contract", "child_contract_update"); changes != nil {
		t.Errorf("a no-op update should carry no changes map, got %+v", changes)
	}
}

// Amend mode returns a *different* contract. The row must say so, otherwise the
// diff silently compares two identities.
func TestContractAudit_Amend_RecordsBothContractIDs(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Audit Amend")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Amend", LastName: "Child", Gender: "female",
		Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)

	// Started before today, so UpdateContract amends instead of editing.
	past := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: models.Today().AddDate(-1, 0, 0), To: nil},
			SectionID:  sectionID,
			Properties: models.ContractProperties{"care_type": "halbtag"},
		},
	}
	if err := db.Create(past).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/:contractId", handler.UpdateContract)

	body := models.ChildContractUpdateRequest{
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, past.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("update: status %d: %s", w.Code, w.Body.String())
	}

	var updated models.ChildContractResponse
	parseResponse(t, w, &updated)
	if updated.ID == past.ID {
		t.Fatalf("expected amend to create a new contract, still got id %d", updated.ID)
	}

	changes := contractAuditChangesFor(t, db, updated.ID, "child_contract", "child_contract_update")
	amended, ok := changes["amended"].(map[string]any)
	if !ok {
		t.Fatalf("expected an `amended` marker, got changes=%+v", changes)
	}
	if uint(amended["closed_contract_id"].(float64)) != past.ID {
		t.Errorf("closed_contract_id = %v, want %d", amended["closed_contract_id"], past.ID)
	}
	if uint(amended["new_contract_id"].(float64)) != updated.ID {
		t.Errorf("new_contract_id = %v, want %d", amended["new_contract_id"], updated.ID)
	}
}

// A property-less contract must not gain a properties diff from an edit that
// hands it an empty map.
//
// Note this passes even without recordPropertiesChange's nil-vs-empty
// short-circuit, because the response DTO carries `properties,omitempty` so an
// empty map round-trips as nil and DeepEqual already reports equality. The
// short-circuit itself is guarded directly in
// TestRecordPropertiesChange (audit_diff_helpers_test.go). This test covers the
// end-to-end behaviour a user sees.
func TestContractAudit_NilVsEmptyProperties_NotAChange(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	childService := createChildService(db)
	handler := NewChildHandler(childService, createAuditService(db))

	org := createTestOrganization(t, db, "Audit NilEmpty")
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID, FirstName: "NilEmpty", LastName: "Child", Gender: "female",
		Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)

	// No properties at all, and starting tomorrow so the update stays in place.
	contract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: models.Today().AddDate(0, 0, 1), To: nil},
			SectionID: sectionID,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/children/:childId/contracts/batch", handler.BatchUpdateContracts)

	// Hand it an explicitly empty map while moving a date.
	newTo := models.Today().AddDate(0, 3, 0)
	body := models.ChildContractBatchUpdateRequest{
		Updates: []models.ChildContractBatchUpdateEntry{
			{ID: contract.ID, ChildContractUpdateRequest: models.ChildContractUpdateRequest{
				To:         &newTo,
				Properties: models.ContractProperties{},
			}},
		},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/batch", org.ID, child.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("batch update: status %d: %s", w.Code, w.Body.String())
	}

	changes := contractAuditChangesFor(t, db, contract.ID, "child_contract", "child_contract_update")
	if _, present := changes["properties"]; present {
		t.Errorf("nil -> empty must not be recorded as a properties change, got %+v", changes["properties"])
	}
	// The date move is still recorded, so this is not passing by writing nothing.
	if _, ok := changes["to"]; !ok {
		t.Errorf("expected the date move to be recorded, got changes=%+v", changes)
	}
}

// Employee contracts diff five fields the child ones do not: staff_category,
// grade, step, weekly_hours and payplan_id. employeeContractChanges was wired up
// without a test — this covers it.
func TestContractAudit_EmployeeSalaryFields_Recorded(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	employeeService := createEmployeeService(db)
	handler := NewEmployeeHandler(employeeService, createAuditService(db))

	org := createTestOrganization(t, db, "Audit Emp Salary")
	sectionID := ensureTestSection(t, db, org.ID)
	payPlanID := ensureTestPayPlan(t, db, org.ID)
	employee := &models.Employee{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Salary", LastName: "Employee", Gender: "male",
		Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(employee)

	contract := &models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: models.Today().AddDate(0, 0, 1), To: nil},
			SectionID:  sectionID,
			Properties: models.ContractProperties{"note": "before"},
		},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          2,
		WeeklyHours:   30,
		PayPlanID:     payPlanID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := setupTestRouter()
	r.PUT("/organizations/:orgId/employees/:employeeId/contracts/batch", handler.BatchUpdateContracts)

	newGrade := "S8b"
	newStep := 3
	newHours := 35.0
	newCategory := "supplementary"
	body := models.EmployeeContractBatchUpdateRequest{
		Updates: []models.EmployeeContractBatchUpdateEntry{
			{ID: contract.ID, EmployeeContractUpdateRequest: models.EmployeeContractUpdateRequest{
				Grade:         &newGrade,
				Step:          &newStep,
				WeeklyHours:   &newHours,
				StaffCategory: &newCategory,
			}},
		},
	}
	w := performRequest(r, "PUT",
		fmt.Sprintf("/organizations/%d/employees/%d/contracts/batch", org.ID, employee.ID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("batch update: status %d: %s", w.Code, w.Body.String())
	}

	changes := contractAuditChangesFor(t, db, contract.ID, "employee_contract", "employee_contract_update")
	if changes == nil {
		t.Fatal("expected a changes map for an employee contract update")
	}

	for _, tc := range []struct {
		field     string
		old, want any
	}{
		{"grade", "S8a", "S8b"},
		{"step", float64(2), float64(3)},
		{"weekly_hours", float64(30), float64(35)},
		{"staff_category", "qualified", "supplementary"},
	} {
		old, nw := oldNew(t, changes, tc.field)
		if old != tc.old {
			t.Errorf("%s.old = %v (%T), want %v", tc.field, old, old, tc.old)
		}
		if nw != tc.want {
			t.Errorf("%s.new = %v (%T), want %v", tc.field, nw, nw, tc.want)
		}
	}

	// Untouched fields must not appear — the salary diff should be readable.
	if _, present := changes["properties"]; present {
		t.Errorf("properties were not changed, should not be recorded: %+v", changes["properties"])
	}
	if _, present := changes["payplan_id"]; present {
		t.Errorf("payplan_id was not changed, should not be recorded: %+v", changes["payplan_id"])
	}
}

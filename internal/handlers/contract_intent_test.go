package handlers

// HTTP-level tests for the intent endpoints.
//
// These go through gin's JSON binding on purpose. The service-level tests
// construct Opt[T] values in Go, which proves the service logic but not the
// wiring — and "absent" only exists on the wire. A request body that mentions
// `section_id` and not `properties` is exactly the payload that used to strip a
// contract's care type and supplements, so it is asserted here as raw JSON.
//
// The audit assertions matter for a second reason: with explicit intents the log
// no longer has to *infer* what happened by comparing contract ids, so what it
// records should be checked against what was actually asked for.

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

// childIntentRouter seeds a child with one contract and registers the intent
// routes exactly as internal/routes does, including the boundary route ahead of
// the `:contractId` ones.
func childIntentRouter(t *testing.T, db *gorm.DB, orgName string, from time.Time, to *time.Time) (
	*models.Organization, *models.Child, *models.ChildContract, *gin.Engine,
) {
	t.Helper()
	handler := NewChildHandler(createChildService(db), createAuditService(db))

	org := createTestOrganization(t, db, orgName)
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Intent", LastName: "Child", Gender: "female",
		Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("seed child: %v", err)
	}

	contract := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: from, To: to},
		SectionID:  sectionID,
		Properties: models.ContractProperties{"care_type": "halbtag", "ndh": "ndh"},
	}}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/contracts/boundary", handler.MoveContractBoundary)
	r.PATCH("/organizations/:orgId/children/:childId/contracts/:contractId", handler.CorrectContract)
	r.POST("/organizations/:orgId/children/:childId/contracts/:contractId/amend", handler.AmendContract)
	r.POST("/organizations/:orgId/children/:childId/contracts/:contractId/end", handler.EndContract)
	return org, child, contract, r
}

// The payload the kanban board sends. On the old PUT this cleared `to` and every
// funding property; here the omission has to mean nothing at all.
func TestContractIntent_Patch_SectionOnly_KeepsEverythingElse(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	org, child, contract, r := childIntentRouter(t, db, "Patch Section", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)
	other := models.Section{OrganizationID: org.ID, Name: "Elefanten"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed second section: %v", err)
	}

	w := performRequestRaw(r, "PATCH",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID),
		fmt.Sprintf(`{"section_id": %d}`, other.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH: status %d: %s", w.Code, w.Body.String())
	}

	var resp models.ChildContractResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.SectionID != other.ID {
		t.Errorf("section_id = %d, want %d", resp.SectionID, other.ID)
	}
	if resp.To == nil || !resp.To.Equal(end) {
		t.Errorf("to = %v, want %v — an omitted field must survive the round trip", resp.To, end)
	}
	if resp.Properties["care_type"] != "halbtag" || resp.Properties["ndh"] != "ndh" {
		t.Errorf("properties = %v, want care_type and ndh intact", resp.Properties)
	}

	// And the audit row names the one thing that changed, not a properties wipe.
	changes := contractAuditChangesFor(t, db, contract.ID, "child_contract", "child_contract_update")
	if _, present := changes["properties"]; present {
		t.Errorf("a section-only correction must not report a properties change: %+v", changes["properties"])
	}
	if _, present := changes["to"]; present {
		t.Errorf("a section-only correction must not report a `to` change: %+v", changes["to"])
	}
	if _, _ = oldNew(t, changes, "section_id"); len(changes) != 1 {
		t.Errorf("expected exactly one changed field, got %+v", changes)
	}
}

// Null on the wire has to reach the service as "clear", not as "absent".
func TestContractIntent_Patch_ExplicitNullClearsTo(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	org, child, contract, r := childIntentRouter(t, db, "Patch Null", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)

	w := performRequestRaw(r, "PATCH",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID),
		`{"to": null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH: status %d: %s", w.Code, w.Body.String())
	}

	var resp models.ChildContractResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.To != nil {
		t.Errorf("to = %v, want nil", resp.To)
	}
	if resp.Properties["care_type"] != "halbtag" {
		t.Errorf("properties = %v, want untouched by a `to`-only clear", resp.Properties)
	}
}

// An amendment is two facts — one contract closed, another created — and the log
// records both, cross-linked, with the closed row carrying its own date diff.
func TestContractIntent_Amend_RecordsUpdateAndCreatePair(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := childIntentRouter(t, db, "Amend Audit", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)

	w := performRequestRaw(r, "POST",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d/amend", org.ID, child.ID, contract.ID),
		`{"effective_from": "2026-05-01T00:00:00Z", "properties": {"care_type": "ganztag"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("amend: status %d: %s", w.Code, w.Body.String())
	}

	var resp models.ChildContractAmendResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Closed.ID != contract.ID {
		t.Errorf("closed id = %d, want the addressed contract %d", resp.Closed.ID, contract.ID)
	}
	if resp.Created.ID == contract.ID {
		t.Error("created id equals the addressed contract; the successor should be a new row")
	}

	// The closed contract: an update, with the link and its own `to` diff.
	changes := contractAuditChangesFor(t, db, resp.Closed.ID, "child_contract", "child_contract_update")
	link, ok := changes["amended"].(map[string]any)
	if !ok {
		t.Fatalf("closed contract's row carries no `amended` link: %+v", changes)
	}
	if uint(link["closed_contract_id"].(float64)) != resp.Closed.ID ||
		uint(link["new_contract_id"].(float64)) != resp.Created.ID {
		t.Errorf("link = %+v, want closed=%d new=%d", link, resp.Closed.ID, resp.Created.ID)
	}
	if _, nw := oldNew(t, changes, "to"); nw == nil {
		t.Error("the closed contract's new end date should be recorded")
	}

	// The successor: a create, not an update of something that never existed.
	testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_contract_create"),
		ResourceType: "child_contract",
		ResourceID:   resp.Created.ID,
	})
}

// One drag, two contracts changed. Both rows have to say so, or an admin reading
// the log sees two unrelated date edits seconds apart.
func TestContractIntent_Boundary_RecordsBothSidesLinked(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, first, r := childIntentRouter(t, db, "Boundary Audit",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)))

	second := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		SectionID:  first.SectionID,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}}
	if err := db.Create(second).Error; err != nil {
		t.Fatalf("seed second: %v", err)
	}

	w := performRequestRaw(r, "POST",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/boundary", org.ID, child.ID),
		fmt.Sprintf(`{"earlier_id": %d, "later_id": %d, "at": "2026-03-01T00:00:00Z"}`, first.ID, second.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("boundary: status %d: %s", w.Code, w.Body.String())
	}

	for _, tc := range []struct {
		id   uint
		role string
	}{{first.ID, "earlier"}, {second.ID, "later"}} {
		changes := contractAuditChangesFor(t, db, tc.id, "child_contract", "child_contract_update")
		link, ok := changes["boundary_moved"].(map[string]any)
		if !ok {
			t.Fatalf("contract %d has no boundary_moved link: %+v", tc.id, changes)
		}
		if link["side"] != tc.role {
			t.Errorf("contract %d side = %v, want %q", tc.id, link["side"], tc.role)
		}
		if uint(link["earlier_contract_id"].(float64)) != first.ID ||
			uint(link["later_contract_id"].(float64)) != second.ID {
			t.Errorf("contract %d link = %+v, want earlier=%d later=%d", tc.id, link, first.ID, second.ID)
		}
	}

	// The earlier side's date really moved, and the later side kept its own
	// open end — the failure mode the old batch payload produced.
	var reloadedLater models.ChildContract
	if err := db.First(&reloadedLater, second.ID).Error; err != nil {
		t.Fatalf("reload later: %v", err)
	}
	if reloadedLater.To != nil {
		t.Errorf("later contract to = %v, want still open-ended", reloadedLater.To)
	}
	if !reloadedLater.From.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("later contract from = %v, want 2026-03-01", reloadedLater.From)
	}
}

// The create path's `weekly_hours` fix hinges on a subtlety of the binding
// layer, so it is asserted there rather than in the service: `required` on a
// float64 rejects 0, but on a *float64 it only checks non-nil. A 0-hour contract
// (parental leave) must therefore bind, while an omitted field must still 400.
func TestEmployeeContractCreate_WeeklyHoursBinding(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	handler := NewEmployeeHandler(createEmployeeService(db), createAuditService(db))
	org := createTestOrganization(t, db, "Hours Binding Org")
	sectionID := ensureTestSection(t, db, org.ID)
	payPlanID := ensureTestPayPlan(t, db, org.ID)
	employee := &models.Employee{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Hours", LastName: "Employee", Gender: "male",
		Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("seed employee: %v", err)
	}

	r := setupTestRouter()
	r.POST("/organizations/:orgId/employees/:employeeId/contracts", handler.CreateContract)
	path := fmt.Sprintf("/organizations/%d/employees/%d/contracts", org.ID, employee.ID)

	t.Run("zero binds and is created", func(t *testing.T) {
		body := fmt.Sprintf(`{"from":"2026-01-01T00:00:00Z","section_id":%d,"staff_category":"qualified",
			"grade":"S8a","step":3,"weekly_hours":0,"payplan_id":%d}`, sectionID, payPlanID)
		w := performRequestRaw(r, "POST", path, body)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
		var resp models.EmployeeContractResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.WeeklyHours != 0 {
			t.Errorf("weekly_hours = %v, want 0", resp.WeeklyHours)
		}
	})

	t.Run("absent is rejected", func(t *testing.T) {
		body := fmt.Sprintf(`{"from":"2027-01-01T00:00:00Z","section_id":%d,"staff_category":"qualified",
			"grade":"S8a","step":3,"payplan_id":%d}`, sectionID, payPlanID)
		w := performRequestRaw(r, "POST", path, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
		}
	})
}

// `{}` on the end endpoint is not "make it ongoing" — that was the ambiguity.
func TestContractIntent_End_EmptyBodyRejected(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := childIntentRouter(t, db, "End Empty", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)

	w := performRequestRaw(r, "POST",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d/end", org.ID, child.ID, contract.ID), `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
	}
}

// Reopening: `{"to": null}` is the deliberate form, and it has to work through
// binding as well as in Go.
func TestContractIntent_End_NullReopens(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	org, child, contract, r := childIntentRouter(t, db, "End Reopen", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end)

	w := performRequestRaw(r, "POST",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d/end", org.ID, child.ID, contract.ID),
		`{"to": null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	var resp models.ChildContractResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.To != nil {
		t.Errorf("to = %v, want nil", resp.To)
	}
}

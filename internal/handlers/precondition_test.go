package handlers

// If-Match preconditions on contract writes.
//
// The version column and the version-guarded stores were added earlier, but on
// their own they protect nothing: a stale write only fails once the client says
// which version it is editing. These tests pin that it must say so, that a wrong
// answer is refused, and that the refusal uses the status matching the situation
// — 428 when nothing was compared, 412 when the comparison failed.
//
// Tested through HTTP because the precondition is a header: parsing, the
// required-ness, and the status codes are all transport behaviour.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// requestWithHeaders performs a request with an arbitrary header set, which
// performRequestRaw does not allow.
func requestWithHeaders(r *gin.Engine, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := mustNewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// preconditionFixture is a child with one contract, plus the intent routes and
// the read routes needed to discover the contract's version.
func preconditionFixture(t *testing.T, db *gorm.DB, orgName string) (
	*models.Organization, *models.Child, *models.ChildContract, *gin.Engine,
) {
	t.Helper()
	handler := NewChildHandler(createChildService(db), createAuditService(db))

	org := createTestOrganization(t, db, orgName)
	sectionID := ensureTestSection(t, db, org.ID)
	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID, FirstName: "Precondition", LastName: "Child", Gender: "female",
		Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("seed child: %v", err)
	}
	contract := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		SectionID:  sectionID,
		Properties: models.ContractProperties{"care_type": "halbtag"},
	}}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/contracts/:contractId", handler.GetContract)
	r.PATCH("/organizations/:orgId/children/:childId/contracts/:contractId", handler.CorrectContract)
	r.POST("/organizations/:orgId/children/:childId/contracts/:contractId/amend", handler.AmendContract)
	r.POST("/organizations/:orgId/children/:childId/contracts/:contractId/end", handler.EndContract)
	r.DELETE("/organizations/:orgId/children/:childId/contracts/:contractId", handler.DeleteContract)
	r.POST("/organizations/:orgId/children/:childId/contracts/boundary", handler.MoveContractBoundary)
	return org, child, contract, r
}

// A client can discover the precondition from the resource it just read, which is
// the whole point of publishing it — otherwise If-Match would be unusable.
func TestPrecondition_GetPublishesETagAndVersion(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "ETag Org")

	w := performRequest(r, "GET",
		fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET: %d: %s", w.Code, w.Body.String())
	}

	if got, want := w.Header().Get("ETag"), strconv.Quote("1"); got != want {
		t.Errorf("ETag = %q, want %q (quoted per RFC 9110)", got, want)
	}

	var resp models.ChildContractResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Version != 1 {
		t.Errorf("version = %d, want 1", resp.Version)
	}
}

// Every contract write must state a precondition. 428 rather than 400 because
// nothing was compared: the client has to read the contract and try again, which
// is a different instruction from "you lost a race".
func TestPrecondition_MissingIfMatchIs428(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Missing IfMatch Org")
	base := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	for name, tc := range map[string]struct {
		method, path, body string
	}{
		"correct": {"PATCH", base, `{"section_id": 1}`},
		"amend":   {"POST", base + "/amend", `{"effective_from":"2026-05-01T00:00:00Z"}`},
		"end":     {"POST", base + "/end", `{"to":"2026-06-30T00:00:00Z"}`},
		"delete":  {"DELETE", base, ""},
	} {
		t.Run(name, func(t *testing.T) {
			w := performRequestRaw(r, tc.method, tc.path, tc.body)
			if w.Code != http.StatusPreconditionRequired {
				t.Fatalf("status = %d, want 428: %s", w.Code, w.Body.String())
			}
			var body models.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Code != "precondition_required" {
				t.Errorf("code = %q, want precondition_required", body.Code)
			}
		})
	}
}

// A stale version is refused, and nothing is written. This is the lost update the
// version column exists to prevent.
func TestPrecondition_StaleIfMatchIs412AndChangesNothing(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Stale IfMatch Org")
	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	// Editor one succeeds, taking the contract to version 2.
	w := requestWithHeaders(r, "PATCH", path, `{"properties":{"care_type":"ganztag"}}`,
		map[string]string{"If-Match": `"1"`})
	if w.Code != http.StatusOK {
		t.Fatalf("first correction: %d: %s", w.Code, w.Body.String())
	}

	// Editor two still holds version 1.
	w = requestWithHeaders(r, "PATCH", path, `{"properties":{"care_type":"teilzeit"}}`,
		map[string]string{"If-Match": `"1"`})
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale correction: status = %d, want 412: %s", w.Code, w.Body.String())
	}
	var body models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "precondition_failed" {
		t.Errorf("code = %q, want precondition_failed", body.Code)
	}

	// The first editor's change is intact — a refused write must not partially land.
	var reloaded models.ChildContract
	if err := db.First(&reloaded, contract.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag", reloaded.Properties["care_type"])
	}
	if reloaded.Version != 2 {
		t.Errorf("version = %d, want 2 (a refused write must not bump it)", reloaded.Version)
	}
}

// The current version is accepted, and the response advertises the new one so a
// client can make a second edit without re-reading.
func TestPrecondition_CurrentIfMatchSucceedsAndAdvancesVersion(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Fresh IfMatch Org")
	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	w := requestWithHeaders(r, "POST", path+"/end", `{"to":"2026-06-30T00:00:00Z"}`,
		map[string]string{"If-Match": `"1"`})
	if w.Code != http.StatusOK {
		t.Fatalf("end: %d: %s", w.Code, w.Body.String())
	}
	var resp models.ChildContractResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Version != 2 {
		t.Errorf("response version = %d, want 2", resp.Version)
	}

	// And the new version is the one that now works.
	w = requestWithHeaders(r, "POST", path+"/end", `{"to":null}`,
		map[string]string{"If-Match": `"2"`})
	if w.Code != http.StatusOK {
		t.Fatalf("second end with version 2: %d: %s", w.Code, w.Body.String())
	}
}

// Header forms that cannot mean a single exact version are rejected rather than
// interpreted, because guessing here would silently weaken the guarantee.
func TestPrecondition_MalformedIfMatch(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Malformed IfMatch Org")
	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	for name, header := range map[string]string{
		"weak validator": `W/"1"`,
		"list of tags":   `"1", "2"`,
		"not a number":   `"abc"`,
		"version zero":   `"0"`,
		"negative":       `"-1"`,
		"empty quotes":   `""`,
	} {
		t.Run(name, func(t *testing.T) {
			w := requestWithHeaders(r, "PATCH", path, `{"section_id":1}`,
				map[string]string{"If-Match": header})
			if w.Code != http.StatusBadRequest {
				t.Errorf("If-Match: %s → status %d, want 400: %s", header, w.Code, w.Body.String())
			}
		})
	}
}

// `*` is RFC 9110's "any current version" — an explicit opt-out, not an accident,
// so it is honoured. It is the one way to write without knowing the version.
func TestPrecondition_WildcardIfMatchAccepted(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Wildcard Org")
	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	w := requestWithHeaders(r, "PATCH", path, `{"properties":{"care_type":"ganztag"}}`,
		map[string]string{"If-Match": "*"})
	if w.Code != http.StatusOK {
		t.Fatalf("wildcard: %d: %s", w.Code, w.Body.String())
	}
}

// A seam move changes two contracts, so one header cannot describe it: the
// versions travel in the body, and both are required.
func TestPrecondition_BoundaryRequiresBothVersions(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, first, r := preconditionFixture(t, db, "Boundary Versions Org")
	// Give the first contract an end date and add an adjacent successor.
	end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	first.To = &end
	if err := db.Save(first).Error; err != nil {
		t.Fatalf("close first: %v", err)
	}
	second := &models.ChildContract{ChildID: child.ID, BaseContract: models.BaseContract{
		Period:     models.Period{From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		SectionID:  first.SectionID,
		Properties: models.ContractProperties{"care_type": "ganztag"},
	}}
	if err := db.Create(second).Error; err != nil {
		t.Fatalf("seed second: %v", err)
	}

	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/boundary", org.ID, child.ID)

	// Missing versions: rejected by binding, before anything is touched.
	w := performRequestRaw(r, "POST", path,
		fmt.Sprintf(`{"earlier_id":%d,"later_id":%d,"at":"2026-03-01T00:00:00Z"}`, first.ID, second.ID))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing versions: status = %d, want 400: %s", w.Code, w.Body.String())
	}

	// A stale version on either side is a 412, named so the user can tell which
	// contract moved on.
	w = performRequestRaw(r, "POST", path, fmt.Sprintf(
		`{"earlier_id":%d,"later_id":%d,"at":"2026-03-01T00:00:00Z","earlier_version":99,"later_version":%d}`,
		first.ID, second.ID, second.Version))
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale earlier version: status = %d, want 412: %s", w.Code, w.Body.String())
	}

	// Correct versions: the move goes through. `first` was saved once above, so it
	// is on version 2 — read them from the DB rather than assuming.
	var e, l models.ChildContract
	if err := db.First(&e, first.ID).Error; err != nil {
		t.Fatalf("reload earlier: %v", err)
	}
	if err := db.First(&l, second.ID).Error; err != nil {
		t.Fatalf("reload later: %v", err)
	}
	w = performRequestRaw(r, "POST", path, fmt.Sprintf(
		`{"earlier_id":%d,"later_id":%d,"at":"2026-03-01T00:00:00Z","earlier_version":%d,"later_version":%d}`,
		first.ID, second.ID, e.Version, l.Version))
	if w.Code != http.StatusOK {
		t.Fatalf("correct versions: status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// Deleting is the write where a lost update is least recoverable, so it carries
// the precondition too — and a stale one must leave the contract in place.
func TestPrecondition_DeleteWithStaleVersionKeepsContract(t *testing.T) {
	db := setupTestDB(t)
	restore := models.SetNow(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC))
	defer restore()

	org, child, contract, r := preconditionFixture(t, db, "Delete IfMatch Org")
	path := fmt.Sprintf("/organizations/%d/children/%d/contracts/%d", org.ID, child.ID, contract.ID)

	// Someone edits it, so the deleter's version 1 is now stale.
	w := requestWithHeaders(r, "PATCH", path, `{"properties":{"care_type":"ganztag"}}`,
		map[string]string{"If-Match": `"1"`})
	if w.Code != http.StatusOK {
		t.Fatalf("setup correction: %d: %s", w.Code, w.Body.String())
	}

	w = requestWithHeaders(r, "DELETE", path, "", map[string]string{"If-Match": `"1"`})
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale delete: status = %d, want 412: %s", w.Code, w.Body.String())
	}

	var still models.ChildContract
	if err := db.First(&still, contract.ID).Error; err != nil {
		t.Fatalf("the contract should still exist: %v", err)
	}

	// With the current version it goes.
	w = requestWithHeaders(r, "DELETE", path, "", map[string]string{"If-Match": `"2"`})
	if w.Code != http.StatusNoContent {
		t.Fatalf("current delete: status = %d, want 204: %s", w.Code, w.Body.String())
	}
}

// ifMatch builds the header set for a write that states a specific version.
func ifMatch(version int64) map[string]string {
	return map[string]string{"If-Match": strconv.Quote(strconv.FormatInt(version, 10))}
}

// anyVersion is for tests asserting something other than concurrency: `*` says
// "whatever the current version is" explicitly, rather than hard-coding a number
// that later edits to the fixture would silently invalidate.
var anyVersion = map[string]string{"If-Match": "*"}

package handlers

// The children list's `active_on` filter, tested at the HTTP layer.
//
// The service-level filter was already covered
// (TestChildService_ListByOrganizationAndSection_ActiveOn). What was not covered
// was the wire: that the query parameter is *named* active_on, that it is parsed,
// and that it overrides the handler's default of today.
//
// That gap is exactly how a bug survived. The frontend sent `contract_on`, which
// no endpoint declares. Gin drops unknown query parameters silently, so the
// request succeeded, the handler fell back to today, and the section board's date
// picker did nothing — moving it to another date returned the same roster. Nothing
// failed, and the existing tests all passed, because none of them asked the
// endpoint for a date other than today.
//
// Each test below distinguishes "the queried date" from "today", so a regression
// to the default cannot pass.

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// activeOnFixture builds two children whose contracts do not overlap: one that was
// enrolled in spring 2025 and left, one enrolled from today. Any query has to pick
// exactly one of them, which is what makes the assertions sharp.
func activeOnFixture(t *testing.T) (uint, *models.Child, *models.Child, *models.Child, func(string) []models.Child) {
	t.Helper()
	db := setupTestDB(t)
	handler := NewChildHandler(createChildService(db), createAuditService(db))

	org := createTestOrganization(t, db, "ActiveOn Org")
	sectionID := ensureTestSection(t, db, org.ID)

	mk := func(name string) *models.Child {
		c := &models.Child{Person: models.Person{
			OrganizationID: org.ID, FirstName: name, LastName: "Test", Gender: "female",
			Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
		return c
	}

	past := mk("Past")
	pastFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pastTo := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ChildContract{ChildID: past.ID, BaseContract: models.BaseContract{
		Period: models.Period{From: pastFrom, To: &pastTo}, SectionID: sectionID,
	}}).Error; err != nil {
		t.Fatalf("seed past contract: %v", err)
	}

	current := mk("Current")
	if err := db.Create(&models.ChildContract{ChildID: current.ID, BaseContract: models.BaseContract{
		Period: models.Period{From: models.Today()}, SectionID: sectionID,
	}}).Error; err != nil {
		t.Fatalf("seed current contract: %v", err)
	}

	// A child with no contract at all must never appear for any date.
	none := mk("NoContract")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	list := func(query string) []models.Child {
		t.Helper()
		url := fmt.Sprintf("/organizations/%d/children", org.ID)
		if query != "" {
			url += "?" + query
		}
		w := performRequest(r, "GET", url, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d: %s", url, w.Code, w.Body.String())
		}
		var resp models.PaginatedResponse[models.Child]
		parseResponse(t, w, &resp)
		return resp.Data
	}
	return org.ID, past, current, none, list
}

func names(children []models.Child) []string {
	out := make([]string, 0, len(children))
	for _, c := range children {
		out = append(out, c.FirstName)
	}
	return out
}

// The parameter has to actually narrow the result to the date asked for — not to
// today. With the old `contract_on` this returned "Current", because the unknown
// parameter was dropped and the handler defaulted.
func TestChildHandler_List_ActiveOnSelectsTheQueriedDate(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	_, _, _, _, list := activeOnFixture(t)

	got := list("active_on=2025-03-01")
	if len(got) != 1 || got[0].FirstName != "Past" {
		t.Fatalf("active_on=2025-03-01 returned %v, want exactly [Past] — the child enrolled on that date", names(got))
	}
}

// The complement: today's roster is a different set. Asserting both directions is
// what makes the first test meaningful, since a filter that always returned "Past"
// would otherwise pass it.
func TestChildHandler_List_ActiveOnToday(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	_, _, _, _, list := activeOnFixture(t)

	got := list(fmt.Sprintf("active_on=%s", models.Today().Format("2006-01-02")))
	if len(got) != 1 || got[0].FirstName != "Current" {
		t.Fatalf("active_on=today returned %v, want exactly [Current]", names(got))
	}
}

// Omitting the parameter defaults to today, which is the documented behaviour and
// the reason the bug was invisible: the wrong parameter name and no parameter at
// all produce identical responses.
func TestChildHandler_List_DefaultsToToday(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	_, _, _, _, list := activeOnFixture(t)

	got := list("")
	if len(got) != 1 || got[0].FirstName != "Current" {
		t.Fatalf("no filter returned %v, want exactly [Current] (the endpoint defaults to today)", names(got))
	}
}

// A date on which nobody was enrolled returns nothing rather than falling back to
// today — the fallback is what a dropped parameter looks like.
func TestChildHandler_List_ActiveOnDateWithNobodyEnrolled(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	_, _, _, _, list := activeOnFixture(t)

	if got := list("active_on=2024-01-01"); len(got) != 0 {
		t.Errorf("active_on=2024-01-01 returned %v, want none", names(got))
	}
}

// Boundary days are inclusive at both ends, which is the same rule contracts use
// everywhere else.
func TestChildHandler_List_ActiveOnIsInclusive(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	_, _, _, _, list := activeOnFixture(t)

	for _, day := range []string{"2025-01-01", "2025-06-30"} {
		got := list("active_on=" + day)
		if len(got) != 1 || got[0].FirstName != "Past" {
			t.Errorf("active_on=%s returned %v, want [Past] — first and last day count as active", day, names(got))
		}
	}
	// And the day either side does not.
	for _, day := range []string{"2024-12-31", "2025-07-01"} {
		if got := list("active_on=" + day); len(got) != 0 {
			t.Errorf("active_on=%s returned %v, want none — outside the contract", day, names(got))
		}
	}
}

// A malformed date is a client error, not a silent fallback to today.
func TestChildHandler_List_ActiveOnRejectsMalformedDate(t *testing.T) {
	restore := models.SetNow(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC))
	defer restore()

	db := setupTestDB(t)
	handler := NewChildHandler(createChildService(db), createAuditService(db))
	org := createTestOrganization(t, db, "Malformed Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children", handler.List)

	w := performRequest(r, "GET",
		fmt.Sprintf("/organizations/%d/children?active_on=not-a-date", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a malformed active_on: %s", w.Code, w.Body.String())
	}
}

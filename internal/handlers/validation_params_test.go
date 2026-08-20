package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/eenemeene/kitamanager-go/internal/i18n"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// TestInvalidParamsReportsEveryFailingField covers the second producer of
// multiple violations.
//
// The bulk-import checks build their violations explicitly, and that path is
// tested in internal/problem. This one comes from validator/v10, which reports
// *every* failing field rather than the first — and until now only the
// single-field case had a test, so nothing proved the response carried more than
// one, nor that each carried its own localized reason.
func TestInvalidParamsReportsEveryFailingField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type req struct {
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
	}

	err := binding.Validator.ValidateStruct(&req{Email: "not-an-email"})
	if err == nil {
		t.Fatal("expected the validator to reject the struct")
	}

	// Through the middleware, so the localizer is the request-scoped one; a bare
	// context silently yields English and would make this test look like it
	// passed for the wrong reason.
	var params []struct{ Field, Rule, Reason, Localized string }
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		for _, p := range invalidParams(c, err) {
			params = append(params, struct{ Field, Rule, Reason, Localized string }{
				p.Field, p.Rule, p.Reason, p.LocalizedReason,
			})
		}
		c.Status(http.StatusOK)
	})
	httpReq := httptest.NewRequest(http.MethodGet, "/x", nil)
	httpReq.Header.Set("Accept-Language", "de")
	r.ServeHTTP(httptest.NewRecorder(), httpReq)

	if len(params) != 3 {
		t.Fatalf("got %d invalid params, want 3 (two missing, one malformed): %+v", len(params), params)
	}

	// The wire name, not the Go name. The frontend resolves `field` as a JSON
	// path against the form's values to mark the offending input, so "FirstName"
	// resolves to nothing and the violation silently goes unmarked. Asserting
	// only that the field is non-empty is what let that through: it passed just
	// as happily on the Go names validator reports by default.
	wantFields := map[string]bool{"first_name": true, "last_name": true, "email": true}
	for _, p := range params {
		if !wantFields[p.Field] {
			t.Errorf("field %q is not a JSON name — the frontend cannot map it to an input", p.Field)
		}
	}

	for _, p := range params {
		if p.Field == "" || p.Rule == "" || p.Reason == "" {
			t.Errorf("incomplete entry %+v", p)
		}
		if p.Localized == "" {
			t.Errorf("entry %q has no German reason, so a German user sees English for that field", p.Field)
		}
		if p.Localized == p.Reason {
			t.Errorf("entry %q was not translated: %q", p.Field, p.Localized)
		}
	}
}

// TestInvalidParamsIsEnglishForEnglishRequests keeps the localized member absent
// rather than echoing the English into it, matching how `localized` behaves at
// the document level.
func TestInvalidParamsIsEnglishForEnglishRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type req struct {
		Name string `json:"name" binding:"required"`
	}
	err := binding.Validator.ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected the validator to reject the struct")
	}

	var localized string
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		localized = invalidParams(c, err)[0].LocalizedReason
		c.Status(http.StatusOK)
	})
	httpReq := httptest.NewRequest(http.MethodGet, "/x", nil)
	httpReq.Header.Set("Accept-Language", "en")
	r.ServeHTTP(httptest.NewRecorder(), httpReq)

	if localized != "" {
		t.Errorf("localized_reason = %q for an English request, want it omitted", localized)
	}
}

// TestForecastOverlayBindingReportsJSONPaths pins the forecast overlay's binding
// tags to the paths its service validators produce.
//
// The overlay is checked twice: by binding tags at the request boundary, and by
// the forecast service, which is a public method that does not assume it was
// reached over HTTP. Two checkers are only tolerable while they agree, and the
// thing a client depends on is the field path — `add_children[0].contracts[1].from`
// is what the frontend maps onto a form input.
//
// Two details make this worth pinning rather than assuming. Slice elements are
// not validated without an explicit `dive`, so dropping that tag would silently
// stop checking every nested contract. And validator spells `min` for both a
// string length and a numeric floor, so a weekly-hours minimum would otherwise
// be reported as a character count.
func TestForecastOverlayBindingReportsJSONPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := models.ForecastRequest{
		AddChildren: []models.ForecastChildInput{{
			Contracts: []models.ForecastChildContractInput{{}},
		}},
		AddEmployees: []models.ForecastEmployeeInput{{
			Contracts: []models.ForecastEmployeeContractInput{{}},
		}},
	}

	err := binding.Validator.ValidateStruct(&req)
	if err == nil {
		t.Fatal("expected the validator to reject an overlay with empty contracts")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	got := make(map[string]string)
	for _, p := range invalidParams(c, err) {
		got[p.Field] = p.Rule
	}

	want := map[string]string{
		"add_children[0].birthdate":                    "required",
		"add_children[0].contracts[0].from":            "required",
		"add_children[0].contracts[0].section_id":      "required",
		"add_employees[0].contracts[0].from":           "required",
		"add_employees[0].contracts[0].section_id":     "required",
		"add_employees[0].contracts[0].staff_category": "required",
		"add_employees[0].contracts[0].grade":          "required",
		"add_employees[0].contracts[0].payplan_id":     "required",
		"add_employees[0].contracts[0].step":           "min_value",
		"add_employees[0].contracts[0].weekly_hours":   "positive",
	}
	for field, rule := range want {
		switch actual, ok := got[field]; {
		case !ok:
			t.Errorf("no violation reported for %s", field)
		case actual != rule:
			t.Errorf("%s: got rule %q, want %q", field, actual, rule)
		}
	}
	for field := range got {
		if _, ok := want[field]; !ok {
			t.Errorf("unexpected violation for %s", field)
		}
	}
}

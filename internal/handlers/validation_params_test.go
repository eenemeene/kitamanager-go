package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestForecastOverlayBindingRejectsIncompleteOverlays is the field-presence
// sweep that used to live in internal/service as
// TestValidateOverlay_FieldValidators.
//
// It moved when the checks did. Field presence is declared once, on the overlay
// DTOs, so this exercises the declaration rather than a hand-written copy of it
// — and it asserts the same field paths the service used to produce, which is
// what a client actually depends on.
//
// Each case leaves exactly one field out of an otherwise valid overlay, so a
// weakened tag surfaces as a missing violation rather than as a request that
// quietly succeeds.
func TestForecastOverlayBindingRejectsIncompleteOverlays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bday := time.Date(2022, 5, 15, 0, 0, 0, 0, time.UTC)

	// A complete contract, so each case can drop a single field from it.
	employeeContract := func(mut func(*models.ForecastEmployeeContractInput)) models.ForecastEmployeeContractInput {
		ct := models.ForecastEmployeeContractInput{
			From: from, SectionID: 1, PayPlanID: 1,
			Grade: "S8a", Step: 3, WeeklyHours: 30, StaffCategory: "qualified",
		}
		mut(&ct)
		return ct
	}

	cases := []struct {
		name      string
		req       models.ForecastRequest
		wantField string
		wantRule  string
	}{
		{
			name: "child_missing_birthdate",
			req: models.ForecastRequest{AddChildren: []models.ForecastChildInput{{
				FirstName: "X", LastName: "Y", Gender: "female",
				Contracts: []models.ForecastChildContractInput{{From: from, SectionID: 1}},
			}}},
			wantField: "add_children[0].birthdate",
			wantRule:  "required",
		},
		{
			name: "child_no_contracts",
			req: models.ForecastRequest{AddChildren: []models.ForecastChildInput{{
				FirstName: "X", LastName: "Y", Gender: "female", Birthdate: bday,
			}}},
			wantField: "add_children[0].contracts",
			wantRule:  "required",
		},
		{
			name: "child_contract_missing_from",
			req: models.ForecastRequest{AddChildren: []models.ForecastChildInput{{
				FirstName: "X", LastName: "Y", Gender: "female", Birthdate: bday,
				Contracts: []models.ForecastChildContractInput{{SectionID: 1}},
			}}},
			wantField: "add_children[0].contracts[0].from",
			wantRule:  "required",
		},
		{
			name: "child_contract_missing_section",
			req: models.ForecastRequest{AddChildren: []models.ForecastChildInput{{
				FirstName: "X", LastName: "Y", Gender: "female", Birthdate: bday,
				Contracts: []models.ForecastChildContractInput{{From: from}},
			}}},
			wantField: "add_children[0].contracts[0].section_id",
			wantRule:  "required",
		},
		{
			name: "child_contract_standalone_missing_from",
			req: models.ForecastRequest{AddChildContracts: []models.ForecastChildContractInput{{
				ChildID: 1, SectionID: 1,
			}}},
			wantField: "add_child_contracts[0].from",
			wantRule:  "required",
		},
		{
			name: "employee_no_contracts",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
			}}},
			wantField: "add_employees[0].contracts",
			wantRule:  "required",
		},
		{
			name: "employee_contract_missing_payplan",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
				Contracts: []models.ForecastEmployeeContractInput{
					employeeContract(func(ct *models.ForecastEmployeeContractInput) { ct.PayPlanID = 0 }),
				},
			}}},
			wantField: "add_employees[0].contracts[0].payplan_id",
			wantRule:  "required",
		},
		{
			name: "employee_contract_missing_grade",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
				Contracts: []models.ForecastEmployeeContractInput{
					employeeContract(func(ct *models.ForecastEmployeeContractInput) { ct.Grade = "" }),
				},
			}}},
			wantField: "add_employees[0].contracts[0].grade",
			wantRule:  "required",
		},
		{
			name: "employee_contract_step_zero",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
				Contracts: []models.ForecastEmployeeContractInput{
					employeeContract(func(ct *models.ForecastEmployeeContractInput) { ct.Step = 0 }),
				},
			}}},
			wantField: "add_employees[0].contracts[0].step",
			// min_value, not min: a step is a magnitude, and reporting it as
			// "at least 1 characters" is what the kind check in ruleAndReason
			// exists to prevent.
			wantRule: "min_value",
		},
		{
			name: "employee_contract_zero_hours",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
				Contracts: []models.ForecastEmployeeContractInput{
					employeeContract(func(ct *models.ForecastEmployeeContractInput) { ct.WeeklyHours = 0 }),
				},
			}}},
			wantField: "add_employees[0].contracts[0].weekly_hours",
			wantRule:  "positive",
		},
		{
			name: "employee_contract_missing_staff_category",
			req: models.ForecastRequest{AddEmployees: []models.ForecastEmployeeInput{{
				FirstName: "X", LastName: "Y", Birthdate: bday,
				Contracts: []models.ForecastEmployeeContractInput{
					employeeContract(func(ct *models.ForecastEmployeeContractInput) { ct.StaffCategory = "" }),
				},
			}}},
			wantField: "add_employees[0].contracts[0].staff_category",
			wantRule:  "required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(&tc.req)
			if err == nil {
				t.Fatalf("expected %s to be rejected", tc.wantField)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			for _, p := range invalidParams(c, err) {
				if p.Field == tc.wantField {
					if p.Rule != tc.wantRule {
						t.Errorf("%s: got rule %q, want %q", tc.wantField, p.Rule, tc.wantRule)
					}
					return
				}
			}
			t.Errorf("no violation reported for %s: %+v", tc.wantField, invalidParams(c, err))
		})
	}
}

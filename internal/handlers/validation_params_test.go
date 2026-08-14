package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/eenemeene/kitamanager-go/internal/i18n"
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

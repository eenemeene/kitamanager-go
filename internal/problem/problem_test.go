package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/i18n"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/problem"
)

// TestRequestIDKeyMatchesMiddleware pins the duplicated constant.
//
// problem cannot import middleware — middleware writes problem documents, so it
// would be a cycle — and copies the context key instead. If someone renames the
// key in one place, every problem document silently loses its request_id and
// nothing else fails. This test is the thing that fails.
//
// It lives in an external test package precisely so it may import both.
func TestRequestIDKeyMatchesMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set(middleware.RequestIDKey, "abc-123")

	got := problem.New(c, http.StatusNotFound, apperror.CodeNotFound, "gone")
	if got.RequestID != "abc-123" {
		t.Fatalf("request_id = %q, want %q — problem's copy of the context key has "+
			"drifted from middleware.RequestIDKey", got.RequestID, "abc-123")
	}
}

func TestWriteProducesFullDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/1/children/42", nil)
	c.Set(middleware.RequestIDKey, "req-9")

	problem.Write(c, http.StatusNotFound, apperror.CodeNotFound, "child 42 not found")

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, problem.ContentType) {
		t.Errorf("Content-Type = %q, want %q", ct, problem.ContentType)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !c.IsAborted() {
		t.Error("Write did not abort the chain; a middleware rejection would fall through to the handler")
	}

	var got models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	want := models.ErrorResponse{
		Type:      problem.TypeBase + "not_found",
		Title:     "Resource not found",
		Status:    http.StatusNotFound,
		Detail:    "child 42 not found",
		Instance:  "/api/v1/organizations/1/children/42",
		Code:      apperror.CodeNotFound,
		RequestID: "req-9",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("document =\n  %+v\nwant\n  %+v", got, want)
	}
}

// TestTypeURIEndsInTheCode guards the one thing that makes the type member worth
// having: it must land on an anchor that exists. Every code is a heading on the
// errors page, so the fragment has to be the code itself, untransformed.
func TestTypeURIEndsInTheCode(t *testing.T) {
	got := problem.TypeURI(apperror.CodeContractConflict)
	if !strings.HasSuffix(got, "#"+apperror.CodeContractConflict) {
		t.Errorf("TypeURI(%q) = %q, which does not end in the code, so it points at an anchor the errors page does not have",
			apperror.CodeContractConflict, got)
	}
}

// TestEveryCodeHasATitle stops a new apperror code from shipping with the code
// string standing in for a title.
func TestEveryCodeHasATitle(t *testing.T) {
	codes := []string{
		apperror.CodeNotFound, apperror.CodeBadRequest, apperror.CodeValidation,
		apperror.CodeConflict, apperror.CodeUnauthorized, apperror.CodeForbidden,
		apperror.CodeTooManyRequests, apperror.CodeInternal, apperror.CodeEmailConflict,
		apperror.CodeContractConflict, apperror.CodeDuplicateBillHash,
		apperror.CodeDuplicateBillMonth, apperror.CodeMethodNotAllowed,
		apperror.CodePreconditionRequired, apperror.CodePreconditionFailed,
	}
	for _, code := range codes {
		if problem.Title(nil, code) == code {
			t.Errorf("code %q has no registered title, so its problem documents would repeat the code as their title", code)
		}
	}
}

// TestTitleIsLocalized checks the one member of the document that step 1 can
// already translate: the title is a fixed phrase per code, so it lives in the
// catalogue, while `detail` is still built as an English sentence at ~368 call
// sites and stays English until those are converted.
func TestTitleIsLocalized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		problem.Write(c, http.StatusNotFound, apperror.CodeNotFound, "child 42 not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Language", "de")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got.Title != "Ressource nicht gefunden" {
		t.Errorf("title = %q, want the German one", got.Title)
	}
	// The code is not translated: it is the client's contract, not prose.
	if got.Code != apperror.CodeNotFound {
		t.Errorf("code = %q, want %q", got.Code, apperror.CodeNotFound)
	}
	// Detail is still English, and this asserts it deliberately — step 1 does
	// not touch the ~368 sites that build it, so a change here is a change in
	// scope, not an improvement that slipped in.
	if got.Detail != "child 42 not found" {
		t.Errorf("detail = %q, want the English sentence", got.Detail)
	}
}

// TestEveryTitleIsTranslated is the drift gate for titles.
//
// Adding an error code means adding a title, and a title with no German entry
// renders in English — correct behaviour, and invisible. Without this the
// catalogue rots one code at a time and nobody notices until a German user
// reports a half-English message. Here it is a build failure naming the code.
//
// It checks the shipped catalogue rather than a fixture: the thing that can be
// wrong is the file we release.
func TestEveryTitleIsTranslated(t *testing.T) {
	codes := []string{
		apperror.CodeNotFound, apperror.CodeBadRequest, apperror.CodeValidation,
		apperror.CodeConflict, apperror.CodeUnauthorized, apperror.CodeForbidden,
		apperror.CodeTooManyRequests, apperror.CodeInternal, apperror.CodeEmailConflict,
		apperror.CodeContractConflict, apperror.CodeDuplicateBillHash,
		apperror.CodeDuplicateBillMonth, apperror.CodeMethodNotAllowed,
		apperror.CodePreconditionRequired, apperror.CodePreconditionFailed,
	}
	for _, code := range codes {
		id := "error.title." + code
		if !i18n.Has(language.English, id, false) {
			t.Errorf("code %q has no English title under id %q in locales/en.json", code, id)
		}
		if !i18n.Has(language.German, id, false) {
			t.Errorf("code %q has no German title under id %q in locales/de.json", code, id)
		}
	}
}

// TestInternalErrorsDoNotLeakDetail is the one security-relevant assertion here.
// err.Error() on a 500 can carry a driver message or a query fragment; that
// belongs in the log the request_id points at.
func TestInternalErrorsDoNotLeakDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	problem.WriteError(c, apperror.Internal("pq: relation \"secret_table\" does not exist"))

	body := rec.Body.String()
	if strings.Contains(body, "secret_table") {
		t.Errorf("500 body leaked the underlying error: %s", body)
	}
	var got models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got.Code != apperror.CodeInternal || got.Status != http.StatusInternalServerError {
		t.Errorf("code/status = %q/%d, want %q/%d", got.Code, got.Status,
			apperror.CodeInternal, http.StatusInternalServerError)
	}
}

// TestDetailIsLocalizedWithArguments is the point of the whole change: a message
// whose specifics are arguments rather than baked-in text can be translated and
// still name the record it is about.
//
// Before the constructors kept the format apart from the arguments, this could
// not work — "child 7 not found in this organization" is a string with no
// structure, so a German reader either saw the English or lost the 7.
func TestDetailIsLocalizedWithArguments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		problem.WriteError(c, apperror.BadRequest("child %d not found in this organization", 7))
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Language", "de")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	const want = "Kind 7 wurde in dieser Organisation nicht gefunden"
	if got.Detail != want {
		t.Errorf("detail = %q, want %q", got.Detail, want)
	}
}

// TestDetailStaysEnglishWhenUntranslated covers the state the tree is actually
// in: most messages have no German entry yet, and those must render in English
// rather than as a blank or a format string.
func TestDetailStaysEnglishWhenUntranslated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		problem.WriteError(c, apperror.BadRequest("widget %d has no translation yet", 3))
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Language", "de")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got.Detail != "widget 3 has no translation yet" {
		t.Errorf("detail = %q, want the English rendering with the argument applied", got.Detail)
	}
}

// TestErrorTextStaysEnglishForLogs pins the other half of the split. Whatever
// language a response was rendered in, err.Error() is English — a German user's
// failure must not produce a log line that an English-speaking operator cannot
// grep for.
func TestErrorTextStaysEnglishForLogs(t *testing.T) {
	err := apperror.BadRequest("child %d not found in this organization", 7)
	if err.Error() != "child 7 not found in this organization" {
		t.Errorf("Error() = %q, want the English rendering", err.Error())
	}
}

package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
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
		if problem.Title(code) == code {
			t.Errorf("code %q has no registered title, so its problem documents would repeat the code as their title", code)
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

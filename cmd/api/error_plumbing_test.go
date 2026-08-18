package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/problem"
)

// The router's own failures — an unrouted path, a wrong method, a panic — are
// the ones no handler test can reach, because no handler runs. They are also
// the ones a client is most likely to meet first: a typo in a path, a POST where
// the API wants a PUT, a bug in production.
//
// Nothing exercised them before. setupRouter needs a database, stores and
// services, so no test ever built it, and the integration suite builds its own
// gin.New() instead. registerErrorPlumbing exists so this file can assert the
// production wiring rather than a re-creation of it.

// newPlumbedRouter builds a router with the real plumbing and one route, which
// exists only so that "wrong method on a path that does exist" is expressible.
func newPlumbedRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// The recovery captures this writer when it is registered, so it has to be
	// swapped before the router is built. gin dumps the panic and its stack
	// through it, which would bury the assertions in the test output.
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = io.Discard
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })

	r := gin.New()
	registerErrorPlumbing(r)
	r.GET("/api/v1/anything", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/boom", func(c *gin.Context) { panic("something went wrong deep inside") })
	return r
}

// do issues a request and decodes the problem document, failing if the response
// is not one.
func do(t *testing.T, r *gin.Engine, method, path, acceptLanguage string) (*httptest.ResponseRecorder, models.ErrorResponse) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, problem.ContentType) {
		t.Fatalf("%s %s: Content-Type = %q, want %s (a client cannot parse anything else)",
			method, path, ct, problem.ContentType)
	}

	var doc models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("%s %s: body is not a problem document: %v\nbody: %s", method, path, err, w.Body.String())
	}
	return w, doc
}

func TestUnroutedPathIsAProblemDocument(t *testing.T) {
	r := newPlumbedRouter(t)
	w, doc := do(t, r, http.MethodGet, "/api/v1/organisations", "")

	// gin's default here is the plain text "404 page not found", which a client
	// parsing JSON cannot read at all.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if doc.Code != apperror.CodeNotFound {
		t.Errorf("code = %q, want %q", doc.Code, apperror.CodeNotFound)
	}
	// The detail must name the path, because "not found" alone does not tell a
	// developer whether the route or the record is missing.
	if !strings.Contains(doc.Detail, "/api/v1/organisations") {
		t.Errorf("detail = %q, want it to name the path that did not match", doc.Detail)
	}
	if doc.RequestID == "" {
		t.Error("request_id is empty: RequestID must run before the router answers")
	}
}

func TestWrongMethodIsFourZeroFiveNotFourZeroFour(t *testing.T) {
	r := newPlumbedRouter(t)
	w, doc := do(t, r, http.MethodDelete, "/api/v1/anything", "")

	// With HandleMethodNotAllowed off — gin's default — this is a 404, and a
	// client cannot tell "you used the wrong verb" from "it isn't there".
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 (HandleMethodNotAllowed must stay on)", w.Code)
	}
	if doc.Code != apperror.CodeMethodNotAllowed {
		t.Errorf("code = %q, want %q", doc.Code, apperror.CodeMethodNotAllowed)
	}
	if !strings.Contains(doc.Detail, http.MethodDelete) {
		t.Errorf("detail = %q, want it to name the method that was rejected", doc.Detail)
	}
}

// TestRouterGeneratedResponsesAreLocalized guards the scope of language
// negotiation, not its position.
//
// gin rebuilds the 404 and 405 chains on every Use, so where i18n.Middleware sits
// among the other global middleware makes no difference. What does make a
// difference is it being global at all: scope it to a route group — the obvious
// move if someone decides only the API needs it — and every handler-produced
// problem is still translated, so the change looks harmless. These two responses,
// and only these two, silently lose their localized member.
func TestRouterGeneratedResponsesAreLocalized(t *testing.T) {
	r := newPlumbedRouter(t)

	cases := []struct {
		name   string
		method string
		path   string
		// names is what the detail has to keep hold of once translated. The
		// specifics are the whole value of these two messages — a German reader
		// told only "nicht gefunden" knows strictly less than an English one
		// told which path did not match, which is the defect this whole line of
		// work started from.
		names string
	}{
		{"unrouted path", http.MethodGet, "/api/v1/nope", "/api/v1/nope"},
		{"wrong method", http.MethodDelete, "/api/v1/anything", http.MethodDelete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, doc := do(t, r, c.method, c.path, "de-DE,de;q=0.9")

			// Both, not just the negotiated one: the body really does carry the
			// English members and the German rendering, and Content-Language
			// names the intended audience of what was sent.
			if got := w.Header().Get("Content-Language"); got != "en, de" {
				t.Errorf("Content-Language = %q, want \"en, de\"", got)
			}
			if doc.Localized == nil {
				t.Fatal("localized is absent: i18n.Middleware must be registered before NoRoute/NoMethod")
			}
			if doc.Localized.Locale != "de" {
				t.Errorf("localized.locale = %q, want de", doc.Localized.Locale)
			}
			if doc.Localized.Title == "" || doc.Localized.Title == doc.Title {
				t.Errorf("localized.title = %q, want a German title distinct from the English %q",
					doc.Localized.Title, doc.Title)
			}
			// The English stays put: it is what a log and a support ticket show.
			if doc.Title == "" {
				t.Error("title is empty: the English members must survive negotiation")
			}

			// Both messages are composed, so they only translate if the format
			// and its arguments reach the catalogue apart. Built by
			// concatenation — as both were — the lookup silently misses and the
			// detail comes back English inside an otherwise German document.
			if doc.Localized.Detail == "" {
				t.Fatal("localized.detail is empty: the detail must be written with a registered format, not assembled by hand")
			}
			if doc.Localized.Detail == doc.Detail {
				t.Errorf("localized.detail = %q, identical to the English: nothing was translated", doc.Localized.Detail)
			}
			if !strings.Contains(doc.Localized.Detail, c.names) {
				t.Errorf("localized.detail = %q, want it to still name %q", doc.Localized.Detail, c.names)
			}
		})
	}
}

func TestUnsupportedLanguageIsAnsweredInEnglish(t *testing.T) {
	r := newPlumbedRouter(t)
	w, doc := do(t, r, http.MethodGet, "/api/v1/nope", "ja-JP")

	if got := w.Header().Get("Content-Language"); got != "en" {
		t.Errorf("Content-Language = %q, want en", got)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Accept-Language") {
		t.Errorf("Vary = %q, want it to list Accept-Language so caches do not serve one language to everyone", got)
	}

	// No catalogue for Japanese, so there is nothing to put in localized and
	// duplicating the English there would be a lie about what was served.
	if doc.Localized != nil {
		t.Errorf("localized = %+v, want absent for a language the catalogue does not cover", doc.Localized)
	}
	if doc.Title == "" {
		t.Error("title is empty")
	}
}

// TestPanicBecomesADocumentWithACorrelationID covers the one response a user is
// most likely to report. gin's own recovery aborts with a bare 500 and no body
// at all, so what a client saw was an empty response for the failure it most
// needed to describe.
func TestPanicBecomesADocumentWithACorrelationID(t *testing.T) {
	r := newPlumbedRouter(t)
	w, doc := do(t, r, http.MethodGet, "/api/v1/boom", "")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if doc.RequestID == "" {
		t.Error("request_id is empty: a 500 with no correlation id cannot be traced to a log line")
	}
	if got := w.Header().Get("X-Request-ID"); got != doc.RequestID {
		t.Errorf("X-Request-ID = %q but body says %q; a user quoting either must reach the same log line", got, doc.RequestID)
	}
	// The panic message must not reach the client.
	if strings.Contains(strings.ToLower(doc.Detail), "deep inside") {
		t.Errorf("detail = %q, leaks the panic message", doc.Detail)
	}
}

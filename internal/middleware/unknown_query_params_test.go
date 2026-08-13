package middleware

// Tests for the unknown-query-parameter warning.
//
// The last test is the one that matters most: it runs the real spec that swaggo
// compiles into the binary against the real `contract_on` request, and asserts a
// warning would have been logged. That is the bug this middleware exists to catch,
// so it is asserted against the actual data rather than a fixture.

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/docs"
)

const testSpec = `{
  "basePath": "/",
  "paths": {
    "/api/v1/organizations/{orgId}/children": {
      "get": {
        "parameters": [
          {"name": "orgId", "in": "path"},
          {"name": "active_on", "in": "query"},
          {"name": "page", "in": "query"}
        ]
      },
      "post": {
        "parameters": [{"name": "orgId", "in": "path"}]
      }
    }
  }
}`

// captureLogs swaps the default slog handler for one writing into a buffer, and
// restores it afterwards.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// routerFor registers the one route the test spec describes, plus an undeclared
// one, both behind the middleware.
func routerFor(spec string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(UnknownQueryParams(spec))
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/api/v1/organizations/:orgId/children", ok)
	r.POST("/api/v1/organizations/:orgId/children", ok)
	r.GET("/healthz", ok)
	return r
}

func get(r *gin.Engine, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestUnknownQueryParams_WarnsOnUndeclaredParameter(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor(testSpec)

	w := get(r, "/api/v1/organizations/1/children?contract_on=2026-03-01")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — the check must never reject a request", w.Code)
	}

	out := logs.String()
	if !strings.Contains(out, "contract_on") {
		t.Errorf("expected a warning naming contract_on, got: %s", out)
	}
	if !strings.Contains(out, "/api/v1/organizations/:orgId/children") {
		t.Errorf("expected the warning to name the route, got: %s", out)
	}
}

func TestUnknownQueryParams_SilentOnDeclaredParameters(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor(testSpec)

	get(r, "/api/v1/organizations/1/children?active_on=2026-03-01&page=2")

	if out := logs.String(); out != "" {
		t.Errorf("declared parameters must not warn, got: %s", out)
	}
}

// A parameter valid on one method is not automatically valid on another, which is
// the accuracy a single global allowlist could not provide.
func TestUnknownQueryParams_IsPerMethod(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor(testSpec)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/1/children?active_on=2026-03-01", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(logs.String(), "active_on") {
		t.Errorf("active_on is declared for GET only; POST should warn. Got: %s", logs.String())
	}
}

// Routes the spec does not describe cannot be judged, so they are left alone
// rather than reported as suspicious.
func TestUnknownQueryParams_SkipsUndescribedRoutes(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor(testSpec)

	get(r, "/healthz?verbose=1")

	if out := logs.String(); out != "" {
		t.Errorf("a route absent from the spec must not warn, got: %s", out)
	}
}

func TestUnknownQueryParams_NoQueryNoWarning(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor(testSpec)

	get(r, "/api/v1/organizations/1/children")

	if out := logs.String(); out != "" {
		t.Errorf("a request without a query string must not warn, got: %s", out)
	}
}

// A spec that cannot be parsed disables the check instead of breaking the API.
func TestUnknownQueryParams_BrokenSpecDisablesTheCheck(t *testing.T) {
	logs := captureLogs(t)
	r := routerFor("{not json")

	w := get(r, "/api/v1/organizations/1/children?whatever=1")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 — a broken spec must not affect serving", w.Code)
	}
	if !strings.Contains(logs.String(), "check disabled") {
		t.Errorf("expected a warning that the check is disabled, got: %s", logs.String())
	}
}

func TestSpecPathToGin(t *testing.T) {
	cases := map[string]string{
		"/api/v1/organizations/{orgId}/children":                                  "/api/v1/organizations/:orgId/children",
		"/api/v1/organizations/{orgId}/children/{childId}/contracts":              "/api/v1/organizations/:orgId/children/:childId/contracts",
		"/api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId}": "/api/v1/organizations/:orgId/children/:childId/contracts/:contractId",
		"/healthz": "/healthz",
	}
	for in, want := range cases {
		if got := specPathToGin(in); got != want {
			t.Errorf("specPathToGin(%q) = %q, want %q", in, got, want)
		}
	}
}

// The regression this middleware was built for, against the spec actually shipped.
//
// Rather than trusting a fixture, this reads docs.SwaggerInfo.ReadDoc() — the same
// string main.go passes in — and checks two things about the real children list:
// that active_on is declared (so the fixed client stays quiet) and that
// contract_on is not (so the old client would have been reported).
func TestUnknownQueryParams_AgainstTheRealSpec(t *testing.T) {
	spec := docs.SwaggerInfo.ReadDoc()
	if spec == "" {
		t.Skip("no embedded spec available")
	}
	declared := parseDeclaredQueryParams(spec)
	if declared == nil {
		t.Fatal("the shipped spec must parse")
	}

	key := routeKey(http.MethodGet, "/api/v1/organizations/:orgId/children")
	params, ok := declared[key]
	if !ok {
		var sample []string
		for k := range declared {
			if strings.Contains(k, "/children") {
				sample = append(sample, k)
			}
		}
		t.Fatalf("children list route missing from the parsed spec; /children routes found: %v", sample)
	}

	if _, fine := params["active_on"]; !fine {
		t.Errorf("active_on should be declared on the children list; got %v", keys(params))
	}
	if _, fine := params["contract_on"]; fine {
		t.Error("contract_on is declared after all — then the bug was a server-side gap, not a client typo")
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

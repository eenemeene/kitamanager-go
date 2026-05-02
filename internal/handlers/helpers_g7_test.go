package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// G7 / I-M-6 — bindJSON rejects payloads with unknown fields.
//
// Without DisallowUnknownFields, the server silently ignored extra
// keys, which (a) hid client typos and (b) allowed callers to smuggle
// fields named after future schema fields, where the existence of the
// API contract is the only review surface.
func TestBindJSON_RejectsUnknownFields(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		_, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := `{"email":"a@example.com","password":"secret123","extra":"sneaky"}`
	w := performRequestRaw(r, "POST", "/login", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "unknown field") {
		t.Errorf("expected error to mention unknown field, got: %s", w.Body.String())
	}
}

// G7 / I-M-6 — body cap protects the JSON decoder from amplification
// attacks (an unbounded io.ReadAll would let a slow client tie up
// goroutines and reflect-allocate megabytes per request).
func TestBindJSON_RejectsOversizedBody(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		_, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Build a JSON body larger than MaxJSONBodySize. The decoder must
	// refuse before the field validator runs.
	pad := strings.Repeat("x", MaxJSONBodySize+1024)
	body := `{"email":"a@example.com","password":"` + pad + `"}`
	w := performRequestRaw(r, "POST", "/login", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "exceeds maximum") {
		t.Errorf("expected size-limit error, got: %s", w.Body.String())
	}
}

// G7 / I-M-6 — a trailing JSON value after the first object must be
// rejected. Without this guard, a request like  {"a":1} {"a":2}
// would silently apply only the first document.
func TestBindJSON_RejectsTrailingDocument(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		_, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := `{"email":"a@example.com","password":"secret123"}{"email":"b@example.com","password":"second"}`
	w := performRequestRaw(r, "POST", "/login", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for trailing document, got %d: %s", w.Code, w.Body.String())
	}
}

// G7 — happy path: a well-formed request still succeeds end-to-end.
// Guards against an over-strict bindJSON regressing every handler.
func TestBindJSON_AcceptsValidPayload(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		req, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"email": req.Email})
	})

	body := models.LoginRequest{Email: "admin@example.com", Password: "secret123"}
	w := performRequest(r, "POST", "/login", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// G7 / I-M-5 — Email is bounded at RFC 5321's 320-byte limit. Anything
// larger (here 400 bytes) must be rejected before the email regex or
// downstream bcrypt verify spend cycles on the payload.
func TestBindJSON_LoginEmailLengthCap(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		_, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	longLocal := strings.Repeat("a", 400)
	body := models.LoginRequest{Email: longLocal + "@example.com", Password: "secret123"}
	w := performRequest(r, "POST", "/login", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-long email, got %d", w.Code)
	}
}

// G7 / I-M-5 — Password is bounded at 256 bytes. bcrypt only consumes
// 72 bytes, so anything larger is wasted CPU at best, DoS amplification
// at worst.
func TestBindJSON_LoginPasswordLengthCap(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		_, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := models.LoginRequest{Email: "a@example.com", Password: strings.Repeat("p", 1024)}
	w := performRequest(r, "POST", "/login", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-long password, got %d", w.Code)
	}
}

// G7 / I-M-2 — decodeYAMLStrict rejects unknown keys. Our YAML
// importers round-trip the matching export shape, so any extra key is
// either a typo or a forgotten rename.
func TestDecodeYAMLStrict_RejectsUnknownFields(t *testing.T) {
	type onlyName struct {
		Name string `yaml:"name"`
	}
	data := []byte("name: foo\nsneaky: yes\n")
	if _, err := decodeYAMLStrict[onlyName](data); err == nil {
		t.Fatal("expected error for unknown YAML field")
	}
}

// G7 / I-M-2 — decodeYAMLStrict refuses YAML files containing more
// than one document. Our importers expect exactly one shape.
func TestDecodeYAMLStrict_RejectsTrailingDocument(t *testing.T) {
	type onlyName struct {
		Name string `yaml:"name"`
	}
	data := []byte("name: foo\n---\nname: bar\n")
	if _, err := decodeYAMLStrict[onlyName](data); err == nil {
		t.Fatal("expected error for multi-document YAML")
	}
}

// G7 — happy path for decodeYAMLStrict; a sole well-formed document
// must continue to decode cleanly.
func TestDecodeYAMLStrict_AcceptsValidDocument(t *testing.T) {
	type onlyName struct {
		Name string `yaml:"name"`
	}
	out, err := decodeYAMLStrict[onlyName]([]byte("name: foo\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.Name != "foo" {
		t.Fatalf("expected Name=foo, got %+v", out)
	}
}

// G7 / I-M-3 — safeAttachmentFilename strips characters that could
// inject into Content-Disposition (quotes, semicolons, equals, etc.)
// and replaces spaces with hyphens. Only [a-z0-9_-] survive.
func TestSafeAttachmentFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"TVöD-SuE", "tvd-sue.yaml"},
		// Spaces become hyphens; quotes / semicolons / equals are stripped.
		{`evil"; download="x.exe`, "evil-downloadxexe.yaml"},
		{"  Mixed Case Plan  ", "mixed-case-plan.yaml"},
		{"", "export.yaml"},
		{strings.Repeat("a", MaxAttachmentSegment+50), strings.Repeat("a", MaxAttachmentSegment) + ".yaml"},
		{"---", "export.yaml"},
	}
	for _, tc := range cases {
		got := safeAttachmentFilename(tc.in, ".yaml")
		if got != tc.want {
			t.Errorf("safeAttachmentFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// G7 / I-M-3 — Content-Disposition end-to-end: a pay plan whose name
// contains injection-friendly characters must produce a sanitised
// header that cannot break out of the quoted-string parameter.
//
// We bypass route wiring by constructing a fake handler body that
// duplicates the relevant 3 lines of payplan.Export.Header — that is
// the surface this test cares about. A higher-level integration test
// in payplan_test.go would also work but adds a lot of fixture cost
// for one assertion.
func TestSafeAttachment_HeaderHasNoInjection(t *testing.T) {
	r := setupTestRouter()
	r.GET("/leak", func(c *gin.Context) {
		filename := safeAttachmentFilename(`evil"; foo="x`, ".yaml")
		c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest("GET", "/leak", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cd := w.Header().Get("Content-Disposition")
	if strings.Contains(cd, `"; foo="`) {
		t.Errorf("Content-Disposition leaked injection: %s", cd)
	}
	// Spaces collapse to hyphens; the rest of the injection payload
	// (quotes, semicolons, equals) is stripped.
	if !strings.HasPrefix(cd, `attachment; filename="evil-foox.yaml"`) {
		t.Errorf("unexpected Content-Disposition: %s", cd)
	}
}

// G7 — sanity: the JSON helper still parses a valid payload using the
// ordinary go encoding (i.e. Decoder.Decode populates fields).
func TestBindJSON_ParsesFields(t *testing.T) {
	r := setupTestRouter()
	captured := models.LoginRequest{}
	r.POST("/login", func(c *gin.Context) {
		req, ok := bindJSON[models.LoginRequest](c)
		if !ok {
			return
		}
		captured = *req
		c.Status(http.StatusOK)
	})
	w := performRequestRaw(r, "POST", "/login", `{"email":"a@b.c","password":"p123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if captured.Email != "a@b.c" || captured.Password != "p123" {
		t.Errorf("captured = %+v", captured)
	}
}

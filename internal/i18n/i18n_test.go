package i18n_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"

	"github.com/eenemeene/kitamanager-go/internal/i18n"
)

// TestMatch covers the header forms that actually arrive, including the ones
// that make hand-written parsers pick the wrong language.
func TestMatch(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   language.Tag
	}{
		{"no header at all", "", language.English},
		{"plain German", "de", language.German},
		{"regional German falls back to German", "de-AT", language.German},
		{
			// The case a naive "first tag wins" parser gets wrong in the other
			// direction, and a naive "split on comma" one gets wrong here: the
			// quality values say German is preferred.
			name:   "quality values decide",
			header: "de-AT;q=0.9, en;q=0.8",
			want:   language.German,
		},
		{"English preferred over German by weight", "de;q=0.2, en;q=0.9", language.English},
		{"unsupported language is answered in English", "fr-FR, ja;q=0.7", language.English},
		{"malformed header does not fail the request", "!!!not a language!!!", language.English},
		{"wildcard", "*", language.English},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := i18n.Match(tt.header); got != tt.want {
				t.Errorf("Match(%q) = %s, want %s", tt.header, got, tt.want)
			}
		})
	}
}

func TestMiddlewareStatesWhatItServed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		accept string
		want   string
	}{
		{"de", "de"},
		{"de-AT", "de"},
		{"fr", "en"},
		{"", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.accept, func(t *testing.T) {
			r := gin.New()
			r.Use(i18n.Middleware())
			r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.accept != "" {
				req.Header.Set("Accept-Language", tt.accept)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if got := rec.Header().Get("Content-Language"); got != tt.want {
				t.Errorf("Content-Language = %q, want %q", got, tt.want)
			}
			// Without Vary, a shared cache serves the first user's language to
			// everyone behind it.
			if got := rec.Header().Get("Vary"); got != "Accept-Language" {
				t.Errorf("Vary = %q, want %q", got, "Accept-Language")
			}
		})
	}
}

// TestPrinterTranslatesForGermanOnly is the end-to-end check on the catalogue:
// a key with a German entry renders German for a German request and stays
// English otherwise.
func TestPrinterTranslatesForGermanOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const key = "Resource not found"
	var german, english string

	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		//nolint:govet // key is a catalogue lookup, not a format literal
		got := i18n.For(c).Sprintf(key)
		if i18n.LanguageFor(c) == language.German {
			german = got
		} else {
			english = got
		}
		c.Status(http.StatusOK)
	})

	for _, accept := range []string{"de", "en"} {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Accept-Language", accept)
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	if german != "Ressource nicht gefunden" {
		t.Errorf("German render = %q, want %q", german, "Ressource nicht gefunden")
	}
	if english != key {
		t.Errorf("English render = %q, want the key itself %q", english, key)
	}
}

// TestUntranslatedKeyRendersAsItself pins the fallback that makes the gettext
// model safe: a message nobody has translated yet is shown in English, not
// blank and not as an identifier.
func TestUntranslatedKeyRendersAsItself(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.Header.Set("Accept-Language", "de")

	const key = "this message has no translation anywhere"
	//nolint:govet // key is a catalogue lookup, not a format literal
	if got := i18n.For(c).Sprintf(key); got != key {
		t.Errorf("untranslated key rendered as %q, want %q", got, key)
	}
}

// TestForWithoutMiddlewareIsEnglish covers the path that must not panic: an
// error being written from a context that never went through the middleware.
func TestForWithoutMiddlewareIsEnglish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	//nolint:govet // key is a catalogue lookup, not a format literal
	if got := i18n.For(c).Sprintf("Resource not found"); got != "Resource not found" {
		t.Errorf("render without middleware = %q, want the English string", got)
	}
	if got := i18n.For(nil).Sprintf("Resource not found"); got != "Resource not found" { //nolint:govet,staticcheck // nil context is the case under test
		t.Errorf("render with nil context = %q, want the English string", got)
	}
}

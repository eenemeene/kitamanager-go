package i18n_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"

	"github.com/eenemeene/kitamanager-go/internal/i18n"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
)

// render drives a message through the real middleware, which is the only way to
// get a request-scoped localizer. Building a bare gin context and setting the
// header on it silently yields the English fallback — a mistake worth designing
// the helper against, since it makes a broken test look like a passing one.
func render(t *testing.T, acceptLanguage, format string, args ...any) (string, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var got string
	var ok bool
	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		got, ok = i18n.Localize(c, format, args...)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
	return got, ok
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   language.Tag
	}{
		{"no header at all", "", language.English},
		{"plain German", "de", language.German},
		{"regional German falls back to German", "de-AT", language.German},
		{"quality values decide", "de-AT;q=0.9, en;q=0.8", language.German},
		{"English preferred by weight", "de;q=0.2, en;q=0.9", language.English},
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
	for _, tt := range []struct{ accept, want string }{
		{"de", "de"}, {"de-AT", "de"}, {"fr", "en"}, {"", "en"},
	} {
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

// TestPluralFormsAreSelected is the reason for this change. The message it checks
// was wrong in English before the swap — "with 1 currently-assigned children" —
// because the previous catalogue was a map[string]string and could not hold two
// forms. German is not a copy of English's rule either, so both languages are
// checked at both counts.
func TestPluralFormsAreSelected(t *testing.T) {
	const msg = "cannot delete section with %d currently-assigned children; reassign them first"

	tests := []struct {
		lang  string
		count int
		want  string
	}{
		{"en", 1, "Cannot delete section with 1 currently-assigned child; reassign them first"},
		{"en", 3, "Cannot delete section with 3 currently-assigned children; reassign them first"},
		{"de", 1, "Der Bereich kann nicht gelöscht werden, da ihm noch 1 Kind zugeordnet ist; bitte zuerst umbuchen"},
		{"de", 3, "Der Bereich kann nicht gelöscht werden, da ihm noch 3 Kinder zugeordnet sind; bitte zuerst umbuchen"},
	}
	for _, tt := range tests {
		got, ok := render(t, tt.lang, msg, tt.count)
		if !ok {
			t.Fatalf("[%s n=%d] message is not registered", tt.lang, tt.count)
		}
		if got != tt.want {
			t.Errorf("[%s n=%d]\n got %q\nwant %q", tt.lang, tt.count, got, tt.want)
		}
	}
}

// TestNamedPlaceholdersAllowReordering covers the second reason for the swap.
// The German sentence puts the subject before the verb phrase, which a bare %d
// cannot express and a named placeholder can.
func TestNamedPlaceholdersAllowReordering(t *testing.T) {
	got, ok := render(t, "de", "child %d not found in this organization", 42)
	if !ok {
		t.Fatal("message is not registered")
	}
	const want = "Kind 42 wurde in dieser Organisation nicht gefunden"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestNumbersUseTheLocalesSeparators guards the one capability go-i18n does not
// have on its own. text/template would print a bare 1234; x/text formats it for
// the negotiated locale, so this is what stops the swap being a regression.
func TestNumbersUseTheLocalesSeparators(t *testing.T) {
	de, ok := render(t, "de", "child %d not found in this organization", 1234)
	if !ok {
		t.Fatal("message is not registered")
	}
	if !strings.Contains(de, "1.234") {
		t.Errorf("German render = %q, want the number as 1.234", de)
	}

	en, _ := render(t, "en", "child %d not found in this organization", 1234)
	if !strings.Contains(en, "1,234") {
		t.Errorf("English render = %q, want the number as 1,234", en)
	}
}

// TestUnregisteredMessageFallsThrough pins what makes the migration incremental:
// a message with no registry entry reports false, and the caller keeps the
// English rendering it already had.
func TestUnregisteredMessageFallsThrough(t *testing.T) {
	if got, ok := render(t, "de", "this message is not in the registry %d", 1); ok {
		t.Errorf("unregistered message reported ok with %q", got)
	}
}

// TestLocalizerKeyMatchesMiddleware would be redundant for a single package, but
// problem reads the localizer out of the same context. This asserts the request
// really carries one, so a rename cannot leave every response silently English.
func TestLocalizerIsRequestScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(i18n.Middleware())
	r.GET("/x", func(c *gin.Context) {
		if i18n.LanguageFor(c) != language.German {
			t.Errorf("LanguageFor = %s, want German", i18n.LanguageFor(c))
		}
		if title := i18n.Title(c, "not_found"); title != "Ressource nicht gefunden" {
			t.Errorf("Title = %q, want the German one", title)
		}
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Language", "de")
	r.ServeHTTP(httptest.NewRecorder(), req)
}

// TestRegistryIsFullyTranslated is the drift gate, and it now covers the details
// as well as the titles — which the previous version did not, so 148 detail
// translations would have been able to rot one at a time.
//
// It checks both catalogues: an entry with no English source renders as its ID,
// and one with no German is silently served in English.
func TestRegistryIsFullyTranslated(t *testing.T) {
	for format, e := range i18n.RegistryEntries() {
		plural := e.Plural != ""
		if !i18n.Has(language.English, e.ID, plural) {
			t.Errorf("id %q (from %q) has no English source in locales/en.json", e.ID, format)
		}
		if !i18n.Has(language.German, e.ID, plural) {
			t.Errorf("id %q (from %q) has no German translation in locales/de.json", e.ID, format)
		}
	}
}

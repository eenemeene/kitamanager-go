// Package i18n negotiates a response language and renders messages in it.
//
// The API is English. This does not change that: `code`, `type`, field names and
// every log line stay English, and English is what a client gets unless it asks
// for something else. What this adds is a way for a client with a German user to
// receive German prose from the component that knows the message.
//
// # Where translation happens
//
// At the edge, in the response writer, never at the site that raises the error.
// Threading a localizer through the ~370 places that construct an error would
// touch every service signature and make err.Error() language-dependent, so a
// German user's failure would write a German log line that nobody grepping the
// logs can find.
//
// # Why go-i18n rather than x/text/message
//
// This started on golang.org/x/text/message with the English string as the
// catalogue key. That is cheap to adopt and wrong for three reasons, all of which
// would have been baked in by translating the remaining messages:
//
//   - No plurals. Its catalogue was a map[string]string here, which cannot hold
//     "1 child" against "2 children" — and one shipped message was already wrong
//     in English for exactly that reason.
//   - No named placeholders. A translator cannot reorder a bare pair of %d to fit
//     German word order.
//   - Rewording the English silently orphaned the translation.
//
// go-i18n gives stable IDs, CLDR plural rules per language, and named template
// data. The two Go products closest to this one — Mattermost and Gitea, both
// user-facing with dozens of locales — use stable IDs feeding a translation
// platform, not gettext-style English keys.
//
// x/text is still here for one job it does better: locale-aware number
// formatting, so a German reader sees 1.234 where an English one sees 1,234.
// go-i18n renders through text/template, which would print a bare 1234.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

//go:embed locales/*.json
var locales embed.FS

// Supported lists the languages this API can answer in, English first so it is
// the fallback for anything unrecognised.
var Supported = []language.Tag{
	language.English,
	language.German,
}

var bundle *goi18n.Bundle

// Context keys. Duplicated from middleware.RequestIDKey's neighbourhood rather
// than imported, because middleware writes problem documents and importing it
// here would be a cycle.
const (
	localizerKey = "i18nLocalizer"
	languageKey  = "i18nLanguage"
)

func init() {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	entries, err := locales.ReadDir("locales")
	if err != nil {
		slog.Error("i18n: cannot read embedded locales; continuing in English only", "error", err)
		return
	}
	for _, e := range entries {
		if _, err := bundle.LoadMessageFileFS(locales, "locales/"+e.Name()); err != nil {
			// A malformed catalogue must not stop the server: every message has
			// an English source, so the worst case is an untranslated response.
			slog.Error("i18n: cannot load locale file", "file", e.Name(), "error", err)
		}
	}
}

// Match negotiates a language from an Accept-Language header value.
//
// Parsing is delegated to x/text: quality values, subtag fallback (de-AT to de)
// and malformed input are all handled there. Hand-rolling any of it is how
// "de-AT;q=0.9, en;q=0.8" ends up selecting English.
func Match(acceptLanguage string) language.Tag {
	if acceptLanguage == "" {
		return language.English
	}
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil {
		return language.English
	}
	_, index, confidence := language.NewMatcher(Supported).Match(tags...)
	if confidence == language.No {
		return language.English
	}
	return Supported[index]
}

// Middleware negotiates the response language and records the decision.
//
// Content-Language states what the client actually got, which is often not what
// it asked for. Vary tells any cache in between that this response is one of
// several — without it, the first German user's response is served to the next
// English one.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tag := Match(c.GetHeader("Accept-Language"))

		c.Set(languageKey, tag)
		c.Set(localizerKey, goi18n.NewLocalizer(bundle, tag.String()))

		c.Header("Content-Language", tag.String())
		c.Writer.Header().Add("Vary", "Accept-Language")

		c.Next()
	}
}

// LanguageFor returns the tag negotiated for this request, English by default.
func LanguageFor(c *gin.Context) language.Tag {
	if c != nil {
		if tag, ok := c.Value(languageKey).(language.Tag); ok {
			return tag
		}
	}
	return language.English
}

// localizerFor returns the request's localizer, falling back to English rather
// than nil: a missing middleware registration should produce an English message,
// not a panic on the error path.
func localizerFor(c *gin.Context) *goi18n.Localizer {
	if c != nil {
		if l, ok := c.Value(localizerKey).(*goi18n.Localizer); ok {
			return l
		}
	}
	return goi18n.NewLocalizer(bundle, language.English.String())
}

// titleID returns the catalogue ID for an error code's title. Derived rather
// than tabulated: the code is already a stable slug, so a new code gets a
// predictable ID and there is no second list to forget to update.
func titleID(code string) string {
	return "error.title." + code
}

// Title returns the localized title for an error code, or "" when the catalogue
// has no entry — the caller then decides what to fall back to.
func Title(c *gin.Context, code string) string {
	out, err := localizerFor(c).Localize(&goi18n.LocalizeConfig{MessageID: titleID(code)})
	if err != nil {
		return ""
	}
	return out
}

// Localize renders an English message in the request's language.
//
// It reports false when the message has no registry entry, which is the common
// case: the caller then uses the English rendering it already has. That is what
// makes the migration incremental — an unregistered message behaves exactly as
// it did before this package existed.
func Localize(c *gin.Context, format string, args ...any) (string, bool) {
	e, ok := registry[format]
	if !ok {
		return "", false
	}
	if len(e.Args) != len(args) {
		// The registry disagrees with the call site about how many arguments
		// this message takes. Rendering it would produce "<no value>", so fall
		// back to English and say so loudly; TestRegistryArity turns this into a
		// build failure rather than a log line nobody reads.
		slog.Error("i18n: registry arity mismatch",
			"id", e.ID, "want", len(e.Args), "got", len(args))
		return "", false
	}

	tag := LanguageFor(c)
	printer := message.NewPrinter(tag)

	data := make(map[string]any, len(e.Args))
	var pluralCount any
	for i, name := range e.Args {
		data[name] = localizeValue(printer, args[i])
		if name == e.Plural {
			pluralCount = args[i]
		}
	}

	cfg := &goi18n.LocalizeConfig{MessageID: e.ID, TemplateData: data}
	if e.Plural != "" {
		cfg.PluralCount = pluralCount
	}
	out, err := localizerFor(c).Localize(cfg)
	if err != nil {
		slog.Error("i18n: cannot localize message", "id", e.ID, "language", tag.String(), "error", err)
		return "", false
	}
	return out, true
}

// localizeValue formats a value for insertion into a message.
//
// Numbers go through x/text so they carry the locale's separators — 1.234 for a
// German reader, 1,234 for an English one. text/template alone would print a
// bare 1234, which is the one thing go-i18n does worse than what it replaced.
func localizeValue(p *message.Printer, v any) any {
	switch n := v.(type) {
	case int:
		return p.Sprint(number.Decimal(n))
	case int32:
		return p.Sprint(number.Decimal(n))
	case int64:
		return p.Sprint(number.Decimal(n))
	case uint:
		return p.Sprint(number.Decimal(n))
	case float32:
		return p.Sprint(number.Decimal(n))
	case float64:
		return p.Sprint(number.Decimal(n))
	default:
		return fmt.Sprint(v)
	}
}

// ruleIDs maps a validator tag to its catalogue entry. Validation reasons are
// not registry messages: they are assembled from a rule and a field rather than
// written as a sentence at a call site, so they are keyed directly.
var ruleIDs = map[string]string{
	"required":  "validation.rule.required",
	"email":     "validation.rule.email",
	"min":       "validation.rule.min",
	"max":       "validation.rule.max",
	"voucher":   "validation.rule.voucher",
	"min_value": "validation.rule.min_value",
	"non_empty": "validation.rule.non_empty",
	"positive":  "validation.rule.positive",
	"mismatch":  "validation.rule.mismatch",
	"":          "validation.rule.invalid",
}

// Rule renders the reason a field was rejected, in the request's language.
//
// Returns "" when the request is English or the rule is unknown, so the caller
// keeps the English reason it already built. param carries the rule's argument
// where it takes one — the bound in "at least 8 characters".
func Rule(c *gin.Context, rule, param string) string {
	if LanguageFor(c) == language.English {
		return ""
	}
	id, ok := ruleIDs[rule]
	if !ok {
		id = ruleIDs[""]
	}
	out, err := localizerFor(c).Localize(&goi18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: map[string]any{"Param": param},
	})
	if err != nil {
		return ""
	}
	return out
}

// Registered reports whether an English message has a catalogue entry. Used by
// the drift tests.
func Registered(format string) (string, bool) {
	e, ok := registry[format]
	return e.ID, ok
}

// RegistryEntries exposes the registry to the drift tests, which check that every
// entry has an English source and a German translation.
func RegistryEntries() map[string]struct {
	ID     string
	Args   []string
	Plural string
} {
	out := make(map[string]struct {
		ID     string
		Args   []string
		Plural string
	}, len(registry))
	for format, e := range registry {
		out[format] = struct {
			ID     string
			Args   []string
			Plural string
		}{ID: e.ID, Args: e.Args, Plural: e.Plural}
	}
	return out
}

// Has reports whether a message ID exists in a language's catalogue, without
// falling back. Used by the drift tests to prove a translation is really there.
func Has(tag language.Tag, id string, plural bool) bool {
	l := goi18n.NewLocalizer(bundle, tag.String())
	cfg := &goi18n.LocalizeConfig{MessageID: id}
	if plural {
		cfg.PluralCount = 2
	}
	out, err := l.Localize(cfg)
	if err != nil || out == "" {
		return false
	}
	if tag == language.English {
		return true
	}
	en := goi18n.NewLocalizer(bundle, language.English.String())
	enCfg := &goi18n.LocalizeConfig{MessageID: id}
	if plural {
		enCfg.PluralCount = 2
	}
	enOut, enErr := en.Localize(enCfg)
	return enErr != nil || out != enOut
}

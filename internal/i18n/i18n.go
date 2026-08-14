// Package i18n negotiates a response language and renders messages in it.
//
// The API is English. This does not change that: `code`, `type`, field names and
// every log line stay English, and English is what a client gets unless it asks
// for something else. What this adds is a way for a client that has a German
// user to receive German prose from the one component that knows the message —
// the server — instead of reimplementing the catalogue on its side.
//
// # Where translation happens
//
// At the edge, in the response writer, never at the site that raises the error.
// The alternative — threading a printer through the ~368 places that construct
// an error — would touch every service signature and, worse, would make
// err.Error() language-dependent, so a German user's failure would produce a
// German log line that nobody grepping the logs can find.
//
// # The English message is the key
//
// There are no invented message IDs. The English string a call site already
// passes is the catalogue key, the gettext model: the sentence stays visible at
// the call site, a reviewer reads code rather than an identifier, and a message
// with no translation renders as itself rather than as a blank or an ID.
package i18n

import (
	"embed"
	"encoding/json"
	"log/slog"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

//go:embed locales/*.json
var locales embed.FS

// Supported lists the languages this API can answer in, most preferred first.
// English is index 0, which makes it the matcher's fallback for anything
// unrecognised.
var Supported = []language.Tag{
	language.English,
	language.German,
}

var (
	matcher = language.NewMatcher(Supported)
	cat     *catalog.Builder
)

// printerKey is the gin context key holding the request's printer.
const printerKey = "i18nPrinter"

// languageKey is the gin context key holding the negotiated tag.
const languageKey = "i18nLanguage"

func init() {
	cat = catalog.NewBuilder(catalog.Fallback(language.English))

	entries, err := locales.ReadDir("locales")
	if err != nil {
		// The files are embedded, so this cannot fail at runtime; if it somehow
		// does, an English-only catalogue is a working API, not a dead one.
		slog.Error("i18n: cannot read embedded locales; continuing in English only", "error", err)
		return
	}

	for _, entry := range entries {
		tag, err := language.Parse(entry.Name()[:len(entry.Name())-len(".json")])
		if err != nil {
			slog.Error("i18n: locale file is not named after a language tag", "file", entry.Name(), "error", err)
			continue
		}
		raw, err := locales.ReadFile("locales/" + entry.Name())
		if err != nil {
			slog.Error("i18n: cannot read locale file", "file", entry.Name(), "error", err)
			continue
		}
		var msgs map[string]string
		if err := json.Unmarshal(raw, &msgs); err != nil {
			slog.Error("i18n: locale file is not valid JSON", "file", entry.Name(), "error", err)
			continue
		}
		for key, translated := range msgs {
			if err := cat.SetString(tag, key, translated); err != nil {
				slog.Error("i18n: cannot register message", "file", entry.Name(), "key", key, "error", err)
			}
		}
	}
}

// Match negotiates a language from an Accept-Language header value.
//
// Parsing is delegated to x/text: quality values, subtag fallback (de-AT to de)
// and malformed input are all handled there, and hand-rolling any of it is how
// "de-AT;q=0.9, en;q=0.8" ends up selecting English.
//
// An empty, unparseable or unsupported header yields English.
func Match(acceptLanguage string) language.Tag {
	if acceptLanguage == "" {
		return language.English
	}
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil {
		return language.English
	}
	_, index, confidence := matcher.Match(tags...)
	if confidence == language.No {
		return language.English
	}
	return Supported[index]
}

// Middleware negotiates the response language and records the decision.
//
// Content-Language states what the client actually got, which matters because it
// is often not what it asked for. Vary tells any cache in between that this
// response is one of several — without it, the first German user's response is
// served to the next English one.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tag := Match(c.GetHeader("Accept-Language"))

		c.Set(languageKey, tag)
		c.Set(printerKey, message.NewPrinter(tag, message.Catalog(cat)))

		c.Header("Content-Language", tag.String())
		c.Writer.Header().Add("Vary", "Accept-Language")

		c.Next()
	}
}

// For returns the printer negotiated for this request.
//
// A nil context, or one that never passed through Middleware, gets an English
// printer rather than nil — a missing middleware registration should render an
// English message, not panic on the error path, which is the one path where a
// second failure is hardest to diagnose.
func For(c *gin.Context) *message.Printer {
	if c != nil {
		if p, ok := c.Value(printerKey).(*message.Printer); ok {
			return p
		}
	}
	return message.NewPrinter(language.English, message.Catalog(cat))
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

// Translated reports whether a key has a translation in a language.
//
// Used by the catalogue completeness test rather than by request handling: at
// request time an untranslated message renders as its English self, which is the
// intended fallback, so nothing needs to ask.
func Translated(tag language.Tag, key string) bool {
	p := message.NewPrinter(tag, message.Catalog(cat))
	english := message.NewPrinter(language.English, message.Catalog(cat))
	return p.Sprintf(key) != english.Sprintf(key) //nolint:govet // key is a catalogue lookup, not a format literal
}

// assert the catalogue builder satisfies what message.Catalog wants, so a
// dependency bump that changes the interface fails here rather than at a call.
var _ catalog.Catalog = (*catalog.Builder)(nil)

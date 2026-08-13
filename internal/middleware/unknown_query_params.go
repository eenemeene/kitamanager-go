package middleware

// Warn when a request carries query parameters the API does not declare.
//
// Gin drops unknown query parameters silently, which makes a misspelled filter
// one of the quietest bugs available: the request succeeds, the handler applies
// its default, and the caller believes it filtered. That is exactly what happened
// with `contract_on` — the frontend sent it for months, the children list
// defaulted to today, and the section board's date picker did nothing. No error,
// no failing test, no log line.
//
// The allowed set is derived from the OpenAPI spec swaggo already compiles into
// the binary, so there is no second list to maintain and it cannot drift from the
// handler annotations. A parameter that reaches here unrecognised is either a
// client bug or a missing @Param annotation — both worth knowing about.
//
// This only ever logs. Rejecting unknown parameters would be a stricter contract
// than the API promises today, and would break any client that appends its own
// tracking parameters.

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

// swaggerDoc is the subset of the spec this needs: query parameter names per
// path and method.
type swaggerDoc struct {
	BasePath string `json:"basePath"`
	Paths    map[string]map[string]struct {
		Parameters []struct {
			Name string `json:"name"`
			In   string `json:"in"`
		} `json:"parameters"`
	} `json:"paths"`
}

// declaredQueryParams maps "METHOD /gin/style/:path" to the query parameters the
// spec declares for it.
type declaredQueryParams map[string]map[string]struct{}

// parseDeclaredQueryParams builds the lookup table from a swagger 2.0 document.
//
// Returns nil when the document cannot be parsed, which disables the check rather
// than failing startup: this is a diagnostic, and a broken spec should not stop
// the API from serving.
func parseDeclaredQueryParams(specJSON string) declaredQueryParams {
	var doc swaggerDoc
	if err := json.Unmarshal([]byte(specJSON), &doc); err != nil {
		slog.Warn("unknown-query-parameter check disabled: cannot parse the embedded OpenAPI spec", "error", err)
		return nil
	}

	out := make(declaredQueryParams, len(doc.Paths))
	for path, ops := range doc.Paths {
		for method, op := range ops {
			names := make(map[string]struct{}, len(op.Parameters))
			for _, p := range op.Parameters {
				if p.In == "query" {
					names[p.Name] = struct{}{}
				}
			}
			out[routeKey(strings.ToUpper(method), specPathToGin(path))] = names
		}
	}
	return out
}

// specPathToGin rewrites OpenAPI's `{orgId}` placeholders into gin's `:orgId`, so
// a spec path can be matched against gin's registered route pattern.
func specPathToGin(path string) string {
	if !strings.ContainsRune(path, '{') {
		return path
	}
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{':
			b.WriteByte(':')
		case '}':
			// the placeholder ends at the next separator; nothing to write
		default:
			b.WriteByte(path[i])
		}
	}
	return b.String()
}

func routeKey(method, path string) string {
	return method + " " + path
}

// UnknownQueryParams logs a warning for each request carrying query parameters
// that its route does not declare.
//
// Pass docs.SwaggerInfo.ReadDoc() — the spec swaggo compiles into the binary.
func UnknownQueryParams(specJSON string) gin.HandlerFunc {
	declared := parseDeclaredQueryParams(specJSON)
	if declared == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		c.Next()

		query := c.Request.URL.Query()
		if len(query) == 0 {
			return
		}

		// FullPath is the registered pattern (empty for an unmatched route). A
		// route the spec does not describe cannot be judged — swagger UI, health,
		// metrics — so it is skipped rather than reported.
		route := c.FullPath()
		if route == "" {
			return
		}
		allowed, ok := declared[routeKey(c.Request.Method, route)]
		if !ok {
			return
		}

		var unknown []string
		for name := range query {
			if _, fine := allowed[name]; !fine {
				unknown = append(unknown, name)
			}
		}
		if len(unknown) == 0 {
			return
		}

		slog.Warn("request sent query parameters the API does not declare; they were ignored",
			"method", c.Request.Method,
			"route", route,
			"unknown_params", unknown,
			"status", c.Writer.Status(),
		)
	}
}

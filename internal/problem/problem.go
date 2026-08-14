// Package problem writes error responses as RFC 9457 problem details.
//
// Every error the API returns goes through Write. That is the point of the
// package: before it existed, five files constructed error bodies by hand and
// gin wrote two more shapes of its own (a plain-text 404 for an unrouted path,
// an empty body for a panic), so a client could not rely on any single field
// being present. A caller can now parse one document type for every failure.
//
// The document carries the RFC 9457 members plus two extensions. "code" is the
// programmatic contract — RFC 9457 deliberately has no opinion on error codes,
// and clients need one that does not involve parsing a URI. "request_id" ties
// the response to the server logs for the same request, which is the first
// thing anyone asks for when a user reports a 500.
package problem

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"golang.org/x/text/language"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/i18n"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ContentType is the media type RFC 9457 defines for these documents.
//
// A client that tests for exactly "application/json" will not match it. That is
// the specification's known cost, and it is why the frontend's response parsing
// checks for the "+json" structured suffix rather than an exact string.
const ContentType = "application/problem+json"

// requestIDKey mirrors middleware.RequestIDKey. It is duplicated rather than
// imported because the middleware package writes problem documents itself, and
// importing it here would be an import cycle. problem_test asserts the two
// constants are equal, so the duplication cannot drift silently.
const requestIDKey = "requestID"

// TypeBase prefixes the type URIs. They resolve to the errors reference in the
// user guide, one anchor per code — a type URI that 404s is the most common way
// this specification gets implemented badly, so these are generated from the
// same code constants the page documents.
const TypeBase = "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#"

// titles maps each error code to the short, human-readable summary that RFC 9457
// requires not to vary between occurrences of the same type. The detail carries
// what was specific to this one.
//
// These are English, and deliberately so: the API speaks English, and the code
// is the key the UI translates by. A German user sees German because the
// frontend looks the code up in its message catalogue, not because the server
// guessed at an Accept-Language header.
var titles = map[string]string{
	apperror.CodeNotFound:             "Resource not found",
	apperror.CodeBadRequest:           "Malformed request",
	apperror.CodeValidation:           "Validation failed",
	apperror.CodeConflict:             "Conflict with current state",
	apperror.CodeUnauthorized:         "Authentication required",
	apperror.CodeForbidden:            "Not permitted",
	apperror.CodeTooManyRequests:      "Too many requests",
	apperror.CodeInternal:             "Internal server error",
	apperror.CodeEmailConflict:        "Email address already in use",
	apperror.CodeContractConflict:     "Contract periods overlap",
	apperror.CodeDuplicateBillHash:    "Bill already imported",
	apperror.CodeDuplicateBillMonth:   "Bill already exists for this month",
	apperror.CodeMethodNotAllowed:     "Method not allowed",
	apperror.CodePreconditionRequired: "If-Match header required",
	apperror.CodePreconditionFailed:   "Resource was modified",
}

// Title returns the registered title for a code, falling back to the code
// itself so an unregistered code still produces a valid document rather than an
// empty title.
//
// The English title is the catalogue key, so a request that negotiated German
// gets the German title and one that did not gets the English string back
// unchanged. RFC 9457 asks for exactly this: a title is the same for every
// occurrence of a type "except for purposes of localization".
// Title returns the English title for a code, falling back to the code itself so
// an unregistered code still produces a valid document.
//
// English, deliberately: the top level of a problem document is the developer's
// view. The reader's language lives under "localized".
func Title(code string) string {
	if t, ok := titles[code]; ok {
		return t
	}
	return code
}

// TypeURI returns the type URI for a code.
//
// The code is appended verbatim: each code is a heading on the errors page, and
// Hugo's GitHub-style anchors keep underscores, so "contract_overlap" is both
// the code and the anchor. Rewriting the separator here would produce a URI that
// loads the right page and lands nowhere on it — the failure mode that makes so
// many implementations of this specification decorative.
//
// The URI points at the English page because the API is English-only. The German
// translation of this page exists for readers, not for machines.
func TypeURI(code string) string {
	return TypeBase + code
}

// New builds a problem document for a request without writing it.
func New(c *gin.Context, status int, code, detail string) models.ErrorResponse {
	p := models.ErrorResponse{
		Type:   TypeURI(code),
		Title:  Title(code),
		Status: status,
		Detail: detail,
		Code:   code,
	}
	if c != nil && c.Request != nil {
		p.Instance = c.Request.URL.Path
	}
	if c != nil {
		if id := c.GetString(requestIDKey); id != "" {
			p.RequestID = id
		}
	}
	p.Localized = localizedFor(c, code, detail, nil)
	return p
}

// localizedFor builds the user-facing view of a problem, or nil when there is
// nothing to add.
//
// Nothing to add means one of two things: the client is reading English, so the
// top level already is its language; or the catalogue has neither a title nor a
// message for this error, so a localized block would only repeat what is above
// it. Both produce an absent member rather than an empty object.
func localizedFor(c *gin.Context, code, messageID string, args []any) *models.LocalizedMessage {
	tag := i18n.LanguageFor(c)
	if tag == language.English {
		return nil
	}

	out := &models.LocalizedMessage{Locale: tag.String()}
	out.Title = i18n.Title(c, code)
	if messageID != "" {
		if detail, ok := i18n.Localize(c, messageID, args...); ok {
			out.Detail = detail
		}
	}
	if out.Title == "" && out.Detail == "" {
		return nil
	}
	return out
}

// Write sends a problem document and aborts the handler chain.
//
// It aborts rather than returning, because every caller was already pairing its
// write with an Abort and forgetting that pairing in middleware is how a
// rejected request goes on to run the handler anyway.
func Write(c *gin.Context, status int, code, detail string) {
	WriteProblem(c, New(c, status, code, detail))
}

// WriteProblem sends an already-built document, for the callers that attach
// extension members such as invalid_params.
func WriteProblem(c *gin.Context, p models.ErrorResponse) {
	c.Abort()
	c.Header("Content-Type", ContentType)
	if p.Localized != nil {
		// The body now genuinely contains two languages, and Content-Language
		// names the intended audience — so listing only the negotiated one would
		// misdescribe it. RFC 9110 allows the list for exactly this case.
		c.Header("Content-Language", "en, "+p.Localized.Locale)
	}
	c.JSON(p.Status, p)
}

// WriteError maps an application error onto a problem document.
//
// 5xx details are replaced with a fixed sentence: the underlying text can carry
// a driver message or a query fragment, and that belongs in the log the
// request_id points at, not in a response.
func WriteError(c *gin.Context, err error) {
	status := apperror.HTTPStatus(err)
	code := apperror.CodeInternal
	var appErr *apperror.AppError
	var params map[string]string
	if errors.As(err, &appErr) {
		code = appErr.GetErrorCode()
		params = appErr.GetParams()
	}

	// Detail is the English rendering, always. AppError keeps the format string
	// and its arguments apart so the localized view can be built from the same
	// pieces without replacing this one — a captured response stays readable by
	// whoever handles the support ticket, whatever language the user was in.
	detail := err.Error()
	if status >= http.StatusInternalServerError {
		slog.Error("Internal error",
			"error", err,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"request_id", c.GetString(requestIDKey),
		)
		detail = "An unexpected error occurred. Quote the request_id when reporting this."
	}

	doc := New(c, status, code, detail)
	// Params are dropped on a 5xx along with the detail: they describe an
	// internal failure, and the same argument applies to both.
	if status < http.StatusInternalServerError && appErr != nil {
		doc.Params = params
		// Re-derive the localized view from the format and its arguments rather
		// than from the rendered English, which New could only look up verbatim.
		doc.Localized = localizedFor(c, code, appErr.MessageID, appErr.Args)
	}
	WriteProblem(c, doc)
}

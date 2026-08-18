package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors for domain operations
var (
	ErrNotFound        = errors.New("resource not found")
	ErrBadRequest      = errors.New("bad request")
	ErrConflict        = errors.New("resource conflict")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrTooManyRequests = errors.New("too many requests")
	ErrInternalServer  = errors.New("internal server error")
)

// Error codes for programmatic handling
const (
	CodeNotFound           = "not_found"
	CodeBadRequest         = "bad_request"
	CodeValidation         = "validation_error"
	CodeConflict           = "conflict"
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeTooManyRequests    = "too_many_requests"
	CodeInternal           = "internal_error"
	CodeEmailConflict      = "email_conflict"
	CodeContractConflict   = "contract_overlap"
	CodeDuplicateBillHash  = "duplicate_bill_hash"
	CodeDuplicateBillMonth = "duplicate_bill_month"
	// CodePreconditionRequired and CodePreconditionFailed are separate because
	// the remedies differ: the first means "send If-Match", the second means
	// "reload, the record moved on".
	// CodeMethodNotAllowed is returned by the router itself: the path exists,
	// this method does not. It has no AppError constructor because no handler
	// ever raises it.
	CodeMethodNotAllowed = "method_not_allowed"

	CodePreconditionRequired = "precondition_required"
	CodePreconditionFailed   = "precondition_failed"
)

// AppError wraps errors with HTTP context
type AppError struct {
	Err       error
	Message   string
	Code      int
	ErrorCode string // machine-readable error code

	// MessageID is the message before formatting — the English format string a
	// call site wrote. It is also the translation key: the response writer
	// re-renders it in the request's language, which is only possible while the
	// arguments are still separate from the text.
	//
	// Message holds the already-rendered English, and stays the value Error()
	// returns, so a log line is English no matter who made the request.
	MessageID string
	// Args are MessageID's formatting arguments, kept for that re-render.
	Args []any

	// Fields carries per-field validation failures as data. When set, the
	// response writer composes the message from them.
	Fields []FieldViolation

	// Params carries the specifics of this occurrence as data rather than as
	// prose: the dates that overlapped, the month already billed.
	//
	// The server renders the localized message itself, from MessageID and Args,
	// so Params is not what makes translation work — it is what lets a client
	// that wants to compose its own wording get at the specifics without parsing
	// them back out of an English sentence.
	Params map[string]string
}

// WithParams attaches the structured form of what went wrong.
func (e *AppError) WithParams(kv ...string) *AppError {
	if len(kv)%2 != 0 {
		panic("apperror: WithParams needs key/value pairs")
	}
	if e.Params == nil {
		e.Params = make(map[string]string, len(kv)/2)
	}
	for i := 0; i < len(kv); i += 2 {
		e.Params[kv[i]] = kv[i+1]
	}
	return e
}

// GetParams returns the structured params, or nil.
func (e *AppError) GetParams() map[string]string { return e.Params }

func (e *AppError) Error() string {
	if e.Message == "" && len(e.Fields) > 0 {
		parts := make([]string, 0, len(e.Fields))
		for _, f := range e.Fields {
			parts = append(parts, f.Field+" "+EnglishReason(f.Rule, f.Param))
		}
		return strings.Join(parts, "; ")
	}
	return e.Message
}

// EnglishReason renders a rule as the English sentence fragment that follows a
// field name. Kept here rather than in the i18n package so that Error(), which
// must never depend on a request, has no reason to reach for a localizer.
func EnglishReason(rule, param string) string {
	switch rule {
	case "required":
		return "is required"
	case "non_empty":
		return "must contain at least one entry"
	case "min":
		// The validator tag, which is a string length.
		return "must be at least " + param + " characters"
	case "min_value":
		// A numeric minimum. Separate from "min" because "at least 1 characters"
		// is wrong for a pay-plan step, and was wrong in both languages.
		return "must be at least " + param
	case "positive":
		return "must be greater than 0"
	case "mismatch":
		return "must match " + param
	default:
		return "is invalid"
	}
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// GetErrorCode returns the machine-readable error code
func (e *AppError) GetErrorCode() string {
	if e.ErrorCode != "" {
		return e.ErrorCode
	}
	// Default error codes based on HTTP status
	switch e.Code {
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusConflict:
		return CodeConflict
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusTooManyRequests:
		return CodeTooManyRequests
	default:
		return CodeInternal
	}
}

// FieldViolation names one request field that failed validation, as data.
//
// The alternative this replaces was a formatted sentence with the field path
// glued onto the front — "add_children[3].contracts[1]: from is required". That
// serves neither audience: a client has to parse prose to learn which field
// failed, and a translation of it is half German and half JSON path. Every
// specification in this space keeps the two apart — JSON:API's source.pointer,
// AIP-193's FieldViolation, RFC 9457's invalid-params, ASP.NET's dictionary
// keyed by path — and so does this.
//
// Rule is a small vocabulary rather than free text, so the reason can be
// rendered in any language: "required", "non_empty", "min", "positive",
// "mismatch". Param carries the rule's argument where it takes one.
type FieldViolation struct {
	Field string
	Rule  string
	Param string
}

// Field builds a violation, formatting the field path from its arguments.
func Field(rule, param, pathFormat string, args ...any) FieldViolation {
	path := pathFormat
	if len(args) > 0 {
		path = fmt.Sprintf(pathFormat, args...)
	}
	return FieldViolation{Field: path, Rule: rule, Param: param}
}

// RequiredField is the common case, kept short because it is most of them.
func RequiredField(pathFormat string, args ...any) *AppError {
	return InvalidFields(Field("required", "", pathFormat, args...))
}

// InvalidFields creates a 400 carrying field violations.
//
// The Message is left empty: the response writer composes it from the
// violations, so the English sentence and the localized one are built the same
// way from the same data instead of one being a translation of the other.
func InvalidFields(violations ...FieldViolation) *AppError {
	return &AppError{
		Err:       ErrBadRequest,
		Code:      http.StatusBadRequest,
		ErrorCode: CodeValidation,
		Fields:    violations,
	}
}

// render builds the English message and keeps the pieces needed to build it
// again in another language.
//
// Constructors take a format string and its arguments rather than an
// already-formatted string. That is what makes a translated message possible:
// "child %d not found" can be looked up and re-rendered, while
// "child 7 not found" can only be shown as-is. A call with no arguments is the
// common case and stays a plain string.
func render(e *AppError, msg string, args []any) *AppError {
	e.MessageID = msg
	e.Args = args
	if len(args) == 0 {
		e.Message = msg
		return e
	}
	e.Message = fmt.Sprintf(msg, args...)
	return e
}

// NotFound creates a not found error
func NotFound(resource string, args ...any) *AppError {
	return render(&AppError{Err: ErrNotFound, Code: http.StatusNotFound, ErrorCode: CodeNotFound}, resource+" not found", args)
}

// BadRequest creates a bad request error
func BadRequest(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrBadRequest, Code: http.StatusBadRequest, ErrorCode: CodeBadRequest}, msg, args)
}

// Validation creates a validation error (subset of bad request)
func Validation(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrBadRequest, Code: http.StatusBadRequest, ErrorCode: CodeValidation}, msg, args)
}

// Conflict creates a conflict error
func Conflict(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrConflict, Code: http.StatusConflict, ErrorCode: CodeConflict}, msg, args)
}

// EmailConflict creates an error for duplicate email
func EmailConflict() *AppError {
	return &AppError{Err: ErrConflict, Message: "email already in use", Code: http.StatusConflict, ErrorCode: CodeEmailConflict}
}

// ContractConflict creates an error for overlapping contracts
func ContractConflict(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrConflict, Code: http.StatusConflict, ErrorCode: CodeContractConflict}, msg, args)
}

// PreconditionRequired creates a 428 for a write that arrived without the
// If-Match precondition it is required to carry. Distinct from 412: nothing was
// compared, so the client has to read the resource and try again with its
// version rather than assume it lost a race.
func PreconditionRequired(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrBadRequest, Code: http.StatusPreconditionRequired, ErrorCode: CodePreconditionRequired}, msg, args)
}

// PreconditionFailed creates a 412 for a write whose If-Match version no longer
// matches the stored one: someone else changed the record since the client read
// it. The remedy is to reload and reapply, which is why this is not a 409 — no
// overlap or constraint was violated.
func PreconditionFailed(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrConflict, Code: http.StatusPreconditionFailed, ErrorCode: CodePreconditionFailed}, msg, args)
}

// TooManyRequests creates a 429 rate-limit error
func TooManyRequests(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrTooManyRequests, Code: http.StatusTooManyRequests, ErrorCode: CodeTooManyRequests}, msg, args)
}

// Forbidden creates a forbidden error
func Forbidden(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrForbidden, Code: http.StatusForbidden, ErrorCode: CodeForbidden}, msg, args)
}

// Internal creates an internal server error
func Internal(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrInternalServer, Code: http.StatusInternalServerError, ErrorCode: CodeInternal}, msg, args)
}

// InternalWrap creates an internal server error that wraps the original error.
// The original error is available via Unwrap() for logging/debugging but is not
// exposed in HTTP responses.
func InternalWrap(err error, msg string, args ...any) *AppError {
	wrapped := render(&AppError{Code: http.StatusInternalServerError, ErrorCode: CodeInternal}, msg, args)
	wrapped.Err = fmt.Errorf("%s: %w", wrapped.Message, err)
	return wrapped
}

// Unauthorized creates an unauthorized error
func Unauthorized(msg string, args ...any) *AppError {
	return render(&AppError{Err: ErrUnauthorized, Code: http.StatusUnauthorized, ErrorCode: CodeUnauthorized}, msg, args)
}

// NewAppError creates a custom AppError with specified code
func NewAppError(err error, msg string, code int) *AppError {
	return &AppError{Err: err, Message: msg, Code: code}
}

// NewAppErrorWithCode creates a custom AppError with specified HTTP code and error code
func NewAppErrorWithCode(err error, msg string, httpCode int, errorCode string) *AppError {
	return &AppError{Err: err, Message: msg, Code: httpCode, ErrorCode: errorCode}
}

// HTTPStatus returns appropriate status code for error
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden
	}
	if errors.Is(err, ErrTooManyRequests) {
		return http.StatusTooManyRequests
	}
	if errors.Is(err, ErrUnauthorized) {
		return http.StatusUnauthorized
	}
	return http.StatusInternalServerError
}

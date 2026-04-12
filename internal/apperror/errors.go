package apperror

import (
	"errors"
	"fmt"
	"net/http"
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
)

// AppError wraps errors with HTTP context
type AppError struct {
	Err       error
	Message   string
	Code      int
	ErrorCode string // machine-readable error code
}

func (e *AppError) Error() string {
	return e.Message
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

// NotFound creates a not found error
func NotFound(resource string) *AppError {
	return &AppError{Err: ErrNotFound, Message: resource + " not found", Code: http.StatusNotFound, ErrorCode: CodeNotFound}
}

// BadRequest creates a bad request error
func BadRequest(msg string) *AppError {
	return &AppError{Err: ErrBadRequest, Message: msg, Code: http.StatusBadRequest, ErrorCode: CodeBadRequest}
}

// Validation creates a validation error (subset of bad request)
func Validation(msg string) *AppError {
	return &AppError{Err: ErrBadRequest, Message: msg, Code: http.StatusBadRequest, ErrorCode: CodeValidation}
}

// Conflict creates a conflict error
func Conflict(msg string) *AppError {
	return &AppError{Err: ErrConflict, Message: msg, Code: http.StatusConflict, ErrorCode: CodeConflict}
}

// EmailConflict creates an error for duplicate email
func EmailConflict() *AppError {
	return &AppError{Err: ErrConflict, Message: "email already in use", Code: http.StatusConflict, ErrorCode: CodeEmailConflict}
}

// ContractConflict creates an error for overlapping contracts
func ContractConflict(msg string) *AppError {
	return &AppError{Err: ErrConflict, Message: msg, Code: http.StatusConflict, ErrorCode: CodeContractConflict}
}

// TooManyRequests creates a 429 rate-limit error
func TooManyRequests(msg string) *AppError {
	return &AppError{Err: ErrTooManyRequests, Message: msg, Code: http.StatusTooManyRequests, ErrorCode: CodeTooManyRequests}
}

// Forbidden creates a forbidden error
func Forbidden(msg string) *AppError {
	return &AppError{Err: ErrForbidden, Message: msg, Code: http.StatusForbidden, ErrorCode: CodeForbidden}
}

// Internal creates an internal server error
func Internal(msg string) *AppError {
	return &AppError{Err: ErrInternalServer, Message: msg, Code: http.StatusInternalServerError, ErrorCode: CodeInternal}
}

// InternalWrap creates an internal server error that wraps the original error.
// The original error is available via Unwrap() for logging/debugging but is not
// exposed in HTTP responses.
func InternalWrap(err error, msg string) *AppError {
	return &AppError{Err: fmt.Errorf("%s: %w", msg, err), Message: msg, Code: http.StatusInternalServerError, ErrorCode: CodeInternal}
}

// Unauthorized creates an unauthorized error
func Unauthorized(msg string) *AppError {
	return &AppError{Err: ErrUnauthorized, Message: msg, Code: http.StatusUnauthorized, ErrorCode: CodeUnauthorized}
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

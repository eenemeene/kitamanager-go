package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNotFound(t *testing.T) {
	err := NotFound("user")
	if err.Error() != "user not found" {
		t.Errorf("expected 'user not found', got %q", err.Error())
	}
	if err.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.Code)
	}
	if err.GetErrorCode() != CodeNotFound {
		t.Errorf("expected code %q, got %q", CodeNotFound, err.GetErrorCode())
	}
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected err to wrap ErrNotFound")
	}
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input")
	if err.Error() != "invalid input" {
		t.Errorf("expected 'invalid input', got %q", err.Error())
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.Code)
	}
	if err.GetErrorCode() != CodeBadRequest {
		t.Errorf("expected code %q, got %q", CodeBadRequest, err.GetErrorCode())
	}
}

func TestValidation(t *testing.T) {
	err := Validation("field required")
	if err.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.Code)
	}
	if err.GetErrorCode() != CodeValidation {
		t.Errorf("expected code %q, got %q", CodeValidation, err.GetErrorCode())
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("duplicate")
	if err.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, err.Code)
	}
	if err.GetErrorCode() != CodeConflict {
		t.Errorf("expected code %q, got %q", CodeConflict, err.GetErrorCode())
	}
}

func TestEmailConflict(t *testing.T) {
	err := EmailConflict()
	if err.Error() != "email already in use" {
		t.Errorf("expected 'email already in use', got %q", err.Error())
	}
	if err.GetErrorCode() != CodeEmailConflict {
		t.Errorf("expected code %q, got %q", CodeEmailConflict, err.GetErrorCode())
	}
}

func TestContractConflict(t *testing.T) {
	err := ContractConflict("overlapping dates")
	if err.GetErrorCode() != CodeContractConflict {
		t.Errorf("expected code %q, got %q", CodeContractConflict, err.GetErrorCode())
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")
	if err.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, err.Code)
	}
	if err.GetErrorCode() != CodeForbidden {
		t.Errorf("expected code %q, got %q", CodeForbidden, err.GetErrorCode())
	}
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("not authenticated")
	if err.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, err.Code)
	}
	if err.GetErrorCode() != CodeUnauthorized {
		t.Errorf("expected code %q, got %q", CodeUnauthorized, err.GetErrorCode())
	}
}

func TestInternal(t *testing.T) {
	err := Internal("something broke")
	if err.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if err.GetErrorCode() != CodeInternal {
		t.Errorf("expected code %q, got %q", CodeInternal, err.GetErrorCode())
	}
}

func TestInternalWrap(t *testing.T) {
	cause := fmt.Errorf("db connection failed")
	err := InternalWrap(cause, "failed to fetch user")

	if err.Error() != "failed to fetch user" {
		t.Errorf("expected 'failed to fetch user', got %q", err.Error())
	}
	if err.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, err.Code)
	}

	// The wrapped error should contain the original cause
	var unwrapped error = err
	if inner := errors.Unwrap(unwrapped); inner == nil {
		t.Error("expected unwrappable error")
	}
}

func TestGetErrorCode_DefaultsByHTTPStatus(t *testing.T) {
	tests := []struct {
		httpCode int
		wantCode string
	}{
		{http.StatusNotFound, CodeNotFound},
		{http.StatusBadRequest, CodeBadRequest},
		{http.StatusConflict, CodeConflict},
		{http.StatusUnauthorized, CodeUnauthorized},
		{http.StatusForbidden, CodeForbidden},
		{http.StatusInternalServerError, CodeInternal},
		{http.StatusServiceUnavailable, CodeInternal}, // unknown maps to internal
	}

	for _, tt := range tests {
		err := &AppError{Err: errors.New("test"), Message: "test", Code: tt.httpCode}
		if got := err.GetErrorCode(); got != tt.wantCode {
			t.Errorf("GetErrorCode() for status %d = %q, want %q", tt.httpCode, got, tt.wantCode)
		}
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"AppError", NotFound("x"), http.StatusNotFound},
		{"AppError bad request", BadRequest("x"), http.StatusBadRequest},
		{"AppError conflict", Conflict("x"), http.StatusConflict},
		{"AppError forbidden", Forbidden("x"), http.StatusForbidden},
		{"AppError unauthorized", Unauthorized("x"), http.StatusUnauthorized},
		{"sentinel ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"sentinel ErrBadRequest", ErrBadRequest, http.StatusBadRequest},
		{"sentinel ErrConflict", ErrConflict, http.StatusConflict},
		{"sentinel ErrForbidden", ErrForbidden, http.StatusForbidden},
		{"sentinel ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"wrapped sentinel", fmt.Errorf("wrap: %w", ErrNotFound), http.StatusNotFound},
		{"unknown error", errors.New("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatus(tt.err); got != tt.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewAppError(t *testing.T) {
	err := NewAppError(errors.New("cause"), "msg", http.StatusTeapot)
	if err.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, err.Code)
	}
}

func TestNewAppErrorWithCode(t *testing.T) {
	err := NewAppErrorWithCode(errors.New("cause"), "msg", http.StatusTeapot, "teapot")
	if err.GetErrorCode() != "teapot" {
		t.Errorf("expected 'teapot', got %q", err.GetErrorCode())
	}
}

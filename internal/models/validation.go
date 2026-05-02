package models

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// VoucherNumberPattern is the canonical Berlin Gutschein format used
// across the codebase: "GB-" + 11 digits + "-" + 2 digits.
//
// The same regex is duplicated in `internal/isbj/parse.go` for parser-side
// validation; keeping them in sync is a manual chore but the two layers
// have different ownership (one rejects malformed XLSX rows, this one
// gates the JSON API). If the format ever changes, update both.
var VoucherNumberPattern = regexp.MustCompile(`^GB-\d{11}-\d{2}$`)

// init registers custom binding validators on package import so every
// caller — production main, unit tests, integration tests — gets the
// same set without explicit setup. Re-registration on subsequent
// imports is safe (RegisterValidation overwrites by tag).
//
// Validators registered:
//
//   - "voucher": matches VoucherNumberPattern. Used by
//     ChildVoucherCreateRequest to gate the API boundary against
//     pattern-violating Gutschein numbers (audit finding I-M-4).
//
// A registration failure here is a programmer error (e.g. empty tag
// name) — panic so it's caught at startup, not as a baffling 500 in
// production.
func init() {
	if err := RegisterCustomValidators(); err != nil {
		panic("models: failed to register custom validators: " + err.Error())
	}
}

// RegisterCustomValidators wires the custom validators into Gin's default
// validator engine. Exposed in addition to init() so callers (notably
// cmd/api/main.go) can surface a structured error to logs instead of
// panicking — even though in practice the underlying call cannot fail.
func RegisterCustomValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	return v.RegisterValidation("voucher", validateVoucher)
}

func validateVoucher(fl validator.FieldLevel) bool {
	return VoucherNumberPattern.MatchString(fl.Field().String())
}

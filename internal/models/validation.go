package models

import (
	"reflect"
	"regexp"
	"strings"

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
	v.RegisterTagNameFunc(jsonFieldName)
	return v.RegisterValidation("voucher", validateVoucher)
}

// jsonFieldName makes validator report the wire name of a field rather than its
// Go name, so `weekly_hours` fails as "weekly_hours" and not "WeeklyHours".
//
// invalid_params is consumed as data: the frontend resolves each entry's `field`
// as a JSON path against the form's values and marks that input. A Go name
// resolves to nothing, so the violation silently fell through to the "unmapped"
// list and the user got a sentence with no field marked — the exact failure
// applyProblemToForm is built to prevent, arriving through the one producer it
// could not see. Every hand-built violation (apperror.RequiredField and friends)
// already used JSON paths; only validator's disagreed.
//
// Fields tagged `json:"-"` report an empty name, which validator treats as
// "use the Go name". They are unexported from the wire anyway, so a caller
// cannot have sent one.
func jsonFieldName(f reflect.StructField) string {
	name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}
	return name
}

func validateVoucher(fl validator.FieldLevel) bool {
	return VoucherNumberPattern.MatchString(fl.Field().String())
}

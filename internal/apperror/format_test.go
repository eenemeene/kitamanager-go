package apperror_test

import (
	"io/fs"

	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestErrorFormatArity checks every apperror call site whose message is a format
// string: the number of verbs must match the number of arguments supplied.
//
// This exists because making the constructors variadic removed a check that used
// to come for free. `apperror.BadRequest(fmt.Sprintf("section %d ...", id))` is a
// direct fmt call, so `go vet` verified it. `apperror.BadRequest("section %d ...",
// id)` is not, and vet does not follow the format string into a user function —
// verified by mutation, and confirmed not to be fixable from configuration:
// listing the constructors in vet's `printf.funcs` (both through golangci-lint
// and as a bare `-printf.funcs` flag, in all three documented name forms)
// produced no diagnostic, because the constructors hand the format to a shared
// helper and vet records no wrapper fact across that hop.
//
// Without this, a mistyped verb reaches a user as "%!d(MISSING)" or
// "%!s(int=7)" in an error message, and nothing fails first.
//
// What it does not cover: argument *types*. A %s given an int has the right
// arity and is not caught here. Closing that would need go/types over every
// package; arity is the common mistake and this is the cheap 90%.
func TestErrorFormatArity(t *testing.T) {
	// Which argument holds the format string, per constructor. Everything else
	// after it is a formatting argument.
	formatArg := map[string]int{
		"BadRequest":           0,
		"Conflict":             0,
		"ContractConflict":     0,
		"Forbidden":            0,
		"Internal":             0,
		"NotFound":             0,
		"PreconditionFailed":   0,
		"PreconditionRequired": 0,
		"TooManyRequests":      0,
		"Unauthorized":         0,
		"Validation":           0,
		// InternalWrap takes the wrapped error first.
		"InternalWrap": 1,
	}

	root := filepath.Join("..", "..")
	checked := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", ".git", "frontend", "website", "docs":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			// A file that does not parse is not this test's problem; the build
			// will say so more clearly.
			return nil //nolint:nilerr // parse failures surface from the compiler
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "apperror" {
				return true
			}
			idx, ok := formatArg[sel.Sel.Name]
			if !ok || len(call.Args) <= idx {
				return true
			}
			lit, ok := call.Args[idx].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				// A message built from a variable cannot be checked statically.
				return true
			}
			format, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}

			wantVerbs := countVerbs(format)
			gotArgs := len(call.Args) - idx - 1
			// A call that spreads a slice (f(msg, args...)) supplies an unknown
			// number of arguments.
			if call.Ellipsis.IsValid() {
				return true
			}
			checked++
			if wantVerbs != gotArgs {
				pos := fset.Position(call.Pos())
				t.Errorf("%s:%d: apperror.%s has %d verb(s) in %q but %d argument(s)",
					pos.Filename, pos.Line, sel.Sel.Name, wantVerbs, format, gotArgs)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the tree: %v", err)
	}

	// A guard against the check silently walking nothing — a wrong root or a
	// changed layout would otherwise make this test pass by inspecting zero
	// call sites.
	if checked < 300 {
		t.Fatalf("only inspected %d apperror call sites, expected 300+; the walk root or layout has changed", checked)
	}
	t.Logf("checked %d apperror call sites", checked)
}

// countVerbs counts formatting verbs, treating %% as a literal percent.
func countVerbs(format string) int {
	n := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			i++
			continue
		}
		// Skip flags, width and precision to land on the verb itself.
		j := i + 1
		for j < len(format) && strings.ContainsRune("+-# 0123456789.*", rune(format[j])) {
			j++
		}
		if j < len(format) {
			n++
			i = j
		}
	}
	return n
}

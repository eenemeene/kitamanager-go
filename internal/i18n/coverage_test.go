package i18n_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/text/language"

	"github.com/eenemeene/kitamanager-go/internal/i18n"
)

// userFacing lists the constructors whose message can reach a client as `detail`,
// with the index of the argument holding it.
//
// Internal and InternalWrap are deliberately absent: a 5xx detail is replaced
// with fixed text before it leaves the process, so those messages are for the
// log and translating them would be work with no reader.
var userFacing = map[string]int{
	"BadRequest": 0, "NotFound": 0, "Validation": 0, "Conflict": 0,
	"ContractConflict": 0, "Forbidden": 0, "Unauthorized": 0,
	"TooManyRequests": 0, "PreconditionFailed": 0, "PreconditionRequired": 0,
}

// TestEveryUserFacingMessageIsRegistered is what makes the English-as-lookup-key
// design safe.
//
// Call sites write English prose and the registry maps it to a stable ID. The
// hazard of that split is silence: reword the English in a later PR and the
// lookup simply misses, the message falls back to English, and no German reader
// is around to file the bug. This turns that into a build failure that names the
// exact string to add.
//
// It is also the gate that lets the frontend stop carrying its own catalogue. A
// fallback there was covering incomplete server coverage; deleting it is only
// safe while coverage is total, and this is the thing that keeps it total.
func TestEveryUserFacingMessageIsRegistered(t *testing.T) {
	root := filepath.Join("..", "..")
	checked := 0
	var unregistered []string

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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
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
			idx, ok := userFacing[sel.Sel.Name]
			if !ok || len(call.Args) <= idx {
				return true
			}
			lit, ok := call.Args[idx].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				// Built from a variable: nothing static to register.
				return true
			}
			msg, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			// NotFound composes "<resource> not found", and that composed string
			// is what reaches the localizer — so it is what must be registered.
			if sel.Sel.Name == "NotFound" {
				msg += " not found"
			}

			checked++
			if _, ok := i18n.Registered(msg); !ok {
				pos := fset.Position(call.Pos())
				unregistered = append(unregistered,
					pos.Filename+":"+strconv.Itoa(pos.Line)+"  "+strconv.Quote(msg))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the tree: %v", err)
	}

	// Guard against the walk finding nothing — a moved layout would otherwise
	// make this pass by inspecting zero call sites.
	if checked < 300 {
		t.Fatalf("only inspected %d user-facing call sites, expected 300+; the walk root or layout has changed", checked)
	}
	if len(unregistered) > 0 {
		t.Errorf("%d user-facing message(s) have no registry entry, so a German reader gets English.\n"+
			"Add each to internal/i18n/registry.go with a stable id, and its text to locales/en.json and de.json:\n  %s",
			len(unregistered), strings.Join(unregistered, "\n  "))
	}
	t.Logf("checked %d user-facing call sites, all registered", checked)
}

// TestEveryRegisteredMessageIsTranslated closes the other direction: an entry can
// exist in the registry with nothing behind it in either catalogue, which renders
// as the message ID.
func TestEveryRegisteredMessageIsTranslated(t *testing.T) {
	for format, e := range i18n.RegistryEntries() {
		plural := e.Plural != ""
		if !i18n.Has(language.English, e.ID, plural) {
			t.Errorf("id %q (from %q) has no English source in locales/en.json", e.ID, format)
		}
		if !i18n.Has(language.German, e.ID, plural) {
			t.Errorf("id %q (from %q) has no German translation in locales/de.json", e.ID, format)
		}
	}
}

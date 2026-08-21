package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

// fieldShape is what the Go source says about one JSON field: whether the
// marshaller may leave it out, and whether it may write null.
type fieldShape struct {
	// Omitted is true when the field carries `omitempty` (or `omitzero`) and
	// has a type that can actually be empty — a pointer, slice or map. An int
	// with omitempty is not interesting here: the spec's `required` claim is
	// still wrong for it in principle, but a missing number reads as 0 either
	// way and no consumer can tell.
	Omitted bool
	// Nullable is true when the field is a pointer that is always written.
	// `*time.Time` with a plain `json:"to"` tag marshals to null when nil.
	Nullable bool
}

// goStructFields parses the model package and reports, per struct, what each
// JSON field's type and tag actually permit.
//
// The spec cannot answer this on its own. swaggo emits no signal for
// `omitempty`, so the fixer's "responses populate every field" rule had no way
// to see the exceptions — and there are 76 of them. Reading the source is the
// only source of truth, and it lives two directories away.
func goStructFields(modelsDir string) (map[string]map[string]fieldShape, error) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", modelsDir, err)
	}

	fset := token.NewFileSet()
	out := map[string]map[string]fieldShape{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(modelsDir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		collectStructFields(file, out)
	}
	if len(out) == 0 {
		// A silent empty map would quietly restore the old all-required
		// behaviour, which is the bug this exists to fix. Fail loudly instead.
		return nil, fmt.Errorf("no structs found in %s", modelsDir)
	}
	return out, nil
}

// collectStructFields records every JSON-tagged field of every struct in one file.
func collectStructFields(file *ast.File, out map[string]map[string]fieldShape) {
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		structType, ok := spec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		fields := map[string]fieldShape{}
		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			jsonTag := tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			parts := strings.Split(jsonTag, ",")
			jsonName := parts[0]
			if jsonName == "" {
				continue
			}
			opts := parts[1:]
			omit := slices.Contains(opts, "omitempty") || slices.Contains(opts, "omitzero")

			ptr, nilable := typeShape(field.Type)
			fields[jsonName] = fieldShape{
				Omitted:  omit && nilable,
				Nullable: ptr && !omit,
			}
		}
		if len(fields) > 0 {
			out[spec.Name.Name] = fields
		}
		return true
	})
}

// typeShape reports whether an expression is a pointer, and whether it is
// nilable at all (pointer, slice, map or interface).
func typeShape(expr ast.Expr) (ptr, nilable bool) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return true, true
	case *ast.ArrayType:
		// A fixed-size array is not nilable; a slice is.
		return false, t.Len == nil
	case *ast.MapType:
		return false, true
	case *ast.InterfaceType:
		return false, true
	case *ast.IndexExpr:
		// A generic instantiation such as Opt[time.Time]. These carry
		// `omitzero` plus an explicit x-nullable extension already, and the
		// omitempty branch above covers them.
		return false, true
	}
	return false, false
}

// modelsDirFrom locates internal/models relative to the repo root the tool was
// invoked from. Kept separate so the swagger-check flow, which writes its spec
// to a temp directory, still reads the models from the working tree.
func modelsDirFrom(root string) string {
	return filepath.Join(root, "internal", "models")
}

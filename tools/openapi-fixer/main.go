// Command openapi-fixer converts the Swagger 2.0 spec produced by swaggo
// (docs/swagger.json) to OpenAPI 3.0 (docs/openapi.json).
//
// The Swagger 2.0 output is the swaggo-native artifact; the 3.0 output is
// the contract for downstream type generation (frontend uses
// openapi-typescript against this file). swaggo cannot emit 3.x natively,
// so this binary owns the conversion step and any post-processing we
// later need to bolt on.
//
// Post-processing applied:
//
//   - Schema names are stripped of the swaggo --parseDependency prefix
//     (e.g. github_com_eenemeene_kitamanager-go_internal_models.User → User).
//     swaggo emits fully-qualified Go module paths to disambiguate types
//     across packages; we don't have that ambiguity in this spec, and
//     the long names produce unreadable generated TypeScript.
//
//   - Schemas whose name ends in `Response` get every property marked
//     required. swaggo only emits `required` from `binding:"required"`
//     tags, which by convention only appear on request DTOs; response
//     fields are always present at runtime but the spec doesn't say so.
//     We assert that runtime contract here so generated TypeScript types
//     don't become useless walls of `field?: T | undefined`.
//
// Run via `make swagger-docs`; CI invokes the same target with a
// dirty-tree check.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

// schemaPrefix is the module-path prefix swaggo --parseDependency stamps
// onto every schema name. It does not appear in JSON wire payloads at
// runtime; it only exists to disambiguate Go types in the spec.
const schemaPrefix = "github_com_eenemeene_kitamanager-go_internal_models."

func main() {
	in := flag.String("in", "docs/swagger.json", "input Swagger 2.0 file")
	out := flag.String("out", "docs/openapi.json", "output OpenAPI 3.0 file")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintf(os.Stderr, "openapi-fixer: %v\n", err)
		os.Exit(1)
	}
}

func run(inPath, outPath string) error {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inPath, err)
	}

	var v2 openapi2.T
	if err := json.Unmarshal(data, &v2); err != nil {
		return fmt.Errorf("parse swagger 2.0: %w", err)
	}

	v3, err := openapi2conv.ToV3(&v2)
	if err != nil {
		return fmt.Errorf("convert to openapi 3.0: %w", err)
	}

	markResponsePropertiesRequired(v3)

	// kin-openapi marshals fields in a stable order via tagged structs;
	// the only non-determinism left would come from unordered maps in the
	// underlying spec. openapi3.T uses orderedmap for Paths/Components so
	// this round-trips deterministically.
	specBytes, err := json.MarshalIndent(v3, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal openapi 3.0: %w", err)
	}

	specBytes = stripSchemaPrefix(specBytes)
	specBytes = append(specBytes, '\n')

	if err := os.WriteFile(outPath, specBytes, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

// markResponsePropertiesRequired walks every schema whose name does NOT
// end in `Request` and adds every defined property to its `required`
// list (deduplicating against any required entries already present).
//
// swaggo derives the `required` array from `binding:"required"` struct
// tags, which by convention only appear on request DTOs. Response and
// supporting (shared) types end up with no required properties, which
// generates useless TypeScript where every consumer must ?? every read.
//
// We trust the convention: response/shared structs populate every field
// they declare. If a response field is genuinely optional (omitempty +
// may be nil), this rule is wrong for that field — checking individual
// fields would require reading Go source. Revisit per-field if false
// positives appear at consumer sites.
//
// Request schemas are left alone: their `required` lists come straight
// from `binding:"required"` tags which is the right truth for inputs.
func markResponsePropertiesRequired(spec *openapi3.T) {
	if spec.Components == nil {
		return
	}
	for name, ref := range spec.Components.Schemas {
		if ref == nil || ref.Value == nil {
			continue
		}
		shortName := strings.TrimPrefix(name, schemaPrefix)
		// Apply the "all properties required" rule selectively. Schemas
		// covered: anything that's a response shape (the Go DTO populates
		// every field at runtime), plus the inline shared types that
		// nest inside responses.
		//
		// Excluded:
		//   - *Request types: their `required` list comes from
		//     `binding:"required"` Go tags, which is the correct truth
		//     for inputs; forcing every field required would break
		//     partial updates.
		//   - *BatchUpdateEntry wrappers (e.g.
		//     ChildContractBatchUpdateEntry): they embed an UpdateRequest
		//     and inherit its partial-update semantics. Other *Entry
		//     types (e.g. ChildBillingSummaryEntry) ARE response shapes
		//     and should keep the all-required treatment.
		if strings.HasSuffix(shortName, "Request") {
			continue
		}
		if strings.HasSuffix(shortName, "BatchUpdateEntry") {
			continue
		}
		schema := ref.Value
		if len(schema.Properties) == 0 {
			continue
		}
		existing := make(map[string]struct{}, len(schema.Required))
		for _, r := range schema.Required {
			existing[r] = struct{}{}
		}
		for prop := range schema.Properties {
			if _, ok := existing[prop]; !ok {
				schema.Required = append(schema.Required, prop)
				existing[prop] = struct{}{}
			}
		}
		sort.Strings(schema.Required)
	}
}

// stripSchemaPrefix rewrites both schema definition keys and $ref strings
// so consumers reference types by short name (User, ChildContractResponse)
// instead of the swaggo-stamped fully qualified path. Operates on the
// JSON byte stream rather than the spec model because the rename touches
// dozens of unrelated $ref sites — string substitution is correct and
// trivially deterministic for these payloads.
//
// A second pattern handles the underscore-joined variant swaggo emits
// inside generic-instance names like
// PaginatedResponse-github_com_eenemeene_kitamanager-go_internal_models_UserResponse.
func stripSchemaPrefix(in []byte) []byte {
	// Dotted form (component-of-paths references): "github_com_..._models.User"
	out := bytes.ReplaceAll(in, []byte(schemaPrefix), nil)

	// Underscored form (inside generic-instance suffixes):
	// "...PaginatedResponse-github_com_..._models_UserResponse"
	underscored := strings.ReplaceAll(schemaPrefix, ".", "_")
	out = bytes.ReplaceAll(out, []byte(underscored), nil)

	// Sanity check: no lingering occurrences of the prefix in either form.
	if leftover := regexp.MustCompile(`github_com_eenemeene_kitamanager-go`).Find(out); leftover != nil {
		panic(fmt.Sprintf("openapi-fixer: schema prefix not fully stripped, found %q", leftover))
	}
	return out
}

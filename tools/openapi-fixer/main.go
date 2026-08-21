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
//   - Response schemas get their properties marked required, except the ones
//     the Go source says may be absent. swaggo only emits `required` from
//     `binding:"required"` tags, which by convention only appear on request
//     DTOs, so response fields carry no requiredness at all and the generated
//     TypeScript would be a useless wall of `field?: T | undefined`.
//
//     The requiredness cannot be read off the spec, because swaggo emits no
//     signal for `omitempty`; it is read from internal/models directly (see
//     gofields.go). A field that is `omitempty` on a pointer, slice or map is
//     left optional, and a pointer that is always written is marked nullable.
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
	"slices"
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
	models := flag.String("models", modelsDirFrom("."), "directory holding the model DTOs")
	flag.Parse()

	if err := run(*in, *out, *models); err != nil {
		fmt.Fprintf(os.Stderr, "openapi-fixer: %v\n", err)
		os.Exit(1)
	}
}

func run(inPath, outPath, modelsDir string) error {
	// #nosec G304 -- inPath is the path the developer typed on the
	// command line to a local dev tool that runs in `make swagger-fix`.
	// There is no network surface, no untrusted input source, and the
	// tool is never deployed.
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

	shapes, err := goStructFields(modelsDir)
	if err != nil {
		return err
	}
	markResponsePropertiesRequired(v3, shapes)
	allowFreeFormObjectProperties(v3)
	assignOperationIDs(v3)
	declareContractETag(v3)
	declareEnrollmentPayloadUnion(v3)
	declareProblemContentType(v3)
	declareServer(v3)
	upgradeTo31(v3)

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

// declareContractETag documents the ETag that single-contract reads return.
//
// The handler sets it (see setVersionETag) because a client needs the contract's
// version to send back as an If-Match precondition, and a header is the standard
// place to publish it. swaggo's @Header annotation produced nothing here — it does
// not reach the emitted spec — so the header is declared at this step instead,
// beside the operationIds, rather than left undocumented.
//
// Undocumented, the concurrency contract is only half stated: the write endpoints
// say they require If-Match without saying where its value comes from.
// declareProblemContentType re-labels error responses as problem documents.
//
// swaggo derives every response's media type from the operation's @Produce
// annotation, which is "application/json" for the whole API — so the generated
// spec claimed plain JSON for the error bodies even though the server sends
// RFC 9457's "application/problem+json". Annotating each of the 400-odd @Failure
// lines individually would be the alternative; doing it here keeps the one fact
// in one place, and it cannot fall out of step with a new endpoint.
//
// Only responses that actually carry the error schema are moved. A 4xx that
// returns something else (there are none today, but a file endpoint could)
// keeps its own media type.
// declareServer replaces the server list the 2.0 conversion produces.
//
// swaggo's @host is "localhost:8080", and the converter defaults the scheme to
// https when none is declared, so the published contract claimed the API lives at
// https://localhost:8080/ — a developer's machine, over a scheme that port does
// not speak. A relative server is both correct and deployment-agnostic: it
// resolves against whatever host served the document, which is what every
// consumer of this file actually wants.
func declareServer(doc *openapi3.T) {
	doc.Servers = openapi3.Servers{{
		URL:         "/",
		Description: "The host serving this document. All paths are absolute from the root.",
	}}
}

// upgradeTo31 turns the 3.0 document the converter emits into 3.1.
//
// The conversion path is swaggo 2.0 -> kin-openapi 3.0, and kin-openapi does not
// target 3.1, so the last step happens here. Only one construct in this spec
// differs between the versions: 3.0's `nullable: true` is not a keyword in 3.1,
// where nullability is expressed by including "null" in the type. There are nine
// of them, all from the presence-aware contract fields.
//
// Verified type-neutral before adopting: openapi-typescript generates a
// byte-identical file from the 3.0 and 3.1 forms of this spec, so the frontend
// contract does not move. What it buys is a document that is valid JSON Schema
// 2020-12, which is what tooling increasingly assumes.
func upgradeTo31(doc *openapi3.T) {
	doc.OpenAPI = "3.1.0"

	seen := map[*openapi3.Schema]bool{}
	var walk func(*openapi3.SchemaRef)
	walk = func(ref *openapi3.SchemaRef) {
		if ref == nil || ref.Value == nil || seen[ref.Value] {
			return
		}
		seen[ref.Value] = true
		sch := ref.Value

		if sch.Nullable {
			sch.Nullable = false
			if sch.Type != nil {
				types := append(sch.Type.Slice(), "null")
				sch.Type = &openapi3.Types{}
				*sch.Type = types
			}
		}

		for _, child := range sch.Properties {
			walk(child)
		}
		walk(sch.Items)
		walk(sch.AdditionalProperties.Schema)
		for _, group := range [][]*openapi3.SchemaRef{sch.AllOf, sch.AnyOf, sch.OneOf} {
			for _, child := range group {
				walk(child)
			}
		}
		walk(sch.Not)
	}

	for _, ref := range doc.Components.Schemas {
		walk(ref)
	}
	for _, item := range doc.Paths.Map() {
		for _, op := range item.Operations() {
			if op == nil {
				continue
			}
			for _, param := range op.Parameters {
				if param.Value != nil {
					walk(param.Value.Schema)
				}
			}
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				for _, mt := range op.RequestBody.Value.Content {
					walk(mt.Schema)
				}
			}
			if op.Responses != nil {
				for _, resp := range op.Responses.Map() {
					if resp.Value == nil {
						continue
					}
					for _, mt := range resp.Value.Content {
						walk(mt.Schema)
					}
				}
			}
		}
	}
}

func declareProblemContentType(doc *openapi3.T) {
	if doc.Paths == nil {
		return
	}
	const problemJSON = "application/problem+json"
	for _, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		for _, op := range item.Operations() {
			if op == nil || op.Responses == nil {
				continue
			}
			for status, ref := range op.Responses.Map() {
				if status < "400" || ref == nil || ref.Value == nil {
					continue
				}
				media := ref.Value.Content["application/json"]
				if media == nil || media.Schema == nil ||
					!strings.HasSuffix(media.Schema.Ref, "ErrorResponse") {
					continue
				}
				delete(ref.Value.Content, "application/json")
				ref.Value.Content[problemJSON] = media
			}
		}
	}
}

func declareContractETag(doc *openapi3.T) {
	if doc.Paths == nil {
		return
	}
	const desc = "The contract's version, quoted. Echo it back as `If-Match` when correcting, " +
		"amending, ending or deleting this contract; it is also on the body as `version`."
	for path, item := range doc.Paths.Map() {
		if item == nil || item.Get == nil || !strings.HasSuffix(path, "/contracts/{contractId}") {
			continue
		}
		ok := item.Get.Responses.Value("200")
		if ok == nil || ok.Value == nil {
			continue
		}
		if ok.Value.Headers == nil {
			ok.Value.Headers = openapi3.Headers{}
		}
		ok.Value.Headers["ETag"] = &openapi3.HeaderRef{Value: &openapi3.Header{Parameter: openapi3.Parameter{
			Description: desc,
			Schema:      openapi3.NewStringSchema().NewRef(),
		}}}
	}
}

// declareEnrollmentPayloadUnion types FactorResponse.enrollment as the union it
// actually is.
//
// The field is `any` in Go — the handler stays generic and each verifier produces
// its own shape — so swaggo emits it as an untyped blob. That left the three
// payload schemas referenced by nothing, and the previous workaround was to list
// them as extra `@Success 200` lines on the enroll endpoint. swaggo keeps only
// the last annotation per status code, so the published contract claimed the
// endpoint returned a bare WebAuthnEnrollmentPayload when it returns a
// FactorResponse. The frontend hand-wrote the correct type and was unaffected;
// the docs and the generated types were simply wrong.
//
// Swagger 2.0 has no oneOf, which is why this could not be said at the
// annotation level. OpenAPI 3.1 does, so it is said here: the payload schemas
// stay reachable because the union references them, and the contract gains a
// discriminated union in place of the blob.
func declareEnrollmentPayloadUnion(doc *openapi3.T) {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return
	}
	factor := doc.Components.Schemas[schemaPrefix+"FactorResponse"]
	if factor == nil || factor.Value == nil {
		return
	}
	enrollment := factor.Value.Properties["enrollment"]
	if enrollment == nil || enrollment.Value == nil {
		return
	}

	// Ordered by factor type as the enum on FactorResponse.type lists them, so
	// the generated union reads in the same order as the discriminator.
	branches := []string{"TOTPEnrollmentPayload", "BackupCodesPayload", "WebAuthnEnrollmentPayload"}
	oneOf := make(openapi3.SchemaRefs, 0, len(branches))
	for _, name := range branches {
		if doc.Components.Schemas[schemaPrefix+name] == nil {
			// The schema was renamed or removed; leave the blob rather than
			// emit a dangling $ref.
			return
		}
		oneOf = append(oneOf, openapi3.NewSchemaRef("#/components/schemas/"+schemaPrefix+name, nil))
	}

	enrollment.Value.OneOf = oneOf
	enrollment.Value.Description = "Populated only on the enrollment response. Which member " +
		"appears follows the factor's type: totp, backup_codes, or webauthn."

	// markResponsePropertiesRequired has already marked it required, on the rule
	// that response fields are always present. This one is the exception the rule
	// admits: it is `omitempty` in Go and every GET omits it, so requiring it
	// would force each caller to override the type back to optional.
	factor.Value.Required = slices.DeleteFunc(factor.Value.Required, func(name string) bool {
		return name == "enrollment"
	})
}

// assignOperationIDs gives every operation a stable, readable id derived from its
// method and path.
//
// swaggo only emits operationId from an explicit `@id` annotation, and this API
// has none across 155 operations. Generators fall back to inventing names from the
// path, which produces things like
// `getApiV1OrganizationsOrgIdChildrenChildIdContractsContractId` and, worse, names
// that change whenever a path does. Deriving them here keeps one rule in one place
// and leaves the handlers uncluttered.
//
// The shape is method + path segments, with parameters marked "By": GET
// /api/v1/organizations/{orgId}/children becomes `getOrganizationsByOrgIdChildren`.
func assignOperationIDs(doc *openapi3.T) {
	if doc.Paths == nil {
		return
	}
	for path, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		for method, op := range item.Operations() {
			if op == nil || op.OperationID != "" {
				continue
			}
			op.OperationID = operationID(method, path)
		}
	}
}

// operationID builds the identifier described on assignOperationIDs.
func operationID(method, path string) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(method))
	for _, seg := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if seg == "" || seg == "api" || seg == "v1" {
			continue
		}
		if strings.HasPrefix(seg, "{") {
			b.WriteString("By")
			seg = strings.Trim(seg, "{}")
		}
		b.WriteString(camel(seg))
	}
	return b.String()
}

// camel upper-cases the first rune and drops separators, so "government-funding"
// becomes "GovernmentFunding".
func camel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

// allowFreeFormObjectProperties states that an object-typed schema with no
// declared properties accepts arbitrary ones.
//
// swaggo emits `{"type": "object"}` for a `swaggertype:"object"` field — its way
// of saying "free-form object", which is how the contract-property maps on the
// correct/amend requests are declared. OpenAPI reads that literally, and strict
// generators take it at its word: openapi-typescript renders it as
// `Record<string, never>`, an object permitted to have *no* properties, which no
// caller can satisfy. The response side does not hit this because it $refs the
// named ContractProperties schema, which carries additionalProperties.
//
// Stating additionalProperties: true says what swaggo meant. The rule is safe
// because an object with neither declared properties nor a $ref carries no
// information otherwise — there is nothing it could describe except a free-form
// map.
func allowFreeFormObjectProperties(doc *openapi3.T) {
	if doc.Components == nil {
		return
	}
	for _, ref := range doc.Components.Schemas {
		if ref == nil || ref.Value == nil {
			continue
		}
		for _, prop := range ref.Value.Properties {
			if prop == nil || prop.Value == nil {
				continue
			}
			v := prop.Value
			if !v.Type.Is("object") || len(v.Properties) > 0 || v.AdditionalProperties.Has != nil || v.AdditionalProperties.Schema != nil {
				continue
			}
			allow := true
			v.AdditionalProperties.Has = &allow
		}
	}
}

// markResponsePropertiesRequired walks every schema whose name does NOT
// end in `Request` and adds every property to its `required` list, except the
// ones the Go source says may be absent — and marks nullable the ones that may
// be written as null.
//
// swaggo derives the `required` array from `binding:"required"` struct tags,
// which by convention only appear on request DTOs. Response and supporting
// (shared) types end up with no required properties, which generates useless
// TypeScript where every consumer must ?? every read.
//
// The blanket "responses populate every field" rule that used to live here was
// close but not true, and the exceptions were not rare: 76 fields across the
// spec carry `omitempty` on a pointer, slice or map, or are pointers that
// marshal to null. The generated types promised `child.contracts: T[]` for a
// child with no contracts and `attendance.check_in_time: string` for a record
// with none, so every consumer that trusted the type was one absent field away
// from a TypeError — and the compiler said the guard was unnecessary.
//
// `shapes` is read straight from internal/models, because the spec carries no
// signal for `omitempty` and there is nowhere else to learn it.
//
// Request schemas are left alone: their `required` lists come straight
// from `binding:"required"` tags which is the right truth for inputs.
func markResponsePropertiesRequired(spec *openapi3.T, shapes map[string]map[string]fieldShape) {
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
		//   - *Request and *Input types: their `required` list comes from
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
		// *Input types are request shapes too: the forecast overlay inputs
		// (ForecastChildInput and friends) describe what a caller sends, so
		// their optional fields must stay optional. They are named Input rather
		// than Request because they are members of ForecastRequest, not
		// requests in their own right.
		if strings.HasSuffix(shortName, "Input") {
			continue
		}
		if strings.HasSuffix(shortName, "BatchUpdateEntry") {
			continue
		}
		schema := ref.Value
		if len(schema.Properties) == 0 {
			continue
		}
		// The Go struct behind this schema, if we can find it. Generic
		// instances (PaginatedResponse-UserResponse) and hand-declared shapes
		// have none; those keep the all-required treatment, which is correct
		// for them — their fields are built by the handler, not marshalled
		// from a tagged struct.
		fields := shapes[shortName]

		existing := make(map[string]struct{}, len(schema.Required))
		for _, r := range schema.Required {
			existing[r] = struct{}{}
		}
		for prop, propRef := range schema.Properties {
			shape := fields[prop]
			if shape.Omitted {
				// `omitempty` on something nilable: the field is genuinely
				// absent from the payload when empty, so claiming it is
				// required would be a lie the compiler enforces.
				continue
			}
			if shape.Nullable && propRef != nil && propRef.Value != nil {
				// A pointer that is always written: present, but possibly null.
				propRef.Value.Nullable = true
			}
			if _, ok := existing[prop]; !ok {
				schema.Required = append(schema.Required, prop)
				existing[prop] = struct{}{}
			}
		}
		// A field can be listed required by a `binding:"required"` tag and still
		// be omitempty on the response side; drop those so the two agree.
		schema.Required = slices.DeleteFunc(schema.Required, func(name string) bool {
			return fields[name].Omitted
		})
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

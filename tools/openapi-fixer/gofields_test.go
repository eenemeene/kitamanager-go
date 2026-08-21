package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// writeModels drops a throwaway models package in a temp dir and parses it.
func writeModels(t *testing.T, src string) map[string]map[string]fieldShape {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "models.go"), []byte(src), 0o600); err != nil {
		t.Fatalf("write models: %v", err)
	}
	shapes, err := goStructFields(dir)
	if err != nil {
		t.Fatalf("goStructFields: %v", err)
	}
	return shapes
}

func TestGoStructFields_ClassifiesEachTagAndType(t *testing.T) {
	shapes := writeModels(t, `package models

import "time"

type SampleResponse struct {
	// Always written, never null.
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`

	// Pointer without omitempty: always written, may be null.
	LastLogin *time.Time `+"`json:\"last_login\"`"+`

	// omitempty on nilable types: may be absent entirely.
	SectionName *string  `+"`json:\"section_name,omitempty\"`"+`
	Contracts   []string `+"`json:\"contracts,omitempty\"`"+`
	Params      map[string]string `+"`json:\"params,omitempty\"`"+`

	// omitempty on a non-nilable type: an absent 0 and a present 0 read the
	// same, so this stays required rather than becoming noise.
	Count int `+"`json:\"count,omitempty\"`"+`

	// Not part of the JSON contract at all.
	Internal string `+"`json:\"-\"`"+`
	Untagged string
}
`)

	fields, ok := shapes["SampleResponse"]
	if !ok {
		t.Fatalf("SampleResponse not found; got %v", shapes)
	}

	cases := []struct {
		field    string
		omitted  bool
		nullable bool
	}{
		{"id", false, false},
		{"name", false, false},
		{"last_login", false, true},
		{"section_name", true, false},
		{"contracts", true, false},
		{"params", true, false},
		{"count", false, false},
	}
	for _, tc := range cases {
		got, ok := fields[tc.field]
		if !ok {
			t.Errorf("%s: missing", tc.field)
			continue
		}
		if got.Omitted != tc.omitted || got.Nullable != tc.nullable {
			t.Errorf("%s: got omitted=%v nullable=%v, want omitted=%v nullable=%v",
				tc.field, got.Omitted, got.Nullable, tc.omitted, tc.nullable)
		}
	}
	if _, ok := fields["-"]; ok {
		t.Error(`json:"-" should not be recorded as a field`)
	}
	if _, ok := fields["Untagged"]; ok {
		t.Error("an untagged field should not be recorded")
	}
}

func TestGoStructFields_RejectsAnEmptyDirectory(t *testing.T) {
	// A silent empty map would restore the all-required behaviour without
	// anybody noticing, which is precisely the bug this machinery removes.
	if _, err := goStructFields(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory with no structs")
	}
}

// schemaWith builds a minimal object schema with the named string properties.
func schemaWith(props ...string) *openapi3.SchemaRef {
	schema := openapi3.NewObjectSchema()
	for _, p := range props {
		schema.Properties[p] = openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	}
	return openapi3.NewSchemaRef("", schema)
}

func specWith(schemas map[string]*openapi3.SchemaRef) *openapi3.T {
	return &openapi3.T{Components: &openapi3.Components{Schemas: schemas}}
}

func TestMarkResponseProperties_OmitsWhatGoMayOmit(t *testing.T) {
	spec := specWith(map[string]*openapi3.SchemaRef{
		"ChildResponse": schemaWith("id", "contracts", "school_entry_date"),
	})
	shapes := map[string]map[string]fieldShape{
		"ChildResponse": {
			"id":                {},
			"contracts":         {Omitted: true},
			"school_entry_date": {Omitted: true},
		},
	}

	markResponsePropertiesRequired(spec, shapes)

	required := spec.Components.Schemas["ChildResponse"].Value.Required
	if len(required) != 1 || required[0] != "id" {
		t.Fatalf("required = %v, want [id]", required)
	}
}

func TestMarkResponseProperties_MarksAlwaysWrittenPointersNullable(t *testing.T) {
	spec := specWith(map[string]*openapi3.SchemaRef{
		"ChildAttendanceResponse": schemaWith("id", "check_in_time"),
	})
	shapes := map[string]map[string]fieldShape{
		"ChildAttendanceResponse": {
			"id":            {},
			"check_in_time": {Nullable: true},
		},
	}

	markResponsePropertiesRequired(spec, shapes)

	schema := spec.Components.Schemas["ChildAttendanceResponse"].Value
	// Present in every payload, so still required -- but it may be null, and a
	// consumer that is not told so will happily call formatTime on nothing.
	if !slices.Contains(schema.Required, "check_in_time") {
		t.Errorf("check_in_time should stay required, got %v", schema.Required)
	}
	if !schema.Properties["check_in_time"].Value.Nullable {
		t.Error("check_in_time should be nullable")
	}
	if schema.Properties["id"].Value.Nullable {
		t.Error("id should not be nullable")
	}
}

func TestMarkResponseProperties_LeavesRequestSchemasAlone(t *testing.T) {
	// A request's `required` list comes from `binding:"required"` and is the
	// right truth for input; forcing every field required breaks partial updates.
	for _, name := range []string{"ChildUpdateRequest", "ForecastChildInput", "FooBatchUpdateEntry"} {
		spec := specWith(map[string]*openapi3.SchemaRef{name: schemaWith("first_name", "last_name")})
		markResponsePropertiesRequired(spec, nil)
		if got := spec.Components.Schemas[name].Value.Required; len(got) != 0 {
			t.Errorf("%s: required = %v, want none", name, got)
		}
	}
}

func TestMarkResponseProperties_KeepsAllRequiredForSchemasWithNoGoStruct(t *testing.T) {
	// Generic instances such as PaginatedResponse-UserResponse have no tagged
	// struct behind them; the handler builds them field by field.
	spec := specWith(map[string]*openapi3.SchemaRef{
		"PaginatedResponse-UserResponse": schemaWith("data", "total", "page"),
	})

	markResponsePropertiesRequired(spec, map[string]map[string]fieldShape{})

	if got := len(spec.Components.Schemas["PaginatedResponse-UserResponse"].Value.Required); got != 3 {
		t.Fatalf("required count = %d, want 3", got)
	}
}

func TestMarkResponseProperties_DropsAPreExistingRequiredThatGoMayOmit(t *testing.T) {
	// A field can carry `binding:"required"` and still be omitempty on the way
	// out. The two claims cannot both hold, and the response side wins here.
	schema := schemaWith("id", "section_name")
	schema.Value.Required = []string{"section_name"}
	spec := specWith(map[string]*openapi3.SchemaRef{"ChildContractResponse": schema})

	markResponsePropertiesRequired(spec, map[string]map[string]fieldShape{
		"ChildContractResponse": {"id": {}, "section_name": {Omitted: true}},
	})

	if got := spec.Components.Schemas["ChildContractResponse"].Value.Required; len(got) != 1 || got[0] != "id" {
		t.Fatalf("required = %v, want [id]", got)
	}
}

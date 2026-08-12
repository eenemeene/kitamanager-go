package models

// The optimistic-concurrency token has to reach the client to be usable: an
// If-Match precondition a caller cannot read is a precondition it cannot send.
// ToResponse maps fields by hand, so a new field on BaseContract does not
// propagate on its own — that is exactly what these tests pin.

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestChildContractToResponse_CarriesVersion(t *testing.T) {
	c := &ChildContract{ChildID: 7, BaseContract: BaseContract{Version: 3}}

	resp := c.ToResponse()
	if resp.Version != 3 {
		t.Errorf("Version = %d, want 3", resp.Version)
	}

	// json, not the struct field: the client reads the wire form, and an
	// `omitempty` here would hide version 0 rather than expose a stale token.
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(out, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if wire["version"] != float64(3) {
		t.Errorf("wire version = %v, want 3 (json: %s)", wire["version"], out)
	}
}

func TestEmployeeContractToResponse_CarriesVersion(t *testing.T) {
	c := &EmployeeContract{EmployeeID: 7, BaseContract: BaseContract{Version: 5}}

	resp := c.ToResponse()
	if resp.Version != 5 {
		t.Errorf("Version = %d, want 5", resp.Version)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(out, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if wire["version"] != float64(5) {
		t.Errorf("wire version = %v, want 5 (json: %s)", wire["version"], out)
	}
}

// The person YAML dumps describe a child's or employee's contracts, not the
// revision of the row storing them. A version key there would be noise the
// importer has to ignore, and would look meaningful to whoever edits the file.
func TestContractResponse_VersionExcludedFromYAML(t *testing.T) {
	for name, resp := range map[string]any{
		"child":    (&ChildContract{BaseContract: BaseContract{Version: 3}}).ToResponse(),
		"employee": (&EmployeeContract{BaseContract: BaseContract{Version: 3}}).ToResponse(),
	} {
		t.Run(name, func(t *testing.T) {
			out, err := yaml.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var wire map[string]any
			if err := yaml.Unmarshal(out, &wire); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if _, present := wire["version"]; present {
				t.Errorf("version leaked into the YAML dump:\n%s", out)
			}
		})
	}
}

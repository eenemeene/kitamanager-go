package models

import "encoding/json"

// Opt is a JSON field that can distinguish three states: absent from the
// request, present and null, and present with a value.
//
// Go's usual `*T` collapses the first two, which is the root of a whole class of
// contract bugs: a PATCH-style request that omits `to` looked identical to one
// asking to clear it, so omitting a field destroyed data. A timeline boundary
// drag that sent only the date it moved silently cleared `to` on the neighbouring
// contract, and the same flaw erased `properties` — care_type and every funding
// supplement — recomputing months of funding at the base rate.
//
// Use Opt for partial-update requests. Create/amend/end requests take plain
// required values and do not need it.
//
//	type CorrectRequest struct {
//	    To         Opt[time.Time]         `json:"to"`
//	    SectionID  Opt[uint]              `json:"section_id"`
//	    Properties Opt[ContractProperties] `json:"properties"`
//	}
//
//	if req.To.Set {          // the caller said something about `to`
//	    contract.To = req.To.Value  // nil here means an explicit null
//	}
//	                        // otherwise leave it alone
type Opt[T any] struct {
	// Set reports whether the field was present in the JSON at all.
	Set bool
	// Value is nil when the field was present but null, or when Set is false.
	Value *T
}

// UnmarshalJSON records that the key was present. encoding/json only calls this
// for keys that actually appear in the payload, which is precisely the signal
// a `*T` field cannot capture.
func (o *Opt[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// MarshalJSON emits the value, or null when the field was explicitly set to
// null. An *unset* Opt never reaches here: IsZero reports it as zero and the
// `omitzero` tag on every Opt field omits it entirely.
//
// That pairing is what makes the type round-trip. Without it, marshalling
// flattened absent and null back together — the very distinction Opt exists to
// carry — so a Go caller that built a request struct and serialized it sent
// `null` for every field it had not touched, and the service rejected it. Tests
// and any future Go client hit that; the frontend did not, because it hand-builds
// its JSON.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	if o.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.Value)
}

// IsZero reports an unset Opt as zero so encoding/json's `omitzero` (Go 1.24+)
// leaves the field out. An explicit null is not zero: it was chosen.
func (o Opt[T]) IsZero() bool {
	return !o.Set
}

// Get returns the value and whether a non-null value was supplied. Convenience
// for the common "apply only if a real value came in" case.
func (o Opt[T]) Get() (T, bool) {
	if o.Value == nil {
		var zero T
		return zero, false
	}
	return *o.Value, true
}

// IsNull reports an explicit JSON null — the caller asking to clear the field,
// as opposed to not mentioning it.
func (o Opt[T]) IsNull() bool {
	return o.Set && o.Value == nil
}

// OptOf builds a set Opt carrying v. For tests and internal callers.
func OptOf[T any](v T) Opt[T] {
	return Opt[T]{Set: true, Value: &v}
}

// OptNull builds an Opt representing an explicit null.
func OptNull[T any]() Opt[T] {
	return Opt[T]{Set: true}
}

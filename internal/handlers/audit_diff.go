package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// auditUpdateWithChanges logs a resource update audit event whose
// Details JSON carries a `changes` map listing the modified fields
// with their old and new values. The map is built by callers using
// recordChange / recordTimeChange so the helper stays type-safe.
//
// Wired up for child and employee personal data, and for contract
// updates (single and batch). For updates with no observable change the
// behaviour degrades to plain auditUpdate.
func auditUpdateWithChanges(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string, changes map[string]any) {
	if len(changes) == 0 {
		auditUpdate(c, svc, resourceType, id, name)
		return
	}
	svc.LogResourceUpdateWithChanges(c.Request.Context(), getUserID(c), getUserEmail(c),
		resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c), changes)
}

// auditDeleteWithSnapshot logs a resource deletion whose Details JSON carries a
// `snapshot` of the record's fields as they were. Degrades to plain auditDelete
// when the snapshot is empty.
func auditDeleteWithSnapshot(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string, snapshot map[string]any) {
	if len(snapshot) == 0 {
		auditDelete(c, svc, resourceType, id, name)
		return
	}
	svc.LogResourceDeleteWithSnapshot(c.Request.Context(), getUserID(c), getUserEmail(c),
		resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c), snapshot)
}

// recordChange adds a `{field: {old, new}}` entry to `m` when before
// and after differ. Generic on any comparable type — covers strings,
// ints, bools, and pointer values used in DTOs. Use recordTimeChange
// for time.Time values (which need instant-equality semantics).
func recordChange[T comparable](m map[string]any, field string, before, after T) {
	if before != after {
		m[field] = map[string]any{"old": before, "new": after}
	}
}

// recordTimeChange is the time.Time-specialised variant. time.Time
// comparison with == checks the wall/monotonic representation, not
// the represented instant; .Equal does the right thing.
func recordTimeChange(m map[string]any, field string, before, after time.Time) {
	if !before.Equal(after) {
		m[field] = map[string]any{
			"old": before.UTC().Format(time.RFC3339),
			"new": after.UTC().Format(time.RFC3339),
		}
	}
}

// recordNullableTimeChange is the *time.Time variant, for fields like a
// contract's `to` where nil means "ongoing". recordChange must not be used
// here: it would compare the pointers rather than the instants, so it would
// both miss real changes and report changes that did not happen.
func recordNullableTimeChange(m map[string]any, field string, before, after *time.Time) {
	switch {
	case before == nil && after == nil:
		return
	case before != nil && after != nil && before.Equal(*after):
		return
	}
	m[field] = map[string]any{
		"old": nullableTimeValue(before),
		"new": nullableTimeValue(after),
	}
}

// nullableTimeValue renders a *time.Time for the audit Details JSON: an RFC3339
// string, or nil so the field marshals as JSON null rather than a zero date.
func nullableTimeValue(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// timeValue renders a time.Time the same way nullableTimeValue renders its
// pointer form, so audit snapshots and diffs agree on the wire format.
func timeValue(t time.Time) any {
	return t.UTC().Format(time.RFC3339)
}

// recordPropertiesChange diffs two ContractProperties maps. recordChange cannot
// be used: ContractProperties is map[string]any, Go maps are not comparable, and
// a property value may itself be a slice (array-valued properties) — so
// reflect.DeepEqual is the correct test.
//
// Absent and empty count as the same state. A contract with no properties has
// not "changed" by being handed an empty map, and recording that as a diff would
// fill the log with noise on every dates-only edit.
//
// This is the field that matters most: care_type sets the base funding rate and
// the supplements (NdH, QM/MSS, Integration A/B) are worth tens to hundreds of
// euros per child per month, so a silent change here moves real money.
func recordPropertiesChange(m map[string]any, field string, before, after models.ContractProperties) {
	if len(before) == 0 && len(after) == 0 {
		return
	}
	if reflect.DeepEqual(before, after) {
		return
	}
	m[field] = map[string]any{"old": before, "new": after}
}

// --- Generic snapshot / diff -------------------------------------------------

// auditIgnoredFields are keys dropped from every snapshot and diff.
//
// created_at and updated_at are bookkeeping, not content: updated_at changes on
// literally every update, so leaving it in would put one guaranteed entry in
// every `changes` map and make "did anything actually change?" unanswerable by
// looking at the map's size.
//
// child_name is dropped for a different reason: it is a denormalized
// convenience field on ChildAttendanceResponse, and child_id already identifies
// the record for anyone entitled to resolve it. Copying a child's full name into
// every attendance audit row — and keeping it for the retention window, past the
// deletion of the child — is exactly the duplication the DSGVO minimisation rule
// exists to stop. Dropping it by name rather than per call site means a DTO that
// grows the field later cannot quietly start recording it.
//
// last_login belongs with the timestamps: it is written by the login path, not
// by whoever is editing the record, so an admin who edits an account while its
// owner happens to sign in would otherwise get a change attributed to them that
// they did not make.
var auditIgnoredFields = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"last_login": true,
	"child_name": true,
}

// auditSnapshot renders a response DTO as the flat map an audit row stores.
//
// Scalars only. Nested objects and arrays are dropped, which is what keeps the
// size of an audit row predictable: PayPlanDetailResponse carries every period
// and every period carries every entry, so a snapshot that followed them would
// copy an entire salary table into one Details column. It is also the right
// semantics — a period and an entry are separately audited resources with their
// own rows, and duplicating a child's fields into its parent's row would mean
// two records of the same change that can disagree.
//
// Numbers are decoded as json.Number rather than float64 so a value makes the
// round trip as the digits it was written with. Money is integer cents and this
// is an audit log; neither is a place to introduce a binary float.
func auditSnapshot(v any) map[string]any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		slog.Error("audit snapshot: marshal failed", "error", err)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		// Not an object — a DTO that marshals to an array or a scalar has
		// no fields to record, which is a caller error rather than a
		// runtime one, so say so loudly in the log and record nothing.
		slog.Error("audit snapshot: response is not a JSON object", "error", err)
		return nil
	}

	for k, val := range m {
		if auditIgnoredFields[k] {
			delete(m, k)
			continue
		}
		switch val.(type) {
		case map[string]any, []any:
			delete(m, k)
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// auditChangesOf diffs two response DTOs into the {field: {old, new}} shape the
// hand-written recordChange helpers produce, so a reader of the audit log sees
// one format regardless of which path wrote the row.
//
// Generic on T rather than taking two `any`s, and that is load-bearing: it makes
// the compiler guarantee both sides are the same type. Only then is "the key is
// present on one side and absent on the other" unambiguously an omitempty field
// going to or from its zero value, which is a real change worth recording. Given
// two different DTO types the same observation would usually mean nothing more
// than that the two types have different fields, and the diff would invent
// changes that never happened. Where a handler's read and write methods return
// different types, use recordChange instead — the compiler will not let this be
// called by mistake.
func auditChangesOf[T any](before, after *T) map[string]any {
	return auditChanges(auditSnapshot(before), auditSnapshot(after))
}

// auditChanges is the map-level half of auditChangesOf, split out so it can be
// tested directly against maps that no DTO produces.
func auditChanges(before, after map[string]any) map[string]any {
	changes := map[string]any{}
	for k, b := range before {
		a, present := after[k]
		if !present {
			// omitempty: the field went to its zero value and vanished.
			changes[k] = map[string]any{"old": b, "new": nil}
			continue
		}
		// DeepEqual rather than ==: auditSnapshot leaves only scalars, so ==
		// would do, but an audit path that panics takes the request down with
		// it and a future caller feeding this a composite value should get a
		// wrong-looking diff rather than a 500.
		if !reflect.DeepEqual(a, b) {
			changes[k] = map[string]any{"old": b, "new": a}
		}
	}
	for k, a := range after {
		if _, present := before[k]; !present {
			changes[k] = map[string]any{"old": nil, "new": a}
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return changes
}

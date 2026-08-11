package handlers

import (
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

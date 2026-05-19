package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/service"
)

// auditUpdateWithChanges logs a resource update audit event whose
// Details JSON carries a `changes` map listing the modified fields
// with their old and new values. The map is built by callers using
// recordChange / recordTimeChange so the helper stays type-safe.
//
// Closes review finding H2 for the high-value handlers (child,
// employee, contracts). For updates with no observable change the
// behaviour degrades to plain auditUpdate.
func auditUpdateWithChanges(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string, changes map[string]any) {
	if len(changes) == 0 {
		auditUpdate(c, svc, resourceType, id, name)
		return
	}
	svc.LogResourceUpdateWithChanges(c.Request.Context(), getUserID(c), getUserEmail(c),
		resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c), changes)
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

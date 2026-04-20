package testutil

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// auditPollDeadline bounds how long AssertAuditLog will wait for an
// async-written audit row to land. AuditService persists logs from a
// background goroutine, so a test that asserts immediately after a handler
// returns may otherwise race the writer.
const auditPollDeadline = 2 * time.Second
const auditPollInterval = 10 * time.Millisecond

// AuditLogQuery selects audit log rows for assertions. Only non-zero fields
// participate in the WHERE clause; e.g. an empty Action matches any action.
// ResourceID == 0 means "do not constrain on resource_id".
type AuditLogQuery struct {
	Action       models.AuditAction
	ResourceType string
	ResourceID   uint
	ActorUserID  uint
}

// AssertAuditLog asserts that exactly one audit log row matching q exists.
// It returns the matched row so callers can make further assertions on
// fields the helper does not constrain (Details, IPAddress, Timestamp).
//
// Use this in mutating-handler tests to verify the audit trail was written.
// Without it, a handler that silently stops calling auditService.Log* would
// pass every existing functional test — and then fail an external compliance
// audit instead of CI, which is the wrong place to learn about the bug.
func AssertAuditLog(t *testing.T, db *gorm.DB, q AuditLogQuery) *models.AuditLog {
	t.Helper()

	deadline := time.Now().Add(auditPollDeadline)
	var rows []models.AuditLog
	for {
		rows = queryAuditLogs(t, db, q)
		if len(rows) >= 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(auditPollInterval)
	}

	switch len(rows) {
	case 1:
		return &rows[0]
	case 0:
		t.Fatalf("AssertAuditLog: no rows matched %+v within %s. Existing audit_logs:\n%s",
			q, auditPollDeadline, dumpAuditLogs(db))
	default:
		t.Fatalf("AssertAuditLog: expected exactly 1 row matching %+v, found %d", q, len(rows))
	}
	return nil
}

// AssertNoAuditLog asserts that no audit log row matches q. Useful for
// negative tests — e.g. a failed authorization must NOT produce a successful
// resource-mutation audit row.
func AssertNoAuditLog(t *testing.T, db *gorm.DB, q AuditLogQuery) {
	t.Helper()

	// Audit logging is asynchronous, so even a "no log" assertion needs a
	// short settle before reading — otherwise a write that the handler
	// queued just before returning could land *after* the assertion and
	// the negative test would silently pass while production is wrong.
	time.Sleep(100 * time.Millisecond)

	rows := queryAuditLogs(t, db, q)
	if len(rows) != 0 {
		t.Fatalf("AssertNoAuditLog: expected 0 rows matching %+v, found %d", q, len(rows))
	}
}

func queryAuditLogs(t *testing.T, db *gorm.DB, q AuditLogQuery) []models.AuditLog {
	t.Helper()
	tx := db.Model(&models.AuditLog{})
	if q.Action != "" {
		tx = tx.Where("action = ?", q.Action)
	}
	if q.ResourceType != "" {
		tx = tx.Where("resource_type = ?", q.ResourceType)
	}
	if q.ResourceID != 0 {
		tx = tx.Where("resource_id = ?", q.ResourceID)
	}
	if q.ActorUserID != 0 {
		tx = tx.Where("user_id = ?", q.ActorUserID)
	}
	var rows []models.AuditLog
	if err := tx.Find(&rows).Error; err != nil {
		t.Fatalf("audit log query failed: %v", err)
	}
	return rows
}

func dumpAuditLogs(db *gorm.DB) string {
	var rows []models.AuditLog
	if err := db.Find(&rows).Error; err != nil {
		return "(failed to dump: " + err.Error() + ")"
	}
	if len(rows) == 0 {
		return "  (table is empty)"
	}
	var b strings.Builder
	for _, r := range rows {
		var actor, rid uint
		if r.UserID != nil {
			actor = *r.UserID
		}
		if r.ResourceID != nil {
			rid = *r.ResourceID
		}
		b.WriteString("  - action=")
		b.WriteString(string(r.Action))
		b.WriteString(" resource_type=")
		b.WriteString(r.ResourceType)
		b.WriteString(" resource_id=")
		b.WriteString(strconv.FormatUint(uint64(rid), 10))
		b.WriteString(" actor=")
		b.WriteString(strconv.FormatUint(uint64(actor), 10))
		b.WriteString("\n")
	}
	return b.String()
}

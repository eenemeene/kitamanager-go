package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// The org-scoped audit feed must not hand an org admin a colleague's exact IP.
//
// These go through the real authorization middleware rather than setting the
// context key by hand, because the defect being fixed was precisely that the
// key the middleware populates had no reader — a test that sets it itself would
// prove nothing about the wiring.

// orgAuditRouter wires GET /organizations/:orgId/audit-logs the way routes.go
// does, through RequirePermission.
func orgAuditRouter(t *testing.T, db *gorm.DB, userID uint) *gin.Engine {
	t.Helper()

	authzMw := setupAuthzMiddleware(t, db)
	handler := NewAuditLogHandler(createAuditService(db))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, userID)
		c.Next()
	})
	r.GET("/api/v1/organizations/:orgId/audit-logs",
		authzMw.RequirePermission(rbac.ResourceAuditLog, rbac.ActionRead),
		handler.ListByOrganization)
	return r
}

// readOrgAuditRows performs the request and decodes the page of rows.
func readOrgAuditRows(t *testing.T, r *gin.Engine, orgID uint) []models.AuditLogResponse {
	t.Helper()

	req, _ := http.NewRequest("GET", "/api/v1/organizations/"+strconv.FormatUint(uint64(orgID), 10)+"/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page models.PaginatedResponse[models.AuditLogResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return page.Data
}

func seedOrgAuditRow(t *testing.T, db *gorm.DB, orgID uint, ip string) {
	t.Helper()
	svc := createAuditService(db)
	svc.LogResourceCreate(t.Context(), 1, "actor@example.com", "child", 42, "Anna", ip, &orgID)
	svc.Shutdown()
}

func TestAuditLogHandler_OrgAdminGetsAnonymizedIP(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestUser(t, db, "Org Admin", "orgadmin-ip@test.com", "password123")
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)
	seedOrgAuditRow(t, db, org.ID, "203.0.113.147")

	rows := readOrgAuditRows(t, orgAuditRouter(t, db, admin.ID), org.ID)
	if len(rows) != 1 {
		t.Fatalf("expected 1 audit row, got %d", len(rows))
	}
	if rows[0].IPAddress != "203.0.113.0" {
		t.Errorf("IPAddress = %q, want the /24 prefix — an org admin must not read a colleague's exact address",
			rows[0].IPAddress)
	}
	if !rows[0].IPAnonymized {
		t.Error("ip_anonymized must be set so the client can tell a prefix from a real address")
	}
}

func TestAuditLogHandler_SuperAdminGetsRecordedIP(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestUser(t, db, "Super Admin", "superadmin-ip@test.com", "password123")
	if err := db.Model(&models.User{}).Where("id = ?", admin.ID).Update("is_superadmin", true).Error; err != nil {
		t.Fatalf("failed to promote to superadmin: %v", err)
	}
	org := createTestOrganization(t, db, "Kita Sonnenschein")
	seedOrgAuditRow(t, db, org.ID, "203.0.113.147")

	rows := readOrgAuditRows(t, orgAuditRouter(t, db, admin.ID), org.ID)
	if len(rows) != 1 {
		t.Fatalf("expected 1 audit row, got %d", len(rows))
	}
	if rows[0].IPAddress != "203.0.113.147" {
		t.Errorf("IPAddress = %q, want the address as recorded", rows[0].IPAddress)
	}
	if rows[0].IPAnonymized {
		t.Error("ip_anonymized must be absent when nothing was reduced")
	}
}

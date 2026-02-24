package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

var (
	auditFallbackTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "audit_entries_fallback_total",
		Help: "Total number of audit entries written via synchronous fallback",
	})
	auditDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "audit_entries_dropped_total",
		Help: "Total number of audit entries dropped (both async and fallback failed)",
	})
)

// auditBufferSize is the capacity of the asynchronous audit log channel.
const auditBufferSize = 4096

// AuditService handles audit logging operations
type AuditService struct {
	store         store.AuditStorer
	logCh         chan *models.AuditLog
	done          chan struct{}
	fallbackCount atomic.Int64
	droppedCount  atomic.Int64
}

// NewAuditService creates a new AuditService
func NewAuditService(store store.AuditStorer) *AuditService {
	s := &AuditService{
		store: store,
		logCh: make(chan *models.AuditLog, auditBufferSize),
		done:  make(chan struct{}),
	}
	go s.processLogs()
	return s
}

// processLogs drains the audit log channel and persists entries
func (s *AuditService) processLogs() {
	defer close(s.done)
	for entry := range s.logCh {
		if err := s.store.Create(context.Background(), entry); err != nil {
			slog.Error("Failed to create audit log", "action", entry.Action, "error", err)
		}
	}
}

// Shutdown closes the log channel and waits for the worker to drain
func (s *AuditService) Shutdown() {
	if s == nil || s.logCh == nil {
		return
	}
	close(s.logCh)
	<-s.done
}

// DroppedCount returns the number of audit entries that were dropped.
func (s *AuditService) DroppedCount() int64 {
	if s == nil {
		return 0
	}
	return s.droppedCount.Load()
}

// FallbackCount returns the number of audit entries written via synchronous fallback.
func (s *AuditService) FallbackCount() int64 {
	if s == nil {
		return 0
	}
	return s.fallbackCount.Load()
}

// mustMarshalJSON marshals v to JSON, returning "{}" on error.
func mustMarshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
		return "{}"
	}
	return string(data)
}

// LogLogin logs a successful login attempt
func (s *AuditService) LogLogin(userID uint, email, ipAddress, userAgent string) {
	s.log(&models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionLogin,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	})
}

// LogLoginFailed logs a failed login attempt
func (s *AuditService) LogLoginFailed(email, ipAddress, userAgent, reason string) {
	s.log(&models.AuditLog{
		UserEmail: email,
		Action:    models.AuditActionLoginFailed,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   mustMarshalJSON(map[string]string{"reason": reason}),
		Success:   false,
	})
}

// LogSuperAdminChange logs a superadmin status change
func (s *AuditService) LogSuperAdminChange(actorID, targetUserID uint, targetEmail string, granted bool, ipAddress string) {
	action := models.AuditActionSuperAdminGrant
	if !granted {
		action = models.AuditActionSuperAdminRevoke
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       action,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"target_user_id":    targetUserID,
			"target_user_email": targetEmail,
			"granted":           granted,
		}),
		Success:      true,
	})
}

// LogUserAddToOrg logs adding a user to an organization
func (s *AuditService) LogUserAddToOrg(actorID, userID, orgID uint, role string, ipAddress string) {
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserAddToOrg,
		ResourceType: "user_organization",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"organization_id": orgID,
			"role":            role,
		}),
		Success: true,
	})
}

// LogUserRemoveFromOrg logs removing a user from an organization
func (s *AuditService) LogUserRemoveFromOrg(actorID, userID, orgID uint, ipAddress string) {
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserRemoveFromOrg,
		ResourceType: "user_organization",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]any{"organization_id": orgID}),
		Success:      true,
	})
}

// LogRoleChange logs a role change for a user in an organization
func (s *AuditService) LogRoleChange(actorID, userID, orgID uint, oldRole, newRole string, ipAddress string) {
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionRoleChange,
		ResourceType: "user_organization",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"organization_id": orgID,
			"old_role":        oldRole,
			"new_role":        newRole,
		}),
		Success: true,
	})
}

// LogResourceDelete logs deletion of a resource (employee, child, org, etc.)
func (s *AuditService) LogResourceDelete(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	var action models.AuditAction
	switch resourceType {
	case "employee":
		action = models.AuditActionEmployeeDelete
	case "child":
		action = models.AuditActionChildDelete
	case "organization":
		action = models.AuditActionOrgDelete
	case "user":
		action = models.AuditActionUserDelete
	default:
		action = models.AuditAction(resourceType + "_delete")
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:      true,
	})
}

// LogResourceCreate logs creation of a resource
func (s *AuditService) LogResourceCreate(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	var action models.AuditAction
	switch resourceType {
	case "user":
		action = models.AuditActionUserCreate
	case "organization":
		action = models.AuditActionOrgCreate
	default:
		action = models.AuditAction(resourceType + "_create")
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:      true,
	})
}

// LogResourceUpdate logs update of a resource
func (s *AuditService) LogResourceUpdate(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditAction(resourceType + "_update"),
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:      true,
	})
}

// LogDataExport logs a bulk data export event
func (s *AuditService) LogDataExport(actorID uint, resourceType string, orgID uint, recordCount int, ipAddress string) {
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditAction(resourceType + "_export"),
		ResourceType: resourceType,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"organization_id": orgID,
			"record_count":    recordCount,
		}),
		Success: true,
	})
}

// GetLogs returns paginated audit logs
func (s *AuditService) GetLogs(ctx context.Context, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs")
	}

	return toResponseList(logs, (*models.AuditLog).ToResponse), total, nil
}

// GetLogsByUser returns audit logs for a specific user
func (s *AuditService) GetLogsByUser(ctx context.Context, userID uint, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs for user")
	}

	return toResponseList(logs, (*models.AuditLog).ToResponse), total, nil
}

// CountRecentFailedLogins counts failed login attempts for an email in the last duration
func (s *AuditService) CountRecentFailedLogins(ctx context.Context, email string, duration time.Duration) (int64, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}

	since := time.Now().UTC().Add(-duration)
	return s.store.CountFailedLoginsSince(ctx, email, since)
}

// log sends an audit log entry to the worker channel.
// If the channel is full, falls back to synchronous write with a timeout.
func (s *AuditService) log(entry *models.AuditLog) {
	if s == nil || s.logCh == nil {
		return
	}

	entry.Timestamp = time.Now().UTC()

	select {
	case s.logCh <- entry:
	default:
		s.fallbackCount.Add(1)
		auditFallbackTotal.Inc()
		// Synchronous fallback with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.Create(ctx, entry); err != nil {
			s.droppedCount.Add(1)
			auditDroppedTotal.Inc()
			slog.Error("Audit log dropped", "action", entry.Action, "error", err)
		}
	}
}

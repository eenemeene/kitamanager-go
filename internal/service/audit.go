package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// AuditService handles audit logging operations
type AuditService struct {
	store store.AuditStorer
	logCh chan *models.AuditLog
	done  chan struct{}
}

// NewAuditService creates a new AuditService
func NewAuditService(store store.AuditStorer) *AuditService {
	s := &AuditService{
		store: store,
		logCh: make(chan *models.AuditLog, 256),
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
	details, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}
	s.log(&models.AuditLog{
		UserEmail: email,
		Action:    models.AuditActionLoginFailed,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   string(details),
		Success:   false,
	})
}

// LogSuperAdminChange logs a superadmin status change
func (s *AuditService) LogSuperAdminChange(actorID, targetUserID uint, targetEmail string, granted bool, ipAddress string) {
	action := models.AuditActionSuperAdminGrant
	if !granted {
		action = models.AuditActionSuperAdminRevoke
	}

	details, err := json.Marshal(map[string]interface{}{
		"target_user_id":    targetUserID,
		"target_user_email": targetEmail,
		"granted":           granted,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       action,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogUserCreate logs a user creation
func (s *AuditService) LogUserCreate(actorID, newUserID uint, newUserEmail, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"new_user_email": newUserEmail,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserCreate,
		ResourceType: "user",
		ResourceID:   &newUserID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogUserDelete logs a user deletion
func (s *AuditService) LogUserDelete(actorID, deletedUserID uint, deletedUserEmail, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"deleted_user_email": deletedUserEmail,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserDelete,
		ResourceType: "user",
		ResourceID:   &deletedUserID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogUserAddToGroup logs adding a user to a group
func (s *AuditService) LogUserAddToGroup(actorID, userID, groupID uint, role string, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"group_id": groupID,
		"role":     role,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserAddToGroup,
		ResourceType: "user_group",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogUserRemoveFromGroup logs removing a user from a group
func (s *AuditService) LogUserRemoveFromGroup(actorID, userID, groupID uint, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"group_id": groupID,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionUserRemoveFromGroup,
		ResourceType: "user_group",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogRoleChange logs a role change for a user in a group
func (s *AuditService) LogRoleChange(actorID, userID, groupID uint, oldRole, newRole string, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"group_id": groupID,
		"old_role": oldRole,
		"new_role": newRole,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionRoleChange,
		ResourceType: "user_group",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogResourceDelete logs deletion of a resource (employee, child, org, etc.)
func (s *AuditService) LogResourceDelete(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"resource_name": resourceName,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	var action models.AuditAction
	switch resourceType {
	case "employee":
		action = models.AuditActionEmployeeDelete
	case "child":
		action = models.AuditActionChildDelete
	case "organization":
		action = models.AuditActionOrgDelete
	default:
		action = models.AuditAction(resourceType + "_delete")
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogOrgCreate logs organization creation
func (s *AuditService) LogOrgCreate(actorID, orgID uint, orgName, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"org_name": orgName,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}

	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditActionOrgCreate,
		ResourceType: "organization",
		ResourceID:   &orgID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogResourceCreate logs creation of a resource
func (s *AuditService) LogResourceCreate(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"resource_name": resourceName,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditAction(resourceType + "_create"),
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// LogResourceUpdate logs update of a resource
func (s *AuditService) LogResourceUpdate(actorID uint, resourceType string, resourceID uint, resourceName, ipAddress string) {
	details, err := json.Marshal(map[string]interface{}{
		"resource_name": resourceName,
	})
	if err != nil {
		slog.Error("Failed to marshal audit details", "error", err)
	}
	s.log(&models.AuditLog{
		UserID:       &actorID,
		Action:       models.AuditAction(resourceType + "_update"),
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      string(details),
		Success:      true,
	})
}

// GetLogs returns paginated audit logs
func (s *AuditService) GetLogs(ctx context.Context, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = log.ToResponse()
	}

	return responses, total, nil
}

// GetLogsByUser returns audit logs for a specific user
func (s *AuditService) GetLogsByUser(ctx context.Context, userID uint, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = log.ToResponse()
	}

	return responses, total, nil
}

// CountRecentFailedLogins counts failed login attempts for an email in the last duration
func (s *AuditService) CountRecentFailedLogins(ctx context.Context, email string, duration time.Duration) (int64, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}

	since := time.Now().Add(-duration)
	return s.store.CountFailedLoginsSince(ctx, email, since)
}

// log sends an audit log entry to the worker channel
func (s *AuditService) log(entry *models.AuditLog) {
	if s == nil || s.logCh == nil {
		return
	}

	entry.Timestamp = time.Now()

	select {
	case s.logCh <- entry:
	default:
		slog.Warn("Audit log channel full, dropping entry", "action", entry.Action)
	}
}

package service

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// AuditService handles audit logging operations
type AuditService struct {
	store *store.AuditStore
}

// NewAuditService creates a new AuditService
func NewAuditService(store *store.AuditStore) *AuditService {
	return &AuditService{store: store}
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
	details, _ := json.Marshal(map[string]string{"reason": reason})
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

	details, _ := json.Marshal(map[string]interface{}{
		"target_user_id":    targetUserID,
		"target_user_email": targetEmail,
		"granted":           granted,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"new_user_email": newUserEmail,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"deleted_user_email": deletedUserEmail,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"group_id": groupID,
		"role":     role,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"group_id": groupID,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"group_id": groupID,
		"old_role": oldRole,
		"new_role": newRole,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"resource_name": resourceName,
	})

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
	details, _ := json.Marshal(map[string]interface{}{
		"org_name": orgName,
	})

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

// GetLogs returns paginated audit logs
func (s *AuditService) GetLogs(limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindAll(limit, offset)
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
func (s *AuditService) GetLogsByUser(userID uint, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindByUser(userID, limit, offset)
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
func (s *AuditService) CountRecentFailedLogins(email string, duration time.Duration) (int64, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}

	since := time.Now().Add(-duration)
	return s.store.CountFailedLoginsSince(email, since)
}

// log creates an audit log entry asynchronously to not block the main request
func (s *AuditService) log(entry *models.AuditLog) {
	// Handle nil receiver gracefully (e.g., in tests)
	if s == nil || s.store == nil {
		return
	}

	entry.Timestamp = time.Now()

	// Log asynchronously to not block the request
	go func() {
		if err := s.store.Create(entry); err != nil {
			slog.Error("Failed to create audit log",
				"action", entry.Action,
				"error", err)
		}
	}()
}

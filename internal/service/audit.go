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
	"github.com/eenemeene/kitamanager-go/internal/middleware"
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.store.Create(ctx, entry); err != nil {
			slog.Error("Failed to create audit log", "action", entry.Action, "error", err)
		}
		cancel()
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
func (s *AuditService) LogLogin(ctx context.Context, userID uint, email, ipAddress, userAgent string) {
	s.log(ctx, &models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionLogin,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	})
}

// LogLogout records a user destroying their current session via the
// /logout endpoint. Best-effort: if the caller did not know the user
// id (e.g. session already expired server-side) the event is still
// emitted with UserID=nil so investigators can correlate by email +
// ip. Idempotent logout attempts against an already-gone session
// must NOT be logged — the handler gates this call on success so we
// don't spam noise rows after a double-click.
func (s *AuditService) LogLogout(ctx context.Context, userID *uint, email, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:    userID,
		UserEmail: email,
		Action:    models.AuditActionLogout,
		IPAddress: ipAddress,
		Success:   true,
	})
}

// LogSessionRevoked records a user deleting one of their OTHER
// sessions from the Active Sessions UI — explicit, intentional, and
// a strong security signal (commonly fired after the user notices a
// device they don't recognise). sessionIDHash identifies the row
// that was killed; we store it in Details rather than ResourceID
// because AuditLog.ResourceID is a uint and the session key is a
// hash string.
func (s *AuditService) LogSessionRevoked(ctx context.Context, userID uint, email, sessionIDHash, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		UserEmail:    email,
		Action:       models.AuditActionSessionRevoked,
		ResourceType: "session",
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]string{"session_id_hash": sessionIDHash}),
		Success:      true,
	})
}

// LogLoginFailed logs a failed login attempt
func (s *AuditService) LogLoginFailed(ctx context.Context, email, ipAddress, userAgent, reason string) {
	s.log(ctx, &models.AuditLog{
		UserEmail: email,
		Action:    models.AuditActionLoginFailed,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   mustMarshalJSON(map[string]string{"reason": reason}),
		Success:   false,
	})
}

// LogPasswordChange logs a successful self-service password rotation.
func (s *AuditService) LogPasswordChange(ctx context.Context, userID uint, email, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		UserEmail:    email,
		Action:       models.AuditActionPasswordChange,
		ResourceType: "user",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Success:      true,
	})
}

// LogPasswordChangeFailed logs a failed /me/password attempt. Used by the
// lockout counter so an attacker with a stolen access token cannot brute-force
// the current password at full API-mutation-rate-limit speed.
func (s *AuditService) LogPasswordChangeFailed(ctx context.Context, userID uint, email, ipAddress, reason string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		UserEmail:    email,
		Action:       models.AuditActionPasswordChangeFailed,
		ResourceType: "user",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]string{"reason": reason}),
		Success:      false,
	})
}

// LogFactorEnrolled logs completion of MFA factor enrollment.
// `factorType` is the factor-generic type string ("totp", etc.) so
// audit queries can pivot on it.
func (s *AuditService) LogFactorEnrolled(ctx context.Context, userID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorEnrolled,
		ResourceType: "factor",
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      true,
	})
}

// LogFactorDeleted logs a user removing their OWN MFA factor.
func (s *AuditService) LogFactorDeleted(ctx context.Context, userID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorDeleted,
		ResourceType: "factor",
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      true,
	})
}

// LogFactorActivationLocked logs a pending factor being auto-deleted
// because activation failures hit the limit. This is a security
// signal: the common cause is an attacker in a hijacked session trying
// codes against a freshly-enrolled pending row.
func (s *AuditService) LogFactorActivationLocked(ctx context.Context, userID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorActivationLocked,
		ResourceType: "factor",
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      false,
	})
}

// LogBackupCodesRegenerated logs a user regenerating their backup
// codes. A spike in these signals a user having trouble with their
// primary factor.
func (s *AuditService) LogBackupCodesRegenerated(ctx context.Context, userID uint) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionBackupCodesRegenerated,
		ResourceType: "factor",
		Success:      true,
	})
}

// LogLoginMFARequired logs the password-accepted-but-MFA-required
// branch of /login. Success=true because the password itself did
// verify — the session just isn't complete yet.
func (s *AuditService) LogLoginMFARequired(ctx context.Context, userID uint, email, ipAddress, userAgent string) {
	s.log(ctx, &models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionLoginMFARequired,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	})
}

// LogMFAChallengeSucceeded logs /auth/mfa/verify accepting a code.
// The `factorType` goes in details so dashboards can pivot on TOTP
// vs backup_codes usage.
func (s *AuditService) LogMFAChallengeSucceeded(ctx context.Context, userID uint, email, factorType, ipAddress, userAgent string) {
	s.log(ctx, &models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionMFAChallengeSucceeded,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:   true,
	})
}

// LogMFAChallengeFailed logs a wrong code on /auth/mfa/verify. This
// is the audit record the per-user rate-limit counter queries.
// factorType is the user-claimed factor (from the request) so a
// distributed attack that iterates through factor IDs is visible.
func (s *AuditService) LogMFAChallengeFailed(ctx context.Context, userID uint, email, factorType, ipAddress, userAgent, reason string) {
	s.log(ctx, &models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionMFAChallengeFailed,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details: mustMarshalJSON(map[string]string{
			"factor_type": factorType,
			"reason":      reason,
		}),
		Success: false,
	})
}

// LogMFAChallengeLocked logs a pending_mfa row being destroyed after
// MFAChallengeFailureLimit wrong codes. Security signal — the user
// will have to restart from the password step.
func (s *AuditService) LogMFAChallengeLocked(ctx context.Context, userID uint, email, ipAddress, userAgent string) {
	s.log(ctx, &models.AuditLog{
		UserID:    &userID,
		UserEmail: email,
		Action:    models.AuditActionMFAChallengeLocked,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   false,
	})
}

// CountRecentFailedMFAChallenges counts `mfa_challenge_failed` audit
// rows for the user in the given window. Used by the verify handler
// to enforce a per-user cap independent of per-pending-row failures —
// an attacker cycling through many pending rows would otherwise blow
// past the per-row limit.
func (s *AuditService) CountRecentFailedMFAChallenges(ctx context.Context, userID uint, window time.Duration) (int64, error) {
	return s.store.CountFailedMFAChallengesSince(ctx, userID, time.Now().UTC().Add(-window))
}

// LogSuperAdminChange logs a superadmin status change
func (s *AuditService) LogSuperAdminChange(ctx context.Context, actorID uint, actorEmail string, targetUserID uint, targetEmail string, granted bool, ipAddress string) {
	action := models.AuditActionSuperAdminGrant
	if !granted {
		action = models.AuditActionSuperAdminRevoke
	}

	s.log(ctx, &models.AuditLog{
		UserID:       &actorID,
		UserEmail:    actorEmail,
		Action:       action,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"target_user_id":    targetUserID,
			"target_user_email": targetEmail,
			"granted":           granted,
		}),
		Success: true,
	})
}

// LogUserAddToOrg logs adding a user to an organization
func (s *AuditService) LogUserAddToOrg(ctx context.Context, actorID uint, actorEmail string, userID, orgID uint, role string, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditActionUserAddToOrg,
		ResourceType:   "user_organization",
		ResourceID:     &userID,
		OrganizationID: &orgID,
		IPAddress:      ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"organization_id": orgID,
			"role":            role,
		}),
		Success: true,
	})
}

// LogUserRemoveFromOrg logs removing a user from an organization
func (s *AuditService) LogUserRemoveFromOrg(ctx context.Context, actorID uint, actorEmail string, userID, orgID uint, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditActionUserRemoveFromOrg,
		ResourceType:   "user_organization",
		ResourceID:     &userID,
		OrganizationID: &orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(map[string]any{"organization_id": orgID}),
		Success:        true,
	})
}

// LogRoleChange logs a role change for a user in an organization
func (s *AuditService) LogRoleChange(ctx context.Context, actorID uint, actorEmail string, userID, orgID uint, oldRole, newRole string, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditActionRoleChange,
		ResourceType:   "user_organization",
		ResourceID:     &userID,
		OrganizationID: &orgID,
		IPAddress:      ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"organization_id": orgID,
			"old_role":        oldRole,
			"new_role":        newRole,
		}),
		Success: true,
	})
}

// LogResourceDelete logs deletion of a resource (employee, child, org, etc.)
// orgID may be nil for identity-level resources (user); pass the owning org
// id for org-scoped resources so org admins can see the event.
func (s *AuditService) LogResourceDelete(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint) {
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

	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		OrganizationID: orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:        true,
	})
}

// LogResourceCreate logs creation of a resource.
// orgID may be nil for identity-level resources (user); pass the owning org
// id for org-scoped resources so org admins can see the event.
func (s *AuditService) LogResourceCreate(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint) {
	var action models.AuditAction
	switch resourceType {
	case "user":
		action = models.AuditActionUserCreate
	case "organization":
		action = models.AuditActionOrgCreate
	default:
		action = models.AuditAction(resourceType + "_create")
	}

	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		OrganizationID: orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:        true,
	})
}

// LogResourceUpdate logs update of a resource.
// orgID may be nil for identity-level resources (user); pass the owning org
// id for org-scoped resources so org admins can see the event.
func (s *AuditService) LogResourceUpdate(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint) {
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditAction(resourceType + "_update"),
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		OrganizationID: orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(map[string]any{"resource_name": resourceName}),
		Success:        true,
	})
}

// LogPasswordReset logs when an admin resets another user's password.
func (s *AuditService) LogPasswordReset(ctx context.Context, actorID uint, actorEmail string, targetUserID uint, targetEmail, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &actorID,
		UserEmail:    actorEmail,
		Action:       models.AuditActionPasswordReset,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"target_user_id":    targetUserID,
			"target_user_email": targetEmail,
		}),
		Success: true,
	})
}

// LogDataExport logs a bulk data export event
func (s *AuditService) LogDataExport(ctx context.Context, actorID uint, actorEmail, resourceType string, orgID uint, recordCount int, ipAddress string) {
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditAction(resourceType + "_export"),
		ResourceType:   resourceType,
		OrganizationID: &orgID,
		IPAddress:      ipAddress,
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

// GetLogsFiltered returns paginated audit logs with optional filters
func (s *AuditService) GetLogsFiltered(ctx context.Context, action string, userID *uint, from *time.Time, to *time.Time, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindAllFiltered(ctx, action, userID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs")
	}

	return toResponseList(logs, (*models.AuditLog).ToResponse), total, nil
}

// GetLogsByOrganization returns audit logs scoped to a single organization
// with optional filters. Identity-level events (org_id IS NULL) are excluded
// — only the superadmin-only GetLogsFiltered path sees those.
func (s *AuditService) GetLogsByOrganization(ctx context.Context, orgID uint, action string, userID *uint, from, to *time.Time, limit, offset int) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindByOrganization(ctx, orgID, action, userID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs")
	}

	return toResponseList(logs, (*models.AuditLog).ToResponse), total, nil
}

// GetLogByID returns a single audit log entry by ID
func (s *AuditService) GetLogByID(ctx context.Context, id uint) (*models.AuditLogResponse, error) {
	if s == nil || s.store == nil {
		return nil, apperror.NotFound("audit log")
	}

	log, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "audit log")
	}

	resp := log.ToResponse()
	return &resp, nil
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

// CountRecentFailedPasswordChanges counts /me/password failures for a user in
// the last duration. Used for the lockout check before bcrypt so an attacker
// holding a stolen access token cannot brute-force the current password.
func (s *AuditService) CountRecentFailedPasswordChanges(ctx context.Context, userID uint, duration time.Duration) (int64, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}

	since := time.Now().UTC().Add(-duration)
	return s.store.CountFailedPasswordChangesSince(ctx, userID, since)
}

// log sends an audit log entry to the worker channel, stamping the
// request id from ctx so every row emitted inside one HTTP request
// carries the same X-Request-ID correlation key. ctx may be nil or
// context.Background() for non-HTTP callers; RequestID then stays
// empty, which is the correct semantic for events that didn't
// originate from an HTTP request (seed imports, CLI, background
// jobs).
//
// If the caller pre-populated entry.RequestID (rare — e.g. a test
// injecting a synthetic id), that wins; ctx is not consulted.
//
// If the channel is full, falls back to a synchronous write with a
// short timeout using a fresh background context so that a client
// disconnect mid-request cannot cancel the write and drop the audit
// row.
func (s *AuditService) log(ctx context.Context, entry *models.AuditLog) {
	if s == nil || s.logCh == nil {
		return
	}

	entry.Timestamp = time.Now().UTC()
	if entry.RequestID == "" {
		entry.RequestID = middleware.RequestIDFromContext(ctx)
	}

	select {
	case s.logCh <- entry:
	default:
		s.fallbackCount.Add(1)
		auditFallbackTotal.Inc()
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.Create(writeCtx, entry); err != nil {
			s.droppedCount.Add(1)
			auditDroppedTotal.Inc()
			slog.Error("Audit log dropped", "action", entry.Action, "error", err)
		}
	}
}

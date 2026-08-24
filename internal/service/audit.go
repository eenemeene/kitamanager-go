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

// LogAuditLogPurged records the periodic retention sweep itself —
// "deleted N audit rows older than <cutoff>". UserID is nil
// because the actor is the system retention job, not a user. The
// row is itself subject to the same retention but the most recent
// one always exists, ratifying the deletion pattern for any
// investigator looking at the table.
func (s *AuditService) LogAuditLogPurged(ctx context.Context, deleted int64, olderThan time.Time) {
	s.log(ctx, &models.AuditLog{
		Action:       models.AuditActionAuditLogPurged,
		ResourceType: "audit_log",
		Details: mustMarshalJSON(map[string]any{
			"deleted_rows": deleted,
			"older_than":   olderThan.Format(time.RFC3339),
		}),
		Success: true,
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

// LogPasswordResetFailed logs a failed /users/:userId/password attempt:
// the actor's actor_password did not match. UserID on the row is the
// actor (the brute-force victim); ResourceID carries the target user
// so investigators can see who the actor was trying to reset.
// Counted by CountRecentFailedPasswordResets to drive the per-actor
// lockout counter.
func (s *AuditService) LogPasswordResetFailed(ctx context.Context, actorID uint, actorEmail, ipAddress string, targetUserID uint, reason string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &actorID,
		UserEmail:    actorEmail,
		Action:       models.AuditActionPasswordResetFailed,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details:      mustMarshalJSON(map[string]string{"reason": reason}),
		Success:      false,
	})
}

// LogFactorEnrolled logs completion of MFA factor enrollment.
// `factorType` is the factor-generic type string ("totp", etc.) so
// audit queries can pivot on it. factorID is recorded as the
// ResourceID so a user with multiple authenticators of the same type
// can still be told apart in the audit trail (review finding M2).
func (s *AuditService) LogFactorEnrolled(ctx context.Context, userID, factorID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorEnrolled,
		ResourceType: "factor",
		ResourceID:   &factorID,
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      true,
	})
}

// LogFactorDeleted logs a user removing their OWN MFA factor. factorID
// goes into ResourceID so it pairs with the LogFactorEnrolled row for
// the same authenticator (review finding M2).
func (s *AuditService) LogFactorDeleted(ctx context.Context, userID, factorID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorDeleted,
		ResourceType: "factor",
		ResourceID:   &factorID,
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      true,
	})
}

// LogFactorLabelUpdated logs a user renaming one of their own MFA factors.
// factorID is recorded as the ResourceID so an investigator can correlate
// the event with the exact factor row.
func (s *AuditService) LogFactorLabelUpdated(ctx context.Context, userID, factorID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorLabelUpdated,
		ResourceType: "factor",
		ResourceID:   &factorID,
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      true,
	})
}

// LogFactorActivationLocked logs a pending factor being auto-deleted
// because activation failures hit the limit. This is a security
// signal: the common cause is an attacker in a hijacked session trying
// codes against a freshly-enrolled pending row. factorID identifies
// which pending row was destroyed — a user with several abandoned
// enrollments otherwise produces indistinguishable lock events
// (review finding M2).
func (s *AuditService) LogFactorActivationLocked(ctx context.Context, userID, factorID uint, factorType string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionFactorActivationLocked,
		ResourceType: "factor",
		ResourceID:   &factorID,
		Details:      mustMarshalJSON(map[string]string{"factor_type": factorType}),
		Success:      false,
	})
}

// LogBackupCodesRegenerated logs a user regenerating their backup
// codes. A spike in these signals a user having trouble with their
// primary factor. factorID identifies the backup_codes factor whose
// codes were rotated (review finding M2).
func (s *AuditService) LogBackupCodesRegenerated(ctx context.Context, userID, factorID uint) {
	s.log(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       models.AuditActionBackupCodesRegenerated,
		ResourceType: "factor",
		ResourceID:   &factorID,
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

// LogSuperAdminChangeFailed logs a /users/:userId/superadmin attempt that
// failed the actor_password step-up. UserID on the row is the actor (the
// brute-force victim in the stolen-session threat model); ResourceID carries
// the target user so investigators can see who was about to be
// promoted/demoted. `granted` is the change that *would* have happened.
func (s *AuditService) LogSuperAdminChangeFailed(ctx context.Context, actorID uint, actorEmail string, targetUserID uint, targetEmail string, granted bool, ipAddress, reason string) {
	s.log(ctx, &models.AuditLog{
		UserID:       &actorID,
		UserEmail:    actorEmail,
		Action:       models.AuditActionSuperAdminChangeFailed,
		ResourceType: "user",
		ResourceID:   &targetUserID,
		IPAddress:    ipAddress,
		Details: mustMarshalJSON(map[string]any{
			"target_user_id":    targetUserID,
			"target_user_email": targetEmail,
			"granted":           granted,
			"reason":            reason,
		}),
		Success: false,
	})
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

// LogResourceDeleteWithSnapshot is LogResourceDelete plus a `snapshot` of the
// record's field values as they were at deletion, in the Details JSON.
//
// Updates can be reconstructed from the surviving row plus the `changes` diff.
// A delete cannot: once the row is gone, "contract 42 was deleted" is all that
// is left, and for a contract that means the care type, the funding supplements
// and the period are unrecoverable. The snapshot is the only record of what the
// deletion removed.
func (s *AuditService) LogResourceDeleteWithSnapshot(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint, snapshot map[string]any) {
	if len(snapshot) == 0 {
		s.LogResourceDelete(ctx, actorID, actorEmail, resourceType, resourceID, resourceName, ipAddress, orgID)
		return
	}

	details := map[string]any{"resource_name": resourceName, "snapshot": snapshot}
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditAction(resourceType + "_delete"),
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		OrganizationID: orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(details),
		Success:        true,
	})
}

// LogResourcePurged logs a hard-delete (purge) event, distinct from
// the soft-delete path that LogResourceDelete emits. Used by the
// retention TTL cleanup job and admin-initiated Art. 17 erasure.
// Currently wired for resourceType "user" and "organization"; other
// types fall through to the "<type>_purged" convention.
func (s *AuditService) LogResourcePurged(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint) {
	var action models.AuditAction
	switch resourceType {
	case "user":
		action = models.AuditActionUserPurged
	case "organization":
		action = models.AuditActionOrgPurged
	default:
		action = models.AuditAction(resourceType + "_purged")
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

// LogResourceUpdateWithChanges logs an update event together with the
// per-field diff so the audit row answers "what changed?", not just
// "who touched what". `changes` is a map keyed by field name; each
// value is typically {"old": <prev>, "new": <next>}. Pass an empty
// map (or nil) for an update with no observable change and the row
// degrades to the behaviour of LogResourceUpdate.
//
// Closes review finding H2: pre-fix LogResourceUpdate stored only
// `{"resource_name": "…"}` in Details, so updates were
// indistinguishable for compliance purposes — an investigator could
// see "Anna's record was edited at 14:03" but not "her voucher number
// changed from X to Y".
func (s *AuditService) LogResourceUpdateWithChanges(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgID *uint, changes map[string]any) {
	details := map[string]any{"resource_name": resourceName}
	if len(changes) > 0 {
		details["changes"] = changes
	}
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditAction(resourceType + "_update"),
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		OrganizationID: orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(details),
		Success:        true,
	})
}

// LogResourceUpdateAcrossOrgs emits one update audit row per org id
// in `orgIDs`, plus exactly one identity-level row (OrganizationID =
// nil) so the superadmin global feed always shows the event even for
// users with no org memberships. Each per-org row gets the same
// timestamp window and Details, so an investigator who pivots from
// any org view to the global view sees a consistent picture.
//
// Closes review finding M4: pre-fix `PUT /users/:userId` recorded
// only one row with OrganizationID = NULL, invisible to the org admin
// of every org the user was a member of. Now an org admin who manages
// users in their org sees the update in their org-scoped audit feed
// the moment it happens.
//
// `changes` carries the per-field diff, in the same shape and for the same
// reason as LogResourceUpdateWithChanges. Without it these rows recorded only
// the resource name — which for a user is the email — so the two edits this
// endpoint exists to make, changing an account's address and deactivating an
// account, both landed as "somebody updated this user" with the new email as
// the only evidence and no way to tell which of the two had happened.
func (s *AuditService) LogResourceUpdateAcrossOrgs(ctx context.Context, actorID uint, actorEmail, resourceType string, resourceID uint, resourceName, ipAddress string, orgIDs []uint, changes map[string]any) {
	// Built once and shared by every row so an investigator who pivots from an
	// org view to the global view cannot see two different accounts of the same
	// edit.
	details := map[string]any{"resource_name": resourceName}
	if len(changes) > 0 {
		details["changes"] = changes
	}
	encoded := mustMarshalJSON(details)

	// De-duplicate so a user that's somehow listed twice in
	// user_organizations (shouldn't happen, but defence in depth)
	// doesn't produce duplicate audit rows.
	seen := make(map[uint]bool, len(orgIDs))
	for _, id := range orgIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		idCopy := id
		s.log(ctx, &models.AuditLog{
			UserID:         &actorID,
			UserEmail:      actorEmail,
			Action:         models.AuditAction(resourceType + "_update"),
			ResourceType:   resourceType,
			ResourceID:     &resourceID,
			OrganizationID: &idCopy,
			IPAddress:      ipAddress,
			Details:        encoded,
			Success:        true,
		})
	}

	// Always emit the identity-level row too: the superadmin global
	// feed must see the event even when the target user has no
	// memberships (e.g. a freshly-created user pre-AddToOrganization).
	s.log(ctx, &models.AuditLog{
		UserID:       &actorID,
		UserEmail:    actorEmail,
		Action:       models.AuditAction(resourceType + "_update"),
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		Details:      encoded,
		Success:      true,
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

// LogAccessDenied records an authenticated request that was refused with 403.
//
// `route` is the Gin route pattern (/api/v1/organizations/:orgId/children) and
// `path` the concrete path that was asked for. The query string is deliberately
// NOT recorded: search parameters on the child and employee list endpoints carry
// names typed by the user, and an audit row is the last place that should
// acquire a copy of a child's name that no request ever successfully returned.
//
// `suppressed` is the number of denials for this actor that the throttle
// swallowed since the previous recorded one. It is written only when non-zero,
// so an ordinary refusal stays a plain row and a burst is still visible as a
// burst rather than silently becoming a single event.
//
// OrganizationID is left NULL and the org id from the URL goes into Details as
// `requested_org_id` instead, which is not the obvious choice and is not a
// shortcut. audit_logs.organization_id is a foreign key onto organizations, and
// the single commonest denial worth investigating — someone walking org ids
// looking for one they can reach — names an organization that does not exist.
// RequirePermission answers those with 403 rather than 404 on purpose, so that
// the endpoint is not an existence oracle; writing the requested id into the FK
// column would make precisely those rows fail to insert. A denial is an
// identity-level event like login and password rotation, and lands in the
// superadmin-only global feed with them.
func (s *AuditService) LogAccessDenied(ctx context.Context, userID *uint, email, method, route, path, code, reason, ipAddress, userAgent string, requestedOrgID *uint, suppressed int) {
	details := map[string]any{
		"method": method,
		"route":  route,
		"path":   path,
		"code":   code,
		"reason": reason,
	}
	if requestedOrgID != nil {
		details["requested_org_id"] = *requestedOrgID
	}
	if suppressed > 0 {
		details["suppressed_since_last"] = suppressed
	}
	s.log(ctx, &models.AuditLog{
		UserID:    userID,
		UserEmail: email,
		Action:    models.AuditActionAccessDenied,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   mustMarshalJSON(details),
		Success:   false,
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

// LogResourceImport logs a bulk import of `resourceType` into `orgID`.
// Mirrors LogDataExport for symmetry; the action is `<resource>_import`.
// Captures enough to answer "where did these rows come from?" from the
// audit trail alone:
//
//   - record_count: how many child/employee rows the import touched
//   - filename: the original upload name (best-effort: the multipart
//     header value, which is client-supplied — informational only)
//   - ids: the entity IDs that were created or updated, so an
//     investigator can pivot from a single suspicious row back to the
//     import event that placed it. Capped to the first 1000 to keep
//     the audit row from blowing past the column TEXT budget on
//     pathologically large imports; record_count remains accurate.
//
// Pre-fix the importer audit calls used `auditCreate(..., 0, "YAML import")`
// which discarded every one of these fields. Closes review finding M1.
func (s *AuditService) LogResourceImport(ctx context.Context, actorID uint, actorEmail, resourceType string, orgID uint, recordCount int, ids []uint, filename, ipAddress string) {
	const maxIDsInDetails = 1000
	truncatedIDs := ids
	idsTruncated := false
	if len(truncatedIDs) > maxIDsInDetails {
		truncatedIDs = truncatedIDs[:maxIDsInDetails]
		idsTruncated = true
	}
	details := map[string]any{
		"organization_id": orgID,
		"record_count":    recordCount,
		"filename":        filename,
		"ids":             truncatedIDs,
	}
	if idsTruncated {
		details["ids_truncated"] = true
	}
	s.log(ctx, &models.AuditLog{
		UserID:         &actorID,
		UserEmail:      actorEmail,
		Action:         models.AuditAction(resourceType + "_import"),
		ResourceType:   resourceType,
		OrganizationID: &orgID,
		IPAddress:      ipAddress,
		Details:        mustMarshalJSON(details),
		Success:        true,
	})
}

// IPVisibility controls how much of an actor's IP address a read may return.
//
// The zero value is IPAnonymized on purpose. Every audit read has to decide
// this, and a caller who forgets — a new endpoint, a refactor that drops an
// argument — gets the answer that protects the data rather than the one that
// publishes it.
//
// Which viewers get IPFull is a routing question, not a service one: the
// handlers resolve it from ctxkeys.IsSuperAdmin, which the authorization
// middleware already populates on every path.
//
// Every read that returns audit rows to a caller therefore takes an
// IPVisibility. There deliberately is no convenience getter without one:
// GetLogs and GetLogsByUser used to exist, were unrouted, and returned rows
// straight from the store with the recorded address intact — a fail-open
// shortcut sitting next to a type whose whole point is to fail closed. Both
// were redundant with GetLogsFiltered (pass no filters, or only a user id),
// so they were removed rather than repaired. Add filters there instead of
// reintroducing a getter that cannot express the redaction.
type IPVisibility int

const (
	// IPAnonymized returns only the network prefix of each address.
	IPAnonymized IPVisibility = iota
	// IPFull returns addresses exactly as recorded.
	IPFull
)

// applyIPVisibility reduces the addresses in a page of audit rows unless the
// viewer is entitled to see them in full.
func applyIPVisibility(rows []models.AuditLogResponse, visibility IPVisibility) []models.AuditLogResponse {
	if visibility == IPFull {
		return rows
	}
	for i := range rows {
		rows[i] = rows[i].WithAnonymizedIP()
	}
	return rows
}

// GetLogsFiltered returns paginated audit logs with optional filters
func (s *AuditService) GetLogsFiltered(ctx context.Context, action string, userID *uint, from *time.Time, to *time.Time, limit, offset int, visibility IPVisibility) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindAllFiltered(ctx, action, userID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs")
	}

	return applyIPVisibility(toResponseList(logs, (*models.AuditLog).ToResponse), visibility), total, nil
}

// GetLogsByOrganization returns audit logs scoped to a single organization
// with optional filters. Identity-level events (org_id IS NULL) are excluded
// — only the superadmin-only GetLogsFiltered path sees those.
func (s *AuditService) GetLogsByOrganization(ctx context.Context, orgID uint, action string, userID *uint, from, to *time.Time, limit, offset int, visibility IPVisibility) ([]models.AuditLogResponse, int64, error) {
	if s == nil || s.store == nil {
		return nil, 0, nil
	}

	logs, total, err := s.store.FindByOrganization(ctx, orgID, action, userID, from, to, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch audit logs")
	}

	return applyIPVisibility(toResponseList(logs, (*models.AuditLog).ToResponse), visibility), total, nil
}

// GetLogByID returns a single audit log entry by ID
func (s *AuditService) GetLogByID(ctx context.Context, id uint, visibility IPVisibility) (*models.AuditLogResponse, error) {
	if s == nil || s.store == nil {
		return nil, apperror.NotFound("audit log")
	}

	log, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "audit log")
	}

	resp := log.ToResponse()
	if visibility != IPFull {
		resp = resp.WithAnonymizedIP()
	}
	return &resp, nil
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

// CountRecentFailedPasswordResets counts /users/:userId/password actor_password
// failures for an actor in the last duration. Used for the per-actor lockout
// check on the admin reset endpoint so an attacker holding a stolen admin
// session cannot iterate actor_password candidates against arbitrary target
// users at full API rate. `actorID` is the row's UserID (the actor under
// brute-force pressure), not the reset target.
func (s *AuditService) CountRecentFailedPasswordResets(ctx context.Context, actorID uint, duration time.Duration) (int64, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}

	since := time.Now().UTC().Add(-duration)
	return s.store.CountFailedPasswordResetsSince(ctx, actorID, since)
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
	// L3: fall back to the request's client IP if the caller didn't
	// set one explicitly. Used by FactorService.LogFactor* and the
	// audit-log retention purge — neither has c.ClientIP() in scope,
	// but the IP is on the request context via the RequestID
	// middleware. Empty stays empty for non-HTTP callers (seed
	// imports, CLI tooling) and for service-layer purges that aren't
	// associated with any single request.
	if entry.IPAddress == "" {
		entry.IPAddress = middleware.ClientIPFromContext(ctx)
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

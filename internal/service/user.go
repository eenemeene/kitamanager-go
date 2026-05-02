package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// UserService handles business logic for user operations
type UserService struct {
	store        store.UserStorer
	userOrgStore store.UserOrganizationStorer
	sessionStore store.SessionStorer
	// auditService is optional. ResetPassword uses it to log
	// password_reset_failed events that drive the per-actor lockout
	// counter (see security audit finding A-M-2). Tests that don't
	// exercise the lockout path may leave it nil.
	auditService *AuditService
}

// NewUserService creates a new user service. `sessionStore` is optional —
// callers that do not need session invalidation (e.g. tests for pure CRUD)
// may omit it.
func NewUserService(store store.UserStorer, userOrgStore store.UserOrganizationStorer, sessionStore ...store.SessionStorer) *UserService {
	svc := &UserService{store: store, userOrgStore: userOrgStore}
	if len(sessionStore) > 0 {
		svc.sessionStore = sessionStore[0]
	}
	return svc
}

// WithAuditService attaches an AuditService for the password-reset
// lockout audit trail. Returns the receiver for builder-style use in
// the wiring done by cmd/api/main.go.
func (s *UserService) WithAuditService(audit *AuditService) *UserService {
	s.auditService = audit
	return s
}

// List returns a paginated list of users visible to the requester.
// Superadmins see all users; other users see only users who share at least one organization.
func (s *UserService) List(ctx context.Context, requesterID uint, search string, limit, offset int) ([]models.UserResponse, int64, error) {
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to check superadmin status")
	}

	if isSuperAdmin {
		users, total, err := s.store.FindAll(ctx, search, limit, offset)
		if err != nil {
			return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
		}
		return toResponseList(users, (*models.User).ToResponse), total, nil
	}

	orgRoles, err := s.userOrgStore.GetUserOrganizationsWithRoles(ctx, requesterID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch requester organizations")
	}

	orgIDs := make([]uint, 0, len(orgRoles))
	for orgID := range orgRoles {
		orgIDs = append(orgIDs, orgID)
	}

	if len(orgIDs) == 0 {
		return []models.UserResponse{}, 0, nil
	}

	users, total, err := s.store.FindByOrganizations(ctx, orgIDs, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
	}

	return toResponseList(users, (*models.User).ToResponse), total, nil
}

// ListByOrganization returns a paginated list of users in a specific organization
func (s *UserService) ListByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.UserResponse, int64, error) {
	users, total, err := s.store.FindByOrganization(ctx, orgID, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
	}

	return toResponseList(users, (*models.User).ToResponse), total, nil
}

// GetByID returns a user by ID. Users can always view themselves.
// For other users, requester must be a superadmin or share an organization.
func (s *UserService) GetByID(ctx context.Context, id uint, requesterID uint) (*models.UserResponse, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}

	if err := s.verifyRequesterCanAccessUser(ctx, requesterID, id); err != nil {
		return nil, apperror.NotFound("user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, req *models.UserCreateRequest, createdBy string) (*models.UserResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}
	// Canonicalize: store in lowercase so the case-insensitive unique index
	// (migration 000009) stays consistent with the stored column.
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to hash password")
	}

	user := &models.User{
		Name:      name,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Active:    req.Active,
		CreatedBy: createdBy,
	}

	if err := s.store.Create(ctx, user); err != nil {
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.EmailConflict()
		}
		return nil, apperror.InternalWrap(err, "failed to create user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, id uint, req *models.UserUpdateRequest, requesterID uint) (*models.UserResponse, error) {
	if err := s.verifyRequesterCanModifyUser(ctx, requesterID, id); err != nil {
		return nil, apperror.NotFound("user")
	}

	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}

	// Trim and validate input. Email is lowercased to keep the stored
	// column aligned with the case-insensitive unique index (migration 000009).
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name != "" {
		if validation.IsWhitespaceOnly(req.Name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		user.Name = req.Name
	}
	if req.Email != "" {
		// Check if email is already used by another user
		exists, err := s.store.EmailExistsForOtherUser(ctx, req.Email, id)
		if err != nil {
			return nil, apperror.InternalWrap(err, "failed to validate email")
		}
		if exists {
			return nil, apperror.EmailConflict()
		}
		user.Email = req.Email
	}
	deactivating := req.Active != nil && !*req.Active && user.Active
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update user")
	}

	// Revoke all sessions when a user is deactivated
	if deactivating && s.sessionStore != nil {
		if err := s.sessionStore.DeleteAllForUser(ctx, id); err != nil {
			slog.Error("failed to revoke sessions after user deactivation", "user_id", id, "error", err)
		}
	}

	resp := user.ToResponse()
	return &resp, nil
}

// passwordResetLockoutThreshold caps consecutive actor_password failures
// against the admin /users/:userId/password endpoint before the actor is
// locked out. Closes audit finding A-M-2: without this the per-IP API
// rate limit (60/min) was the only brake on brute-forcing the actor's
// password from a stolen admin session.
const passwordResetLockoutThreshold int64 = 5

// passwordResetLockoutWindow is the rolling window inside which the
// failure counter is consulted. Mirrors passwordChangeLockoutWindow.
const passwordResetLockoutWindow = 15 * time.Minute

// ResetPassword sets a new password for a user (admin-initiated). The
// requester must be a superadmin or an admin in an organization the
// target user belongs to. Non-superadmin requesters cannot reset a
// superadmin's password.
//
// **Self-target is rejected** (BadRequest) — closes audit finding
// A-H-1. A stolen admin session would otherwise have rotated the
// admin's own password without proof of the current password,
// bypassing the M1 step-up that the self-service /me/password
// endpoint enforces. Self password rotation goes through
// AuthService.ChangePassword which has the proper lockout +
// session-revocation machinery.
//
// `actorPassword` is always required and verified via bcrypt before
// any mutation runs. Repeated wrong actor_password values from the
// same actor are counted via password_reset_failed audit events;
// once `passwordResetLockoutThreshold` failures land within
// `passwordResetLockoutWindow`, further attempts return 429 without
// touching bcrypt — closes audit finding A-M-2.
func (s *UserService) ResetPassword(ctx context.Context, userID uint, newPassword, actorPassword string, requesterID uint, ipAddress string) error {
	if requesterID == userID {
		return apperror.BadRequest("use /me/password to change your own password")
	}

	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}

	// Step-up authentication: the requester must prove they currently know
	// their own password. A stolen access token therefore cannot be used to
	// rotate other users' credentials.
	if actorPassword == "" {
		return apperror.BadRequest("actor_password is required")
	}
	requester, err := s.store.FindByID(ctx, requesterID)
	if err != nil {
		return classifyStoreError(err, "requester")
	}

	// Lockout check. If the actor has burned through their failure budget
	// within the rolling window, fail closed without touching bcrypt so
	// timing analysis cannot tell "locked out" from "wrong password".
	if s.auditService != nil {
		failedCount, countErr := s.auditService.CountRecentFailedPasswordResets(ctx, requesterID, passwordResetLockoutWindow)
		if countErr == nil && failedCount >= passwordResetLockoutThreshold {
			s.auditService.LogPasswordResetFailed(ctx, requesterID, requester.Email, ipAddress, userID, "actor locked - too many failed actor_password attempts")
			return apperror.TooManyRequests("too many failed attempts, please try again later")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(requester.Password), []byte(actorPassword)); err != nil {
		if s.auditService != nil {
			s.auditService.LogPasswordResetFailed(ctx, requesterID, requester.Email, ipAddress, userID, "invalid actor_password")
		}
		return apperror.Unauthorized("actor password is incorrect")
	}

	// Prevent non-superadmin from resetting a superadmin's password
	if user.IsSuperAdmin {
		requesterIsSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check superadmin status")
		}
		if !requesterIsSuperAdmin {
			return apperror.Forbidden("only superadmins can reset a superadmin's password")
		}
	}

	// Verify the requester has admin-level access to the target user
	if err := s.verifyRequesterCanModifyUser(ctx, requesterID, userID); err != nil {
		return apperror.NotFound("user")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.InternalWrap(err, "failed to hash password")
	}

	user.Password = string(hashedPassword)
	if err := s.store.Update(ctx, user); err != nil {
		return apperror.InternalWrap(err, "failed to update password")
	}
	return nil
}

// Delete soft-deletes a user — the row is tombstoned with
// deleted_at and becomes invisible to every GORM-model query, but
// physically remains in the DB for a retention window. Every
// active session for the user is hard-revoked so the user is
// signed out immediately across every device (the DeletedAt filter
// in SessionStore.Lookup is belt-and-braces for any session we
// miss, but an explicit revoke avoids the request window).
//
// Users cannot delete themselves. The last superadmin cannot be
// deleted. Hard-deletion (purge) is available via HardDelete,
// used by the Art. 17 erasure flow and the retention TTL job.
func (s *UserService) Delete(ctx context.Context, id uint, requesterID uint) error {
	if id == requesterID {
		return apperror.BadRequest("cannot delete your own account")
	}

	if err := s.verifyRequesterCanModifyUser(ctx, requesterID, id); err != nil {
		return apperror.NotFound("user")
	}

	// Prevent deleting the last superadmin
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return classifyStoreError(err, "user")
	}
	if user.IsSuperAdmin {
		count, err := s.userOrgStore.CountSuperAdmins(ctx)
		if err != nil {
			return apperror.InternalWrap(err, "failed to count superadmins")
		}
		if count <= 1 {
			return apperror.BadRequest("cannot delete the last superadmin")
		}
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete user")
	}
	// Force sign-out across all devices. Sessions are not soft-
	// deletable (the token loses its meaning the moment we tombstone
	// the user row) so we hard-delete them here.
	if s.sessionStore != nil {
		if err := s.sessionStore.DeleteAllForUser(ctx, id); err != nil {
			// Non-fatal: the user is already soft-deleted, and the
			// Lookup filter rejects stale sessions pointing at a
			// tombstoned user. Log and continue.
			slog.Error("failed to revoke sessions after soft-delete", "user_id", id, "error", err)
		}
	}
	return nil
}

// HardDelete permanently removes a user and cascades through the FK
// graph (sessions, factors, user_organizations — see migration
// 000001 + 000014). Bypasses the soft-delete tombstone. Irreversible.
// Used by the Art. 17 right-to-erasure endpoint (Phase 4) and by the
// retention TTL cleanup job (Phase 3). Keeps the same caller-safety
// invariants as Delete (cannot purge self, cannot purge last
// superadmin).
func (s *UserService) HardDelete(ctx context.Context, id uint, requesterID uint) error {
	if id == requesterID {
		return apperror.BadRequest("cannot purge your own account")
	}
	if err := s.verifyRequesterCanModifyUser(ctx, requesterID, id); err != nil {
		return apperror.NotFound("user")
	}
	// Fetch with Unscoped so that purge works whether the user is
	// live or already tombstoned — the retention job targets
	// tombstoned rows; Art. 17 sometimes fires against a live user.
	var user models.User
	if err := s.store.FindByIDUnscoped(ctx, id, &user); err != nil {
		return classifyStoreError(err, "user")
	}
	if user.IsSuperAdmin {
		count, err := s.userOrgStore.CountSuperAdmins(ctx)
		if err != nil {
			return apperror.InternalWrap(err, "failed to count superadmins")
		}
		if count <= 1 {
			return apperror.BadRequest("cannot purge the last superadmin")
		}
	}
	if err := s.store.HardDelete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to purge user")
	}
	return nil
}

// verifyRequesterCanModifyUser checks that the requester can modify the target user.
// Superadmins can modify all users. A user can always modify themselves.
// Others must be an admin in at least one organization the target user belongs to.
func (s *UserService) verifyRequesterCanModifyUser(ctx context.Context, requesterID, targetUserID uint) error {
	if requesterID == targetUserID {
		return nil
	}

	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return nil
	}

	isAdmin, err := s.store.IsAdminInSharedOrg(ctx, requesterID, targetUserID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check admin access")
	}
	if !isAdmin {
		return apperror.NotFound("user")
	}
	return nil
}

// verifyRequesterCanAccessUser checks that the requester can access the target user.
// Superadmins can access all users. A user can always access themselves.
// Others can only access users who share at least one organization.
func (s *UserService) verifyRequesterCanAccessUser(ctx context.Context, requesterID, targetUserID uint) error {
	if requesterID == targetUserID {
		return nil
	}

	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return nil
	}

	shares, err := s.store.SharesOrganization(ctx, requesterID, targetUserID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check shared organization")
	}
	if !shares {
		return apperror.NotFound("user")
	}
	return nil
}

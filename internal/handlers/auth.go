package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// Cookie names. Duplicated in middleware/auth.go (session) and
// middleware/csrf.go (csrf_token) to avoid middleware→handlers imports;
// they are part of the HTTP contract and change together.
const (
	sessionCookie   = "session"
	csrfTokenCookie = "csrf_token"
)

type AuthHandler struct {
	authService   *service.AuthService
	secureCookies bool
}

func NewAuthHandler(authService *service.AuthService, secureCookies bool) *AuthHandler {
	return &AuthHandler{authService: authService, secureCookies: secureCookies}
}

// Login godoc
// @Summary Login user (step 1: password)
// @Description Authenticate with email and password. If the user has
// @Description no MFA factor, returns `{status:"authenticated"}` and
// @Description sets the session cookie. If the user has an active
// @Description factor, returns `{status:"mfa_required", pending_token,
// @Description expires_at, factors}` and the caller must follow up
// @Description with POST /auth/mfa/verify. No cookie is set on the
// @Description MFA branch.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse "Authenticated"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, ok := bindJSON[models.LoginRequest](c)
	if !ok {
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		respondError(c, err)
		return
	}

	if result.Pending != nil {
		// MFA required — return the pending handle + factor list. No
		// cookie is set: the session only materializes after
		// /auth/mfa/verify succeeds.
		c.JSON(http.StatusOK, models.LoginMFARequiredResponse{
			Status:       models.LoginStatusMFARequired,
			PendingToken: result.Pending.PendingToken,
			ExpiresAt:    result.Pending.ExpiresAt.UTC().Format(http.TimeFormat),
			Factors:      result.Pending.Factors,
		})
		return
	}

	h.setAuthCookies(c, result.Authenticated)

	c.JSON(http.StatusOK, models.LoginResponse{
		Status:    models.LoginStatusAuthenticated,
		ExpiresIn: result.Authenticated.ExpiresIn,
	})
}

// MFAVerify godoc
// @Summary Login step 2: verify MFA code
// @Description Exchanges a pending_token from the /login MFA-required
// @Description response + a TOTP code (or a backup code) for a real
// @Description session. Sets the session cookie on success.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.MFAVerifyRequest true "Pending token + factor id + code"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/auth/mfa/verify [post]
func (h *AuthHandler) MFAVerify(c *gin.Context) {
	req, ok := bindJSON[models.MFAVerifyRequest](c)
	if !ok {
		return
	}
	result, err := h.authService.VerifyMFALogin(
		c.Request.Context(),
		req.PendingToken,
		req.FactorID,
		req.Code,
		req.WebAuthnResponse,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		respondError(c, err)
		return
	}

	h.setAuthCookies(c, result)

	c.JSON(http.StatusOK, models.LoginResponse{
		Status:    models.LoginStatusAuthenticated,
		ExpiresIn: result.ExpiresIn,
	})
}

// MFAChallenge godoc
// @Summary Login step 2a: request a WebAuthn challenge
// @Description For WebAuthn factors, the client must fetch a server-
// @Description issued challenge via this endpoint before calling
// @Description navigator.credentials.get(). Returns the
// @Description PublicKeyCredentialRequestOptions JSON unchanged. TOTP
// @Description and backup-code factors never hit this endpoint.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.MFAChallengeRequest true "Pending token + factor id"
// @Success 200 {object} models.MFAChallengeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/auth/mfa/challenge [post]
func (h *AuthHandler) MFAChallenge(c *gin.Context) {
	req, ok := bindJSON[models.MFAChallengeRequest](c)
	if !ok {
		return
	}
	options, err := h.authService.BeginMFAChallenge(c.Request.Context(), req.PendingToken, req.FactorID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.MFAChallengeResponse{RequestOptions: options})
}

// Logout godoc
// @Summary Logout user
// @Description Delete the current session and clear authentication cookies.
// @Tags auth
// @Produce json
// @Success 200 {object} models.MessageResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionToken, _ := c.Cookie(sessionCookie)

	h.authService.Logout(c.Request.Context(), sessionToken)

	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "logged out successfully",
	})
}

// Me godoc
// @Summary Get current user
// @Description Returns the currently authenticated user.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userIDValue, exists := c.Get(ctxkeys.UserID)
	if !exists {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		respondError(c, apperror.Internal("invalid user ID type"))
		return
	}

	user, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ChangePassword godoc
// @Summary Change current user's password
// @Description Authenticated user changes their own password. Requires current password verification. Revokes every other session the user has.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserPasswordChangeRequest true "Password change data"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/me/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return
	}

	req, ok := bindJSON[models.UserPasswordChangeRequest](c)
	if !ok {
		return
	}

	currentSessionIDHash, _ := c.Get(ctxkeys.SessionIDHash)
	sessionIDHash, _ := currentSessionIDHash.(string)

	result, err := h.authService.ChangePassword(
		c.Request.Context(),
		userID,
		req.CurrentPassword,
		req.NewPassword,
		sessionIDHash,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		respondError(c, err)
		return
	}

	h.setAuthCookies(c, result)

	c.JSON(http.StatusOK, models.LoginResponse{
		Status:    models.LoginStatusAuthenticated,
		ExpiresIn: result.ExpiresIn,
	})
}

// ListSessions godoc
// @Summary List the caller's active sessions
// @Description Returns every non-expired session belonging to the caller,
// @Description with a `current` flag marking the one that served this
// @Description request. Used by the UI to show "signed in on these devices"
// @Description and to let the user revoke individual sessions.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserSessionsResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/me/sessions [get]
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return
	}

	current, _ := c.Get(ctxkeys.SessionIDHash)
	currentHash, _ := current.(string)

	sessions, err := h.authService.ListSessions(c.Request.Context(), userID, currentHash)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.UserSessionsResponse{Sessions: sessions})
}

// RevokeSession godoc
// @Summary Revoke one of the caller's own sessions
// @Description Deletes the session with the given id (sha256 hex of the
// @Description cookie value). Scoped to the caller — a user cannot revoke
// @Description another user's session, and the response is 404 in that
// @Description case so the endpoint does not leak session existence. If
// @Description the caller revokes their own current session, their next
// @Description request will 401 and the frontend will clear cookies.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session id (sha256 hex)"
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/me/sessions/{id} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return
	}

	id := c.Param("id")

	if err := h.authService.RevokeSession(c.Request.Context(), userID, id); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// setAuthCookies sets the session and CSRF cookies from an AuthResult.
func (h *AuthHandler) setAuthCookies(c *gin.Context, result *service.AuthResult) {
	maxAge := int(result.ExpiresIn)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookie, result.SessionToken, maxAge, "/", "", h.secureCookies, true)
	c.SetCookie(csrfTokenCookie, result.CSRFToken, maxAge, "/", "", h.secureCookies, false)
}

// clearAuthCookies clears the session and CSRF cookies.
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookie, "", -1, "/", "", h.secureCookies, true)
	c.SetCookie(csrfTokenCookie, "", -1, "/", "", h.secureCookies, false)
}

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
// @Summary Login user
// @Description Authenticate user with email and password. Sets a session cookie on success.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
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

	h.setAuthCookies(c, result)

	c.JSON(http.StatusOK, models.LoginResponse{
		ExpiresIn: result.ExpiresIn,
	})
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
		ExpiresIn: result.ExpiresIn,
	})
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

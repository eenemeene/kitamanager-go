package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// Cookie names
const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	csrfTokenCookie    = "csrf_token"
)

// refreshCookiePath restricts the refresh cookie to only be sent to the refresh endpoint.
const refreshCookiePath = "/api/v1/refresh"

type AuthHandler struct {
	authService   *service.AuthService
	secureCookies bool
}

func NewAuthHandler(authService *service.AuthService, secureCookies bool) *AuthHandler {
	return &AuthHandler{authService: authService, secureCookies: secureCookies}
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with email and password, returns JWT token
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

// Refresh godoc
// @Summary Refresh access token
// @Description Exchange a valid refresh token (from HttpOnly cookie) for new access and refresh tokens
// @Tags auth
// @Produce json
// @Success 200 {object} models.RefreshResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenStr, err := c.Cookie(refreshTokenCookie)
	if err != nil || refreshTokenStr == "" {
		respondError(c, apperror.Unauthorized("missing refresh token"))
		return
	}

	result, authErr := h.authService.Refresh(c.Request.Context(), refreshTokenStr)
	if authErr != nil {
		respondError(c, authErr)
		return
	}

	h.setAuthCookies(c, result)

	c.JSON(http.StatusOK, models.RefreshResponse{
		ExpiresIn: result.ExpiresIn,
	})
}

// Logout godoc
// @Summary Logout user
// @Description Clear authentication cookies to log out the user
// @Tags auth
// @Produce json
// @Success 200 {object} models.MessageResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	accessTokenStr, _ := c.Cookie(accessTokenCookie)
	refreshTokenStr, _ := c.Cookie(refreshTokenCookie)

	h.authService.Logout(c.Request.Context(), accessTokenStr, refreshTokenStr)

	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "logged out successfully",
	})
}

// Me godoc
// @Summary Get current user
// @Description Returns the currently authenticated user based on the JWT token
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
// @Description Authenticated user changes their own password. Requires current password verification. Revokes all existing tokens.
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

	result, err := h.authService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword, c.ClientIP())
	if err != nil {
		respondError(c, err)
		return
	}

	h.setAuthCookies(c, result)

	c.JSON(http.StatusOK, models.LoginResponse{
		ExpiresIn: result.ExpiresIn,
	})
}

// setAuthCookies sets the authentication cookies from an AuthResult.
func (h *AuthHandler) setAuthCookies(c *gin.Context, result *service.AuthResult) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(accessTokenCookie, result.AccessToken, int(service.AccessTokenExpiry.Seconds()), "/", "", h.secureCookies, true)
	c.SetCookie(refreshTokenCookie, result.RefreshToken, int(service.RefreshTokenExpiry.Seconds()), refreshCookiePath, "", h.secureCookies, true)
	c.SetCookie(csrfTokenCookie, result.CSRFToken, int(service.AccessTokenExpiry.Seconds()), "/", "", h.secureCookies, false)
}

// clearAuthCookies clears all authentication cookies.
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(accessTokenCookie, "", -1, "/", "", h.secureCookies, true)
	c.SetCookie(refreshTokenCookie, "", -1, refreshCookiePath, "", h.secureCookies, true)
	c.SetCookie(csrfTokenCookie, "", -1, "/", "", h.secureCookies, false)
}

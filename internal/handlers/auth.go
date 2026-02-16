package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Token expiration settings
const (
	accessTokenExpiry  = 1 * time.Hour      // Access tokens expire in 1 hour
	refreshTokenExpiry = 7 * 24 * time.Hour // Refresh tokens expire in 7 days
)

// Cookie names
const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	csrfTokenCookie    = "csrf_token"
)

type AuthHandler struct {
	userStore    store.UserStorer
	tokenStore   store.TokenStorer
	jwtSecret    string
	auditService *service.AuditService
}

func NewAuthHandler(userStore store.UserStorer, tokenStore store.TokenStorer, jwtSecret string, auditService *service.AuditService) *AuthHandler {
	return &AuthHandler{
		userStore:    userStore,
		tokenStore:   tokenStore,
		jwtSecret:    jwtSecret,
		auditService: auditService,
	}
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
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	const (
		lockoutThreshold = 5
		lockoutWindow    = 15 * time.Minute
	)

	req, ok := bindJSON[models.LoginRequest](c)
	if !ok {
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Check for account lockout
	failedCount, err := h.auditService.CountRecentFailedLogins(c.Request.Context(), req.Email, lockoutWindow)
	if err == nil && failedCount >= lockoutThreshold {
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "account locked - too many failed attempts")
		respondError(c, apperror.TooManyRequests("too many failed login attempts, please try again later"))
		return
	}

	user, err := h.userStore.FindByEmail(c.Request.Context(), req.Email)
	if err != nil {
		// Log failed login attempt
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "user not found")
		// Use generic message to prevent user enumeration
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	if !user.Active {
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "user inactive")
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "invalid password")
		// Use generic message to prevent password guessing
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	// Update last login timestamp
	if err := h.userStore.UpdateLastLogin(c.Request.Context(), user.ID); err != nil {
		// Log but don't fail login if last_login update fails
		_ = c.Error(err)
	}

	// Generate access token
	accessToken, err := h.generateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate access token"))
		return
	}

	// Generate refresh token
	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate refresh token"))
		return
	}

	// Log successful login
	h.auditService.LogLogin(user.ID, user.Email, ipAddress, userAgent)

	// Derive CSRF token from access token (binds CSRF to session)
	csrfToken := middleware.ComputeCSRFToken(accessToken, h.jwtSecret)

	// Set HttpOnly cookies for tokens
	h.setAuthCookies(c, accessToken, refreshToken, csrfToken)

	c.JSON(http.StatusOK, models.LoginResponse{
		ExpiresIn: int64(accessTokenExpiry.Seconds()),
	})
}

// generateAccessToken creates a short-lived JWT for API access
func (h *AuthHandler) generateAccessToken(userID uint, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"type":    "access",
		"exp":     time.Now().Add(accessTokenExpiry).Unix(),
		"iat":     time.Now().Unix(),
	})
	return token.SignedString([]byte(h.jwtSecret))
}

// generateRefreshToken creates a long-lived JWT for token refresh
func (h *AuthHandler) generateRefreshToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(refreshTokenExpiry).Unix(),
		"iat":     time.Now().Unix(),
	})
	return token.SignedString([]byte(h.jwtSecret))
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
	// Read refresh token from HttpOnly cookie
	refreshTokenStr, err := c.Cookie(refreshTokenCookie)
	if err != nil || refreshTokenStr == "" {
		respondError(c, apperror.Unauthorized("missing refresh token"))
		return
	}

	// Parse and validate the refresh token
	token, err := jwt.Parse(refreshTokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		respondError(c, apperror.Unauthorized("invalid refresh token"))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		respondError(c, apperror.Unauthorized("invalid token claims"))
		return
	}

	// Verify this is a refresh token
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		respondError(c, apperror.Unauthorized("invalid token type"))
		return
	}

	// Get user ID from claims
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		respondError(c, apperror.Unauthorized("invalid user ID in token"))
		return
	}
	userID := uint(userIDFloat)

	// Check for token replay before processing
	if h.tokenStore != nil {
		oldHash := middleware.HashToken(refreshTokenStr)
		revoked, err := h.tokenStore.IsRevoked(c.Request.Context(), oldHash)
		if err == nil && revoked {
			// Replay detected — revoke all tokens for this user as precaution
			_ = h.tokenStore.RevokeAllForUser(c.Request.Context(), userID)
			respondError(c, apperror.Unauthorized("token reuse detected, all sessions have been revoked"))
			return
		}

		userRevoked, err := h.tokenStore.IsUserRevoked(c.Request.Context(), userID)
		if err == nil && userRevoked {
			respondError(c, apperror.Unauthorized("all sessions have been revoked, please login again"))
			return
		}
	}

	// Verify user still exists and is active
	user, err := h.userStore.FindByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, apperror.Unauthorized("invalid refresh token"))
		return
	}

	if !user.Active {
		respondError(c, apperror.Unauthorized("invalid refresh token"))
		return
	}

	// Revoke the old refresh token
	if h.tokenStore != nil {
		expFloat, _ := claims["exp"].(float64)
		oldExpiresAt := time.Unix(int64(expFloat), 0)
		oldHash := middleware.HashToken(refreshTokenStr)
		_ = h.tokenStore.RevokeToken(c.Request.Context(), oldHash, user.ID, oldExpiresAt)
	}

	// Generate new tokens
	accessToken, err := h.generateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate access token"))
		return
	}

	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate refresh token"))
		return
	}

	// Derive CSRF token from access token (binds CSRF to session)
	csrfToken := middleware.ComputeCSRFToken(accessToken, h.jwtSecret)

	// Set HttpOnly cookies for tokens
	h.setAuthCookies(c, accessToken, refreshToken, csrfToken)

	c.JSON(http.StatusOK, models.RefreshResponse{
		ExpiresIn: int64(accessTokenExpiry.Seconds()),
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
	// Revoke current tokens if token store is available
	if h.tokenStore != nil {
		// Revoke access token
		if cookie, err := c.Cookie(accessTokenCookie); err == nil && cookie != "" {
			h.revokeTokenString(c, cookie)
		}
		// Revoke refresh token
		if cookie, err := c.Cookie(refreshTokenCookie); err == nil && cookie != "" {
			h.revokeTokenString(c, cookie)
		}
	}

	// Clear all auth cookies
	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "logged out successfully",
	})
}

// revokeTokenString parses a JWT to extract user_id and exp, then revokes it.
func (h *AuthHandler) revokeTokenString(c *gin.Context, tokenStr string) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return // Can't revoke invalid tokens
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return
	}
	userIDFloat, _ := claims["user_id"].(float64)
	expFloat, _ := claims["exp"].(float64)
	expiresAt := time.Unix(int64(expFloat), 0)
	tokenHash := middleware.HashToken(tokenStr)
	_ = h.tokenStore.RevokeToken(c.Request.Context(), tokenHash, uint(userIDFloat), expiresAt)
}

// setAuthCookies sets the authentication cookies (access token, refresh token, CSRF token)
func (h *AuthHandler) setAuthCookies(c *gin.Context, accessToken, refreshToken, csrfToken string) {
	// Determine if we should use secure cookies (HTTPS)
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// Access token cookie - HttpOnly (not accessible from JavaScript)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		accessTokenCookie,
		accessToken,
		int(accessTokenExpiry.Seconds()),
		"/",
		"",     // domain - empty uses request host
		secure, // secure
		true,   // httpOnly
	)

	// Refresh token cookie - HttpOnly (not accessible from JavaScript)
	c.SetCookie(
		refreshTokenCookie,
		refreshToken,
		int(refreshTokenExpiry.Seconds()),
		"/",
		"",
		secure,
		true, // httpOnly
	)

	// CSRF token cookie - NOT HttpOnly (must be readable by JavaScript)
	c.SetCookie(
		csrfTokenCookie,
		csrfToken,
		int(accessTokenExpiry.Seconds()),
		"/",
		"",
		secure,
		false, // NOT httpOnly - JS needs to read this
	)
}

// clearAuthCookies clears all authentication cookies
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetSameSite(http.SameSiteStrictMode)

	// Clear access token
	c.SetCookie(accessTokenCookie, "", -1, "/", "", secure, true)

	// Clear refresh token
	c.SetCookie(refreshTokenCookie, "", -1, "/", "", secure, true)

	// Clear CSRF token
	c.SetCookie(csrfTokenCookie, "", -1, "/", "", secure, false)
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
	// Get user ID from context (set by auth middleware)
	userIDValue, exists := c.Get("userID")
	if !exists {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		respondError(c, apperror.Internal("invalid user ID type"))
		return
	}

	user, err := h.userStore.FindByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, apperror.NotFound("user not found"))
		return
	}

	c.JSON(http.StatusOK, models.UserResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		Active:       user.Active,
		IsSuperAdmin: user.IsSuperAdmin,
		LastLogin:    user.LastLogin,
		CreatedAt:    user.CreatedAt,
		CreatedBy:    user.CreatedBy,
		UpdatedAt:    user.UpdatedAt,
	})
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

	user, err := h.userStore.FindByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, apperror.NotFound("user not found"))
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		respondError(c, apperror.Unauthorized("current password is incorrect"))
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, apperror.Internal("failed to hash password"))
		return
	}

	user.Password = string(hashedPassword)
	if err := h.userStore.Update(c.Request.Context(), user); err != nil {
		respondError(c, apperror.Internal("failed to update password"))
		return
	}

	// Revoke all existing tokens for this user
	if h.tokenStore != nil {
		_ = h.tokenStore.RevokeAllForUser(c.Request.Context(), userID)
	}

	// Generate new tokens so the user stays logged in
	accessToken, err := h.generateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate access token"))
		return
	}
	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		respondError(c, apperror.Internal("failed to generate refresh token"))
		return
	}
	csrfToken := middleware.ComputeCSRFToken(accessToken, h.jwtSecret)

	h.setAuthCookies(c, accessToken, refreshToken, csrfToken)

	h.auditService.LogResourceUpdate(userID, "user_password", userID, user.Email, c.ClientIP())

	c.JSON(http.StatusOK, models.LoginResponse{
		ExpiresIn: int64(accessTokenExpiry.Seconds()),
	})
}

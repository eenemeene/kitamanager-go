package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Token expiration settings
const (
	accessTokenExpiry  = 1 * time.Hour      // Access tokens expire in 1 hour
	refreshTokenExpiry = 7 * 24 * time.Hour // Refresh tokens expire in 7 days
)

type AuthHandler struct {
	userStore    *store.UserStore
	jwtSecret    string
	auditService *service.AuditService
}

func NewAuthHandler(userStore *store.UserStore, jwtSecret string, auditService *service.AuditService) *AuthHandler {
	return &AuthHandler{
		userStore:    userStore,
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
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	user, err := h.userStore.FindByEmail(req.Email)
	if err != nil {
		// Log failed login attempt
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "user not found")
		// Use generic message to prevent user enumeration
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	if !user.Active {
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "user inactive")
		respondError(c, apperror.Unauthorized("user is inactive"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.auditService.LogLoginFailed(req.Email, ipAddress, userAgent, "invalid password")
		// Use generic message to prevent password guessing
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	// Update last login timestamp
	if err := h.userStore.UpdateLastLogin(user.ID); err != nil {
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

	c.JSON(http.StatusOK, models.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenExpiry.Seconds()),
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
// @Description Exchange a valid refresh token for new access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshRequest true "Refresh token"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	// Parse and validate the refresh token
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
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

	// Verify user still exists and is active
	user, err := h.userStore.FindByID(userID)
	if err != nil {
		respondError(c, apperror.Unauthorized("user not found"))
		return
	}

	if !user.Active {
		respondError(c, apperror.Unauthorized("user is inactive"))
		return
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

	c.JSON(http.StatusOK, models.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenExpiry.Seconds()),
	})
}

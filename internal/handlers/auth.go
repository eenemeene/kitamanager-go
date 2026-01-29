package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Note: net/http is still needed for http.StatusOK in the successful response

type AuthHandler struct {
	userStore *store.UserStore
	jwtSecret string
}

func NewAuthHandler(userStore *store.UserStore, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userStore: userStore,
		jwtSecret: jwtSecret,
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

	user, err := h.userStore.FindByEmail(req.Email)
	if err != nil {
		// Use generic message to prevent user enumeration
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	if !user.Active {
		respondError(c, apperror.Unauthorized("user is inactive"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// Use generic message to prevent password guessing
		respondError(c, apperror.Unauthorized("invalid credentials"))
		return
	}

	// Update last login timestamp
	if err := h.userStore.UpdateLastLogin(user.ID); err != nil {
		// Log but don't fail login if last_login update fails
		_ = c.Error(err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		respondError(c, apperror.Internal("failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{Token: tokenString})
}

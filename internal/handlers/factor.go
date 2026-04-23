package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// FactorHandler serves /api/v1/users/:userId/factors/*. The same
// handler serves both the self-addressed form (`:userId=me`) and the
// admin-addressed form (`:userId=<int>`). Admin-only endpoints get an
// additional middleware gate in routes.go.
type FactorHandler struct {
	service *service.FactorService
}

// NewFactorHandler constructs the handler.
func NewFactorHandler(svc *service.FactorService) *FactorHandler {
	return &FactorHandler{service: svc}
}

// resolveTargetUserID decodes the :userId path param. `me` is a
// convenience alias resolved to the caller's own user id. A numeric
// id must match the caller unless the request is already authorised
// by upstream middleware (e.g. admin routes). For this PR we only
// permit self-addressing; admin endpoints arrive in a later PR.
func resolveTargetUserID(c *gin.Context) (uint, bool) {
	callerID := getUserID(c)
	if callerID == 0 {
		respondError(c, apperror.Unauthorized("not authenticated"))
		return 0, false
	}
	raw := c.Param("userId")
	if raw == "me" {
		return callerID, true
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid user id"))
		return 0, false
	}
	id := uint(n)
	if id != callerID {
		// Self-scope only — admin endpoints are wired separately in a
		// future PR and carry their own permission check.
		respondError(c, apperror.NotFound("factor"))
		return 0, false
	}
	return id, true
}

// parseFactorID decodes :id into a uint.
func parseFactorID(c *gin.Context) (uint, bool) {
	raw := c.Param("id")
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n == 0 {
		// Treat a malformed / zero id as not-found so probes can't
		// distinguish "no such id" from "bad id format."
		respondError(c, apperror.NotFound("factor"))
		return 0, false
	}
	return uint(n), true
}

// getUserEmailForTOTPLabel returns the email to embed in the
// otpauth URI as the account label. Defaults to the empty string
// if unavailable (shouldn't happen in practice — middleware
// populates ctxkeys.UserEmail).
func getUserEmailForTOTPLabel(c *gin.Context) string {
	v, _ := c.Get(ctxkeys.UserEmail)
	email, _ := v.(string)
	return email
}

// List godoc
// @Summary List the caller's MFA factors
// @Description Returns every activated factor owned by the addressed user.
// @Description Pending (not-yet-activated) factors are hidden.
// @Tags factors
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Success 200 {object} models.FactorListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors [get]
func (h *FactorHandler) List(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factors, err := h.service.ListForUser(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.FactorListResponse{Factors: factors})
}

// Get godoc
// @Summary Get a single factor
// @Tags factors
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param id path int true "Factor id"
// @Success 200 {object} models.FactorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors/{id} [get]
func (h *FactorHandler) Get(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factorID, ok := parseFactorID(c)
	if !ok {
		return
	}
	f, err := h.service.GetForUser(c.Request.Context(), userID, factorID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, f)
}

// Enroll godoc
// @Summary Start MFA enrollment
// @Description Creates a pending factor. For TOTP, returns the
// @Description base32 secret and otpauth URI that the client displays
// @Description as a QR code. The factor is not yet usable — call
// @Description /activate with a valid code to enable it.
// @Description Password re-entry is required (step-up) to prevent a
// @Description stolen session from installing a backdoor authenticator.
// @Tags factors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param request body models.FactorEnrollRequest true "Factor type + step-up password"
// @Success 200 {object} models.FactorResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors [post]
func (h *FactorHandler) Enroll(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	req, ok := bindJSON[models.FactorEnrollRequest](c)
	if !ok {
		return
	}
	switch req.Type {
	case models.FactorTypeTOTP:
		email := getUserEmailForTOTPLabel(c)
		resp, err := h.service.EnrollTOTP(c.Request.Context(), userID, req.Label, req.Password, email)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	case models.FactorTypeBackupCodes:
		respondError(c, apperror.BadRequest("backup_codes factors are auto-created; do not enroll directly"))
	default:
		respondError(c, apperror.BadRequest("unsupported factor type"))
	}
}

// Activate godoc
// @Summary Activate a pending factor
// @Description Verifies the first code (proving the user can read
// @Description their authenticator) and enables the factor. If this
// @Description is the user's first primary factor, a fresh set of
// @Description backup codes is generated and returned — save them NOW.
// @Tags factors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param id path int true "Factor id"
// @Param request body models.FactorActivateRequest true "Code from the authenticator"
// @Success 200 {object} models.FactorActivateResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors/{id}/activate [post]
func (h *FactorHandler) Activate(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factorID, ok := parseFactorID(c)
	if !ok {
		return
	}
	req, ok := bindJSON[models.FactorActivateRequest](c)
	if !ok {
		return
	}
	resp, err := h.service.ActivateFactor(c.Request.Context(), userID, factorID, req.Code)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Regenerate godoc
// @Summary Regenerate backup codes
// @Description Replaces every backup code (used or unused) with a
// @Description fresh set. Password re-entry required. Only valid for
// @Description backup_codes factors.
// @Tags factors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param id path int true "Factor id"
// @Param request body models.FactorRegenerateRequest true "Step-up password"
// @Success 200 {object} models.BackupCodesPayload
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors/{id}/regenerate [post]
func (h *FactorHandler) Regenerate(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factorID, ok := parseFactorID(c)
	if !ok {
		return
	}
	req, ok := bindJSON[models.FactorRegenerateRequest](c)
	if !ok {
		return
	}
	payload, err := h.service.RegenerateBackupCodes(c.Request.Context(), userID, factorID, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// UpdateLabel godoc
// @Summary Edit a factor's label
// @Description Rename a factor. No step-up required.
// @Tags factors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param id path int true "Factor id"
// @Param request body models.FactorLabelUpdateRequest true "New label"
// @Success 200 {object} models.FactorResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors/{id} [patch]
func (h *FactorHandler) UpdateLabel(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factorID, ok := parseFactorID(c)
	if !ok {
		return
	}
	req, ok := bindJSON[models.FactorLabelUpdateRequest](c)
	if !ok {
		return
	}
	resp, err := h.service.UpdateLabel(c.Request.Context(), userID, factorID, req.Label)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Delete godoc
// @Summary Remove a factor
// @Description Disables a factor. Password re-entry required
// @Description (step-up). A code from any active factor is additionally
// @Description required when removing your last primary factor —
// @Description otherwise a stolen session could wipe your 2FA and
// @Description leave you with password-only protection.
// @Tags factors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User id or 'me'"
// @Param id path int true "Factor id"
// @Param request body models.FactorDeleteRequest true "Step-up password + optional code"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/factors/{id} [delete]
func (h *FactorHandler) Delete(c *gin.Context) {
	userID, ok := resolveTargetUserID(c)
	if !ok {
		return
	}
	factorID, ok := parseFactorID(c)
	if !ok {
		return
	}
	req, ok := bindJSON[models.FactorDeleteRequest](c)
	if !ok {
		return
	}
	if err := h.service.DeleteFactor(c.Request.Context(), userID, factorID, req.Password, req.Code); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

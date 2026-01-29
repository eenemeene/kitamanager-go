package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check godoc
// @Summary Health check
// @Description Check the health status of the API and its dependencies
// @Tags health
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Failure 503 {object} models.HealthResponse
// @Router /api/v1/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	services := make(map[string]string)
	healthy := true

	// Check database connectivity
	sqlDB, err := h.db.DB()
	if err != nil {
		services["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else if err := sqlDB.Ping(); err != nil {
		services["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["database"] = "healthy"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, models.HealthResponse{
		Status:   status,
		Version:  "1.0.0",
		Services: services,
	})
}

// Ready godoc
// @Summary Readiness check
// @Description Check if the API is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	// Check database connectivity
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Live godoc
// @Summary Liveness check
// @Description Check if the API process is alive
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

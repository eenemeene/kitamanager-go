package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestHealthHandler_Check(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHealthHandler(db)

	r := gin.New()
	r.GET("/health", handler.Check)

	t.Run("returns healthy when database is connected", func(t *testing.T) {
		w := performRequest(r, "GET", "/health", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.HealthResponse
		parseResponse(t, w, &response)

		if response.Status != "healthy" {
			t.Errorf("expected status 'healthy', got '%s'", response.Status)
		}
		if response.Services["database"] != "healthy" {
			t.Errorf("expected database service 'healthy', got '%s'", response.Services["database"])
		}
		if response.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", response.Version)
		}
	})
}

func TestHealthHandler_Ready(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHealthHandler(db)

	r := gin.New()
	r.GET("/ready", handler.Ready)

	t.Run("returns ready when database is connected", func(t *testing.T) {
		w := performRequest(r, "GET", "/ready", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		parseResponse(t, w, &response)

		if response["status"] != "ready" {
			t.Errorf("expected status 'ready', got '%s'", response["status"])
		}
	})
}

func TestHealthHandler_Live(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHealthHandler(db)

	r := gin.New()
	r.GET("/live", handler.Live)

	t.Run("returns alive", func(t *testing.T) {
		w := performRequest(r, "GET", "/live", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		parseResponse(t, w, &response)

		if response["status"] != "alive" {
			t.Errorf("expected status 'alive', got '%s'", response["status"])
		}
	})
}

func TestNewHealthHandler(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHealthHandler(db)

	if handler == nil {
		t.Error("expected handler to be created, got nil")
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func getModelPath(t *testing.T) string {
	t.Helper()

	paths := []string{
		"../../configs/rbac_model.conf",
		"configs/rbac_model.conf",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	t.Fatal("Could not find rbac_model.conf")
	return ""
}

func setupTestEnforcer(t *testing.T) *rbac.Enforcer {
	t.Helper()

	modelPath := getModelPath(t)

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	if err := os.WriteFile(policyFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create temp policy file: %v", err)
	}

	adapter := fileadapter.NewAdapter(policyFile)

	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		t.Fatalf("failed to load model: %v", err)
	}

	enforcer, err := rbac.NewEnforcerWithAdapter(adapter, modelPath)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	enforcer.SetModel(m)

	if err := enforcer.SeedDefaultPolicies(); err != nil {
		t.Fatalf("failed to seed policies: %v", err)
	}

	return enforcer
}

func TestAuthorizationMiddleware_RequirePermission_Allowed(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleAdmin, 1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId/employees",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/1/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddleware_RequirePermission_Forbidden(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleManager, 1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.PUT("/organizations/:orgId/settings",
		middleware.RequirePermission(rbac.ResourceOrganizations, rbac.ActionUpdate),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("PUT", "/organizations/1/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_RequirePermission_NoUserID(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	// No userID set
	r.GET("/organizations/:orgId/employees",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/1/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthorizationMiddleware_RequirePermission_InvalidOrgID(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleAdmin, 1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId/employees",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/invalid/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuthorizationMiddleware_RequirePermission_SuperAdminBypass(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignSuperAdmin(1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.DELETE("/organizations/:orgId/employees/:id",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionDelete),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	// Superadmin can access any org
	req, _ := http.NewRequest("DELETE", "/organizations/999/employees/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddleware_RequireSuperAdmin_Allowed(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignSuperAdmin(1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.POST("/organizations",
		middleware.RequireSuperAdmin(),
		func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

	req, _ := http.NewRequest("POST", "/organizations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireSuperAdmin_Forbidden(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleAdmin, 1) // Admin, not superadmin

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.POST("/organizations",
		middleware.RequireSuperAdmin(),
		func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

	req, _ := http.NewRequest("POST", "/organizations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireOrgAccess_Allowed(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleManager, 1)

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId",
		middleware.RequireOrgAccess(),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireOrgAccess_Forbidden(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleManager, 1) // Only has access to org 1

	middleware := NewAuthorizationMiddleware(enforcer)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId",
		middleware.RequireOrgAccess(),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/2", nil) // Trying to access org 2
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_OrgIDSetInContext(t *testing.T) {
	enforcer := setupTestEnforcer(t)
	_ = enforcer.AssignRole(1, rbac.RoleAdmin, 42) // Assign admin role for org 42

	middleware := NewAuthorizationMiddleware(enforcer)

	var capturedOrgID uint

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/organizations/:orgId/employees",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionRead),
		func(c *gin.Context) {
			if orgID, exists := c.Get("orgID"); exists {
				capturedOrgID = orgID.(uint)
			}
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/organizations/42/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedOrgID != uint(42) {
		t.Errorf("expected orgID 42 in context, got %v", capturedOrgID)
	}
}

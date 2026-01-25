package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v3/model"
	fileadapter "github.com/casbin/casbin/v3/persist/file-adapter"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/store"
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

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
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

func setupTestPermissionService(t *testing.T, db *gorm.DB, enforcer *rbac.Enforcer) *rbac.PermissionService {
	t.Helper()
	userGroupStore := store.NewUserGroupStore(db)
	return rbac.NewPermissionService(userGroupStore, enforcer)
}

// assignRole adds a user to a group with a role in the database
func assignRole(t *testing.T, db *gorm.DB, userID uint, role models.Role, orgID uint) {
	t.Helper()

	// Create organization if it doesn't exist
	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		org = models.Organization{Name: "Test Org", Active: true}
		org.ID = orgID
		if err := db.Create(&org).Error; err != nil {
			t.Fatalf("failed to create organization: %v", err)
		}
	}

	// Create group if it doesn't exist
	var group models.Group
	if err := db.Where("organization_id = ?", orgID).First(&group).Error; err != nil {
		group = models.Group{Name: "Test Group", OrganizationID: orgID, Active: true}
		if err := db.Create(&group).Error; err != nil {
			t.Fatalf("failed to create group: %v", err)
		}
	}

	// Create user if it doesn't exist
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		user = models.User{Name: "Test User", Email: "test@example.com", Password: "password", Active: true}
		user.ID = userID
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
	}

	// Add user to group with role
	userGroup := models.UserGroup{
		UserID:    userID,
		GroupID:   group.ID,
		Role:      role,
		CreatedBy: "test",
	}
	if err := db.Create(&userGroup).Error; err != nil {
		t.Fatalf("failed to add user to group: %v", err)
	}
}

// assignSuperAdmin sets a user as superadmin
func assignSuperAdmin(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()

	// Create user if it doesn't exist
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		user = models.User{Name: "Superadmin", Email: "admin@example.com", Password: "password", Active: true, IsSuperAdmin: true}
		user.ID = userID
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("failed to create superadmin user: %v", err)
		}
	} else {
		user.IsSuperAdmin = true
		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("failed to update user to superadmin: %v", err)
		}
	}
}

func TestAuthorizationMiddleware_RequirePermission_Allowed(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignSuperAdmin(t, db, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignSuperAdmin(t, db, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1) // Admin, not superadmin
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1) // Only has access to org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 42) // Assign admin role for org 42
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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

// Tests for RequireGlobalPermission middleware

func TestAuthorizationMiddleware_RequireGlobalPermission_SuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignSuperAdmin(t, db, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_AdminCanCreateUsers(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1) // Admin in org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.POST("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionCreate),
		func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

	req, _ := http.NewRequest("POST", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_ManagerCannotCreateUsers(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1) // Manager in org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.POST("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionCreate),
		func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

	req, _ := http.NewRequest("POST", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_ManagerCanReadUsers(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1) // Manager in org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.GET("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_NoRole(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	// User 99 has no role - create the user but don't assign any role
	user := models.User{Name: "No Role User", Email: "norole@example.com", Password: "password", Active: true}
	user.ID = 99
	db.Create(&user)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(99)) // User with no roles
		c.Next()
	})
	r.GET("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_NoUserID(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	// No userID set
	r.GET("/users",
		middleware.RequireGlobalPermission(rbac.ResourceUsers, rbac.ActionRead),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_ManagerCannotUpdateOrganizations(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleManager, 1) // Manager in org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.PUT("/organizations/:orgId",
		middleware.RequireGlobalPermission(rbac.ResourceOrganizations, rbac.ActionUpdate),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("PUT", "/organizations/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for manager trying to update org, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthorizationMiddleware_RequireGlobalPermission_AdminCanUpdateOrganizations(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleAdmin, 1) // Admin in org 1
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.PUT("/organizations/:orgId",
		middleware.RequireGlobalPermission(rbac.ResourceOrganizations, rbac.ActionUpdate),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("PUT", "/organizations/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for admin updating org, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// Test member role
func TestAuthorizationMiddleware_RequirePermission_MemberCanRead(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleMember, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

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

func TestAuthorizationMiddleware_RequirePermission_MemberCannotCreate(t *testing.T) {
	db := setupTestDB(t)
	enforcer := setupTestEnforcer(t)
	assignRole(t, db, 1, models.RoleMember, 1)
	permissionService := setupTestPermissionService(t, db, enforcer)

	middleware := NewAuthorizationMiddleware(permissionService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	r.POST("/organizations/:orgId/employees",
		middleware.RequirePermission(rbac.ResourceEmployees, rbac.ActionCreate),
		func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "success"})
		})

	req, _ := http.NewRequest("POST", "/organizations/1/employees", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

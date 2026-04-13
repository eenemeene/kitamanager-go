package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// --- test types for generic helpers ---

type testNestedReq struct {
	Name string `json:"name" binding:"required"`
}

type testNestedResp struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// testAuditConfig returns a auditConfig with a real audit service backed by an in-memory DB.
func testAuditConfig(t *testing.T, resourceType, parentLabel string) auditConfig {
	t.Helper()
	db := setupTestDB(t)
	createTestSuperAdmin(t, db)
	return auditConfig{
		auditService: createAuditService(db),
		resourceType: resourceType,
		parentLabel:  parentLabel,
	}
}

// newTestContext creates a gin context with URL params and optional JSON body.
func newTestContext(method, path string, params gin.Params, body any) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	c.Params = params
	c.Request = req
	// Set user context for audit helpers
	c.Set(ctxkeys.UserID, uint(1))
	c.Set(ctxkeys.UserEmail, "test@test.com")
	return c, w
}

// --- handleOrgNestedCreate ---

func TestHandleOrgNestedCreate_Success(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, testNestedReq{Name: "test"})

	called := false
	handleOrgNestedCreate(c, "parentId", testAuditConfig(t, "test", "item"), func(_ context.Context, parentID, orgID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if orgID != 1 {
			t.Errorf("orgID = %d, want 1", orgID)
		}
		if parentID != 2 {
			t.Errorf("parentID = %d, want 2", parentID)
		}
		if req.Name != "test" {
			t.Errorf("req.Name = %q, want %q", req.Name, "test")
		}
		return &testNestedResp{ID: 10, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("createFn was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleOrgNestedCreate_InvalidOrgID(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/abc/items/2/sub", gin.Params{
		{Key: "orgId", Value: "abc"},
		{Key: "parentId", Value: "2"},
	}, testNestedReq{Name: "test"})

	handleOrgNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOrgNestedCreate_InvalidParentID(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/abc/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "abc"},
	}, testNestedReq{Name: "test"})

	handleOrgNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOrgNestedCreate_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "orgId", Value: "1"}, {Key: "parentId", Value: "2"}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{invalid`))
	c.Request.Header.Set("Content-Type", "application/json")

	handleOrgNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOrgNestedCreate_ServiceError(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, testNestedReq{Name: "test"})

	handleOrgNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		return nil, apperror.NotFound("item")
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- handleOrgNestedGet ---

func TestHandleOrgNestedGet_Success(t *testing.T) {
	c, w := newTestContext("GET", "/organizations/1/items/2/sub/3", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "3"},
	}, nil)

	called := false
	handleOrgNestedGet(c, "parentId", "subId", func(_ context.Context, nestedID, parentID, orgID uint) (*testNestedResp, error) {
		called = true
		if orgID != 1 || parentID != 2 || nestedID != 3 {
			t.Errorf("IDs = (%d, %d, %d), want (1, 2, 3)", orgID, parentID, nestedID)
		}
		return &testNestedResp{ID: 3, Name: "item"}, nil
	})

	if !called {
		t.Fatal("getFn was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleOrgNestedGet_InvalidNestedID(t *testing.T) {
	c, w := newTestContext("GET", "/organizations/1/items/2/sub/abc", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "abc"},
	}, nil)

	handleOrgNestedGet(c, "parentId", "subId", func(_ context.Context, _, _, _ uint) (*testNestedResp, error) {
		t.Fatal("getFn should not be called")
		return nil, nil
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOrgNestedGet_NotFound(t *testing.T) {
	c, w := newTestContext("GET", "/organizations/1/items/2/sub/999", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "999"},
	}, nil)

	handleOrgNestedGet(c, "parentId", "subId", func(_ context.Context, _, _, _ uint) (*testNestedResp, error) {
		return nil, apperror.NotFound("sub item")
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- handleOrgNestedUpdate ---

func TestHandleOrgNestedUpdate_Success(t *testing.T) {
	c, w := newTestContext("PUT", "/organizations/1/items/2/sub/3", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "3"},
	}, testNestedReq{Name: "updated"})

	called := false
	handleOrgNestedUpdate(c, "parentId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, parentID, orgID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if orgID != 1 || parentID != 2 || nestedID != 3 {
			t.Errorf("IDs = (%d, %d, %d), want (1, 2, 3)", orgID, parentID, nestedID)
		}
		return &testNestedResp{ID: nestedID, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("updateFn was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- handleOrgNestedDelete ---

func TestHandleOrgNestedDelete_Success(t *testing.T) {
	c, _ := newTestContext("DELETE", "/organizations/1/items/2/sub/3", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "3"},
	}, nil)

	called := false
	handleOrgNestedDelete(c, "parentId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, parentID, orgID uint) error {
		called = true
		if orgID != 1 || parentID != 2 || nestedID != 3 {
			t.Errorf("IDs = (%d, %d, %d), want (1, 2, 3)", orgID, parentID, nestedID)
		}
		return nil
	})

	if !called {
		t.Fatal("deleteFn was not called")
	}
	// c.Status() without a body doesn't flush to the recorder; check gin's internal status
	if c.Writer.Status() != http.StatusNoContent {
		t.Errorf("status = %d, want %d", c.Writer.Status(), http.StatusNoContent)
	}
}

func TestHandleOrgNestedDelete_ServiceError(t *testing.T) {
	c, w := newTestContext("DELETE", "/organizations/1/items/2/sub/3", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "3"},
	}, nil)

	handleOrgNestedDelete(c, "parentId", "subId", auditConfig{}, func(_ context.Context, _, _, _ uint) error {
		return apperror.NotFound("sub item")
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- handleOrgNestedList ---

func TestHandleOrgNestedList_Success(t *testing.T) {
	c, w := newTestContext("GET", "/organizations/1/items/2/sub?page=1&limit=10", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, nil)
	c.Request = httptest.NewRequest("GET", "/organizations/1/items/2/sub?page=1&limit=10", nil)

	called := false
	handleOrgNestedList(c, "parentId", func(_ context.Context, parentID, orgID uint, limit, offset int) ([]testNestedResp, int64, error) {
		called = true
		if orgID != 1 || parentID != 2 {
			t.Errorf("IDs = (%d, %d), want (1, 2)", orgID, parentID)
		}
		return []testNestedResp{{ID: 1, Name: "a"}}, 1, nil
	})

	if !called {
		t.Fatal("listFn was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result models.PaginatedResponse[testNestedResp]
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("data length = %d, want 1", len(result.Data))
	}
}

// --- handleOrgDeepNestedCreate ---

func TestHandleOrgDeepNestedCreate_Success(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/mid/3/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "midId", Value: "3"},
	}, testNestedReq{Name: "deep"})

	called := false
	handleOrgDeepNestedCreate(c, "parentId", "midId", testAuditConfig(t, "test", "item"), func(_ context.Context, midID, parentID, orgID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if orgID != 1 || parentID != 2 || midID != 3 {
			t.Errorf("IDs = (%d, %d, %d), want (1, 2, 3)", orgID, parentID, midID)
		}
		return &testNestedResp{ID: 10, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("createFn was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleOrgDeepNestedCreate_InvalidMidID(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/mid/abc/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "midId", Value: "abc"},
	}, testNestedReq{Name: "deep"})

	handleOrgDeepNestedCreate(c, "parentId", "midId", auditConfig{}, func(_ context.Context, _, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handleOrgDeepNestedDelete ---

func TestHandleOrgDeepNestedDelete_Success(t *testing.T) {
	c, _ := newTestContext("DELETE", "/organizations/1/items/2/mid/3/sub/4", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "midId", Value: "3"},
		{Key: "subId", Value: "4"},
	}, nil)

	called := false
	handleOrgDeepNestedDelete(c, "parentId", "midId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, midID, parentID, orgID uint) error {
		called = true
		if orgID != 1 || parentID != 2 || midID != 3 || nestedID != 4 {
			t.Errorf("IDs = (%d, %d, %d, %d), want (1, 2, 3, 4)", orgID, parentID, midID, nestedID)
		}
		return nil
	})

	if !called {
		t.Fatal("deleteFn was not called")
	}
	if c.Writer.Status() != http.StatusNoContent {
		t.Errorf("status = %d, want %d", c.Writer.Status(), http.StatusNoContent)
	}
}

// --- handleGlobalNestedCreate ---

func TestHandleGlobalNestedCreate_Success(t *testing.T) {
	c, w := newTestContext("POST", "/items/5/sub", gin.Params{
		{Key: "parentId", Value: "5"},
	}, testNestedReq{Name: "global"})

	called := false
	handleGlobalNestedCreate(c, "parentId", testAuditConfig(t, "test", "item"), func(_ context.Context, parentID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if parentID != 5 {
			t.Errorf("parentID = %d, want 5", parentID)
		}
		return &testNestedResp{ID: 20, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("createFn was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleGlobalNestedCreate_InvalidID(t *testing.T) {
	c, w := newTestContext("POST", "/items/abc/sub", gin.Params{
		{Key: "parentId", Value: "abc"},
	}, testNestedReq{Name: "global"})

	handleGlobalNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handleGlobalNestedDelete ---

func TestHandleGlobalNestedDelete_Success(t *testing.T) {
	c, _ := newTestContext("DELETE", "/items/5/sub/6", gin.Params{
		{Key: "parentId", Value: "5"},
		{Key: "subId", Value: "6"},
	}, nil)

	called := false
	handleGlobalNestedDelete(c, "parentId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, parentID uint) error {
		called = true
		if nestedID != 6 || parentID != 5 {
			t.Errorf("IDs = (%d, %d), want (6, 5)", nestedID, parentID)
		}
		return nil
	})

	if !called {
		t.Fatal("deleteFn was not called")
	}
	if c.Writer.Status() != http.StatusNoContent {
		t.Errorf("status = %d, want %d", c.Writer.Status(), http.StatusNoContent)
	}
}

// --- handleGlobalDeepNestedCreate ---

func TestHandleGlobalDeepNestedCreate_Success(t *testing.T) {
	c, w := newTestContext("POST", "/items/5/mid/6/sub", gin.Params{
		{Key: "parentId", Value: "5"},
		{Key: "midId", Value: "6"},
	}, testNestedReq{Name: "deep-global"})

	called := false
	handleGlobalDeepNestedCreate(c, "parentId", "midId", testAuditConfig(t, "test", "item"), func(_ context.Context, parentID, midID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if parentID != 5 || midID != 6 {
			t.Errorf("IDs = (%d, %d), want (5, 6)", parentID, midID)
		}
		return &testNestedResp{ID: 30, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("createFn was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

// --- handleGlobalDeepNestedDelete ---

func TestHandleGlobalDeepNestedDelete_Success(t *testing.T) {
	c, _ := newTestContext("DELETE", "/items/5/mid/6/sub/7", gin.Params{
		{Key: "parentId", Value: "5"},
		{Key: "midId", Value: "6"},
		{Key: "subId", Value: "7"},
	}, nil)

	called := false
	handleGlobalDeepNestedDelete(c, "parentId", "midId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, midID, parentID uint) error {
		called = true
		if nestedID != 7 || midID != 6 || parentID != 5 {
			t.Errorf("IDs = (%d, %d, %d), want (7, 6, 5)", nestedID, midID, parentID)
		}
		return nil
	})

	if !called {
		t.Fatal("deleteFn should be called")
	}
	if c.Writer.Status() != http.StatusNoContent {
		t.Errorf("status = %d, want %d", c.Writer.Status(), http.StatusNoContent)
	}
}

// --- Response body verification ---

func TestHandleOrgNestedCreate_ResponseBody(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, testNestedReq{Name: "verify-body"})

	handleOrgNestedCreate(c, "parentId", testAuditConfig(t, "test", "item"), func(_ context.Context, _, _ uint, req *testNestedReq) (*testNestedResp, error) {
		return &testNestedResp{ID: 42, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp testNestedResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.ID != 42 || resp.Name != "verify-body" {
		t.Errorf("resp = %+v, want {ID:42 Name:verify-body}", resp)
	}
}

// --- Error message propagation ---

func TestHandleOrgNestedDelete_ErrorMessagePreserved(t *testing.T) {
	c, w := newTestContext("DELETE", "/organizations/1/items/2/sub/3", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
		{Key: "subId", Value: "3"},
	}, nil)

	handleOrgNestedDelete(c, "parentId", "subId", auditConfig{}, func(_ context.Context, _, _, _ uint) error {
		return apperror.Conflict("period overlaps")
	})

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var errResp models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Message != "period overlaps" {
		t.Errorf("message = %q, want %q", errResp.Message, "period overlaps")
	}
}

// --- Validation binding error ---

func TestHandleOrgNestedCreate_MissingRequiredField(t *testing.T) {
	c, w := newTestContext("POST", "/organizations/1/items/2/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, map[string]string{}) // empty body, missing required "name"

	handleOrgNestedCreate(c, "parentId", auditConfig{}, func(_ context.Context, _, _ uint, _ *testNestedReq) (*testNestedResp, error) {
		t.Fatal("createFn should not be called for validation failure")
		return nil, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handleGlobalNestedUpdate ---

func TestHandleGlobalNestedUpdate_Success(t *testing.T) {
	c, w := newTestContext("PUT", "/items/5/sub/6", gin.Params{
		{Key: "parentId", Value: "5"},
		{Key: "subId", Value: "6"},
	}, testNestedReq{Name: "updated"})

	called := false
	handleGlobalNestedUpdate(c, "parentId", "subId", testAuditConfig(t, "test", "item"), func(_ context.Context, nestedID, parentID uint, req *testNestedReq) (*testNestedResp, error) {
		called = true
		if nestedID != 6 || parentID != 5 {
			t.Errorf("IDs = (%d, %d), want (6, 5)", nestedID, parentID)
		}
		return &testNestedResp{ID: nestedID, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	if !called {
		t.Fatal("updateFn was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Audit logging ---

func TestHandleOrgNestedCreate_WithAuditService(t *testing.T) {
	db := setupTestDB(t)
	createTestSuperAdmin(t, db)
	auditSvc := createAuditService(db)

	c, w := newTestContext("POST", "/organizations/1/items/2/sub", gin.Params{
		{Key: "orgId", Value: "1"},
		{Key: "parentId", Value: "2"},
	}, testNestedReq{Name: "audited"})

	audit := auditConfig{
		auditService: auditSvc,
		resourceType: "test_resource",
		parentLabel:  "item",
	}

	handleOrgNestedCreate(c, "parentId", audit, func(_ context.Context, _, _ uint, req *testNestedReq) (*testNestedResp, error) {
		return &testNestedResp{ID: 99, Name: req.Name}, nil
	}, func(r *testNestedResp) uint { return r.ID })

	// Verify the handler still succeeds with a real audit service (async log processing)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

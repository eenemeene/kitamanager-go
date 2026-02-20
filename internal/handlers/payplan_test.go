package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestPayPlanHandler_CRUD(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans", handler.List)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId", handler.Get)
	r.POST("/organizations/:orgId/pay-plans", handler.Create)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId", handler.Update)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId", handler.Delete)

	var createdID uint

	// Test Create
	t.Run("Create", func(t *testing.T) {
		body := models.PayPlanCreateRequest{Name: "TVöD-SuE"}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans", org.ID), body)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var result models.PayPlanResponse
		parseResponse(t, w, &result)
		if result.Name != "TVöD-SuE" {
			t.Errorf("expected name 'TVöD-SuE', got '%s'", result.Name)
		}
		if result.OrganizationID != org.ID {
			t.Errorf("expected org ID %d, got %d", org.ID, result.OrganizationID)
		}
		createdID = result.ID
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans", org.ID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.PayPlanResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 1 {
			t.Errorf("expected 1 payplan, got %d", len(response.Data))
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, createdID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.PayPlanDetailResponse
		parseResponse(t, w, &result)
		if result.Name != "TVöD-SuE" {
			t.Errorf("expected name 'TVöD-SuE', got '%s'", result.Name)
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		updName := "TVöD-VKA"
		body := models.PayPlanUpdateRequest{Name: &updName}
		w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, createdID), body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.PayPlanResponse
		parseResponse(t, w, &result)
		if result.Name != "TVöD-VKA" {
			t.Errorf("expected name 'TVöD-VKA', got '%s'", result.Name)
		}
	})

	// Test Get after Update
	t.Run("GetAfterUpdate", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, createdID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.PayPlanDetailResponse
		parseResponse(t, w, &result)
		if result.Name != "TVöD-VKA" {
			t.Errorf("expected name 'TVöD-VKA', got '%s'", result.Name)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, createdID), nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	// Test Get after Delete (should 404)
	t.Run("GetAfterDelete", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, createdID), nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestPayPlanHandler_OrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans", handler.List)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId", handler.Get)
	r.POST("/organizations/:orgId/pay-plans", handler.Create)

	// Create payplan in org1
	body := models.PayPlanCreateRequest{Name: "Org1 PayPlan"}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans", org1.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create payplan: %s", w.Body.String())
	}
	var created models.PayPlanResponse
	parseResponse(t, w, &created)

	// Try to access org1's payplan from org2 - should 404
	t.Run("CrossOrgAccessDenied", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d", org2.ID, created.ID), nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d for cross-org access, got %d", http.StatusNotFound, w.Code)
		}
	})

	// List payplans in org2 - should be empty
	t.Run("ListOtherOrgEmpty", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans", org2.ID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.PayPlanResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 0 {
			t.Errorf("expected 0 payplans in org2, got %d", len(response.Data))
		}
	})
}

func TestPayPlanHandler_Period_CRUD(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	// Create payplan directly in DB
	payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Test PayPlan"}
	db.Create(payplan)

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/organizations/:orgId/pay-plans/:payPlanId/periods", handler.CreatePeriod)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.GetPeriod)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.UpdatePeriod)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.DeletePeriod)

	var periodID uint

	// Test Create Period
	t.Run("CreatePeriod", func(t *testing.T) {
		toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		body := models.PayPlanPeriodCreateRequest{
			From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:          &toDate,
			WeeklyHours: 39.0,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods", org.ID, payplan.ID), body)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var result models.PayPlanPeriodResponse
		parseResponse(t, w, &result)
		if result.WeeklyHours != 39.0 {
			t.Errorf("expected weekly_hours 39.0, got %f", result.WeeklyHours)
		}
		expectedFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if !result.From.Equal(expectedFrom) {
			t.Errorf("expected from %v, got %v", expectedFrom, result.From)
		}
		periodID = result.ID
	})

	// Test Get Period
	t.Run("GetPeriod", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d", org.ID, payplan.ID, periodID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.PayPlanPeriodResponse
		parseResponse(t, w, &result)
		if result.WeeklyHours != 39.0 {
			t.Errorf("expected weekly_hours 39.0, got %f", result.WeeklyHours)
		}
	})

	// Test Update Period
	t.Run("UpdatePeriod", func(t *testing.T) {
		toDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		body := models.PayPlanPeriodUpdateRequest{
			From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:          &toDate,
			WeeklyHours: 40.0,
		}
		w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d", org.ID, payplan.ID, periodID), body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.PayPlanPeriodResponse
		parseResponse(t, w, &result)
		if result.WeeklyHours != 40.0 {
			t.Errorf("expected weekly_hours 40.0, got %f", result.WeeklyHours)
		}
	})

	// Test Delete Period
	t.Run("DeletePeriod", func(t *testing.T) {
		w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d", org.ID, payplan.ID, periodID), nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	// Test Get after Delete (should 404)
	t.Run("GetPeriodAfterDelete", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d", org.ID, payplan.ID, periodID), nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestPayPlanHandler_Entry_CRUD(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	// Create payplan and period directly in DB
	payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Test PayPlan"}
	db.Create(payplan)

	period := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period)

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries", handler.CreateEntry)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.GetEntry)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.UpdateEntry)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.DeleteEntry)

	var entryID uint

	// Test Create Entry
	t.Run("CreateEntry", func(t *testing.T) {
		body := models.PayPlanEntryCreateRequest{
			Grade:         "S8a",
			Step:          3,
			MonthlyAmount: 350000, // 3500.00 EUR in cents
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries", org.ID, payplan.ID, period.ID), body)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var result models.PayPlanEntryResponse
		parseResponse(t, w, &result)
		if result.Grade != "S8a" {
			t.Errorf("expected grade 'S8a', got '%s'", result.Grade)
		}
		if result.Step != 3 {
			t.Errorf("expected step 3, got %d", result.Step)
		}
		if result.MonthlyAmount != 350000 {
			t.Errorf("expected monthly_amount 350000, got %d", result.MonthlyAmount)
		}
		entryID = result.ID
	})

	// Test Get Entry
	t.Run("GetEntry", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, payplan.ID, period.ID, entryID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.PayPlanEntryResponse
		parseResponse(t, w, &result)
		if result.Grade != "S8a" {
			t.Errorf("expected grade 'S8a', got '%s'", result.Grade)
		}
	})

	// Test Update Entry
	t.Run("UpdateEntry", func(t *testing.T) {
		body := models.PayPlanEntryUpdateRequest{
			Grade:         "S8a",
			Step:          4,
			MonthlyAmount: 380000, // 3800.00 EUR in cents
		}
		w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, payplan.ID, period.ID, entryID), body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.PayPlanEntryResponse
		parseResponse(t, w, &result)
		if result.Step != 4 {
			t.Errorf("expected step 4, got %d", result.Step)
		}
		if result.MonthlyAmount != 380000 {
			t.Errorf("expected monthly_amount 380000, got %d", result.MonthlyAmount)
		}
	})

	// Test Delete Entry
	t.Run("DeleteEntry", func(t *testing.T) {
		w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, payplan.ID, period.ID, entryID), nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	// Test Get after Delete (should 404)
	t.Run("GetEntryAfterDelete", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/%d", org.ID, payplan.ID, period.ID, entryID), nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestPayPlanHandler_GetWithPeriodsAndEntries(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	// Create payplan
	payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Complete PayPlan"}
	db.Create(payplan)

	// Create period
	period := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period)

	// Create multiple entries
	entries := []models.PayPlanEntry{
		{PeriodID: period.ID, Grade: "S8a", Step: 1, MonthlyAmount: 300000},
		{PeriodID: period.ID, Grade: "S8a", Step: 2, MonthlyAmount: 320000},
		{PeriodID: period.ID, Grade: "S8a", Step: 3, MonthlyAmount: 350000},
		{PeriodID: period.ID, Grade: "S11b", Step: 1, MonthlyAmount: 380000},
	}
	for i := range entries {
		db.Create(&entries[i])
	}

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans/:payPlanId", handler.Get)

	t.Run("GetWithNestedData", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, payplan.ID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.PayPlanDetailResponse
		parseResponse(t, w, &result)

		if result.Name != "Complete PayPlan" {
			t.Errorf("expected name 'Complete PayPlan', got '%s'", result.Name)
		}
		if len(result.Periods) != 1 {
			t.Errorf("expected 1 period, got %d", len(result.Periods))
		}
		if len(result.Periods[0].Entries) != 4 {
			t.Errorf("expected 4 entries, got %d", len(result.Periods[0].Entries))
		}
	})
}

func TestPayPlanHandler_DeleteCascade(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	// Create payplan with period and entries
	payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Cascade Test"}
	db.Create(payplan)

	period := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period)

	entry := &models.PayPlanEntry{PeriodID: period.ID, Grade: "S8a", Step: 1, MonthlyAmount: 300000}
	db.Create(entry)

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId", handler.Delete)

	t.Run("DeletePayPlanCascades", func(t *testing.T) {
		w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d", org.ID, payplan.ID), nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		// Verify period was deleted
		var periodCount int64
		db.Model(&models.PayPlanPeriod{}).Where("id = ?", period.ID).Count(&periodCount)
		if periodCount != 0 {
			t.Error("period should have been deleted")
		}

		// Verify entry was deleted
		var entryCount int64
		db.Model(&models.PayPlanEntry{}).Where("id = ?", entry.ID).Count(&entryCount)
		if entryCount != 0 {
			t.Error("entry should have been deleted")
		}
	})
}

func TestPayPlanHandler_InvalidRequests(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/organizations/:orgId/pay-plans", handler.Create)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId", handler.Get)
	r.POST("/organizations/:orgId/pay-plans/:payPlanId/periods", handler.CreatePeriod)

	t.Run("CreateWithEmptyName", func(t *testing.T) {
		body := models.PayPlanCreateRequest{Name: ""}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans", org.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for empty name, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/99999", org.ID), nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d for non-existent, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("CreatePeriodInvalidDate", func(t *testing.T) {
		// First create a payplan
		payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
		db.Create(payplan)

		body := map[string]interface{}{
			"from":         "not-a-date",
			"weekly_hours": 39.0,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods", org.ID, payplan.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for invalid date, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("CreatePeriodZeroWeeklyHours", func(t *testing.T) {
		payplan := &models.PayPlan{OrganizationID: org.ID, Name: "Test2"}
		db.Create(payplan)

		body := models.PayPlanPeriodCreateRequest{
			From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			WeeklyHours: 0, // Invalid - should be > 0
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods", org.ID, payplan.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for zero weekly hours, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestPayPlanHandler_MultiplePayPlansPerOrg(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans", handler.List)
	r.POST("/organizations/:orgId/pay-plans", handler.Create)

	// Create multiple payplans
	names := []string{"TVöD-SuE", "TVöD-VKA", "AVR-DD"}
	for _, name := range names {
		body := models.PayPlanCreateRequest{Name: name}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans", org.ID), body)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create payplan %s: %s", name, w.Body.String())
		}
	}

	t.Run("ListAllPayPlans", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans", org.ID), nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.PayPlanResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 3 {
			t.Errorf("expected 3 payplans, got %d", len(response.Data))
		}
		if response.Total != 3 {
			t.Errorf("expected total 3, got %d", response.Total)
		}
	})
}

// setupPayPlanTestFull creates a full payplan test setup with all routes registered.
func setupPayPlanTestFull(t *testing.T) (*models.Organization, *PayPlanHandler, *gin.Engine, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	payplanStore := store.NewPayPlanStore(db)
	svc := service.NewPayPlanService(payplanStore)
	handler := NewPayPlanHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/pay-plans", handler.List)
	r.POST("/organizations/:orgId/pay-plans", handler.Create)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId", handler.Get)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId", handler.Update)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId", handler.Delete)
	r.POST("/organizations/:orgId/pay-plans/:payPlanId/periods", handler.CreatePeriod)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.GetPeriod)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.UpdatePeriod)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId", handler.DeletePeriod)
	r.POST("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries", handler.CreateEntry)
	r.GET("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.GetEntry)
	r.PUT("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.UpdateEntry)
	r.DELETE("/organizations/:orgId/pay-plans/:payPlanId/periods/:periodId/entries/:entryId", handler.DeleteEntry)

	return org, handler, r, db
}

func TestPayPlanHandler_List_InvalidOrgID(t *testing.T) {
	_, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "GET", "/organizations/abc/pay-plans", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_Update_InvalidID(t *testing.T) {
	_, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "PUT", "/organizations/abc/pay-plans/1", map[string]string{"name": "x"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_Update_InvalidBody(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequestRaw(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1", org.ID), "not json")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_Update_NotFound(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	notFoundName := "x"
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/9999", org.ID), models.PayPlanUpdateRequest{Name: &notFoundName})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_Delete_NotFound(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/9999", org.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_Delete_InvalidID(t *testing.T) {
	_, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "DELETE", "/organizations/abc/pay-plans/1", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_GetPeriod_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_GetPeriod_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/9999", org.ID, pp.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdatePeriod_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc", org.ID), map[string]string{"from": "2024-01-01"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdatePeriod_InvalidBody(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequestRaw(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1", org.ID), "not json")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdatePeriod_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/9999", org.ID, pp.ID), models.PayPlanPeriodUpdateRequest{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), WeeklyHours: 39})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_DeletePeriod_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_DeletePeriod_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/9999", org.ID, pp.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_CreateEntry_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc/entries", org.ID), map[string]interface{}{"level": 1, "step": 1, "monthly_salary": 100000})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_CreateEntry_InvalidBody(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequestRaw(r, "POST", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1/entries", org.ID), "not json")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_GetEntry_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc/entries/1", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_GetEntry_InvalidEntryID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1/entries/abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_GetEntry_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)
	period := &models.PayPlanPeriod{PayPlanID: pp.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}, WeeklyHours: 39}
	db.Create(period)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/9999", org.ID, pp.ID, period.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdateEntry_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc/entries/1", org.ID), map[string]interface{}{"monthly_salary": 100000})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdateEntry_InvalidEntryID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1/entries/abc", org.ID), map[string]interface{}{"monthly_salary": 100000})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdateEntry_InvalidBody(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequestRaw(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1/entries/1", org.ID), "not json")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_UpdateEntry_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)
	period := &models.PayPlanPeriod{PayPlanID: pp.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}, WeeklyHours: 39}
	db.Create(period)

	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/9999", org.ID, pp.ID, period.ID), models.PayPlanEntryUpdateRequest{Grade: "S8a", Step: 1, MonthlyAmount: 100000})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_DeleteEntry_InvalidPeriodID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/abc/entries/1", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_DeleteEntry_InvalidEntryID(t *testing.T) {
	org, _, r, _ := setupPayPlanTestFull(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/1/periods/1/entries/abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPayPlanHandler_DeleteEntry_NotFound(t *testing.T) {
	org, _, r, db := setupPayPlanTestFull(t)

	pp := &models.PayPlan{OrganizationID: org.ID, Name: "Test"}
	db.Create(pp)
	period := &models.PayPlanPeriod{PayPlanID: pp.ID, Period: models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}, WeeklyHours: 39}
	db.Create(period)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/pay-plans/%d/periods/%d/entries/9999", org.ID, pp.ID, period.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

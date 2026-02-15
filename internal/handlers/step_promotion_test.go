package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestStepPromotionHandler_GetStepPromotions(t *testing.T) {
	db := setupTestDB(t)
	payPlanStore := store.NewPayPlanStore(db)
	employeeStore := store.NewEmployeeStore(db)
	svc := service.NewStepPromotionService(payPlanStore, employeeStore)
	handler := NewStepPromotionHandler(svc)

	org := createTestOrganization(t, db, "Test Org")

	// Create payplan with entries
	payPlan := createTestPayPlan(t, db, "TVoeD-SuE", org.ID)
	period := &models.PayPlanPeriod{
		PayPlanID:   payPlan.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period)

	step0, step1, step3 := 0, 1, 3
	db.Create(&models.PayPlanEntry{PeriodID: period.ID, Grade: "S8a", Step: 1, MonthlyAmount: 314847, StepMinYears: &step0})
	db.Create(&models.PayPlanEntry{PeriodID: period.ID, Grade: "S8a", Step: 2, MonthlyAmount: 329947, StepMinYears: &step1})
	db.Create(&models.PayPlanEntry{PeriodID: period.ID, Grade: "S8a", Step: 3, MonthlyAmount: 350089, StepMinYears: &step3})

	// Create employee eligible for promotion
	emp := &models.Employee{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Anna", LastName: "Mueller", Gender: "female", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(emp)

	from := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Create(&models.EmployeeContract{
		EmployeeID:    emp.ID,
		BaseContract:  models.BaseContract{SectionID: 1, Period: models.Period{From: from}},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          2,
		WeeklyHours:   39.0,
		PayPlanID:     payPlan.ID,
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/step-promotions", handler.GetStepPromotions)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/step-promotions", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.StepPromotionsResponse
	parseResponse(t, w, &result)

	if len(result.Promotions) != 1 {
		t.Fatalf("expected 1 promotion, got %d", len(result.Promotions))
	}
	if result.Promotions[0].CurrentStep != 2 {
		t.Errorf("expected current step 2, got %d", result.Promotions[0].CurrentStep)
	}
	if result.Promotions[0].EligibleStep != 3 {
		t.Errorf("expected eligible step 3, got %d", result.Promotions[0].EligibleStep)
	}
}

func TestStepPromotionHandler_GetStepPromotions_WithDate(t *testing.T) {
	db := setupTestDB(t)
	payPlanStore := store.NewPayPlanStore(db)
	employeeStore := store.NewEmployeeStore(db)
	svc := service.NewStepPromotionService(payPlanStore, employeeStore)
	handler := NewStepPromotionHandler(svc)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/step-promotions", handler.GetStepPromotions)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/step-promotions?date=2025-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.StepPromotionsResponse
	parseResponse(t, w, &result)

	if result.Date != "2025-06-15" {
		t.Errorf("expected date 2025-06-15, got %s", result.Date)
	}
}

func TestStepPromotionHandler_GetStepPromotions_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	payPlanStore := store.NewPayPlanStore(db)
	employeeStore := store.NewEmployeeStore(db)
	svc := service.NewStepPromotionService(payPlanStore, employeeStore)
	handler := NewStepPromotionHandler(svc)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/step-promotions", handler.GetStepPromotions)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/employees/step-promotions?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestStepPromotionHandler_GetStepPromotions_InvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	payPlanStore := store.NewPayPlanStore(db)
	employeeStore := store.NewEmployeeStore(db)
	svc := service.NewStepPromotionService(payPlanStore, employeeStore)
	handler := NewStepPromotionHandler(svc)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/employees/step-promotions", handler.GetStepPromotions)

	w := performRequest(r, "GET", "/organizations/invalid/employees/step-promotions", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

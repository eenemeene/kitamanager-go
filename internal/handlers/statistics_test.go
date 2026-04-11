package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestStatisticsHandler_GetStaffingHours_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours", handler.GetStaffingHours)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours?from=2024-01-01&to=2024-03-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.StaffingHoursResponse
	parseResponse(t, w, &response)
	if len(response.DataPoints) == 0 {
		t.Error("expected data points")
	}
}

func TestStatisticsHandler_GetStaffingHours_WithQueryParams(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours", handler.GetStaffingHours)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours?from=2024-06-01&to=2024-09-01&section_id=1", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.StaffingHoursResponse
	parseResponse(t, w, &response)
	// With custom from/to spanning 4 months (Jun, Jul, Aug, Sep), expect 4 data points
	if len(response.DataPoints) != 4 {
		t.Errorf("expected 4 data points, got %d", len(response.DataPoints))
	}
}

func TestStatisticsHandler_GetStaffingHours_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours", handler.GetStaffingHours)

	w := performRequest(r, "GET", "/organizations/abc/statistics/staffing-hours", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetStaffingHours_InvalidDates(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours", handler.GetStaffingHours)

	// Invalid from date
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours?from=not-a-date", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid from date, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid to date
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours?to=2024-13-99", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid to date, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetEmployeeStaffingHours tests ---

func TestStatisticsHandler_GetEmployeeStaffingHours_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours/employees", handler.GetEmployeeStaffingHours)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours/employees?from=2024-01-01&to=2024-03-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.EmployeeStaffingHoursResponse
	parseResponse(t, w, &response)
	if response.Dates == nil {
		t.Error("expected dates field to be present")
	}
}

func TestStatisticsHandler_GetEmployeeStaffingHours_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours/employees", handler.GetEmployeeStaffingHours)

	w := performRequest(r, "GET", "/organizations/abc/statistics/staffing-hours/employees", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetEmployeeStaffingHours_InvalidDates(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours/employees", handler.GetEmployeeStaffingHours)

	// Invalid from date
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours/employees?from=not-a-date", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid from date, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid to date
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours/employees?to=2024-13-99", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid to date, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetEmployeeStaffingHours_InvalidSectionId(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/staffing-hours/employees", handler.GetEmployeeStaffingHours)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/staffing-hours/employees?section_id=abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetOccupancy tests ---

func TestStatisticsHandler_GetOccupancy_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/occupancy", handler.GetOccupancy)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/occupancy?from=2024-01-01&to=2024-03-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.OccupancyResponse
	parseResponse(t, w, &response)
	if len(response.DataPoints) == 0 {
		t.Error("expected data points")
	}
}

func TestStatisticsHandler_GetOccupancy_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/occupancy", handler.GetOccupancy)

	w := performRequest(r, "GET", "/organizations/abc/statistics/occupancy", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetOccupancy_InvalidDates(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/occupancy", handler.GetOccupancy)

	// Invalid from date
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/occupancy?from=not-a-date", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid from date, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid to date
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/occupancy?to=2024-13-99", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid to date, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetOccupancy_InvalidSectionId(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/occupancy", handler.GetOccupancy)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/occupancy?section_id=abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetFinancials tests ---

func TestStatisticsHandler_GetFinancials_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/financials?from=2024-01-01&to=2024-03-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.FinancialResponse
	parseResponse(t, w, &response)
	if len(response.DataPoints) == 0 {
		t.Error("expected data points")
	}
}

func TestStatisticsHandler_GetFinancials_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", "/organizations/abc/statistics/financials", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetFinancials_InvalidDates(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	// Invalid from date
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/financials?from=not-a-date", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid from date, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid to date
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/financials?to=2024-13-99", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid to date, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatisticsHandler_GetFinancials_WithQueryParams(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc, createChildService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/financials?from=2024-06-01&to=2024-09-01", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.FinancialResponse
	parseResponse(t, w, &response)
	// With custom from/to spanning 4 months (Jun, Jul, Aug, Sep), expect 4 data points
	if len(response.DataPoints) != 4 {
		t.Errorf("expected 4 data points, got %d", len(response.DataPoints))
	}
}

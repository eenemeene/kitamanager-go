package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestStatisticsHandler_GetStaffingHours_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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
	handler := NewStatisticsHandler(svc)

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

// Financials are organization-wide. section_id was accepted here once and
// produced a figure that charged every fixed budget item in full to each
// section; the parameter is gone from the documented API and an explicit
// rejection keeps a caller who still sends it from quietly receiving
// organization-wide numbers under a section heading.
func TestStatisticsHandler_GetFinancials_RejectsSectionID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", fmt.Sprintf(
		"/organizations/%d/statistics/financials?from=2024-01-01&to=2024-03-01&section_id=1", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for section_id on financials, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "section_id") {
		t.Errorf("error body should name the rejected parameter; got %s", w.Body.String())
	}
}

// The rejection must key on the parameter being present, not on it parsing —
// a caller sending garbage deserves the same "not supported" answer rather
// than a misleading "must be a positive integer".
func TestStatisticsHandler_GetFinancials_RejectsNonNumericSectionID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", fmt.Sprintf(
		"/organizations/%d/statistics/financials?section_id=abc", org.ID), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not supported") {
		t.Errorf("expected the not-supported message, got %s", w.Body.String())
	}
}

// An empty section_id= is indistinguishable from omitting it, and rejecting
// it would break a caller that builds query strings from optional fields.
func TestStatisticsHandler_GetFinancials_EmptySectionIDIsIgnored(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	svc := createStatisticsService(db)
	handler := NewStatisticsHandler(svc)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/financials", handler.GetFinancials)

	w := performRequest(r, "GET", fmt.Sprintf(
		"/organizations/%d/statistics/financials?from=2024-01-01&to=2024-03-01&section_id=", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for an empty section_id, got %d: %s", w.Code, w.Body.String())
	}
}

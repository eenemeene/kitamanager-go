package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// setupAttendanceTest sets up the common test fixtures for attendance tests.
func setupAttendanceTest(t *testing.T) (*models.Organization, *models.Child, *ChildAttendanceHandler, *gin.Engine, *gorm.DB) {
	t.Helper()

	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Kita")

	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Emma",
			LastName:       "Schmidt",
			Gender:         "female",
			Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)

	attendanceService := createAttendanceService(db)
	auditService := createAuditService(db)
	handler := NewChildAttendanceHandler(attendanceService, auditService)

	r := setupTestRouter()
	// Register non-parameterized routes BEFORE parameterized ones to avoid gin routing conflicts
	r.GET("/organizations/:orgId/children/attendance", handler.ListByDate)
	r.GET("/organizations/:orgId/children/attendance/summary", handler.GetDailySummary)
	r.POST("/organizations/:orgId/children/:id/attendance", handler.Create)
	r.GET("/organizations/:orgId/children/:id/attendance", handler.ListByChild)
	r.GET("/organizations/:orgId/children/:id/attendance/:attendanceId", handler.Get)
	r.PUT("/organizations/:orgId/children/:id/attendance/:attendanceId", handler.Update)
	r.DELETE("/organizations/:orgId/children/:id/attendance/:attendanceId", handler.Delete)

	return org, child, handler, r, db
}

func TestChildAttendanceHandler_Create(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp models.ChildAttendanceResponse
	parseResponse(t, w, &resp)
	if resp.Status != "present" {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
	if resp.ChildID != child.ID {
		t.Errorf("expected child_id %d, got %d", child.ID, resp.ChildID)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, resp.OrganizationID)
	}
	if resp.Date != "2024-06-15" {
		t.Errorf("expected date '2024-06-15', got '%s'", resp.Date)
	}
}

func TestChildAttendanceHandler_Create_InvalidBody(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Empty body - status is required
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), map[string]interface{}{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Create_InvalidOrgID(t *testing.T) {
	_, _, _, r, _ := setupAttendanceTest(t)

	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", "/organizations/abc/children/1/attendance", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Create_ChildNotFound(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/9999/attendance", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Create_DuplicateDate(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}

	// First create should succeed
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create failed: status %d: %s", w.Code, w.Body.String())
	}

	// Second create on the same date should return 409
	w = performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)
	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Get(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create a record first
	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d: %s", w.Code, w.Body.String())
	}

	var created models.ChildAttendanceResponse
	parseResponse(t, w, &created)

	// GET the record
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance/%d", org.ID, child.ID, created.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.ChildAttendanceResponse
	parseResponse(t, w, &resp)
	if resp.ID != created.ID {
		t.Errorf("expected id %d, got %d", created.ID, resp.ID)
	}
	if resp.Status != "present" {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
}

func TestChildAttendanceHandler_Get_NotFound(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance/9999", org.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Get_InvalidID(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance/abc", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Get_InvalidOrgID(t *testing.T) {
	_, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/abc/children/%d/attendance/1", child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Get_InvalidChildID(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/abc/attendance/1", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Update(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create a record first
	createBody := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d: %s", w.Code, w.Body.String())
	}

	var created models.ChildAttendanceResponse
	parseResponse(t, w, &created)

	// Update the record
	newStatus := "sick"
	updateBody := models.ChildAttendanceUpdateRequest{
		Status: &newStatus,
	}
	w = performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/attendance/%d", org.ID, child.ID, created.ID), updateBody)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.ChildAttendanceResponse
	parseResponse(t, w, &resp)
	if resp.Status != "sick" {
		t.Errorf("expected status 'sick', got '%s'", resp.Status)
	}
}

func TestChildAttendanceHandler_Update_InvalidOrgID(t *testing.T) {
	_, child, _, r, _ := setupAttendanceTest(t)

	newStatus := "sick"
	updateBody := models.ChildAttendanceUpdateRequest{
		Status: &newStatus,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/abc/children/%d/attendance/1", child.ID), updateBody)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Update_InvalidChildID(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	newStatus := "sick"
	updateBody := models.ChildAttendanceUpdateRequest{
		Status: &newStatus,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/abc/attendance/1", org.ID), updateBody)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Update_InvalidAttendanceID(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	newStatus := "sick"
	updateBody := models.ChildAttendanceUpdateRequest{
		Status: &newStatus,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/attendance/abc", org.ID, child.ID), updateBody)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Update_InvalidBody(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create a record first
	createBody := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d: %s", w.Code, w.Body.String())
	}

	var created models.ChildAttendanceResponse
	parseResponse(t, w, &created)

	// Send invalid JSON body
	w = performRequestRaw(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/attendance/%d", org.ID, child.ID, created.ID), "not json")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Update_NotFound(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	newStatus := "sick"
	updateBody := models.ChildAttendanceUpdateRequest{
		Status: &newStatus,
	}
	w := performRequest(r, "PUT", fmt.Sprintf("/organizations/%d/children/%d/attendance/9999", org.ID, child.ID), updateBody)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Delete(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create a record first
	createBody := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d: %s", w.Code, w.Body.String())
	}

	var created models.ChildAttendanceResponse
	parseResponse(t, w, &created)

	// Delete the record
	w = performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/attendance/%d", org.ID, child.ID, created.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify it's gone
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance/%d", org.ID, child.ID, created.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d after delete, got %d", http.StatusNotFound, w.Code)
	}
}

func TestChildAttendanceHandler_Delete_InvalidOrgID(t *testing.T) {
	_, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/abc/children/%d/attendance/1", child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Delete_InvalidChildID(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/abc/attendance/1", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Delete_InvalidAttendanceID(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/attendance/abc", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_Delete_NotFound(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/attendance/9999", org.ID, child.ID), nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByChild(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create two records
	for _, date := range []string{"2024-06-15", "2024-06-16"} {
		body := models.ChildAttendanceCreateRequest{
			Date:   date,
			Status: "present",
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed for date %s: status %d: %s", date, w.Code, w.Body.String())
		}
	}

	// List with date range
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance?from=2024-01-01&to=2024-12-31", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.ChildAttendanceResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 2 {
		t.Errorf("expected 2 records, got %d", len(response.Data))
	}
}

func TestChildAttendanceHandler_ListByChild_MissingFrom(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance?to=2024-12-31", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByChild_MissingTo(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance?from=2024-01-01", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByChild_InvalidDate(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance?from=not-a-date&to=2024-12-31", org.ID, child.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByChild_Empty(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Query a date range with no records
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/attendance?from=2023-01-01&to=2023-01-31", org.ID, child.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.ChildAttendanceResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 0 {
		t.Errorf("expected 0 records, got %d", len(response.Data))
	}
}

func TestChildAttendanceHandler_ListByDate(t *testing.T) {
	org, child, _, r, _ := setupAttendanceTest(t)

	// Create a record
	body := models.ChildAttendanceCreateRequest{
		Date:   "2024-06-15",
		Status: "present",
	}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, child.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d: %s", w.Code, w.Body.String())
	}

	// List by date
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance?date=2024-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.ChildAttendanceResponse]
	parseResponse(t, w, &response)
	if len(response.Data) != 1 {
		t.Errorf("expected 1 record, got %d", len(response.Data))
	}
}

func TestChildAttendanceHandler_ListByDate_InvalidOrgID(t *testing.T) {
	_, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", "/organizations/abc/children/attendance?date=2024-06-15", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByDate_InvalidDate(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_ListByDate_DefaultDate(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	// GET without date param should use today and return 200
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.PaginatedResponse[models.ChildAttendanceResponse]
	parseResponse(t, w, &response)
	// Should return empty data for today (no records created for today)
	if response.Data == nil {
		t.Errorf("expected non-nil data array")
	}
}

func TestChildAttendanceHandler_GetDailySummary(t *testing.T) {
	org, child, _, r, db := setupAttendanceTest(t)

	// Create a second child
	child2 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Max",
			LastName:       "Mueller",
			Gender:         "male",
			Birthdate:      time.Date(2019, 7, 20, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child2)

	// Create a third child
	child3 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Lena",
			LastName:       "Fischer",
			Gender:         "female",
			Birthdate:      time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child3)

	// Create records with different statuses
	statuses := []struct {
		childID uint
		status  string
	}{
		{child.ID, "present"},
		{child2.ID, "sick"},
		{child3.ID, "absent"},
	}

	for _, s := range statuses {
		body := models.ChildAttendanceCreateRequest{
			Date:   "2024-06-15",
			Status: s.status,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/attendance", org.ID, s.childID), body)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed for child %d: status %d: %s", s.childID, w.Code, w.Body.String())
		}
	}

	// Get daily summary
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance/summary?date=2024-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var summary models.ChildAttendanceDailySummaryResponse
	parseResponse(t, w, &summary)
	if summary.TotalChildren != 3 {
		t.Errorf("expected total_children 3, got %d", summary.TotalChildren)
	}
	if summary.Present != 1 {
		t.Errorf("expected present 1, got %d", summary.Present)
	}
	if summary.Sick != 1 {
		t.Errorf("expected sick 1, got %d", summary.Sick)
	}
	if summary.Absent != 1 {
		t.Errorf("expected absent 1, got %d", summary.Absent)
	}
	if summary.Vacation != 0 {
		t.Errorf("expected vacation 0, got %d", summary.Vacation)
	}
}

func TestChildAttendanceHandler_GetDailySummary_InvalidOrgID(t *testing.T) {
	_, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", "/organizations/abc/children/attendance/summary?date=2024-06-15", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_GetDailySummary_InvalidDate(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance/summary?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestChildAttendanceHandler_GetDailySummary_Empty(t *testing.T) {
	org, _, _, r, _ := setupAttendanceTest(t)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/attendance/summary?date=2023-01-01", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var summary models.ChildAttendanceDailySummaryResponse
	parseResponse(t, w, &summary)
	if summary.TotalChildren != 0 {
		t.Errorf("expected total_children 0, got %d", summary.TotalChildren)
	}
	if summary.Present != 0 {
		t.Errorf("expected present 0, got %d", summary.Present)
	}
	if summary.Absent != 0 {
		t.Errorf("expected absent 0, got %d", summary.Absent)
	}
	if summary.Sick != 0 {
		t.Errorf("expected sick 0, got %d", summary.Sick)
	}
	if summary.Vacation != 0 {
		t.Errorf("expected vacation 0, got %d", summary.Vacation)
	}
}

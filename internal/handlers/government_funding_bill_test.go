package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func createGovBillService(db *gorm.DB) *service.GovernmentFundingBillService {
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	return service.NewGovernmentFundingBillService(childStore, billPeriodStore)
}

func createGovBillHandler(db *gorm.DB) *GovernmentFundingBillHandler {
	svc := createGovBillService(db)
	return NewGovernmentFundingBillHandler(svc, createAuditService(db))
}

func setupBillRouter(db *gorm.DB) (*gin.Engine, *GovernmentFundingBillHandler) {
	return setupBillRouterWithUser(db, 1)
}

func setupBillRouterWithUser(db *gorm.DB, userID uint) (*gin.Engine, *GovernmentFundingBillHandler) {
	handler := createGovBillHandler(db)
	r := setupTestRouterWithUser(userID)
	org := r.Group("/organizations/:orgId/government-funding-bills")
	{
		org.GET("", handler.List)
		org.GET("/:billId", handler.Get)
		org.POST("", handler.UploadISBJ)
		org.DELETE("/:billId", handler.Delete)
	}
	return r, handler
}

func createBillPeriodInDB(t *testing.T, db *gorm.DB, orgID, userID uint, facilityName string, month time.Month) *models.GovernmentFundingBillPeriod {
	t.Helper()
	to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
	period := &models.GovernmentFundingBillPeriod{
		OrganizationID:    orgID,
		Period:            models.Period{From: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:          fmt.Sprintf("abrechnung_%02d-25.xlsx", month),
		FileSha256:        fmt.Sprintf("hash_%02d", month),
		FacilityName:      facilityName,
		FacilityTotal:     300000,
		ContractBooking:   280000,
		CorrectionBooking: 20000,
		CreatedBy:         userID,
		Children: []models.GovernmentFundingBillChild{
			{
				VoucherNumber: fmt.Sprintf("GB-0000000000%d-01", month),
				ChildName:     "Kind, Test",
				BirthDate:     "01.20",
				District:      1,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: 150000},
					{Key: "ndh", Value: "ndh", Amount: 5000},
				},
			},
		},
	}
	if err := db.Create(period).Error; err != nil {
		t.Fatalf("setup: create bill period error = %v", err)
	}
	return period
}

func TestGovernmentFundingBillHandler_List(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billhandler1@example.com", "password")

	createBillPeriodInDB(t, db, org.ID, user.ID, "Kita A", 10)
	createBillPeriodInDB(t, db, org.ID, user.ID, "Kita B", 11)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Data  []models.GovernmentFundingBillPeriodListResponse `json:"data"`
		Total int64                                            `json:"total"`
	}
	parseResponse(t, w, &response)

	if response.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(response.Data))
	}
}

func TestGovernmentFundingBillHandler_ListEmpty(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Data  []models.GovernmentFundingBillPeriodListResponse `json:"data"`
		Total int64                                            `json:"total"`
	}
	parseResponse(t, w, &response)

	if response.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Total)
	}
}

func TestGovernmentFundingBillHandler_ListPagination(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billhandler2@example.com", "password")

	for m := time.Month(1); m <= 5; m++ {
		createBillPeriodInDB(t, db, org.ID, user.ID, "Kita", m)
	}

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?page=1&limit=2", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Data  []models.GovernmentFundingBillPeriodListResponse `json:"data"`
		Total int64                                            `json:"total"`
	}
	parseResponse(t, w, &response)

	if response.Total != 5 {
		t.Errorf("expected total 5, got %d", response.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 items (page 1, limit 2), got %d", len(response.Data))
	}
}

func TestGovernmentFundingBillHandler_ListOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billhandler3@example.com", "password")

	createBillPeriodInDB(t, db, org1.ID, user.ID, "Org1 Kita", 10)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills", org2.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Total int64 `json:"total"`
	}
	parseResponse(t, w, &response)
	if response.Total != 0 {
		t.Errorf("org2 should see 0 bills, got %d", response.Total)
	}
}

func TestGovernmentFundingBillHandler_ListInvalidOrgID(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)

	w := performRequest(r, "GET", "/organizations/invalid/government-funding-bills", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGovernmentFundingBillHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billhandler4@example.com", "password")

	period := createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Detail", 11)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org.ID, period.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result models.GovernmentFundingBillPeriodResponse
	parseResponse(t, w, &result)

	if result.ID != period.ID {
		t.Errorf("expected ID %d, got %d", period.ID, result.ID)
	}
	if result.FacilityName != "Kita Detail" {
		t.Errorf("expected facility name 'Kita Detail', got %q", result.FacilityName)
	}
	if result.ChildrenCount != 1 {
		t.Errorf("expected children count 1, got %d", result.ChildrenCount)
	}
}

func TestGovernmentFundingBillHandler_GetNotFound(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/99999", org.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_GetWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billhandler5@example.com", "password")

	period := createBillPeriodInDB(t, db, org1.ID, user.ID, "Org1 Kita", 11)

	// Try to get from org2
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org2.ID, period.ID), nil)
	// Should fail (either 404 or 500 depending on error handling)
	if w.Code == http.StatusOK {
		t.Error("expected non-200 status when accessing bill from wrong org")
	}
}

func TestGovernmentFundingBillHandler_GetInvalidIDs(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)

	tests := []struct {
		name string
		path string
	}{
		{"invalid orgId", "/organizations/abc/government-funding-bills/1"},
		{"invalid id", "/organizations/1/government-funding-bills/xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", tt.path, nil)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestGovernmentFundingBillHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billhandler6@example.com", "password")

	period := createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Delete", 11)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org.ID, period.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org.ID, period.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestGovernmentFundingBillHandler_DeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/government-funding-bills/99999", org.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_DeleteWrongOrg(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "billhandler7@example.com", "password")

	period := createBillPeriodInDB(t, db, org1.ID, user.ID, "Protected Kita", 11)

	// Try to delete from org2
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org2.ID, period.ID), nil)
	if w.Code == http.StatusNoContent {
		t.Error("expected non-204 when deleting from wrong org")
	}

	// Verify still exists
	w = performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org1.ID, period.ID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("period should still exist, got status %d", w.Code)
	}
}

func TestGovernmentFundingBillHandler_UploadISBJ(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Upload User", "upload@example.com", "password")
	r, _ := setupBillRouterWithUser(db, user.ID)

	// Read the real test file
	testFile := "../isbj/testdata/Abrechnung_11-25_0770_anonymized.xlsx"
	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Skipf("test file not available: %v", err)
	}

	// Build multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "Abrechnung_11-25.xlsx")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var result models.GovernmentFundingBillResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.ID == 0 {
		t.Error("expected non-zero ID after upload")
	}
	if result.FacilityName == "" {
		t.Error("expected non-empty facility name")
	}
	if result.ChildrenCount == 0 {
		t.Error("expected non-zero children count")
	}

	// Verify it was persisted - fetch via GET
	w2 := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org.ID, result.ID), nil)
	if w2.Code != http.StatusOK {
		t.Errorf("expected GET to succeed after upload, got %d", w2.Code)
	}
}

func TestGovernmentFundingBillHandler_UploadISBJNoFile(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	// POST without file
	req, _ := http.NewRequest("POST", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing file, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_UploadISBJInvalidFile(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	// Upload a non-Excel file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "not-excel.xlsx")
	_, _ = part.Write([]byte("this is not an excel file"))
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid file, got %d", w.Code)
	}
}

func TestGovernmentFundingBillHandler_DeleteCascade(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billhandler8@example.com", "password")

	period := createBillPeriodInDB(t, db, org.ID, user.ID, "Cascade Kita", 11)
	periodID := period.ID

	// Verify children exist before
	var childCount int64
	db.Model(&models.GovernmentFundingBillChild{}).Where("period_id = ?", periodID).Count(&childCount)
	if childCount == 0 {
		t.Fatal("expected children to exist before delete")
	}

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/government-funding-bills/%d", org.ID, periodID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	// Verify cascade delete of children
	db.Model(&models.GovernmentFundingBillChild{}).Where("period_id = ?", periodID).Count(&childCount)
	if childCount != 0 {
		t.Errorf("expected 0 children after cascade delete, got %d", childCount)
	}

	// Verify cascade delete of payments
	var paymentCount int64
	db.Model(&models.GovernmentFundingBillPayment{}).
		Joins("JOIN government_funding_bill_children ON government_funding_bill_children.id = government_funding_bill_payments.child_id").
		Where("government_funding_bill_children.period_id = ?", periodID).
		Count(&paymentCount)
	if paymentCount != 0 {
		t.Errorf("expected 0 payments after cascade delete, got %d", paymentCount)
	}
}

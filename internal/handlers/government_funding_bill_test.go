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
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/testutil"
)

func createGovBillService(db *gorm.DB) *service.GovernmentFundingBillService {
	childStore := store.NewChildStore(db)
	billPeriodStore := store.NewGovernmentFundingBillPeriodStore(db)
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	childVoucherStore := store.NewChildVoucherStore(db)
	return service.NewGovernmentFundingBillService(childStore, childVoucherStore, billPeriodStore, orgStore, fundingStore, store.NewTransactor(db))
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
		org.GET("/:billId/compare", handler.Compare)
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
		CreatedBy:         &userID,
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

func TestGovernmentFundingBillHandler_Compare(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "billcompare1@example.com", "password")

	period := createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Compare", 11)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/%d/compare", org.ID, period.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result models.FundingComparisonResponse
	parseResponse(t, w, &result)

	if result.BillID != period.ID {
		t.Errorf("expected bill_id %d, got %d", period.ID, result.BillID)
	}
	if result.FacilityName != "Kita Compare" {
		t.Errorf("expected facility name 'Kita Compare', got %q", result.FacilityName)
	}
}

func TestGovernmentFundingBillHandler_Compare_NotFound(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/99999/compare", org.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGovernmentFundingBillHandler_UnmatchedBillChildren verifies the
// happy path: bill rows whose voucher has no child_vouchers row anywhere
// surface as one entry per voucher. The earliest bill is the one whose
// metadata lands in the response.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "unmatched-bill@example.com", "password")

	// Three bills covering Jan, Feb, March 2025.
	mkBill := func(month time.Month, children []models.GovernmentFundingBillChild) {
		to := time.Date(2025, month+1, 0, 0, 0, 0, 0, time.UTC)
		period := &models.GovernmentFundingBillPeriod{
			OrganizationID: org.ID,
			Period:         models.Period{From: time.Date(2025, month, 1, 0, 0, 0, 0, time.UTC), To: &to},
			FileName:       fmt.Sprintf("abrechnung_%02d-25.xlsx", month),
			FileSha256:     fmt.Sprintf("hash_%02d_%d", month, org.ID),
			FacilityName:   "Kita Test",
			FacilityTotal:  300000,
			CreatedBy:      &user.ID,
			Children:       children,
		}
		if err := db.Create(period).Error; err != nil {
			t.Fatalf("setup: create bill: %v", err)
		}
	}

	// Voucher A: present in Feb + March bills, no child_vouchers row → unmatched, earliest = Feb
	mkBill(time.February, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-AAAAAAAAAAA-01", ChildName: "Beispiel,Anna", BirthDate: "03.20", District: 1},
	})
	mkBill(time.March, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-AAAAAAAAAAA-01", ChildName: "Beispiel,Anna", BirthDate: "03.20", District: 1},
		{VoucherNumber: "GB-BBBBBBBBBBB-01", ChildName: "Muster,Bert", BirthDate: "07.21", District: 11},
	})
	// Voucher C: in Jan bill, has a matching child_voucher → MUST be excluded.
	matched := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Carla", LastName: "Match", Gender: "female", Birthdate: time.Date(2020, 5, 5, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(matched)
	db.Create(&models.ChildVoucher{ChildID: matched.ID, VoucherNumber: "GB-CCCCCCCCCCC-01", FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	mkBill(time.January, []models.GovernmentFundingBillChild{
		{VoucherNumber: "GB-CCCCCCCCCCC-01", ChildName: "Match,Carla", BirthDate: "05.20", District: 2},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 2 {
		t.Fatalf("expected 2 unmatched vouchers (A + B), got %d: %+v", len(resp), resp)
	}

	// Order: earliest-bill-first, then voucher_number for stability.
	// A first seen Feb 1; B first seen March 1.
	if resp[0].VoucherNumber != "GB-AAAAAAAAAAA-01" {
		t.Errorf("expected first row voucher A, got %s", resp[0].VoucherNumber)
	}
	if resp[0].FirstSeenBillFrom != "2025-02-01" {
		t.Errorf("expected A first_seen 2025-02-01, got %s", resp[0].FirstSeenBillFrom)
	}
	if resp[0].FirstName != "Anna" || resp[0].LastName != "Beispiel" {
		t.Errorf("name parse wrong: first=%q last=%q", resp[0].FirstName, resp[0].LastName)
	}
	if resp[0].BillBirthDate != "03.20" {
		t.Errorf("expected bill_birth_date 03.20, got %s", resp[0].BillBirthDate)
	}

	if resp[1].VoucherNumber != "GB-BBBBBBBBBBB-01" {
		t.Errorf("expected second row voucher B, got %s", resp[1].VoucherNumber)
	}
	if resp[1].FirstSeenBillFrom != "2025-03-01" {
		t.Errorf("expected B first_seen 2025-03-01, got %s", resp[1].FirstSeenBillFrom)
	}

	// Voucher C MUST NOT be present (it has a child_vouchers row).
	for _, row := range resp {
		if row.VoucherNumber == "GB-CCCCCCCCCCC-01" {
			t.Errorf("voucher C is matched and must NOT appear in unmatched list, got %+v", row)
		}
	}
}

// When the bill row's name + birth-month match an existing KitaManager
// child (active OR ended contract), the response carries an
// existing_child_match pointer so the dashboard can route the user to
// "Link voucher" instead of "Create new child". Covers the residual-
// settlement case for departed children.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_LinkToExistingChild(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "unmatched-existing@example.com", "password")

	// Existing child in KitaManager — no voucher row; could be a child
	// who left and whose voucher was never linked, or a current child
	// who slipped past auto-discover.
	existing := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(existing)

	// Bill row with the same name and matching MM.YY birth date.
	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "abrechnung_02-25.xlsx",
		FileSha256:     "hash-link-existing",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-LINK00000001-01", ChildName: "Berger,Anna", BirthDate: "03.20", District: 1},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp))
	}
	if resp[0].ExistingChildMatch == nil {
		t.Fatalf("expected existing_child_match populated, got nil")
	}
	if resp[0].ExistingChildMatch.ID != existing.ID {
		t.Errorf("expected existing match id=%d, got %d", existing.ID, resp[0].ExistingChildMatch.ID)
	}
	if resp[0].ExistingChildMatch.FirstName != "Anna" || resp[0].ExistingChildMatch.LastName != "Berger" {
		t.Errorf("unexpected match identity: %+v", resp[0].ExistingChildMatch)
	}
}

// Truncated-middle-name case: bill has full name "First Mid Mid Last",
// KitaManager has just "First Last". Fuzzy NameSimilarity scores ≥0.65
// → match should still surface.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_LinkToExistingChild_TruncatedName(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "unmatched-truncated@example.com", "password")

	existing := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(existing)

	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "abrechnung_02-25.xlsx",
		FileSha256:     "hash-truncated",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			// Bill carries extra middle names the KitaManager record
			// doesn't have — strict equality would miss this.
			{VoucherNumber: "GB-TRUNC0000001-01", ChildName: "Berger,Anna Maria Lena", BirthDate: "03.20", District: 1},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp))
	}
	if resp[0].ExistingChildMatch == nil {
		t.Fatalf("expected existing_child_match populated for truncated-name case, got nil")
	}
	if resp[0].ExistingChildMatch.ID != existing.ID {
		t.Errorf("expected existing match id=%d, got %d", existing.ID, resp[0].ExistingChildMatch.ID)
	}
}

// ±1 month birth-date drift: bill says March, DB says April. The fuzzy
// matcher accepts up to ±2 months with a score penalty; a perfect
// name match (1.0 - 0.3 = 0.7) survives the threshold.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_LinkToExistingChild_BirthMonthDrift(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "unmatched-drift@example.com", "password")

	existing := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Anna",
			LastName:       "Berger",
			Gender:         "female",
			Birthdate:      time.Date(2020, 4, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(existing)

	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "abrechnung_02-25.xlsx",
		FileSha256:     "hash-drift",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-DRIFT0000001-01", ChildName: "Berger,Anna", BirthDate: "03.20", District: 1},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org.ID), nil)
	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 1 || resp[0].ExistingChildMatch == nil {
		t.Fatalf("expected ±1 month drift to still match, got %+v", resp)
	}
}

// Two existing children that BOTH score above threshold for the same
// bill row → ambiguous; the response must NOT pick one. The user
// resolves manually via the Vouchers dialog instead of the dashboard
// guessing.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_LinkToExistingChild_AmbiguousNoMatch(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "User", "unmatched-ambig@example.com", "password")

	// Two children with identical name + birth-month — both will tie
	// at fuzzy score 1.0.
	for i := range 2 {
		c := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      "Anna",
				LastName:       "Berger",
				Gender:         "female",
				Birthdate:      time.Date(2020, 3, 10+i, 0, 0, 0, 0, time.UTC),
			},
		}
		db.Create(c)
	}

	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "abrechnung_02-25.xlsx",
		FileSha256:     "hash-ambig",
		FacilityName:   "Kita Test",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-AMBIG0000001-01", ChildName: "Berger,Anna", BirthDate: "03.20", District: 1},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org.ID), nil)
	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 unmatched bill row, got %d", len(resp))
	}
	if resp[0].ExistingChildMatch != nil {
		t.Errorf("expected NO existing match (ambiguous), got %+v", resp[0].ExistingChildMatch)
	}
}

// Empty / cross-org isolation case: an unmatched voucher in another org
// must not leak into this org's response.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_CrossOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "unmatched-cross@example.com", "password")

	// Unmatched voucher only in org2.
	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org2.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "org2-feb.xlsx",
		FileSha256:     "hash-org2",
		FacilityName:   "Kita Other",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-OTHERORG001-01", ChildName: "Foo,Bar", BirthDate: "01.20", District: 5},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	// Querying org1 must return empty.
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org1.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 0 {
		t.Errorf("expected 0 (cross-org leak guard), got %d", len(resp))
	}
}

// A voucher that's in child_vouchers in a DIFFERENT org must STILL be
// excluded from this org's unmatched list — voucher_number is globally
// unique so AssignVoucher would 409 anyway. Save the user a doomed POST.
func TestGovernmentFundingBillHandler_UnmatchedBillChildren_ExcludesGloballyAssigned(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	user := createTestUser(t, db, "User", "unmatched-global@example.com", "password")

	// Voucher X is in a child_vouchers row in org2.
	otherChild := &models.Child{
		Person: models.Person{OrganizationID: org2.ID, FirstName: "X", LastName: "Y", Gender: "male", Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(otherChild)
	db.Create(&models.ChildVoucher{ChildID: otherChild.ID, VoucherNumber: "GB-GLOBAL00001-01", FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})

	// The same voucher_number appears in org1's bill — but is "claimed" globally.
	to := time.Date(2025, 3, 0, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingBillPeriod{
		OrganizationID: org1.ID,
		Period:         models.Period{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: &to},
		FileName:       "org1-feb.xlsx",
		FileSha256:     "hash-org1",
		FacilityName:   "Kita One",
		CreatedBy:      &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-GLOBAL00001-01", ChildName: "Foo,Bar", BirthDate: "01.20", District: 5},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/government-funding-bills/unmatched-children", handler.UnmatchedBillChildren)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills/unmatched-children", org1.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.UnmatchedBillChildResponse
	parseResponse(t, w, &resp)
	if len(resp) != 0 {
		t.Errorf("expected 0 (voucher globally assigned to other org), got %d: %+v", len(resp), resp)
	}
}

func TestGovernmentFundingBillHandler_AssignVoucher(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	body := models.ChildVoucherCreateRequest{VoucherNumber: "GB-12345678901-01"}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify voucher was created
	var voucher models.ChildVoucher
	if err := db.Where("child_id = ? AND voucher_number = ?", child.ID, "GB-12345678901-01").First(&voucher).Error; err != nil {
		t.Fatalf("voucher not found in database: %v", err)
	}
}

func TestGovernmentFundingBillHandler_AssignVoucher_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	body := models.ChildVoucherCreateRequest{VoucherNumber: "GB-12345678901-01"}

	// First call
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("first call: expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Second call — should still succeed (idempotent)
	w = performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("second call: expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Only one voucher entry should exist
	var count int64
	db.Model(&models.ChildVoucher{}).Where("child_id = ?", child.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 voucher entry, got %d", count)
	}
}

func TestGovernmentFundingBillHandler_AssignVoucher_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	body := models.ChildVoucherCreateRequest{VoucherNumber: "GB-12345678901-01"}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/99999/vouchers", org.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_AssignVoucher_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	body := models.ChildVoucherCreateRequest{VoucherNumber: "GB-12345678901-01"}
	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org2.ID, child.ID), body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_AssignVoucher_MissingVoucherNumber(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), map[string]any{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// G7 / I-M-4 — voucher numbers must match the canonical Berlin format
// `GB-DDDDDDDDDDD-NN`. Previously only the database `size:17` boundary
// would have rejected truly oversized values; freeform garbage like
// "totally-wrong" reached the audit log and the store layer first.
func TestGovernmentFundingBillHandler_AssignVoucher_RejectsBadPattern(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.POST("/organizations/:orgId/children/:childId/vouchers", handler.AssignVoucher)

	bad := []string{
		"totally-wrong",
		"GB-1-1",                       // too few digits
		"GB-12345678901-1",             // 1-digit suffix
		"GB-12345678901-001",           // 3-digit suffix
		"gb-12345678901-01",            // wrong case
		"GB-12345678901-01\n",          // trailing newline
		" GB-12345678901-01",           // leading space
		"GB-12345678901-01;DROP TABLE", // SQLi-shaped
	}
	for _, voucher := range bad {
		t.Run(voucher, func(t *testing.T) {
			body := models.ChildVoucherCreateRequest{VoucherNumber: voucher}
			w := performRequest(r, "POST", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %q, got %d: %s", voucher, w.Code, w.Body.String())
			}

			// Even the malformed voucher must NOT have hit the database
			// table — gating happens at JSON binding, before the service.
			var count int64
			db.Model(&models.ChildVoucher{}).Where("voucher_number = ?", voucher).Count(&count)
			if count != 0 {
				t.Errorf("voucher %q persisted despite 400 response", voucher)
			}
		})
	}
}

func TestGovernmentFundingBillHandler_ListChildVouchers(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	// Insert in non-canonical order — handler must return them sorted by first_seen ascending.
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: "GB-12345678901-02", FirstSeen: newer})
	db.Create(&models.ChildVoucher{ChildID: child.ID, VoucherNumber: "GB-12345678901-01", FirstSeen: older})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/vouchers", handler.ListChildVouchers)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []models.ChildVoucherResponse
	parseResponse(t, w, &resp)
	if len(resp) != 2 {
		t.Fatalf("expected 2 vouchers, got %d", len(resp))
	}
	// first_seen ascending → -01 (older) comes first.
	if resp[0].VoucherNumber != "GB-12345678901-01" {
		t.Errorf("expected first voucher GB-12345678901-01, got %s", resp[0].VoucherNumber)
	}
	if resp[1].VoucherNumber != "GB-12345678901-02" {
		t.Errorf("expected second voucher GB-12345678901-02, got %s", resp[1].VoucherNumber)
	}
}

func TestGovernmentFundingBillHandler_ListChildVouchers_Empty(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/vouchers", handler.ListChildVouchers)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org.ID, child.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []models.ChildVoucherResponse
	parseResponse(t, w, &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d vouchers", len(resp))
	}
}

func TestGovernmentFundingBillHandler_ListChildVouchers_ChildNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/vouchers", handler.ListChildVouchers)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/99999/vouchers", org.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_ListChildVouchers_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.GET("/organizations/:orgId/children/:childId/vouchers", handler.ListChildVouchers)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/children/%d/vouchers", org2.ID, child.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (cross-org leak guard), got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_RemoveChildVoucher(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)
	voucher := &models.ChildVoucher{ChildID: child.ID, VoucherNumber: "GB-12345678901-01", FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	db.Create(voucher)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/vouchers/:voucherId", handler.RemoveChildVoucher)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/vouchers/%d", org.ID, child.ID, voucher.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Voucher row gone.
	var count int64
	db.Model(&models.ChildVoucher{}).Where("id = ?", voucher.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected voucher row to be hard-deleted, found %d row(s)", count)
	}

	// Audit log captures resource_id + voucher_number.
	row := testutil.AssertAuditLog(t, db, testutil.AuditLogQuery{
		Action:       models.AuditAction("child_voucher_delete"),
		ResourceType: "child_voucher",
		ResourceID:   voucher.ID,
	})
	// Resource name is stored in the JSON Details field.
	if !strings.Contains(row.Details, "GB-12345678901-01") {
		t.Errorf("audit row Details = %q, expected to contain voucher number", row.Details)
	}

	// Freed unique slot — same number can be reassigned to another child.
	other := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Other", LastName: "Child", Gender: "male", Birthdate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(other)
	if err := db.Create(&models.ChildVoucher{ChildID: other.ID, VoucherNumber: "GB-12345678901-01", FirstSeen: time.Now().UTC()}).Error; err != nil {
		t.Errorf("expected freed voucher slot to allow re-insert, got error: %v", err)
	}
}

func TestGovernmentFundingBillHandler_RemoveChildVoucher_NotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/vouchers/:voucherId", handler.RemoveChildVoucher)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/vouchers/99999", org.ID, child.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// Voucher exists but on a different child than the URL claims. Must be
// 404 (not 403) so the response is identical to "voucher does not exist"
// — leaks no information about other children's voucher IDs.
func TestGovernmentFundingBillHandler_RemoveChildVoucher_WrongChild(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org := createTestOrganization(t, db, "Test Org")
	childA := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "A", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childA)
	childB := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "B", LastName: "Child", Gender: "male", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(childB)
	voucher := &models.ChildVoucher{ChildID: childA.ID, VoucherNumber: "GB-12345678901-01", FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	db.Create(voucher)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/vouchers/:voucherId", handler.RemoveChildVoucher)

	// Try to delete childA's voucher under childB's URL.
	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/vouchers/%d", org.ID, childB.ID, voucher.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (cross-child leak guard), got %d: %s", w.Code, w.Body.String())
	}
	// Voucher must still exist.
	var count int64
	db.Model(&models.ChildVoucher{}).Where("id = ?", voucher.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected voucher to be untouched, found %d rows", count)
	}
}

func TestGovernmentFundingBillHandler_RemoveChildVoucher_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	handler := createGovBillHandler(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	child := &models.Child{
		Person: models.Person{OrganizationID: org1.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)
	voucher := &models.ChildVoucher{ChildID: child.ID, VoucherNumber: "GB-12345678901-01", FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	db.Create(voucher)

	r := setupTestRouter()
	r.DELETE("/organizations/:orgId/children/:childId/vouchers/:voucherId", handler.RemoveChildVoucher)

	w := performRequest(r, "DELETE", fmt.Sprintf("/organizations/%d/children/%d/vouchers/%d", org2.ID, child.ID, voucher.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (cross-org leak guard), got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingBillHandler_List_Search(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Search User", "billsearch@example.com", "password")

	createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Sonnenschein", time.January)
	createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Regenbogen", time.February)
	createBillPeriodInDB(t, db, org.ID, user.ID, "Hort Abenteuer", time.March)

	t.Run("case-insensitive match", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=sonnenschein", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 1 {
			t.Errorf("expected 1 bill matching 'sonnenschein', got %d", len(response.Data))
		}
		if response.Total != 1 {
			t.Errorf("expected total 1, got %d", response.Total)
		}
	})

	t.Run("partial match", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=Kita", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 2 {
			t.Errorf("expected 2 bills matching 'Kita', got %d", len(response.Data))
		}
	})

	t.Run("no match", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=nonexistent", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 0 {
			t.Errorf("expected 0 bills, got %d", len(response.Data))
		}
	})

	t.Run("empty search returns all", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 3 {
			t.Errorf("expected 3 bills without search, got %d", len(response.Data))
		}
	})

	t.Run("search preserves org isolation", func(t *testing.T) {
		otherOrg := createTestOrganization(t, db, "Other Org")
		createBillPeriodInDB(t, db, otherOrg.ID, user.ID, "Kita Sonnenschein Other", time.April)

		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=Sonnenschein", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if len(response.Data) != 1 {
			t.Errorf("expected 1 bill from own org, got %d", len(response.Data))
		}
	})

	t.Run("search too long returns 400", func(t *testing.T) {
		longSearch := strings.Repeat("a", 256)
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=%s", org.ID, longSearch), nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for search > 255 chars, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGovernmentFundingBillHandler_List_Search_BillChildren(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupBillRouter(db)
	org := createTestOrganization(t, db, "Test Org")
	user := createTestUser(t, db, "Search User", "billsearchchildren@example.com", "password")

	// Period 1: vanilla, single "Kind, Test" child with voucher GB-00000000001-01.
	createBillPeriodInDB(t, db, org.ID, user.ID, "Kita Sonnenschein", time.January)

	// Period 2: distinctive child name "Mustermann, Anna" with distinctive voucher prefix.
	feb := time.February
	febTo := time.Date(2025, feb+1, 0, 0, 0, 0, 0, time.UTC)
	period2 := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, feb, 1, 0, 0, 0, 0, time.UTC), To: &febTo},
		FileName:          "abrechnung_02-25.xlsx",
		FileSha256:        "hash_search_children_02",
		FacilityName:      "Kita Regenbogen",
		FacilityTotal:     300000,
		ContractBooking:   280000,
		CorrectionBooking: 20000,
		CreatedBy:         &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-99999999991-01", ChildName: "Mustermann, Anna", BirthDate: "01.20", District: 1},
		},
	}
	if err := db.Create(period2).Error; err != nil {
		t.Fatalf("setup: create period2: %v", err)
	}

	// Period 3: two children sharing a last name, to verify EXISTS de-duplicates.
	mar := time.March
	marTo := time.Date(2025, mar+1, 0, 0, 0, 0, 0, time.UTC)
	period3 := &models.GovernmentFundingBillPeriod{
		OrganizationID:    org.ID,
		Period:            models.Period{From: time.Date(2025, mar, 1, 0, 0, 0, 0, time.UTC), To: &marTo},
		FileName:          "abrechnung_03-25.xlsx",
		FileSha256:        "hash_search_children_03",
		FacilityName:      "Hort Abenteuer",
		FacilityTotal:     300000,
		ContractBooking:   280000,
		CorrectionBooking: 20000,
		CreatedBy:         &user.ID,
		Children: []models.GovernmentFundingBillChild{
			{VoucherNumber: "GB-11111111111-01", ChildName: "Schulze, Ben", BirthDate: "01.20", District: 1},
			{VoucherNumber: "GB-22222222222-01", ChildName: "Schulze, Ida", BirthDate: "02.21", District: 1},
		},
	}
	if err := db.Create(period3).Error; err != nil {
		t.Fatalf("setup: create period3: %v", err)
	}

	t.Run("matches by child name", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=Mustermann", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if response.Total != 1 || len(response.Data) != 1 {
			t.Fatalf("expected 1 bill matching child 'Mustermann', got total=%d data=%d", response.Total, len(response.Data))
		}
		if response.Data[0].FacilityName != "Kita Regenbogen" {
			t.Errorf("expected Kita Regenbogen, got %q", response.Data[0].FacilityName)
		}
	})

	t.Run("matches by voucher number", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=GB-9999", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if response.Total != 1 || len(response.Data) != 1 {
			t.Fatalf("expected 1 bill matching voucher 'GB-9999', got total=%d data=%d", response.Total, len(response.Data))
		}
		if response.Data[0].FacilityName != "Kita Regenbogen" {
			t.Errorf("expected Kita Regenbogen, got %q", response.Data[0].FacilityName)
		}
	})

	t.Run("does not duplicate periods when multiple children match", func(t *testing.T) {
		// Both Schulze children belong to period3 — EXISTS must return that period exactly once.
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=Schulze", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if response.Total != 1 || len(response.Data) != 1 {
			t.Fatalf("expected 1 bill (de-duplicated), got total=%d data=%d", response.Total, len(response.Data))
		}
	})

	t.Run("LIKE wildcards in search are matched literally", func(t *testing.T) {
		// Underscores are LIKE wildcards. If escapeLIKE is broken, "_____" matches any 5+ chars
		// and would return all 3 bills; with proper escaping, it matches only literal underscores.
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=_____", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if response.Total != 0 || len(response.Data) != 0 {
			t.Errorf("expected 0 (underscore wildcards must be escaped), got total=%d data=%d", response.Total, len(response.Data))
		}
	})

	t.Run("facility name match still works alongside child search", func(t *testing.T) {
		// Regression: existing facility-name behaviour must not regress with the new OR clause.
		w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/government-funding-bills?search=Sonnenschein", org.ID), nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
		parseResponse(t, w, &response)
		if response.Total != 1 || len(response.Data) != 1 {
			t.Fatalf("expected 1 bill matching facility 'Sonnenschein', got total=%d data=%d", response.Total, len(response.Data))
		}
	})
}

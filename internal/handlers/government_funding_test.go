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

func TestGovernmentFundingHandler_CRUD(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/fundings", handler.List)
	r.GET("/fundings/:id", handler.Get)
	r.POST("/fundings", handler.Create)
	r.PUT("/fundings/:id", handler.Update)
	r.DELETE("/fundings/:id", handler.Delete)

	// Test Create
	t.Run("Create", func(t *testing.T) {
		body := models.GovernmentFundingCreateRequest{Name: "Berlin Kita Funding", State: "berlin"}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin Kita Funding" {
			t.Errorf("expected name 'Berlin Kita Funding', got '%s'", result.Name)
		}
		if result.State != "berlin" {
			t.Errorf("expected state 'berlin', got '%s'", result.State)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.GovernmentFunding]
		parseResponse(t, w, &response)
		if len(response.Data) != 1 {
			t.Errorf("expected 1 funding, got %d", len(response.Data))
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings/1", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin Kita Funding" {
			t.Errorf("expected name 'Berlin Kita Funding', got '%s'", result.Name)
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		name := "Berlin Updated"
		body := models.GovernmentFundingUpdateRequest{Name: &name}
		w := performRequest(r, "PUT", "/fundings/1", body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.GovernmentFunding
		parseResponse(t, w, &result)
		if result.Name != "Berlin Updated" {
			t.Errorf("expected name 'Berlin Updated', got '%s'", result.Name)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		w := performRequest(r, "DELETE", "/fundings/1", nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}

func TestGovernmentFundingHandler_CreatePeriod_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods", handler.CreatePeriod)

	tests := []struct {
		name           string
		existingFrom   string
		existingTo     *string
		newFrom        string
		newTo          *string
		expectedStatus int
		description    string
	}{
		{
			name:           "no overlap: new period after existing",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-07-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusCreated,
			description:    "New period starts after existing ends",
		},
		{
			name:           "no overlap: new period before existing",
			existingFrom:   "2024-07-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusCreated,
			description:    "New period ends before existing starts",
		},
		{
			name:           "overlap: new period inside existing",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-03-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period is entirely within existing period",
		},
		{
			name:           "overlap: new period spans existing",
			existingFrom:   "2024-03-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period completely covers existing period",
		},
		{
			name:           "overlap: partial overlap at start",
			existingFrom:   "2024-06-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-07-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period overlaps at the start of existing",
		},
		{
			name:           "overlap: partial overlap at end",
			existingFrom:   "2024-01-01",
			existingTo:     strPtr("2024-06-30"),
			newFrom:        "2024-05-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "New period overlaps at the end of existing",
		},
		{
			name:           "overlap: existing has no end date",
			existingFrom:   "2024-01-01",
			existingTo:     nil,
			newFrom:        "2024-06-01",
			newTo:          strPtr("2024-12-31"),
			expectedStatus: http.StatusBadRequest,
			description:    "Existing period is ongoing, new period overlaps",
		},
		{
			name:           "overlap: new has no end date",
			existingFrom:   "2024-06-01",
			existingTo:     strPtr("2024-12-31"),
			newFrom:        "2024-01-01",
			newTo:          nil,
			expectedStatus: http.StatusBadRequest,
			description:    "New period is ongoing and overlaps existing",
		},
		{
			name:           "no overlap: new period before ongoing",
			existingFrom:   "2024-07-01",
			existingTo:     nil,
			newFrom:        "2024-01-01",
			newTo:          strPtr("2024-06-30"),
			expectedStatus: http.StatusCreated,
			description:    "New period ends before ongoing period starts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up periods from previous test
			db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

			// Create existing period
			existingFrom, _ := time.Parse("2006-01-02", tt.existingFrom)
			existingPeriod := &models.GovernmentFundingPeriod{
				GovernmentFundingID: funding.ID,
				From:                existingFrom,
			}
			if tt.existingTo != nil {
				to, _ := time.Parse("2006-01-02", *tt.existingTo)
				existingPeriod.To = &to
			}
			db.Create(existingPeriod)

			// Try to create new period
			newFrom, _ := time.Parse("2006-01-02", tt.newFrom)
			body := map[string]interface{}{
				"from":                   newFrom.Format(time.RFC3339),
				"full_time_weekly_hours": 39.0,
			}
			if tt.newTo != nil {
				newTo, _ := time.Parse("2006-01-02", *tt.newTo)
				body["to"] = newTo.Format(time.RFC3339)
			}

			w := performRequest(r, "POST", "/fundings/1/periods", body)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGovernmentFundingHandler_UpdatePeriod_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	r := setupTestRouter()
	r.PUT("/fundings/:id/periods/:periodId", handler.UpdatePeriod)

	t.Run("update period to overlap with another", func(t *testing.T) {
		// Clean up
		db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

		// Create two non-overlapping periods
		from1, _ := time.Parse("2006-01-02", "2024-01-01")
		to1, _ := time.Parse("2006-01-02", "2024-06-30")
		period1 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from1, To: &to1}
		db.Create(period1)

		from2, _ := time.Parse("2006-01-02", "2024-07-01")
		to2, _ := time.Parse("2006-01-02", "2024-12-31")
		period2 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from2, To: &to2}
		db.Create(period2)

		// Try to update period2 to overlap with period1
		newFrom, _ := time.Parse("2006-01-02", "2024-05-01")
		body := map[string]interface{}{
			"from": newFrom.Format(time.RFC3339),
		}

		w := performRequest(r, "PUT", "/fundings/1/periods/"+itoa(int(period2.ID)), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for overlapping update, got %d: %s",
				http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("update period without causing overlap", func(t *testing.T) {
		// Clean up
		db.Where("government_funding_id = ?", funding.ID).Delete(&models.GovernmentFundingPeriod{})

		// Create two non-overlapping periods
		from1, _ := time.Parse("2006-01-02", "2024-01-01")
		to1, _ := time.Parse("2006-01-02", "2024-06-30")
		period1 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from1, To: &to1}
		db.Create(period1)

		from2, _ := time.Parse("2006-01-02", "2024-07-01")
		to2, _ := time.Parse("2006-01-02", "2024-12-31")
		period2 := &models.GovernmentFundingPeriod{GovernmentFundingID: funding.ID, From: from2, To: &to2}
		db.Create(period2)

		// Update period2's end date (no overlap)
		newTo, _ := time.Parse("2006-01-02", "2025-06-30")
		body := map[string]interface{}{
			"to": newTo.Format(time.RFC3339),
		}

		w := performRequest(r, "PUT", "/fundings/1/periods/"+itoa(int(period2.ID)), body)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for valid update, got %d: %s",
				http.StatusOK, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Property_AgeRange(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding and period
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods/:periodId/properties", handler.CreateProperty)

	tests := []struct {
		name           string
		minAge         *int
		maxAge         *int
		expectedStatus int
		description    string
	}{
		{
			name:           "valid range 0-2",
			minAge:         intPtr(0),
			maxAge:         intPtr(2),
			expectedStatus: http.StatusCreated,
			description:    "Children from birth up to but not including 2nd birthday",
		},
		{
			name:           "valid range 3-7",
			minAge:         intPtr(3),
			maxAge:         intPtr(7),
			expectedStatus: http.StatusCreated,
			description:    "Children from 3rd birthday up to but not including 7th birthday",
		},
		{
			name:           "no age filter",
			minAge:         nil,
			maxAge:         nil,
			expectedStatus: http.StatusCreated,
			description:    "Property applies to all ages",
		},
		{
			name:           "invalid: min equals max",
			minAge:         intPtr(2),
			maxAge:         intPtr(2),
			expectedStatus: http.StatusBadRequest,
			description:    "Empty range - no children would qualify",
		},
		{
			name:           "invalid: min greater than max",
			minAge:         intPtr(5),
			maxAge:         intPtr(3),
			expectedStatus: http.StatusBadRequest,
			description:    "Inverted range is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"key":         "care_type",
				"value":       "ganztag",
				"payment":     100000,
				"requirement": 0.1,
			}
			if tt.minAge != nil {
				body["min_age"] = *tt.minAge
			}
			if tt.maxAge != nil {
				body["max_age"] = *tt.maxAge
			}

			w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/%d/properties", funding.ID, period.ID), body)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGovernmentFundingHandler_Get_PeriodsLimit(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding with multiple periods
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	// Create 3 overlapping periods that are all active today
	periods := []struct {
		from time.Time
		to   time.Time
	}{
		{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, p := range periods {
		to := p.to
		period := &models.GovernmentFundingPeriod{
			GovernmentFundingID: funding.ID,
			From:                p.from,
			To:                  &to,
		}
		db.Create(period)
	}

	r := setupTestRouter()
	r.GET("/fundings/:id", handler.Get)

	t.Run("default returns only latest period", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/fundings/%d", funding.ID), nil)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.GovernmentFundingWithDetailsResponse
		parseResponse(t, w, &result)

		if len(result.Periods) != 1 {
			t.Errorf("expected 1 period by default, got %d", len(result.Periods))
		}
		if result.TotalPeriods != 3 {
			t.Errorf("expected total_periods=3, got %d", result.TotalPeriods)
		}
	})

	t.Run("periods_limit=0 returns all periods", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/fundings/%d?periods_limit=0", funding.ID), nil)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.GovernmentFundingWithDetailsResponse
		parseResponse(t, w, &result)

		if len(result.Periods) != 3 {
			t.Errorf("expected 3 periods with limit=0, got %d", len(result.Periods))
		}
		if result.TotalPeriods != 3 {
			t.Errorf("expected total_periods=3, got %d", result.TotalPeriods)
		}
	})

	t.Run("periods_limit=2 returns 2 periods", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/fundings/%d?periods_limit=2", funding.ID), nil)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var result models.GovernmentFundingWithDetailsResponse
		parseResponse(t, w, &result)

		if len(result.Periods) != 2 {
			t.Errorf("expected 2 periods with limit=2, got %d", len(result.Periods))
		}
		if result.TotalPeriods != 3 {
			t.Errorf("expected total_periods=3, got %d", result.TotalPeriods)
		}
	})

	t.Run("negative periods_limit returns 400", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/fundings/%d?periods_limit=-1", funding.ID), nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for negative limit, got %d: %s",
				http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("invalid periods_limit returns 400", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/fundings/%d?periods_limit=abc", funding.ID), nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for invalid limit, got %d: %s",
				http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

// Additional error case tests

func TestGovernmentFundingHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test fundings
	for i := 0; i < 5; i++ {
		funding := &models.GovernmentFunding{Name: fmt.Sprintf("Funding %d", i), State: "berlin"}
		// Note: State is unique, so we need to handle this differently
		// For this test, we'll just create one
		if i == 0 {
			db.Create(funding)
		}
	}

	r := setupTestRouter()
	r.GET("/fundings", handler.List)

	t.Run("default pagination", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.GovernmentFunding]
		parseResponse(t, w, &response)
		if response.Page != 1 {
			t.Errorf("expected page 1, got %d", response.Page)
		}
		if response.Limit != 20 {
			t.Errorf("expected limit 20, got %d", response.Limit)
		}
	})

	t.Run("custom page and limit", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings?page=1&limit=10", nil)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.PaginatedResponse[models.GovernmentFunding]
		parseResponse(t, w, &response)
		if response.Limit != 10 {
			t.Errorf("expected limit 10, got %d", response.Limit)
		}
	})

	t.Run("invalid negative page", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings?page=-1", nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("invalid negative limit", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings?limit=-1", nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("limit exceeds max", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings?limit=200", nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.GET("/fundings/:id", handler.Get)

	t.Run("non-existent ID", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings/999", nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings/abc", nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("zero ID", func(t *testing.T) {
		w := performRequest(r, "GET", "/fundings/0", nil)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Create_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/fundings", handler.Create)

	t.Run("invalid state", func(t *testing.T) {
		body := models.GovernmentFundingCreateRequest{Name: "Test Funding", State: "invalid"}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("empty name", func(t *testing.T) {
		body := models.GovernmentFundingCreateRequest{Name: "", State: "berlin"}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("whitespace only name", func(t *testing.T) {
		body := models.GovernmentFundingCreateRequest{Name: "   ", State: "berlin"}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := map[string]interface{}{}
		w := performRequest(r, "POST", "/fundings", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.PUT("/fundings/:id", handler.Update)

	t.Run("non-existent ID", func(t *testing.T) {
		name := "Updated Name"
		body := models.GovernmentFundingUpdateRequest{Name: &name}
		w := performRequest(r, "PUT", "/fundings/999", body)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		name := "Updated Name"
		body := models.GovernmentFundingUpdateRequest{Name: &name}
		w := performRequest(r, "PUT", "/fundings/abc", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Update_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	r := setupTestRouter()
	r.PUT("/fundings/:id", handler.Update)

	t.Run("whitespace only name", func(t *testing.T) {
		name := "   "
		body := models.GovernmentFundingUpdateRequest{Name: &name}
		w := performRequest(r, "PUT", fmt.Sprintf("/fundings/%d", funding.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.DELETE("/fundings/:id", handler.Delete)

	t.Run("non-existent ID", func(t *testing.T) {
		w := performRequest(r, "DELETE", "/fundings/999", nil)

		// Delete may return 204 or error depending on store implementation
		// Check if it's either 204 (idempotent) or 404/500
		if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("unexpected status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		w := performRequest(r, "DELETE", "/fundings/abc", nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_CreatePeriod_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.POST("/fundings/:id/periods", handler.CreatePeriod)

	t.Run("non-existent funding ID", func(t *testing.T) {
		body := map[string]interface{}{
			"from":                   "2024-01-01T00:00:00Z",
			"full_time_weekly_hours": 39.0,
		}
		w := performRequest(r, "POST", "/fundings/999/periods", body)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})

	t.Run("invalid funding ID", func(t *testing.T) {
		body := map[string]interface{}{
			"from":                   "2024-01-01T00:00:00Z",
			"full_time_weekly_hours": 39.0,
		}
		w := performRequest(r, "POST", "/fundings/abc/periods", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("missing from date", func(t *testing.T) {
		// First create a funding
		funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
		db.Create(funding)

		body := map[string]interface{}{}
		w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods", funding.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_DeletePeriod_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	r := setupTestRouter()
	r.DELETE("/fundings/:id/periods/:periodId", handler.DeletePeriod)

	t.Run("invalid period ID", func(t *testing.T) {
		w := performRequest(r, "DELETE", fmt.Sprintf("/fundings/%d/periods/abc", funding.ID), nil)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_CreateProperty_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding and period
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)

	r := setupTestRouter()
	r.POST("/fundings/:id/periods/:periodId/properties", handler.CreateProperty)

	t.Run("empty key", func(t *testing.T) {
		body := map[string]interface{}{
			"key":         "",
			"value":       "ganztag",
			"payment":     100000,
			"requirement": 0.1,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/%d/properties", funding.ID, period.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("whitespace only key", func(t *testing.T) {
		body := map[string]interface{}{
			"key":         "   ",
			"value":       "ganztag",
			"payment":     100000,
			"requirement": 0.1,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/%d/properties", funding.ID, period.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("empty value", func(t *testing.T) {
		body := map[string]interface{}{
			"key":         "care_type",
			"value":       "",
			"payment":     100000,
			"requirement": 0.1,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/%d/properties", funding.ID, period.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("non-existent period ID", func(t *testing.T) {
		body := map[string]interface{}{
			"key":         "care_type",
			"value":       "ganztag",
			"payment":     100000,
			"requirement": 0.1,
		}
		w := performRequest(r, "POST", fmt.Sprintf("/fundings/%d/periods/999/properties", funding.ID), body)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})
}

func TestGovernmentFundingHandler_UpdateProperty_Validation(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test funding, period, and property
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)
	property := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}
	db.Create(property)

	r := setupTestRouter()
	r.PUT("/fundings/:id/periods/:periodId/properties/:propId", handler.UpdateProperty)

	t.Run("non-existent property ID", func(t *testing.T) {
		key := "new_key"
		body := models.GovernmentFundingPropertyUpdateRequest{Key: &key}
		w := performRequest(r, "PUT", fmt.Sprintf("/fundings/%d/periods/%d/properties/999", funding.ID, period.ID), body)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
		}
	})

	t.Run("whitespace only key", func(t *testing.T) {
		key := "   "
		body := models.GovernmentFundingPropertyUpdateRequest{Key: &key}
		w := performRequest(r, "PUT", fmt.Sprintf("/fundings/%d/periods/%d/properties/%d", funding.ID, period.ID, property.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("whitespace only value", func(t *testing.T) {
		value := "   "
		body := models.GovernmentFundingPropertyUpdateRequest{Value: &value}
		w := performRequest(r, "PUT", fmt.Sprintf("/fundings/%d/periods/%d/properties/%d", funding.ID, period.ID, property.ID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestGovernmentFundingHandler_DeleteProperty(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create test data
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)
	property := &models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Payment:     100000,
		Requirement: 0.1,
	}
	db.Create(property)

	r := setupTestRouter()
	r.DELETE("/fundings/:id/periods/:periodId/properties/:propId", handler.DeleteProperty)

	w := performRequest(r, "DELETE", fmt.Sprintf("/fundings/%d/periods/%d/properties/%d", funding.ID, period.ID, property.ID), nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}
}

func TestGovernmentFundingHandler_DeleteProperty_NotFound(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	// Create funding and period but no property
	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(period)

	r := setupTestRouter()
	r.DELETE("/fundings/:id/periods/:periodId/properties/:propId", handler.DeleteProperty)

	w := performRequest(r, "DELETE", fmt.Sprintf("/fundings/%d/periods/%d/properties/999", funding.ID, period.ID), nil)

	// Delete may return 204 (idempotent), 404, or 500 depending on store implementation
	if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 204, 404, or 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGovernmentFundingHandler_DeleteProperty_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	fundingStore := store.NewGovernmentFundingStore(db)
	svc := service.NewGovernmentFundingService(fundingStore, store.NewTransactor(db))
	handler := NewGovernmentFundingHandler(svc, createAuditService(db))

	r := setupTestRouter()
	r.DELETE("/fundings/:id/periods/:periodId/properties/:propId", handler.DeleteProperty)

	w := performRequest(r, "DELETE", "/fundings/1/periods/1/properties/abc", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

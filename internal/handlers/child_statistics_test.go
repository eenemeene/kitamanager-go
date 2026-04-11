package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildStatisticsHandler_GetAgeDistribution(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create children with different ages
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Child age 3 (born 2022-01-28)
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=%s", org.ID, refDate.Format("2006-01-02")), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 1 {
		t.Errorf("expected total count 1, got %d", response.TotalCount)
	}

	if response.Date != "2025-01-28" {
		t.Errorf("expected date '2025-01-28', got '%s'", response.Date)
	}

	if len(response.Distribution) != 7 {
		t.Errorf("expected 7 buckets, got %d", len(response.Distribution))
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	// No date parameter - should default to today
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	// Just check it returns a valid response with today's date
	if response.Date == "" {
		t.Error("expected date to be set to today")
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid date, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", "/organizations/invalid/statistics/age-distribution", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0, got %d", response.TotalCount)
	}

	// All buckets should have 0
	for _, bucket := range response.Distribution {
		if bucket.Count != 0 {
			t.Errorf("bucket %s should have 0 count, got %d", bucket.AgeLabel, bucket.Count)
		}
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildStatisticsHandler_GetAgeDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org1.ID,
			FirstName:      "Test",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	// Query org2 - should not see org1's children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2025-01-28", org2.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("SECURITY: expected total count 0 for org2 (child in org1), got %d", response.TotalCount)
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_AllBuckets(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create one child for each age bucket (0-6+)
	ages := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	for _, age := range ages {
		child := &models.Child{
			Person: models.Person{
				OrganizationID: org.ID,
				FirstName:      fmt.Sprintf("Age%d", age),
				LastName:       "Child",
				Birthdate:      refDate.AddDate(-age, 0, 0),
			},
		}
		db.Create(child)
		db.Create(&models.ChildContract{
			ChildID: child.ID,
			BaseContract: models.BaseContract{
				SectionID: sectionID,
				Period:    models.Period{From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		})
	}

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	// Total should be 9
	if response.TotalCount != 9 {
		t.Errorf("expected total count 9, got %d", response.TotalCount)
	}

	// Check distribution: 0-5 should have 1 each, 6+ should have 3
	expected := map[string]int{
		"0":  1,
		"1":  1,
		"2":  1,
		"3":  1,
		"4":  1,
		"5":  1,
		"6+": 3, // ages 6, 7, 8
	}

	for _, bucket := range response.Distribution {
		if bucket.Count != expected[bucket.AgeLabel] {
			t.Errorf("bucket %s: expected %d, got %d", bucket.AgeLabel, expected[bucket.AgeLabel], bucket.Count)
		}
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_ExpiredContract(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with expired contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Expired",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	// Query date after contract expired
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0 (contract expired), got %d", response.TotalCount)
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with future contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Future",
			LastName:       "Child",
			Birthdate:      time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}, // Starts in future
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	// Query date before contract starts
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2025-01-28", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 0 {
		t.Errorf("expected total count 0 (contract not started), got %d", response.TotalCount)
	}
}

func TestChildStatisticsHandler_GetAgeDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with historical contract
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      "Historical",
			LastName:       "Child",
			Birthdate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID: sectionID,
			Period:    models.Period{From: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), To: &to},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/age-distribution", handler.GetAgeDistribution)

	// Query historical date when contract was active
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/age-distribution?date=2023-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AgeDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalCount != 1 {
		t.Errorf("expected total count 1 (contract active on historical date), got %d", response.TotalCount)
	}

	// Child should be age 3 on 2023-06-15 (born 2020-01-01)
	for _, bucket := range response.Distribution {
		if bucket.AgeLabel == "3" && bucket.Count != 1 {
			t.Errorf("expected age 3 bucket count 1, got %d", bucket.Count)
		}
	}
}

// =========================================
// GetFunding Tests
// =========================================

func TestChildStatisticsHandler_GetFunding(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)

	// Create child with active contract
	child := &models.Child{
		Person: models.Person{OrganizationID: org.ID, FirstName: "Test", LastName: "Child", Gender: "female", Birthdate: time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	// Create GovernmentFunding matching org state "berlin"
	funding := &models.GovernmentFunding{
		Name:  "Berlin Kita Funding",
		State: "berlin",
	}
	db.Create(funding)

	// Create funding period
	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	db.Create(period)

	// Create funding property
	db.Create(&models.GovernmentFundingProperty{
		PeriodID:    period.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     166847,
		Requirement: 0.261,
	})

	r := setupTestRouter()
	// Register funding route BEFORE per-child routes
	r.GET("/organizations/:orgId/statistics/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/funding", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestChildStatisticsHandler_GetFunding_NoContracts(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/funding", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestChildStatisticsHandler_GetFunding_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/funding", handler.GetFunding)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/funding?date=invalid", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// =============================================================================
// Contract Properties Distribution Tests
// =============================================================================

func TestChildStatisticsHandler_GetContractPropertiesDistribution(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")
	sectionID := ensureTestSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Child 1: care_type=ganztag, supplements=["ndh","mss"]
	child1 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child1", LastName: "Test",
			Gender: "female", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child1)
	db.Create(&models.ChildContract{
		ChildID: child1.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh", "mss"}},
		},
	})

	// Child 2: care_type=halbtag
	child2 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child2", LastName: "Test",
			Gender: "male", Birthdate: time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child2)
	db.Create(&models.ChildContract{
		ChildID: child2.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "halbtag"},
		},
	})

	// Child 3: care_type=ganztag, supplements=["ndh"]
	child3 := &models.Child{
		Person: models.Person{
			OrganizationID: org.ID, FirstName: "Child3", LastName: "Test",
			Gender: "female", Birthdate: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child3)
	db.Create(&models.ChildContract{
		ChildID: child3.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag", "supplements": []string{"ndh"}},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties?date=%s", org.ID, refDate.Format("2006-01-02")), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 3 {
		t.Errorf("expected total_children 3, got %d", response.TotalChildren)
	}

	if response.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", response.Date)
	}

	// Expected properties (sorted by key, then value):
	// care_type/ganztag: 2, care_type/halbtag: 1, supplements/mss: 1, supplements/ndh: 2
	expected := map[string]int{
		"care_type:ganztag": 2,
		"care_type:halbtag": 1,
		"supplements:mss":   1,
		"supplements:ndh":   2,
	}
	if len(response.Properties) != len(expected) {
		t.Errorf("expected %d property entries, got %d", len(expected), len(response.Properties))
	}
	for _, p := range response.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildStatisticsHandler_GetContractPropertiesDistribution_DefaultDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	// No date parameter - should default to today
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.Date == "" {
		t.Error("expected date to be set to today")
	}
}

func TestChildStatisticsHandler_GetContractPropertiesDistribution_CustomDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties?date=2025-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", response.Date)
	}
}

func TestChildStatisticsHandler_GetContractPropertiesDistribution_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties?date=not-a-date", org.ID), nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid date, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildStatisticsHandler_GetContractPropertiesDistribution_InvalidOrgId(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", "/organizations/abc/statistics/contract-properties", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid org ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChildStatisticsHandler_GetContractPropertiesDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org := createTestOrganization(t, db, "Test Org")

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties?date=2025-06-15", org.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 0 {
		t.Errorf("expected total_children 0, got %d", response.TotalChildren)
	}

	if len(response.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(response.Properties))
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildStatisticsHandler_GetContractPropertiesDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	handler := NewStatisticsHandler(createStatisticsService(db))

	org1 := createTestOrganization(t, db, "Org 1")
	sectionID := ensureTestSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	// Create child in org1
	child := &models.Child{
		Person: models.Person{
			OrganizationID: org1.ID, FirstName: "Test", LastName: "Child",
			Gender: "female", Birthdate: time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	db.Create(child)
	db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			SectionID:  sectionID,
			Period:     models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			Properties: models.ContractProperties{"care_type": "ganztag"},
		},
	})

	r := setupTestRouter()
	r.GET("/organizations/:orgId/statistics/contract-properties", handler.GetContractPropertiesDistribution)

	// Query org2 - should not see org1's children
	w := performRequest(r, "GET", fmt.Sprintf("/organizations/%d/statistics/contract-properties?date=2025-01-28", org2.ID), nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ContractPropertiesDistributionResponse
	parseResponse(t, w, &response)

	if response.TotalChildren != 0 {
		t.Errorf("SECURITY: expected total_children 0 for org2 (child in org1), got %d", response.TotalChildren)
	}
}

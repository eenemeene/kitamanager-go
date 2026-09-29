package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// =========================================
// Age Distribution Tests
// =========================================

func TestChildService_GetAgeDistribution_BasicDistribution(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create children with different ages
	// Child age 2 (born 2023-01-28)
	child1 := createTestChild(t, db, "Child1", "Age2", org.ID)
	child1.Birthdate = time.Date(2023, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child1)

	// Child age 3 (born 2022-01-28)
	child2 := createTestChild(t, db, "Child2", "Age3", org.ID)
	child2.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child2)

	// Child age 5 (born 2020-01-28)
	child3 := createTestChild(t, db, "Child3", "Age5", org.ID)
	child3.Birthdate = time.Date(2020, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child3)

	// Create contracts for all children
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, childID := range []uint{child1.ID, child2.ID, child3.ID} {
		_, err := svc.CreateContract(ctx, childID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
		if err != nil {
			t.Fatalf("failed to create contract: %v", err)
		}
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", stats.TotalCount)
	}

	if stats.Date != "2025-01-28" {
		t.Errorf("Date = %s, want 2025-01-28", stats.Date)
	}

	// Check distribution: should have 7 buckets (0-6+)
	if len(stats.Distribution) != 7 {
		t.Errorf("expected 7 buckets, got %d", len(stats.Distribution))
	}

	// Verify specific counts
	expected := map[string]int{
		"0":  0,
		"1":  0,
		"2":  1, // child1
		"3":  1, // child2
		"4":  0,
		"5":  1, // child3
		"6+": 0,
	}

	for _, bucket := range stats.Distribution {
		if bucket.Count != expected[bucket.AgeLabel] {
			t.Errorf("bucket %s: expected count %d, got %d", bucket.AgeLabel, expected[bucket.AgeLabel], bucket.Count)
		}
	}
}

func TestChildService_GetAgeDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", stats.TotalCount)
	}

	// All buckets should have 0 count
	for _, bucket := range stats.Distribution {
		if bucket.Count != 0 {
			t.Errorf("bucket %s: expected count 0, got %d", bucket.AgeLabel, bucket.Count)
		}
	}
}

func TestChildService_GetAgeDistribution_ChildWithNoContract(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child without contract
	child := createTestChild(t, db, "NoContract", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (child has no contract)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_ContractExpiredBeforeDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child with expired contract
	child := createTestChild(t, db, "Expired", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC) // Expired before refDate
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (contract expired)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_ContractStartsAfterDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child with future contract
	child := createTestChild(t, db, "Future", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) // Starts after refDate
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 (contract starts in future)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_OldestBucket(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create children ages 6, 7, 8 - should all be in 6+ bucket
	ages := []int{6, 7, 8}
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, age := range ages {
		child := createTestChild(t, db, "Child", "OldBucket", org.ID)
		child.Birthdate = refDate.AddDate(-age, 0, 0)
		db.Save(child)

		_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
		if err != nil {
			t.Fatalf("failed to create contract: %v", err)
		}
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", stats.TotalCount)
	}

	// Find 6+ bucket
	var sixPlusBucket *models.AgeDistributionBucket
	for i := range stats.Distribution {
		if stats.Distribution[i].AgeLabel == "6+" {
			sixPlusBucket = &stats.Distribution[i]
			break
		}
	}

	if sixPlusBucket == nil {
		t.Fatal("6+ bucket not found")
	}

	if sixPlusBucket.Count != 3 {
		t.Errorf("6+ bucket count = %d, want 3", sixPlusBucket.Count)
	}

	if sixPlusBucket.MaxAge != nil {
		t.Errorf("6+ bucket MaxAge = %v, want nil (open-ended)", sixPlusBucket.MaxAge)
	}
}

func TestChildService_GetAgeDistribution_YoungestBucket(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child age 0 (born recently)
	child := createTestChild(t, db, "Baby", "Child", org.ID)
	child.Birthdate = time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC) // 7 months old
	db.Save(child)

	from := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})
	if err != nil {
		t.Fatalf("failed to create contract: %v", err)
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find age 0 bucket
	var zeroBucket *models.AgeDistributionBucket
	for i := range stats.Distribution {
		if stats.Distribution[i].AgeLabel == "0" {
			zeroBucket = &stats.Distribution[i]
			break
		}
	}

	if zeroBucket == nil {
		t.Fatal("age 0 bucket not found")
	}

	if zeroBucket.Count != 1 {
		t.Errorf("age 0 bucket count = %d, want 1", zeroBucket.Count)
	}
}

func TestChildService_GetAgeDistribution_BirthdayEdgeCase(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Child born 2022-01-28
	child := createTestChild(t, db, "Birthday", "Child", org.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})

	// Test day before birthday - should be age 2
	dayBefore := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	stats, _ := statsSvc.GetAgeDistribution(ctx, org.ID, dayBefore)

	age2Count := 0
	age3Count := 0
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "2" {
			age2Count = bucket.Count
		}
		if bucket.AgeLabel == "3" {
			age3Count = bucket.Count
		}
	}
	if age2Count != 1 {
		t.Errorf("day before birthday: expected age 2 count = 1, got %d", age2Count)
	}
	if age3Count != 0 {
		t.Errorf("day before birthday: expected age 3 count = 0, got %d", age3Count)
	}

	// Test on birthday - should be age 3
	onBirthday := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)
	stats, _ = statsSvc.GetAgeDistribution(ctx, org.ID, onBirthday)

	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "2" {
			age2Count = bucket.Count
		}
		if bucket.AgeLabel == "3" {
			age3Count = bucket.Count
		}
	}
	if age2Count != 0 {
		t.Errorf("on birthday: expected age 2 count = 0, got %d", age2Count)
	}
	if age3Count != 1 {
		t.Errorf("on birthday: expected age 3 count = 1, got %d", age3Count)
	}
}

// SECURITY TEST: Cross-organization isolation
func TestChildService_GetAgeDistribution_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create child in org1 with contract
	child := createTestChild(t, db, "Org1", "Child", org1.ID)
	child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org1.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
	})

	// Query stats for org2 - should not include org1's children
	stats, err := statsSvc.GetAgeDistribution(ctx, org2.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 0 {
		t.Errorf("SECURITY: TotalCount = %d, want 0 (child in different org)", stats.TotalCount)
	}
}

func TestChildService_GetAgeDistribution_MultipleChildrenSameAge(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	// Create 5 children all age 3
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for range 5 {
		child := createTestChild(t, db, "Child", "Age3", org.ID)
		child.Birthdate = time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC) // Age 3
		db.Save(child)

		_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
			SectionID: 1,
			From:      from,
		})
	}

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", stats.TotalCount)
	}

	// Find age 3 bucket
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "3" {
			if bucket.Count != 5 {
				t.Errorf("age 3 bucket count = %d, want 5", bucket.Count)
			}
			break
		}
	}
}

func TestChildService_GetAgeDistribution_BucketMetadata(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC)

	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify bucket metadata
	expectedBuckets := []struct {
		label  string
		minAge int
		maxAge *int
	}{
		{"0", 0, intPtr(0)},
		{"1", 1, intPtr(1)},
		{"2", 2, intPtr(2)},
		{"3", 3, intPtr(3)},
		{"4", 4, intPtr(4)},
		{"5", 5, intPtr(5)},
		{"6+", 6, nil},
	}

	for i, expected := range expectedBuckets {
		bucket := stats.Distribution[i]
		if bucket.AgeLabel != expected.label {
			t.Errorf("bucket %d: AgeLabel = %s, want %s", i, bucket.AgeLabel, expected.label)
		}
		if bucket.MinAge != expected.minAge {
			t.Errorf("bucket %d: MinAge = %d, want %d", i, bucket.MinAge, expected.minAge)
		}
		if expected.maxAge == nil {
			if bucket.MaxAge != nil {
				t.Errorf("bucket %d: MaxAge = %v, want nil", i, bucket.MaxAge)
			}
		} else {
			if bucket.MaxAge == nil || *bucket.MaxAge != *expected.maxAge {
				t.Errorf("bucket %d: MaxAge = %v, want %v", i, bucket.MaxAge, *expected.maxAge)
			}
		}
	}
}

func TestChildService_GetAgeDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Create child born 2020-01-01
	child := createTestChild(t, db, "Historical", "Child", org.ID)
	child.Birthdate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Save(child)

	// Contract from 2022-01-01 to 2023-12-31
	from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateContract(ctx, child.ID, org.ID, &models.ChildContractCreateRequest{
		SectionID: 1,
		From:      from,
		To:        &to,
	})

	// Query for a date when contract was active
	refDate := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	stats, err := statsSvc.GetAgeDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 (contract active on historical date)", stats.TotalCount)
	}

	// Child should be age 3 on 2023-06-15 (born 2020-01-01)
	for _, bucket := range stats.Distribution {
		if bucket.AgeLabel == "3" {
			if bucket.Count != 1 {
				t.Errorf("age 3 bucket count = %d, want 1", bucket.Count)
			}
			break
		}
	}
}

// =============================================================================
// Contract Properties Distribution Tests
// =============================================================================

func TestChildService_GetContractPropertiesDistribution_BasicScalar(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 2 children with care_type=ganztag, 1 with care_type=halbtag
	child1 := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child1.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	child2 := createTestChild(t, db, "Child2", "B", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	child3 := createTestChild(t, db, "Child3", "C", org.ID)
	createTestChildContract(t, db, child3.ID, from, nil, section.ID, models.ContractProperties{"care_type": "halbtag"})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 3 {
		t.Errorf("TotalChildren = %d, want 3", stats.TotalChildren)
	}

	expected := map[string]int{
		"care_type:ganztag": 2,
		"care_type:halbtag": 1,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_ArrayProperties(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with array property supplements=["ndh","mss"]
	child := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"supplements": []string{"ndh", "mss"},
	})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	expected := map[string]int{
		"supplements:mss": 1,
		"supplements:ndh": 1,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_MixedScalarAndArray(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child 1: scalar care_type + array supplements
	child1 := createTestChild(t, db, "Child1", "A", org.ID)
	createTestChildContract(t, db, child1.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh", "mss"},
	})

	// Child 2: scalar care_type + array supplements (overlapping)
	child2 := createTestChild(t, db, "Child2", "B", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
	})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 2 {
		t.Errorf("TotalChildren = %d, want 2", stats.TotalChildren)
	}

	expected := map[string]int{
		"care_type:ganztag": 2,
		"supplements:mss":   1,
		"supplements:ndh":   2,
	}

	if len(stats.Properties) != len(expected) {
		t.Fatalf("expected %d properties, got %d", len(expected), len(stats.Properties))
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_NoChildren(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0", stats.TotalChildren)
	}

	if len(stats.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(stats.Properties))
	}
}

func TestChildService_GetContractPropertiesDistribution_ChildWithNoContract(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Create child without contract
	createTestChild(t, db, "NoContract", "Child", org.ID)

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// FindByOrganizationWithActiveOn only returns children with active contracts
	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (child has no contract)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_ContractWithNoProperties(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with active contract but nil properties
	child := createTestChild(t, db, "NoProps", "Child", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, nil)

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1 (child counted even without properties)", stats.TotalChildren)
	}

	if len(stats.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(stats.Properties))
	}
}

func TestChildService_GetContractPropertiesDistribution_ExpiredContract(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Contract expired before refDate
	child := createTestChild(t, db, "Expired", "Child", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, &to, section.ID, models.ContractProperties{"care_type": "ganztag"})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract expired)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_FutureContract(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Contract starts after refDate
	child := createTestChild(t, db, "Future", "Child", org.ID)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract starts in future)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_MultipleChildrenSameValue(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 5 children all with care_type=ganztag
	for i := range 5 {
		child := createTestChild(t, db, fmt.Sprintf("Child%d", i), "Same", org.ID)
		createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})
	}

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 5 {
		t.Errorf("TotalChildren = %d, want 5", stats.TotalChildren)
	}

	if len(stats.Properties) != 1 {
		t.Fatalf("expected 1 property entry, got %d", len(stats.Properties))
	}

	if stats.Properties[0].Count != 5 {
		t.Errorf("expected count 5, got %d", stats.Properties[0].Count)
	}
}

func TestChildService_GetContractPropertiesDistribution_MultiplePropertiesPerChild(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with 3 different property keys
	child := createTestChild(t, db, "Multi", "Props", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"care_type":   "ganztag",
		"supplements": []string{"ndh"},
		"lunch":       "yes",
	})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	// 3 keys: care_type, lunch, supplements
	if len(stats.Properties) != 3 {
		t.Fatalf("expected 3 property entries, got %d", len(stats.Properties))
	}

	expected := map[string]int{
		"care_type:ganztag": 1,
		"lunch:yes":         1,
		"supplements:ndh":   1,
	}

	for _, p := range stats.Properties {
		key := p.Key + ":" + p.Value
		if expected[key] != p.Count {
			t.Errorf("property %s: expected count %d, got %d", key, expected[key], p.Count)
		}
	}
}

func TestChildService_GetContractPropertiesDistribution_CrossOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	section1 := getDefaultSection(t, db, org1.ID)
	org2 := createTestOrganization(t, db, "Org 2")

	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create child in org1
	child := createTestChild(t, db, "Child1", "Org1", org1.ID)
	createTestChildContract(t, db, child.ID, from, nil, section1.ID, models.ContractProperties{"care_type": "ganztag"})

	// Query org2 - should not see org1's children
	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org2.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (child in different org)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_HistoricalDate(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	// Contract active from 2024-01-01 to 2024-06-30
	child := createTestChild(t, db, "Historical", "Child", org.ID)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, from, &to, section.ID, models.ContractProperties{"care_type": "ganztag"})

	// Query during active period
	activeDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, activeDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1 (contract active on historical date)", stats.TotalChildren)
	}

	// Query after contract ended
	afterDate := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)
	stats, err = statsSvc.GetContractPropertiesDistribution(ctx, org.ID, afterDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 0 {
		t.Errorf("TotalChildren = %d, want 0 (contract expired by this date)", stats.TotalChildren)
	}
}

func TestChildService_GetContractPropertiesDistribution_SortedOutput(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create child with properties that should be sorted: z_key before a_key alphabetically
	child := createTestChild(t, db, "Sort", "Test", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{
		"z_key": "beta",
		"a_key": "alpha",
	})

	// Another child with z_key=alpha (to have multiple values for z_key)
	child2 := createTestChild(t, db, "Sort2", "Test", org.ID)
	createTestChildContract(t, db, child2.ID, from, nil, section.ID, models.ContractProperties{
		"z_key": "alpha",
	})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Expected order: a_key:alpha, z_key:alpha, z_key:beta
	if len(stats.Properties) != 3 {
		t.Fatalf("expected 3 property entries, got %d", len(stats.Properties))
	}

	expectedOrder := []struct {
		Key   string
		Value string
	}{
		{"a_key", "alpha"},
		{"z_key", "alpha"},
		{"z_key", "beta"},
	}

	for i, exp := range expectedOrder {
		if stats.Properties[i].Key != exp.Key || stats.Properties[i].Value != exp.Value {
			t.Errorf("position %d: expected %s:%s, got %s:%s", i, exp.Key, exp.Value, stats.Properties[i].Key, stats.Properties[i].Value)
		}
	}
}

// (Removed TestChildService_GetContractPropertiesDistribution_MultipleActiveContracts.
// The state it set up — two simultaneously active contracts for one child —
// is now prevented at the DB layer by migration 000022's EXCLUDE constraint,
// so the scenario can no longer occur in production or in tests.)

func TestChildService_GetContractPropertiesDistribution_EmptyStringValue(t *testing.T) {
	db := setupTestDB(t)
	statsSvc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	refDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Child with empty string value
	child := createTestChild(t, db, "Empty", "Value", org.ID)
	createTestChildContract(t, db, child.ID, from, nil, section.ID, models.ContractProperties{"care_type": ""})

	stats, err := statsSvc.GetContractPropertiesDistribution(ctx, org.ID, refDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.TotalChildren != 1 {
		t.Errorf("TotalChildren = %d, want 1", stats.TotalChildren)
	}

	if len(stats.Properties) != 1 {
		t.Fatalf("expected 1 property entry, got %d", len(stats.Properties))
	}

	if stats.Properties[0].Key != "care_type" || stats.Properties[0].Value != "" {
		t.Errorf("expected care_type with empty value, got %s:%s", stats.Properties[0].Key, stats.Properties[0].Value)
	}
}

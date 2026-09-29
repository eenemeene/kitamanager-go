package service

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// --- GetOccupancy Tests ---

func TestStatisticsService_GetOccupancy_Basic(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Create funding with age groups and care types
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)

	// care_type properties with age ranges
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 8000, 0.15, 3, 6)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "halbtag", "Halbtag", 6000, 0.20, 0, 2)

	// supplement property
	createTestFundingPropertyFull(t, db, period.ID, "integration", "integration_a", "Integration A", 5000, 0.0, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child 1: born 2022 (age ~2 in Jan 2024), ganztag + integration_a
	child1 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Young", LastName: "Child", Birthdate: time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(child1)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID,
		models.ContractProperties{"care_type": "ganztag", "integration": "integration_a"})

	// Child 2: born 2020 (age ~3 in Jan 2024), ganztag, no supplements
	child2 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Older", LastName: "Child", Birthdate: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child2)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have age groups, care types, and supplement types
	if len(result.AgeGroups) == 0 {
		t.Error("expected age groups")
	}
	if len(result.CareTypes) == 0 {
		t.Error("expected care types")
	}
	if len(result.SupplementTypes) == 0 {
		t.Error("expected supplement types")
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}

	dp := result.DataPoints[0]
	if dp.Total != 2 {
		t.Errorf("expected total=2, got %d", dp.Total)
	}

	// Check supplement count
	if dp.BySupplement["integration_a"] != 1 {
		t.Errorf("expected integration_a=1, got %d", dp.BySupplement["integration_a"])
	}
}

func TestStatisticsService_GetOccupancy_NoFunding(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// No funding configured => should still return without error, empty structure
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result.DataPoints))
	}

	// All totals should be 0
	for _, dp := range result.DataPoints {
		if dp.Total != 0 {
			t.Errorf("expected total=0, got %d for %s", dp.Total, dp.Date)
		}
	}
}

func TestStatisticsService_GetOccupancy_SectionFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section1 := getDefaultSection(t, db, org.ID)
	section2 := createTestSection(t, db, "Krippe", org.ID, false)

	// Child in section1
	child1 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "S1", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child1)
	createTestChildContract(t, db, child1.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section1.ID,
		models.ContractProperties{"care_type": "ganztag"})

	// Child in section2
	child2 := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "S2", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child2)
	createTestChildContract(t, db, child2.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, section2.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Filter section2 => only 1 child
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, &section2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.DataPoints[0].Total != 1 {
		t.Errorf("expected total=1, got %d", result.DataPoints[0].Total)
	}
}

func TestStatisticsService_GetOccupancy_ContractEndsDuringRange(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child with contract ending in Feb 2024
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Leaving", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractEnd := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &contractEnd, section.ID,
		models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Jan=1, Feb=1, Mar=0, Apr=0
	expected := []int{1, 1, 0, 0}
	for i, want := range expected {
		if result.DataPoints[i].Total != want {
			t.Errorf("month %d: total=%d, want %d", i, result.DataPoints[i].Total, want)
		}
	}
}

func TestStatisticsService_GetOccupancy_ChildAgeGroupTransition(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding with two age groups:
	// U3 (ages 0-2): care_type=ganztag
	// Ü3 (ages 3-6): care_type=ganztag
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	toDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &toDate, 39.0)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, period.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child born 2021-06-15: age 2 on 2024-06-01, age 3 on 2024-07-01
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Transition", LastName: "Child", Birthdate: time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// Age group labels: "0/1/2" for ages 0-2, "3+" for ages 3-6
	u3Label := formatAgeGroupLabel(0, 2)
	u6Label := formatAgeGroupLabel(3, 6)

	// Jan-Jun (months 0-5): child is 2 -> counted in U3 age group
	for i := range 6 {
		dp := result.DataPoints[i]
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
		u3Count := dp.ByAgeAndCareType[u3Label]["ganztag"]
		if u3Count != 1 {
			t.Errorf("month %d (%s): U3 ganztag count = %d, want 1", i+1, dp.Date, u3Count)
		}
		u6Count := dp.ByAgeAndCareType[u6Label]["ganztag"]
		if u6Count != 0 {
			t.Errorf("month %d (%s): Ü3 ganztag count = %d, want 0", i+1, dp.Date, u6Count)
		}
	}

	// Jul-Dec (months 6-11): child is 3 -> counted in Ü3 age group
	for i := 6; i < 12; i++ {
		dp := result.DataPoints[i]
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
		u3Count := dp.ByAgeAndCareType[u3Label]["ganztag"]
		if u3Count != 0 {
			t.Errorf("month %d (%s): U3 ganztag count = %d, want 0", i+1, dp.Date, u3Count)
		}
		u6Count := dp.ByAgeAndCareType[u6Label]["ganztag"]
		if u6Count != 1 {
			t.Errorf("month %d (%s): Ü3 ganztag count = %d, want 1", i+1, dp.Date, u6Count)
		}
	}
}

func TestStatisticsService_GetOccupancy_MultiplePeriodsWithDifferentRequirements(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Two funding periods with different care type configurations:
	// Period 1 (Jan-Jun): ganztag + halbtag available for ages 0-6
	// Period 2 (Jul-Dec): ganztag + halbtag available for ages 0-6 (same structure, different values)
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	p1To := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	p1 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &p1To, 39.0)
	createTestFundingPropertyFull(t, db, p1.ID, "care_type", "ganztag", "Ganztag", 10000, 0.25, 0, 6)
	createTestFundingPropertyFull(t, db, p1.ID, "care_type", "halbtag", "Halbtag", 6000, 0.15, 0, 6)

	p2To := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	p2 := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), &p2To, 39.0)
	createTestFundingPropertyFull(t, db, p2.ID, "care_type", "ganztag", "Ganztag", 12000, 0.30, 0, 6)
	createTestFundingPropertyFull(t, db, p2.ID, "care_type", "halbtag", "Halbtag", 7000, 0.18, 0, 6)

	section := getDefaultSection(t, db, org.ID)

	// Child with ganztag contract covering all of 2024
	child := &models.Child{Person: models.Person{OrganizationID: org.ID, FirstName: "Stable", LastName: "Child", Birthdate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}
	db.Create(child)
	contractFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestChildContract(t, db, child.ID, contractFrom, nil, section.ID, models.ContractProperties{"care_type": "ganztag"})

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 12 {
		t.Fatalf("expected 12 data points, got %d", len(result.DataPoints))
	}

	// The child should be counted in every month regardless of which funding period is active.
	// Occupancy counts are about headcount, not funding amounts.
	for i, dp := range result.DataPoints {
		if dp.Total != 1 {
			t.Errorf("month %d (%s): Total = %d, want 1", i+1, dp.Date, dp.Total)
		}
	}

	// Verify the age group structure is derived (from the most recent period per extractOccupancyStructure)
	if len(result.AgeGroups) == 0 {
		t.Error("expected at least one age group")
	}
	if len(result.CareTypes) == 0 {
		t.Error("expected at least one care type")
	}

	// The ganztag count should be 1 for every month in the appropriate age group
	ageLabel := formatAgeGroupLabel(0, 6)
	for i, dp := range result.DataPoints {
		ganztag := dp.ByAgeAndCareType[ageLabel]["ganztag"]
		if ganztag != 1 {
			t.Errorf("month %d (%s): ganztag count in %s = %d, want 1", i+1, dp.Date, ageLabel, ganztag)
		}
	}
}

// Test 7: Occupancy full matrix — multiple children across age groups, care types, and supplements.
func TestStatisticsService_GetOccupancy_FullMatrix(t *testing.T) {
	db := setupTestDB(t)
	svc := createStatisticsService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	db.Model(org).Update("state", "berlin")

	// Funding: U3 (0-2) and Ü3 (3-6) × ganztag + halbtag, plus supplements integration_a, ndh
	funding := createTestGovernmentFunding(t, db, "Berlin Funding")
	fpTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	fp := createTestFundingPeriod(t, db, funding.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), &fpTo, 39.0)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag U3", 200000, 0.25, 0, 2)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "ganztag", "Ganztag Ü3", 100000, 0.15, 3, 6)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "halbtag", "Halbtag U3", 120000, 0.15, 0, 2)
	createTestFundingPropertyFull(t, db, fp.ID, "care_type", "halbtag", "Halbtag Ü3", 80000, 0.10, 3, 6)
	createTestFundingPropertyFull(t, db, fp.ID, "integration", "integration_a", "Integration A", 50000, 0.10, -1, -1)
	createTestFundingPropertyFull(t, db, fp.ID, "supplements", "ndh", "NDH", 30000, 0.05, -1, -1)

	section := getDefaultSection(t, db, org.ID)
	contractFrom := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	type childSpec struct {
		birthYear int
		birthMon  time.Month
		props     models.ContractProperties
	}
	children := []childSpec{
		// Child A: age 1 (U3), ganztag, integration_a
		{2023, 1, models.ContractProperties{"care_type": "ganztag", "integration": "integration_a"}},
		// Child B: age 2 (U3), halbtag, ndh
		{2022, 1, models.ContractProperties{"care_type": "halbtag", "supplements": "ndh"}},
		// Child C: age 2 (U3), ganztag (no supplements)
		{2022, 6, models.ContractProperties{"care_type": "ganztag"}},
		// Child D: age 4 (Ü3), ganztag, ndh
		{2020, 1, models.ContractProperties{"care_type": "ganztag", "supplements": "ndh"}},
		// Child E: age 5 (Ü3), halbtag
		{2019, 1, models.ContractProperties{"care_type": "halbtag"}},
		// Child F: age 3 (Ü3), ganztag, integration_a + ndh
		{2021, 1, models.ContractProperties{"care_type": "ganztag", "integration": "integration_a", "supplements": "ndh"}},
	}

	for i, spec := range children {
		c := &models.Child{Person: models.Person{
			OrganizationID: org.ID,
			FirstName:      string(rune('A' + i)),
			LastName:       "Child",
			Birthdate:      time.Date(spec.birthYear, spec.birthMon, 1, 0, 0, 0, 0, time.UTC),
		}}
		db.Create(c)
		createTestChildContract(t, db, c.ID, contractFrom, nil, section.ID, spec.props)
	}

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetOccupancy(ctx, org.ID, &from, &to, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(result.DataPoints))
	}
	dp := result.DataPoints[0]

	if dp.Total != 6 {
		t.Errorf("Total = %d, want 6", dp.Total)
	}

	u3Label := formatAgeGroupLabel(0, 2) // "0/1/2"
	u6Label := formatAgeGroupLabel(3, 6) // "3+"

	// ByAgeAndCareType checks
	checks := []struct {
		ageLabel string
		careType string
		want     int
	}{
		{u3Label, "ganztag", 2}, // A, C
		{u3Label, "halbtag", 1}, // B
		{u6Label, "ganztag", 2}, // D, F
		{u6Label, "halbtag", 1}, // E
	}
	for _, check := range checks {
		got := dp.ByAgeAndCareType[check.ageLabel][check.careType]
		if got != check.want {
			t.Errorf("ByAgeAndCareType[%s][%s] = %d, want %d", check.ageLabel, check.careType, got, check.want)
		}
	}

	// BySupplement checks
	// integration_a: A, F → 2
	if dp.BySupplement["integration_a"] != 2 {
		t.Errorf("BySupplement[integration_a] = %d, want 2", dp.BySupplement["integration_a"])
	}
	// ndh: B, D, F → 3
	if dp.BySupplement["ndh"] != 3 {
		t.Errorf("BySupplement[ndh] = %d, want 3", dp.BySupplement["ndh"])
	}

	// Verify structure metadata
	if len(result.AgeGroups) != 2 {
		t.Errorf("expected 2 age groups, got %d", len(result.AgeGroups))
	}
	if len(result.CareTypes) != 2 {
		t.Errorf("expected 2 care types, got %d", len(result.CareTypes))
	}
	if len(result.SupplementTypes) != 2 {
		t.Errorf("expected 2 supplement types, got %d", len(result.SupplementTypes))
	}
}

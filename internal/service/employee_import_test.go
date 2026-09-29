package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestEmployeeService_Import_NewEmployee(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "Mustermann",
				Gender:    "male",
				Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						To:            &to,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						Grade:         "S8a",
						Step:          3,
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FirstName != "Max" {
		t.Errorf("FirstName = %v, want Max", results[0].FirstName)
	}
	if len(results[0].Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(results[0].Contracts))
	}
	c := results[0].Contracts[0]
	if c.StaffCategory != "qualified" {
		t.Errorf("StaffCategory = %v, want qualified", c.StaffCategory)
	}
	if c.WeeklyHours != 39 {
		t.Errorf("WeeklyHours = %v, want 39", c.WeeklyHours)
	}
	if c.PayPlanID != payPlan.ID {
		t.Errorf("PayPlanID = %d, want %d", c.PayPlanID, payPlan.ID)
	}
}

func TestEmployeeService_Import_UpsertReplacesContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name
	birthdate := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)

	baseContract := models.EmployeeContractResponse{
		SectionName:   &sectionName,
		PayPlanName:   &payPlanName,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		WeeklyHours:   39,
	}

	// First import: one contract.
	c1 := baseContract
	c1.From = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lisa",
				LastName:  "Schulz",
				Gender:    "female",
				Birthdate: birthdate,
				Contracts: []models.EmployeeContractResponse{c1},
			},
		},
	}
	results1, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if len(results1[0].Contracts) != 1 {
		t.Fatalf("first import: expected 1 contract, got %d", len(results1[0].Contracts))
	}
	employeeID := results1[0].ID

	// Second import: two contracts replace the original.
	c2a := baseContract
	c2a.From = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to2a := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	c2a.To = &to2a

	c2b := baseContract
	c2b.From = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	data2 := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lisa",
				LastName:  "Schulz",
				Gender:    "female",
				Birthdate: birthdate,
				Contracts: []models.EmployeeContractResponse{c2a, c2b},
			},
		},
	}
	results2, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if results2[0].ID != employeeID {
		t.Errorf("expected same employee ID %d, got %d", employeeID, results2[0].ID)
	}
	if len(results2[0].Contracts) != 2 {
		t.Errorf("second import: expected 2 contracts, got %d", len(results2[0].Contracts))
	}
}

func TestEmployeeService_Import_SectionAutoCreation(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	payPlanName := payPlan.Name
	newSectionName := "Elementar"

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Tom",
				LastName:  "Klein",
				Gender:    "male",
				Birthdate: time.Date(1988, 7, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &newSectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify section was auto-created.
	var section models.Section
	if err := db.Where("organization_id = ? AND name = ?", org.ID, newSectionName).First(&section).Error; err != nil {
		t.Fatalf("auto-created section not found: %v", err)
	}
	if results[0].Contracts[0].SectionID != section.ID {
		t.Errorf("contract SectionID = %d, want %d", results[0].Contracts[0].SectionID, section.ID)
	}
}

func TestEmployeeService_Import_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Import(ctx, org.ID, &models.EmployeeImportExportData{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingName(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "",
				Gender:    "male",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Max",
				LastName:  "Mustermann",
				Gender:    "male",
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing birthdate, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingContractFrom(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	payPlanName := payPlan.Name
	sectionName := section.Name

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Jan",
				LastName:  "Bauer",
				Gender:    "male",
				Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing contract From, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_MissingPayPlan(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	sectionName := section.Name
	missingPlan := "NonExistent Plan"

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Kim",
				LastName:  "Lehmann",
				Gender:    "female",
				Birthdate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &missingPlan,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for missing pay plan, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_InvalidStaffCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Lea",
				LastName:  "Richter",
				Gender:    "female",
				Birthdate: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "invalid_category",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for invalid staff category, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_NegativeWeeklyHours(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Nico",
				LastName:  "Wolf",
				Gender:    "male",
				Birthdate: time.Date(1993, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   -5,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for negative weekly hours, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEmployeeService_Import_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createEmployeeService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	payPlan := createTestPayPlanWithCoverage(t, db, "TVöD-SuE", org.ID)
	sectionName := section.Name
	payPlanName := payPlan.Name

	from := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // To before From

	data := &models.EmployeeImportExportData{
		Employees: []models.EmployeeResponse{
			{
				FirstName: "Ole",
				LastName:  "Krause",
				Gender:    "male",
				Birthdate: time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.EmployeeContractResponse{
					{
						From:          from,
						To:            &to,
						SectionName:   &sectionName,
						PayPlanName:   &payPlanName,
						StaffCategory: "qualified",
						WeeklyHours:   39,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for invalid period (to before from), got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

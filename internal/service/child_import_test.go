package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestChildService_Import_NewChild(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	sectionName := section.Name

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Anna",
				LastName:  "Schmidt",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{
						From:        from,
						To:          &to,
						SectionName: &sectionName,
						Properties:  models.ContractProperties{"care_type": "ganztag"},
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
	if results[0].FirstName != "Anna" {
		t.Errorf("FirstName = %v, want Anna", results[0].FirstName)
	}
	if len(results[0].Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(results[0].Contracts))
	}
	if results[0].Contracts[0].Properties["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag", results[0].Contracts[0].Properties["care_type"])
	}
}

func TestChildService_Import_UpsertReplacesContracts(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	sectionName := section.Name

	birthdate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// First import: one contract.
	from1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data1 := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Ben",
				LastName:  "Müller",
				Gender:    "male",
				Birthdate: birthdate,
				Contracts: []models.ChildContractResponse{
					{From: from1, SectionName: &sectionName},
				},
			},
		},
	}
	results1, err := svc.Import(ctx, org.ID, data1)
	if err != nil {
		t.Fatalf("first import: expected no error, got %v", err)
	}
	if len(results1[0].Contracts) != 1 {
		t.Fatalf("first import: expected 1 contract, got %d", len(results1[0].Contracts))
	}
	childID := results1[0].ID

	// Second import: two contracts replace the original.
	from2a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to2a := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	from2b := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	data2 := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Ben",
				LastName:  "Müller",
				Gender:    "male",
				Birthdate: birthdate,
				Contracts: []models.ChildContractResponse{
					{From: from2a, To: &to2a, SectionName: &sectionName},
					{From: from2b, SectionName: &sectionName},
				},
			},
		},
	}
	results2, err := svc.Import(ctx, org.ID, data2)
	if err != nil {
		t.Fatalf("second import: expected no error, got %v", err)
	}
	if results2[0].ID != childID {
		t.Errorf("expected same child ID %d, got %d", childID, results2[0].ID)
	}
	if len(results2[0].Contracts) != 2 {
		t.Errorf("second import: expected 2 contracts, got %d", len(results2[0].Contracts))
	}
}

func TestChildService_Import_SectionAutoCreation(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newSectionName := "Krippe"

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Clara",
				LastName:  "Weber",
				Gender:    "female",
				Birthdate: time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{From: from, SectionName: &newSectionName},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results[0].Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(results[0].Contracts))
	}

	// Verify section was auto-created by checking the contract's section
	var section models.Section
	if err := db.Where("organization_id = ? AND name = ?", org.ID, newSectionName).First(&section).Error; err != nil {
		t.Fatalf("auto-created section not found: %v", err)
	}
	if results[0].Contracts[0].SectionID != section.ID {
		t.Errorf("contract SectionID = %d, want %d", results[0].Contracts[0].SectionID, section.ID)
	}
}

func TestChildService_Import_AutoApplyProperties(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)
	sectionName := section.Name
	setupAutoApplyFunding(t, db)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Dana",
				LastName:  "Fischer",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{
						From:        from,
						SectionName: &sectionName,
						Properties:  models.ContractProperties{"care_type": "ganztag"},
					},
				},
			},
		},
	}

	results, err := svc.Import(ctx, org.ID, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	props := results[0].Contracts[0].Properties
	if props["care_type"] != "ganztag" {
		t.Errorf("care_type = %v, want ganztag", props["care_type"])
	}
	if props["parent"] != "meals" {
		t.Errorf("parent = %v, want meals (auto-applied via import)", props["parent"])
	}
}

func TestChildService_Import_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.Import(ctx, org.ID, &models.ChildImportExportData{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_Import_MissingName(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "",
				LastName:  "Schmidt",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
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

func TestChildService_Import_MissingBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Eva",
				LastName:  "Braun",
				Gender:    "female",
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

func TestChildService_Import_MissingContractFrom(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Finn",
				LastName:  "Becker",
				Gender:    "male",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{SectionID: section.ID},
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

// TestChildService_Import_ContractBeforeBirthdate verifies that Import() delegates contract
// creation to CreateContract(), inheriting its birthdate validation.
func TestChildService_Import_ContractBeforeBirthdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Early",
				LastName:  "Contract",
				Birthdate: time.Date(2020, 3, 10, 0, 0, 0, 0, time.UTC),
				Gender:    "female",
				Contracts: []models.ChildContractResponse{
					{
						From:      time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
						SectionID: section.ID,
					},
				},
			},
		},
	}

	_, err := svc.Import(ctx, org.ID, data)
	if err == nil {
		t.Fatal("expected error for contract before birthdate, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestChildService_Import_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := getDefaultSection(t, db, org.ID)

	from := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // To before From

	data := &models.ChildImportExportData{
		Children: []models.ChildResponse{
			{
				FirstName: "Greta",
				LastName:  "Hoffmann",
				Gender:    "female",
				Birthdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Contracts: []models.ChildContractResponse{
					{From: from, To: &to, SectionID: section.ID},
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

func TestChildService_Import_RoundTripsSchoolEntryDate(t *testing.T) {
	db := setupTestDB(t)
	svc := createChildService(db)
	ctx := context.Background()
	org := createTestOrganization(t, db, "Test Org")

	bd, _ := time.Parse(models.DateFormat, "2020-05-15")
	imported, err := svc.Import(ctx, org.ID, &models.ChildImportExportData{
		Children: []models.ChildResponse{{
			FirstName: "Mira", LastName: "Sonnenschein", Gender: "female",
			Birthdate:       bd,
			SchoolEntryDate: schoolEntry(t, "2027-08-01"),
		}},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("imported %d children, want 1", len(imported))
	}
	if imported[0].SchoolEntryDate == nil {
		t.Fatal("import dropped SchoolEntryDate")
	}
	if got := imported[0].SchoolEntryDate.Format(models.DateFormat); got != "2027-08-01" {
		t.Errorf("SchoolEntryDate = %s, want 2027-08-01", got)
	}
}

package seed

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// SeedAdmin creates an initial admin user if SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD are set.
// If the user already exists, it will be skipped.
// The user will be assigned the superadmin role (in database).
func SeedAdmin(cfg *config.Config, userStore *store.UserStore, userGroupStore *store.UserGroupStore, enforcer *rbac.Enforcer) error {
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminPassword == "" {
		slog.Info("Admin seeding skipped: SEED_ADMIN_EMAIL or SEED_ADMIN_PASSWORD not set")
		return nil
	}

	// Check if user already exists
	existingUser, err := userStore.FindByEmail(cfg.SeedAdminEmail)
	if err == nil && existingUser != nil {
		slog.Info("Admin user already exists", "email", cfg.SeedAdminEmail)

		// Ensure superadmin is set in database
		if !existingUser.IsSuperAdmin {
			if err := userGroupStore.SetSuperAdmin(existingUser.ID, true); err != nil {
				slog.Warn("Failed to ensure superadmin status in database", "error", err)
			} else {
				slog.Info("Superadmin status set in database", "userId", existingUser.ID)
			}
		}

		// Also keep Casbin assignment for backwards compatibility during migration
		if err := enforcer.AssignSuperAdmin(existingUser.ID); err != nil {
			slog.Warn("Failed to ensure superadmin role in Casbin", "error", err)
		}
		return nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create admin user with superadmin flag
	user := &models.User{
		Name:         cfg.SeedAdminName,
		Email:        cfg.SeedAdminEmail,
		Password:     string(hashedPassword),
		Active:       true,
		IsSuperAdmin: true,
		CreatedBy:    "system",
	}

	if err := userStore.Create(user); err != nil {
		return err
	}

	slog.Info("Admin user created", "email", cfg.SeedAdminEmail, "id", user.ID)

	// Also assign superadmin role in Casbin for backwards compatibility during migration
	if err := enforcer.AssignSuperAdmin(user.ID); err != nil {
		slog.Warn("Failed to assign superadmin role in Casbin", "error", err)
	}

	slog.Info("Superadmin role assigned", "userId", user.ID)

	return nil
}

// SeedGovernmentFunding imports a government funding from YAML if GOVERNMENT_FUNDING_SEED_PATH is set.
// If the government funding already exists, it will be skipped.
func SeedGovernmentFunding(cfg *config.Config, db *gorm.DB, fundingStore *store.GovernmentFundingStore) error {
	if cfg.GovernmentFundingSeedPath == "" {
		slog.Info("Government funding seeding skipped: GOVERNMENT_FUNDING_SEED_PATH not set")
		return nil
	}

	governmentFundingImporter := importer.NewGovernmentFundingImporter(db, fundingStore)

	fundingID, err := governmentFundingImporter.ImportGovernmentFundingFromFile(cfg.GovernmentFundingSeedPath, cfg.GovernmentFundingSeedName)
	if err != nil {
		if errors.Is(err, importer.ErrGovernmentFundingExists) {
			slog.Info("Government funding already seeded", "name", cfg.GovernmentFundingSeedName, "id", fundingID)
			return nil
		}
		return err
	}

	slog.Info("Government funding seeded successfully", "name", cfg.GovernmentFundingSeedName, "id", fundingID, "path", cfg.GovernmentFundingSeedPath)
	return nil
}

// German first names for children
var firstNames = []string{
	"Emma", "Mia", "Hannah", "Sofia", "Emilia", "Lina", "Anna", "Marie", "Lea", "Lena",
	"Ben", "Paul", "Leon", "Finn", "Elias", "Noah", "Luis", "Felix", "Lukas", "Max",
	"Clara", "Ella", "Mila", "Amelie", "Emily", "Lara", "Laura", "Johanna", "Nele", "Sarah",
	"Jonas", "Henry", "Theo", "Moritz", "Oskar", "Emil", "Anton", "Jakob", "David", "Julian",
	"Charlotte", "Frieda", "Greta", "Ida", "Mathilda", "Paula", "Rosa", "Victoria", "Helena", "Lilly",
}

// German last names
var lastNames = []string{
	"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker", "Schulz", "Hoffmann",
	"Schäfer", "Koch", "Bauer", "Richter", "Klein", "Wolf", "Schröder", "Neumann", "Schwarz", "Zimmermann",
	"Braun", "Krüger", "Hofmann", "Hartmann", "Lange", "Schmitt", "Werner", "Schmitz", "Krause", "Meier",
}

// Contract attribute combinations
// These must match the property names in the Berlin government funding YAML
var attributeCombinations = [][]string{
	{"ganztag"},
	{"ganztag", "ndh"},
	{"ganztag", "integration a"},
	{"ganztag", "ndh", "integration a"},
	{"halbtag"},
	{"halbtag", "ndh"},
	{"teilzeit"},
	{"teilzeit", "ndh"},
}

// SeedTestData creates test data for development:
// - Berlin government funding plan
// - Organization "Kita Sonnenschein" with Berlin funding assigned
// - Manager user "manager@example.com" (password: "supersecret")
// - 50 children distributed by age with contracts
func SeedTestData(cfg *config.Config, db *gorm.DB, fundingStore *store.GovernmentFundingStore) error {
	if !cfg.SeedTestData {
		slog.Info("Test data seeding skipped: SEED_TEST_DATA not set to true")
		return nil
	}

	// Check if test org already exists
	var existingOrg models.Organization
	if err := db.Where("name = ?", "Kita Sonnenschein").First(&existingOrg).Error; err == nil {
		slog.Info("Test organization already exists", "name", existingOrg.Name, "id", existingOrg.ID)
		return nil
	}

	slog.Info("Seeding test data...")

	// Import Berlin government funding plan
	var fundingID *uint
	governmentFundingImporter := importer.NewGovernmentFundingImporter(db, fundingStore)
	id, err := governmentFundingImporter.ImportGovernmentFundingFromFile("configs/government-fundings/berlin.yaml", "Berlin")
	if err != nil {
		if errors.Is(err, importer.ErrGovernmentFundingExists) {
			slog.Info("Berlin government funding already exists", "id", id)
			fundingID = &id
		} else {
			return fmt.Errorf("failed to import Berlin government funding: %w", err)
		}
	} else {
		slog.Info("Berlin government funding imported", "id", id)
		fundingID = &id
	}

	// Create organization with Berlin funding
	org := &models.Organization{
		Name:                "Kita Sonnenschein",
		Active:              true,
		GovernmentFundingID: fundingID,
		CreatedBy:           "seed",
	}
	if err := db.Create(org).Error; err != nil {
		return err
	}
	slog.Info("Created test organization", "name", org.Name, "id", org.ID, "fundingId", fundingID)

	// Create default group for the organization
	group := &models.Group{
		Name:           "Mitarbeiter",
		OrganizationID: org.ID,
		IsDefault:      true,
		Active:         true,
		CreatedBy:      "seed",
	}
	if err := db.Create(group).Error; err != nil {
		return err
	}
	slog.Info("Created test group", "name", group.Name, "id", group.ID)

	// Create manager user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Check if manager user already exists
	var existingUser models.User
	if err := db.Where("email = ?", "manager@example.com").First(&existingUser).Error; err == nil {
		slog.Info("Manager user already exists", "email", existingUser.Email)
	} else {
		manager := &models.User{
			Name:      "Manager",
			Email:     "manager@example.com",
			Password:  string(hashedPassword),
			Active:    true,
			CreatedBy: "seed",
		}
		if err := db.Create(manager).Error; err != nil {
			return err
		}
		existingUser = *manager
		slog.Info("Created manager user", "email", manager.Email, "id", manager.ID)
	}

	// Add manager to group with manager role
	userGroup := &models.UserGroup{
		UserID:    existingUser.ID,
		GroupID:   group.ID,
		Role:      models.RoleManager,
		CreatedBy: "seed",
	}
	if err := db.Create(userGroup).Error; err != nil {
		slog.Warn("Failed to add manager to group (may already exist)", "error", err)
	} else {
		slog.Info("Added manager to group", "userId", existingUser.ID, "groupId", group.ID, "role", models.RoleManager)
	}

	// Create 50 children with age distribution
	children := createTestChildren(org.ID, 50)
	for i := range children {
		if err := db.Create(&children[i]).Error; err != nil {
			return err
		}
	}
	slog.Info("Created test children", "count", len(children))

	// Create contracts for all children
	// ~30% of children get multiple contracts (contract history)
	contractCount := 0
	for i, child := range children {
		contracts := createTestContracts(child.ID, child.Birthdate, i%3 == 0) // every 3rd child gets history
		for _, contract := range contracts {
			if err := db.Create(&contract).Error; err != nil {
				return err
			}
			contractCount++
		}
	}
	slog.Info("Created test contracts", "count", contractCount)

	slog.Info("Test data seeding completed",
		"organization", org.Name,
		"managerEmail", "manager@example.com",
		"managerPassword", "supersecret",
		"childrenCount", len(children),
	)

	return nil
}

//nolint:gosec // G404: math/rand is fine for test data generation
func createTestChildren(orgID uint, count int) []models.Child {
	children := make([]models.Child, count)
	now := time.Now()

	// Age distribution for a typical Kita:
	// 0-1 years: 10%, 1-2 years: 15%, 2-3 years: 20%,
	// 3-4 years: 20%, 4-5 years: 20%, 5-6 years: 15%
	ageDistribution := []struct {
		minMonths int
		maxMonths int
		percent   int
	}{
		{6, 12, 10},
		{12, 24, 15},
		{24, 36, 20},
		{36, 48, 20},
		{48, 60, 20},
		{60, 72, 15},
	}

	idx := 0
	for _, dist := range ageDistribution {
		childrenInGroup := count * dist.percent / 100
		for i := 0; i < childrenInGroup && idx < count; i++ {
			ageMonths := dist.minMonths + rand.Intn(dist.maxMonths-dist.minMonths)
			birthdate := now.AddDate(0, -ageMonths, -rand.Intn(28))

			children[idx] = models.Child{
				Person: models.Person{
					OrganizationID: orgID,
					FirstName:      firstNames[rand.Intn(len(firstNames))],
					LastName:       lastNames[rand.Intn(len(lastNames))],
					Birthdate:      birthdate,
				},
			}
			idx++
		}
	}

	// Fill remaining slots
	for idx < count {
		ageMonths := 24 + rand.Intn(36)
		birthdate := now.AddDate(0, -ageMonths, -rand.Intn(28))

		children[idx] = models.Child{
			Person: models.Person{
				OrganizationID: orgID,
				FirstName:      firstNames[rand.Intn(len(firstNames))],
				LastName:       lastNames[rand.Intn(len(lastNames))],
				Birthdate:      birthdate,
			},
		}
		idx++
	}

	return children
}

//nolint:gosec // G404: math/rand is fine for test data generation
func createTestContracts(childID uint, birthdate time.Time, withHistory bool) []models.ChildContract {
	now := time.Now()

	// First contract starts 6-18 months after birth
	contractStart := birthdate.AddDate(0, 6+rand.Intn(12), 0)
	contractStart = time.Date(contractStart.Year(), contractStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	if contractStart.After(now) {
		contractStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if !withHistory {
		// Single contract (open-ended)
		return []models.ChildContract{{
			ChildID: childID,
			Period: models.Period{
				From: contractStart,
				To:   nil,
			},
			Attributes: attributeCombinations[rand.Intn(len(attributeCombinations))],
		}}
	}

	// Create 2-3 contracts with history
	numContracts := 2 + rand.Intn(2) // 2 or 3 contracts
	contracts := make([]models.ChildContract, 0, numContracts)

	currentStart := contractStart
	for i := 0; i < numContracts; i++ {
		// Each contract lasts 6-18 months
		duration := 6 + rand.Intn(12)
		contractEnd := currentStart.AddDate(0, duration, 0)
		contractEnd = time.Date(contractEnd.Year(), contractEnd.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1) // last day of prev month

		isLast := i == numContracts-1
		if contractEnd.After(now) || isLast {
			// Last contract or would end in future: make it open-ended
			contracts = append(contracts, models.ChildContract{
				ChildID: childID,
				Period: models.Period{
					From: currentStart,
					To:   nil,
				},
				Attributes: attributeCombinations[rand.Intn(len(attributeCombinations))],
			})
			break
		}

		contracts = append(contracts, models.ChildContract{
			ChildID: childID,
			Period: models.Period{
				From: currentStart,
				To:   &contractEnd,
			},
			Attributes: attributeCombinations[rand.Intn(len(attributeCombinations))],
		})

		// Next contract starts the day after
		currentStart = contractEnd.AddDate(0, 0, 1)
	}

	return contracts
}

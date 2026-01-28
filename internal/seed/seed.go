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

// randInt returns a random integer in [0, n) for test data generation.
// #nosec G404 - math/rand is acceptable for non-security test data
func randInt(n int) int {
	return rand.Intn(n)
}

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

	fundingID, err := governmentFundingImporter.ImportGovernmentFundingFromFile(cfg.GovernmentFundingSeedPath, cfg.GovernmentFundingSeedState)
	if err != nil {
		if errors.Is(err, importer.ErrGovernmentFundingExists) {
			slog.Info("Government funding already seeded", "state", cfg.GovernmentFundingSeedState, "id", fundingID)
			return nil
		}
		return err
	}

	slog.Info("Government funding seeded successfully", "state", cfg.GovernmentFundingSeedState, "id", fundingID, "path", cfg.GovernmentFundingSeedPath)
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
// - Test users with different roles (all with password "supersecret"):
//   - superadmin@example.com (superadmin - full system access)
//   - admin@example.com (admin role in organization)
//   - manager@example.com (manager role in organization)
//
// - 200 children distributed over the last 4 years with contracts
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
	governmentFundingImporter := importer.NewGovernmentFundingImporter(db, fundingStore)
	id, err := governmentFundingImporter.ImportGovernmentFundingFromFile("configs/government-fundings/berlin.yaml", "berlin")
	if err != nil {
		if errors.Is(err, importer.ErrGovernmentFundingExists) {
			slog.Info("Berlin government funding already exists", "id", id)
		} else {
			return fmt.Errorf("failed to import Berlin government funding: %w", err)
		}
	} else {
		slog.Info("Berlin government funding imported", "id", id)
	}

	// Create organization with Berlin state (funding is looked up by state automatically)
	org := &models.Organization{
		Name:      "Kita Sonnenschein",
		Active:    true,
		State:     string(models.StateBerlin),
		CreatedBy: "seed",
	}
	if err := db.Create(org).Error; err != nil {
		return err
	}
	slog.Info("Created test organization", "name", org.Name, "id", org.ID, "state", org.State)

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

	// Hash password for all test users
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Define test users with their roles
	testUsers := []struct {
		name         string
		email        string
		isSuperAdmin bool
		groupRole    models.Role // empty string means no group membership
	}{
		{"Super Admin", "superadmin@example.com", true, ""},
		{"Admin", "admin@example.com", false, models.RoleAdmin},
		{"Manager", "manager@example.com", false, models.RoleManager},
	}

	// Create test users
	for _, tu := range testUsers {
		var user models.User
		if err := db.Where("email = ?", tu.email).First(&user).Error; err == nil {
			slog.Info("User already exists", "email", user.Email)
		} else {
			user = models.User{
				Name:         tu.name,
				Email:        tu.email,
				Password:     string(hashedPassword),
				Active:       true,
				IsSuperAdmin: tu.isSuperAdmin,
				CreatedBy:    "seed",
			}
			if err := db.Create(&user).Error; err != nil {
				return err
			}
			slog.Info("Created user", "email", user.Email, "id", user.ID, "isSuperAdmin", tu.isSuperAdmin)
		}

		// Add user to group with specified role (if applicable)
		if tu.groupRole != "" {
			userGroup := &models.UserGroup{
				UserID:    user.ID,
				GroupID:   group.ID,
				Role:      tu.groupRole,
				CreatedBy: "seed",
			}
			if err := db.Create(userGroup).Error; err != nil {
				slog.Warn("Failed to add user to group (may already exist)", "email", tu.email, "error", err)
			} else {
				slog.Info("Added user to group", "email", tu.email, "groupId", group.ID, "role", tu.groupRole)
			}
		}
	}

	// Create 200 children distributed over the last 4 years
	children := createTestChildren(org.ID, 200)
	for i := range children {
		if err := db.Create(&children[i]).Error; err != nil {
			return err
		}
	}
	slog.Info("Created test children", "count", len(children))

	// Create contracts for all children distributed over 4 years
	// Some children have left (contracts ended), some are current
	contractCount := 0
	for i, child := range children {
		contracts := createTestContractsDistributed(child.ID, child.Birthdate, i)
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
		"users", "superadmin@example.com, admin@example.com, manager@example.com",
		"password", "supersecret",
		"childrenCount", len(children),
	)

	return nil
}

//nolint:gosec // G404: math/rand is fine for test data generation
func createTestChildren(orgID uint, count int) []models.Child {
	children := make([]models.Child, count)
	now := time.Now()

	// For 200 children distributed over 4 years, we need birthdates going back further
	// Children can be 0-10 years old to cover those who started 4 years ago and have since left
	// Age distribution:
	// 0-1 years: 5%, 1-2 years: 10%, 2-3 years: 15%, 3-4 years: 20%,
	// 4-5 years: 15%, 5-6 years: 15%, 6-8 years: 15%, 8-10 years: 5%
	ageDistribution := []struct {
		minMonths int
		maxMonths int
		percent   int
	}{
		{6, 12, 5},
		{12, 24, 10},
		{24, 36, 15},
		{36, 48, 20},
		{48, 60, 15},
		{60, 72, 15},
		{72, 96, 15},
		{96, 120, 5},
	}

	idx := 0
	for _, dist := range ageDistribution {
		childrenInGroup := count * dist.percent / 100
		for i := 0; i < childrenInGroup && idx < count; i++ {
			ageMonths := dist.minMonths + randInt(dist.maxMonths-dist.minMonths)
			birthdate := now.AddDate(0, -ageMonths, -randInt(28))

			children[idx] = models.Child{
				Person: models.Person{
					OrganizationID: orgID,
					FirstName:      firstNames[randInt(len(firstNames))],
					LastName:       lastNames[randInt(len(lastNames))],
					Birthdate:      birthdate,
				},
			}
			idx++
		}
	}

	// Fill remaining slots
	for idx < count {
		ageMonths := 24 + randInt(60)
		birthdate := now.AddDate(0, -ageMonths, -randInt(28))

		children[idx] = models.Child{
			Person: models.Person{
				OrganizationID: orgID,
				FirstName:      firstNames[randInt(len(firstNames))],
				LastName:       lastNames[randInt(len(lastNames))],
				Birthdate:      birthdate,
			},
		}
		idx++
	}

	return children
}

// createTestContractsDistributed creates contracts for children distributed over the last 4 years.
// - Some children started years ago and have left (ended contracts on July 31st - typical Kita exit)
// - Some children are currently enrolled (ongoing contracts)
// - ~30% of children have multiple contracts (contract history)
//
//nolint:gosec // G404: math/rand is fine for test data generation
func createTestContractsDistributed(childID uint, birthdate time.Time, childIndex int) []models.ChildContract {
	now := time.Now()

	// Determine when this child's contract should start based on their index
	// Distribute contract starts over the last 4 years (48 months)
	monthsAgo := randInt(48) // Random start within last 4 years

	// Contract must start at least 6 months after birth
	earliestStart := birthdate.AddDate(0, 6, 0)
	contractStart := now.AddDate(0, -monthsAgo, 0)
	contractStart = time.Date(contractStart.Year(), contractStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	if contractStart.Before(earliestStart) {
		contractStart = time.Date(earliestStart.Year(), earliestStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if contractStart.After(now) {
		contractStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Calculate child's current age in months
	childAgeMonths := int(now.Sub(birthdate).Hours() / 24 / 30)

	// Determine if child has left (contract ended) or is still enrolled
	// Children over 6 years old (72 months) have typically left for school
	hasLeft := false
	if childAgeMonths > 72 {
		hasLeft = randInt(100) < 90 // 90% of school-age children have left
	} else if childAgeMonths > 60 {
		hasLeft = randInt(100) < 30 // 30% of 5-6 year olds have left
	}

	withHistory := childIndex%3 == 0 // ~30% get contract history

	if !withHistory {
		if hasLeft {
			// Contract ended on July 31st (typical Kita exit date)
			contractEnd := findJuly31stForExit(contractStart, now, birthdate)

			// Ensure end is after start
			if !contractEnd.After(contractStart) {
				contractEnd = time.Date(contractStart.Year(), time.July, 31, 0, 0, 0, 0, time.UTC)
				if !contractEnd.After(contractStart) {
					contractEnd = time.Date(contractStart.Year()+1, time.July, 31, 0, 0, 0, 0, time.UTC)
				}
			}

			return []models.ChildContract{{
				ChildID: childID,
				Period: models.Period{
					From: contractStart,
					To:   &contractEnd,
				},
				Attributes: attributeCombinations[randInt(len(attributeCombinations))],
			}}
		}

		// Active contract (open-ended)
		return []models.ChildContract{{
			ChildID: childID,
			Period: models.Period{
				From: contractStart,
				To:   nil,
			},
			Attributes: attributeCombinations[randInt(len(attributeCombinations))],
		}}
	}

	// Create 2-3 contracts with history
	numContracts := 2 + randInt(2)
	contracts := make([]models.ChildContract, 0, numContracts)

	currentStart := contractStart
	for i := 0; i < numContracts; i++ {
		isLast := i == numContracts-1

		if isLast {
			if hasLeft {
				// Last contract ended on July 31st
				contractEnd := findJuly31stForExit(currentStart, now, birthdate)
				if !contractEnd.After(currentStart) {
					contractEnd = time.Date(currentStart.Year()+1, time.July, 31, 0, 0, 0, 0, time.UTC)
				}
				contracts = append(contracts, models.ChildContract{
					ChildID: childID,
					Period: models.Period{
						From: currentStart,
						To:   &contractEnd,
					},
					Attributes: attributeCombinations[randInt(len(attributeCombinations))],
				})
			} else {
				// Last contract is open-ended (still enrolled)
				contracts = append(contracts, models.ChildContract{
					ChildID: childID,
					Period: models.Period{
						From: currentStart,
						To:   nil,
					},
					Attributes: attributeCombinations[randInt(len(attributeCombinations))],
				})
			}
			break
		}

		// Non-last contracts end on July 31st of some year (contract renewals)
		contractEnd := time.Date(currentStart.Year(), time.July, 31, 0, 0, 0, 0, time.UTC)
		if !contractEnd.After(currentStart) {
			contractEnd = time.Date(currentStart.Year()+1, time.July, 31, 0, 0, 0, 0, time.UTC)
		}

		if contractEnd.After(now) {
			// Would end in future, just make it open-ended
			contracts = append(contracts, models.ChildContract{
				ChildID: childID,
				Period: models.Period{
					From: currentStart,
					To:   nil,
				},
				Attributes: attributeCombinations[randInt(len(attributeCombinations))],
			})
			break
		}

		contracts = append(contracts, models.ChildContract{
			ChildID: childID,
			Period: models.Period{
				From: currentStart,
				To:   &contractEnd,
			},
			Attributes: attributeCombinations[randInt(len(attributeCombinations))],
		})

		// Next contract starts August 1st
		currentStart = time.Date(contractEnd.Year(), time.August, 1, 0, 0, 0, 0, time.UTC)
	}

	return contracts
}

// findJuly31stForExit finds the appropriate July 31st exit date for a child.
// Children typically leave when they turn ~6 years old and start school.
func findJuly31stForExit(contractStart, now time.Time, birthdate time.Time) time.Time {
	// Child would typically leave at age 6 (school start)
	// Find the July 31st when the child is around 6 years old
	schoolStartYear := birthdate.Year() + 6

	// If child was born after July, they'd start school a year later
	if birthdate.Month() > time.July {
		schoolStartYear++
	}

	exitDate := time.Date(schoolStartYear, time.July, 31, 0, 0, 0, 0, time.UTC)

	// Make sure exit date is after contract start and before now
	if exitDate.Before(contractStart) || exitDate.After(now) {
		// Find the most recent July 31st before now
		exitDate = time.Date(now.Year(), time.July, 31, 0, 0, 0, 0, time.UTC)
		if exitDate.After(now) {
			exitDate = time.Date(now.Year()-1, time.July, 31, 0, 0, 0, 0, time.UTC)
		}
	}

	return exitDate
}

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

// randomGender returns a random gender for test data.
// Distribution: ~49% male, ~49% female, ~2% diverse
//
//nolint:gosec // G404: math/rand is fine for test data generation
func randomGender() string {
	r := rand.Intn(100)
	if r < 49 {
		return string(models.GenderMale)
	} else if r < 98 {
		return string(models.GenderFemale)
	}
	return string(models.GenderDiverse)
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

	// Create default section for the organization
	section := &models.Section{
		Name:           "Unassigned",
		OrganizationID: org.ID,
		IsDefault:      true,
		CreatedBy:      "seed",
	}
	if err := db.Create(section).Error; err != nil {
		return err
	}
	slog.Info("Created default section", "name", section.Name, "id", section.ID)

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
	children := createTestChildren(org.ID, section.ID, 200)
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

	// Create TVöD-SuE PayPlan with current rates
	payPlan := &models.PayPlan{
		OrganizationID: org.ID,
		Name:           "TVöD-SuE 2024",
	}
	if err := db.Create(payPlan).Error; err != nil {
		return err
	}
	slog.Info("Created PayPlan", "name", payPlan.Name, "id", payPlan.ID)

	// Create current pay period (valid from 2024-01-01)
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	payPeriod := &models.PayPlanPeriod{
		PayPlanID:   payPlan.ID,
		From:        periodStart,
		To:          nil, // ongoing
		WeeklyHours: 39.0,
	}
	if err := db.Create(payPeriod).Error; err != nil {
		return err
	}
	slog.Info("Created PayPlan period", "from", payPeriod.From, "weeklyHours", payPeriod.WeeklyHours)

	// Create pay entries for common grades (S3-S18)
	// TVöD-SuE 2024 approximate rates
	payEntries := []struct {
		grade  string
		step   int
		amount int // cents
	}{
		// S8a (Erzieher)
		{"S8a", 1, 314847}, {"S8a", 2, 329947}, {"S8a", 3, 350089},
		{"S8a", 4, 365134}, {"S8a", 5, 385229}, {"S8a", 6, 398317},
		// S8b (Erzieher mit schwieriger Tätigkeit)
		{"S8b", 1, 339902}, {"S8b", 2, 354655}, {"S8b", 3, 370125},
		{"S8b", 4, 385592}, {"S8b", 5, 401058}, {"S8b", 6, 416526},
		// S4 (Kinderpfleger)
		{"S4", 1, 267400}, {"S4", 2, 282700}, {"S4", 3, 298000},
		{"S4", 4, 313300}, {"S4", 5, 328600}, {"S4", 6, 343900},
		// S9 (Sozialarbeiter)
		{"S9", 1, 344800}, {"S9", 2, 360100}, {"S9", 3, 385200},
		{"S9", 4, 400500}, {"S9", 5, 420700}, {"S9", 6, 435000},
	}
	for _, e := range payEntries {
		entry := &models.PayPlanEntry{
			PeriodID:      payPeriod.ID,
			Grade:         e.grade,
			Step:          e.step,
			MonthlyAmount: e.amount,
		}
		if err := db.Create(entry).Error; err != nil {
			return err
		}
	}
	slog.Info("Created PayPlan entries", "count", len(payEntries))

	// Create employees
	employees := createTestEmployees(org.ID, section.ID, 20)
	for i := range employees {
		if err := db.Create(&employees[i]).Error; err != nil {
			return err
		}
	}
	slog.Info("Created test employees", "count", len(employees))

	// Create employee contracts with varied scenarios:
	// - Some with single current contract
	// - Some with multiple contracts (past + current)
	// - Some with only past contracts (left the organization)
	// - Some with future contracts (starting soon)
	now := time.Now()
	employeeContractCount := 0

	// Employee 0: Single current contract (started 2 years ago)
	if err := createEmployeeContract(db, employees[0].ID, "Kinderpfleger", "S4", 2, 30,
		now.AddDate(-2, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 1: Multiple contracts - past contract ended, current ongoing
	pastEnd := now.AddDate(-1, 0, 0)
	if err := createEmployeeContract(db, employees[1].ID, "Praktikant", "S4", 1, 20,
		now.AddDate(-3, 0, 0), &pastEnd); err != nil {
		return err
	}
	employeeContractCount++
	if err := createEmployeeContract(db, employees[1].ID, "Erzieher", "S8a", 3, 39,
		now.AddDate(-1, 0, 1), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 2: Three contracts showing career progression
	end1 := now.AddDate(-4, 0, 0)
	if err := createEmployeeContract(db, employees[2].ID, "Auszubildende", "S4", 1, 39,
		now.AddDate(-6, 0, 0), &end1); err != nil {
		return err
	}
	employeeContractCount++
	end2 := now.AddDate(-1, 0, 0)
	if err := createEmployeeContract(db, employees[2].ID, "Erzieher", "S8a", 2, 39,
		now.AddDate(-4, 0, 1), &end2); err != nil {
		return err
	}
	employeeContractCount++
	if err := createEmployeeContract(db, employees[2].ID, "Erzieher", "S8a", 4, 39,
		now.AddDate(-1, 0, 1), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 3: Only past contract (left the organization)
	leftEnd := now.AddDate(0, -3, 0)
	if err := createEmployeeContract(db, employees[3].ID, "Erzieher", "S8b", 2, 35,
		now.AddDate(-2, 0, 0), &leftEnd); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 4: Current contract, senior position
	if err := createEmployeeContract(db, employees[4].ID, "Gruppenleitung", "S8a", 5, 39,
		now.AddDate(-5, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 5: Part-time current contract
	if err := createEmployeeContract(db, employees[5].ID, "Sozialarbeiter", "S9", 3, 30,
		now.AddDate(-3, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 6: Future contract (starts next month)
	futureStart := now.AddDate(0, 1, 0)
	if err := createEmployeeContract(db, employees[6].ID, "Erzieher", "S8a", 1, 39,
		futureStart, nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 7: Current contract with planned end (temporary)
	tempEnd := now.AddDate(0, 6, 0)
	if err := createEmployeeContract(db, employees[7].ID, "Erzieher", "S8a", 6, 39,
		now.AddDate(-1, 0, 0), &tempEnd); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 8: Mini-job (low hours)
	if err := createEmployeeContract(db, employees[8].ID, "Kinderpfleger", "S4", 4, 20,
		now.AddDate(-1, 6, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 9: Multiple past contracts, no current (on leave/gap)
	oldEnd1 := now.AddDate(-2, 0, 0)
	if err := createEmployeeContract(db, employees[9].ID, "Erzieher", "S8b", 1, 39,
		now.AddDate(-4, 0, 0), &oldEnd1); err != nil {
		return err
	}
	employeeContractCount++
	oldEnd2 := now.AddDate(0, -6, 0)
	if err := createEmployeeContract(db, employees[9].ID, "Erzieher", "S8b", 3, 39,
		now.AddDate(-1, 6, 0), &oldEnd2); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 10: New hire, current contract
	if err := createEmployeeContract(db, employees[10].ID, "Erzieher", "S8a", 1, 39,
		now.AddDate(0, -3, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 11: Long-term employee, single contract
	if err := createEmployeeContract(db, employees[11].ID, "Erzieher", "S8a", 6, 39,
		now.AddDate(-8, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 12: Part-time, current contract
	if err := createEmployeeContract(db, employees[12].ID, "Kinderpfleger", "S4", 3, 25,
		now.AddDate(-2, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 13: Recently promoted (2 contracts)
	promoEnd := now.AddDate(0, -2, 0)
	if err := createEmployeeContract(db, employees[13].ID, "Erzieher", "S8a", 3, 39,
		now.AddDate(-3, 0, 0), &promoEnd); err != nil {
		return err
	}
	employeeContractCount++
	if err := createEmployeeContract(db, employees[13].ID, "Gruppenleitung", "S9", 1, 39,
		now.AddDate(0, -2, 1), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 14: Retired last year
	retiredEnd := now.AddDate(-1, 0, 0)
	if err := createEmployeeContract(db, employees[14].ID, "Erzieher", "S8a", 6, 39,
		now.AddDate(-15, 0, 0), &retiredEnd); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 15: Current contract, standard
	if err := createEmployeeContract(db, employees[15].ID, "Erzieher", "S8b", 2, 39,
		now.AddDate(-1, 6, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 16: Maternity cover (temporary, ends in 3 months)
	maternityEnd := now.AddDate(0, 3, 0)
	if err := createEmployeeContract(db, employees[16].ID, "Erzieher", "S8a", 2, 39,
		now.AddDate(0, -9, 0), &maternityEnd); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 17: No contract yet (just hired, starts in 2 weeks)
	futureStart2 := now.AddDate(0, 0, 14)
	if err := createEmployeeContract(db, employees[17].ID, "Praktikant", "S4", 1, 39,
		futureStart2, nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 18: Multiple contracts with step increases
	step1End := now.AddDate(-3, 0, 0)
	if err := createEmployeeContract(db, employees[18].ID, "Erzieher", "S8a", 1, 39,
		now.AddDate(-5, 0, 0), &step1End); err != nil {
		return err
	}
	employeeContractCount++
	step2End := now.AddDate(-1, 0, 0)
	if err := createEmployeeContract(db, employees[18].ID, "Erzieher", "S8a", 2, 39,
		now.AddDate(-3, 0, 1), &step2End); err != nil {
		return err
	}
	employeeContractCount++
	if err := createEmployeeContract(db, employees[18].ID, "Erzieher", "S8a", 3, 39,
		now.AddDate(-1, 0, 1), nil); err != nil {
		return err
	}
	employeeContractCount++

	// Employee 19: Current full-time
	if err := createEmployeeContract(db, employees[19].ID, "Sozialarbeiter", "S9", 4, 39,
		now.AddDate(-4, 0, 0), nil); err != nil {
		return err
	}
	employeeContractCount++

	slog.Info("Created employee contracts", "count", employeeContractCount)

	slog.Info("Test data seeding completed",
		"organization", org.Name,
		"users", "superadmin@example.com, admin@example.com, manager@example.com",
		"password", "supersecret",
		"childrenCount", len(children),
		"employeeCount", len(employees),
		"payPlan", payPlan.Name,
	)

	return nil
}

//nolint:gosec // G404: math/rand is fine for test data generation
func createTestChildren(orgID uint, sectionID uint, count int) []models.Child {
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
					SectionID:      &sectionID,
					FirstName:      firstNames[randInt(len(firstNames))],
					LastName:       lastNames[randInt(len(lastNames))],
					Gender:         randomGender(),
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
				SectionID:      &sectionID,
				FirstName:      firstNames[randInt(len(firstNames))],
				LastName:       lastNames[randInt(len(lastNames))],
				Gender:         randomGender(),
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

// createTestEmployees creates test employees with realistic German names
func createTestEmployees(orgID uint, sectionID uint, count int) []models.Employee {
	employeeFirstNames := []string{
		"Anna", "Thomas", "Maria", "Michael", "Julia",
		"Stefan", "Sabine", "Martin", "Petra", "Andreas",
		"Claudia", "Frank", "Susanne", "Christian", "Monika",
		"Jürgen", "Katrin", "Wolfgang", "Birgit", "Uwe",
	}
	employeeLastNames := []string{
		"Müller", "Schmidt", "Weber", "Fischer", "Meyer",
		"Wagner", "Becker", "Schulz", "Hoffmann", "Koch",
		"Richter", "Braun", "Schröder", "Neumann", "Klein",
		"Wolf", "Krüger", "Hartmann", "Lang", "Zimmermann",
	}

	employees := make([]models.Employee, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		// Employees are typically 25-55 years old
		ageYears := 25 + randInt(30)
		birthdate := now.AddDate(-ageYears, -randInt(12), -randInt(28))

		employees[i] = models.Employee{
			Person: models.Person{
				OrganizationID: orgID,
				SectionID:      &sectionID,
				FirstName:      employeeFirstNames[i%len(employeeFirstNames)],
				LastName:       employeeLastNames[i%len(employeeLastNames)],
				Gender:         randomGender(),
				Birthdate:      birthdate,
			},
		}
	}

	return employees
}

// createEmployeeContract is a helper to create an employee contract
func createEmployeeContract(db *gorm.DB, employeeID uint, position, grade string, step int, weeklyHours float64, from time.Time, to *time.Time) error {
	contract := models.EmployeeContract{
		EmployeeID:  employeeID,
		Period:      models.Period{From: from, To: to},
		Position:    position,
		Grade:       grade,
		Step:        step,
		WeeklyHours: weeklyHours,
	}
	return db.Create(&contract).Error
}

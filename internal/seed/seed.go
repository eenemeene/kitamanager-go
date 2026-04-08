package seed

import (
	"context"
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
	"github.com/eenemeene/kitamanager-go/internal/service"
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
	r := rand.Intn(100) // #nosec G404
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
func SeedAdmin(cfg *config.Config, userStore *store.UserStore, userOrgStore *store.UserOrganizationStore, enforcer *rbac.Enforcer) error {
	ctx := context.Background()
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminPassword == "" {
		slog.Info("Admin seeding skipped: SEED_ADMIN_EMAIL or SEED_ADMIN_PASSWORD not set")
		return nil
	}

	// Check if user already exists
	existingUser, err := userStore.FindByEmail(ctx, cfg.SeedAdminEmail)
	if err == nil && existingUser != nil {
		slog.Info("Admin user already exists", "email", cfg.SeedAdminEmail)

		// Ensure superadmin is set in database
		if !existingUser.IsSuperAdmin {
			if err := userOrgStore.SetSuperAdmin(ctx, existingUser.ID, true); err != nil {
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

	if err != nil && !errors.Is(err, store.ErrNotFound) {
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

	if err := userStore.Create(ctx, user); err != nil {
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
// If the government funding already exists, it performs an incremental update.
func SeedGovernmentFunding(cfg *config.Config, imp *importer.GovernmentFundingImporter) error {
	if cfg.GovernmentFundingSeedPath == "" {
		slog.Info("Government funding seeding skipped: GOVERNMENT_FUNDING_SEED_PATH not set")
		return nil
	}

	ctx := context.Background()
	result, err := imp.ImportGovernmentFundingFromFile(ctx, cfg.GovernmentFundingSeedPath, cfg.GovernmentFundingSeedState)
	if err != nil {
		return err
	}

	slog.Info("Government funding seeded successfully",
		"state", cfg.GovernmentFundingSeedState,
		"id", result.FundingID,
		"created", result.Created,
		"path", cfg.GovernmentFundingSeedPath,
	)
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

// Contract property combinations
// These must match the Key/Value structure in the Berlin government funding YAML
// Keys are property categories (care_type, ndh, integration), values are specific options
var propertyCombinations = []models.ContractProperties{
	{"care_type": "ganztag"},
	{"care_type": "ganztag", "ndh": "ndh"},
	{"care_type": "ganztag", "integration": "integration a"},
	{"care_type": "ganztag", "ndh": "ndh", "integration": "integration a"},
	{"care_type": "halbtag"},
	{"care_type": "halbtag", "ndh": "ndh"},
	{"care_type": "teilzeit"},
	{"care_type": "teilzeit", "ndh": "ndh"},
}

// SeedTestData creates realistic test data for development:
// - Berlin government funding plan
// - Organization "Kita Sonnenschein" with Berlin funding assigned
// - Test users with different roles (all with password "supersecret")
// - 120 currently active children across 3 sections with 3 years of history (~200 total)
// - ~35 employees (active, former, and upcoming)
func SeedTestData(cfg *config.Config, db *gorm.DB, imp *importer.GovernmentFundingImporter) error {
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
	ctx := context.Background()
	result, err := imp.ImportGovernmentFundingFromFile(ctx, "configs/government-fundings/berlin.yaml", "berlin")
	if err != nil {
		return fmt.Errorf("failed to import Berlin government funding: %w", err)
	}
	slog.Info("Berlin government funding imported", "id", result.FundingID, "created", result.Created)

	// Create organization with Berlin state
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

	// Create default section for the organization
	defaultSection := &models.Section{
		Name:           "Unassigned",
		OrganizationID: org.ID,
		IsDefault:      true,
		CreatedBy:      "seed",
	}
	if err := db.Create(defaultSection).Error; err != nil {
		return err
	}

	// Create named sections for typical German Kita age groups
	type namedSectionDef struct {
		name         string
		minAgeMonths *int
		maxAgeMonths *int
	}
	namedSectionDefs := []namedSectionDef{
		{"Nest", intPtr(0), intPtr(24)},
		{"Nestflüchter", intPtr(24), intPtr(36)},
		{"Große", intPtr(36), nil},
	}
	var sections []*models.Section // Nest, Nestflüchter, Große
	for _, def := range namedSectionDefs {
		sec := &models.Section{
			Name:           def.name,
			OrganizationID: org.ID,
			CreatedBy:      "seed",
			MinAgeMonths:   def.minAgeMonths,
			MaxAgeMonths:   def.maxAgeMonths,
		}
		if err := db.Create(sec).Error; err != nil {
			return err
		}
		slog.Info("Created section", "name", sec.Name, "id", sec.ID)
		sections = append(sections, sec)
	}

	// Hash password for all test users
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create test users
	testUsers := []struct {
		name         string
		email        string
		isSuperAdmin bool
		orgRole      models.Role
	}{
		{"Super Admin", "superadmin@example.com", true, models.Role("")},
		{"Admin", "admin@example.com", false, models.RoleAdmin},
		{"Manager", "manager@example.com", false, models.RoleManager},
	}
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
		}
		if tu.orgRole != "" {
			userOrg := &models.UserOrganization{
				UserID:         user.ID,
				OrganizationID: org.ID,
				Role:           tu.orgRole,
				CreatedBy:      "seed",
			}
			if err := db.Create(userOrg).Error; err != nil {
				slog.Warn("Failed to add user to organization (may already exist)", "email", tu.email, "error", err)
			}
		}
	}

	// Create TVöD-SuE PayPlan
	payPlan := &models.PayPlan{
		OrganizationID: org.ID,
		Name:           "TVöD-SuE 2024",
	}
	if err := db.Create(payPlan).Error; err != nil {
		return err
	}
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	payPeriod := &models.PayPlanPeriod{
		PayPlanID:                payPlan.ID,
		Period:                   models.Period{From: periodStart},
		WeeklyHours:              39.0,
		EmployerContributionRate: 2200, // 22.00%
	}
	if err := db.Create(payPeriod).Error; err != nil {
		return err
	}
	payEntries := []struct {
		grade        string
		step         int
		amount       int
		stepMinYears *int
	}{
		{"S8a", 1, 314847, intPtr(0)}, {"S8a", 2, 329947, intPtr(1)}, {"S8a", 3, 350089, intPtr(3)},
		{"S8a", 4, 365134, intPtr(6)}, {"S8a", 5, 385229, intPtr(10)}, {"S8a", 6, 398317, intPtr(15)},
		{"S8b", 1, 339902, intPtr(0)}, {"S8b", 2, 354655, intPtr(1)}, {"S8b", 3, 370125, intPtr(3)},
		{"S8b", 4, 385592, intPtr(6)}, {"S8b", 5, 401058, intPtr(10)}, {"S8b", 6, 416526, intPtr(15)},
		{"S4", 1, 267400, intPtr(0)}, {"S4", 2, 282700, intPtr(1)}, {"S4", 3, 298000, intPtr(3)},
		{"S4", 4, 313300, intPtr(6)}, {"S4", 5, 328600, intPtr(10)}, {"S4", 6, 343900, intPtr(15)},
		{"S9", 1, 344800, intPtr(0)}, {"S9", 2, 360100, intPtr(1)}, {"S9", 3, 385200, intPtr(3)},
		{"S9", 4, 400500, intPtr(6)}, {"S9", 5, 420700, intPtr(10)}, {"S9", 6, 435000, intPtr(15)},
	}
	for _, e := range payEntries {
		entry := &models.PayPlanEntry{
			PeriodID:      payPeriod.ID,
			Grade:         e.grade,
			Step:          e.step,
			MonthlyAmount: e.amount,
			StepMinYears:  e.stepMinYears,
		}
		if err := db.Create(entry).Error; err != nil {
			return err
		}
	}
	slog.Info("Created PayPlan", "name", payPlan.Name, "entries", len(payEntries))

	// Create Minijob PayPlan
	minijobPayPlan := &models.PayPlan{
		OrganizationID: org.ID,
		Name:           "Minijob 2024",
	}
	if err := db.Create(minijobPayPlan).Error; err != nil {
		return err
	}
	minijobPeriod := &models.PayPlanPeriod{
		PayPlanID:                minijobPayPlan.ID,
		Period:                   models.Period{From: periodStart},
		WeeklyHours:              10.0,
		EmployerContributionRate: 3100, // 31.00% (pension 15% + health 13% + tax 2% + U1/U2/U3 ~1%)
	}
	if err := db.Create(minijobPeriod).Error; err != nil {
		return err
	}
	minijobEntry := &models.PayPlanEntry{
		PeriodID:      minijobPeriod.ID,
		Grade:         "Minijob",
		Step:          1,
		MonthlyAmount: 55600, // €556.00/month for 10h/week
	}
	if err := db.Create(minijobEntry).Error; err != nil {
		return err
	}
	slog.Info("Created PayPlan", "name", minijobPayPlan.Name, "entries", 1)

	// Seed budget item: Garden maintenance expense (1000 EUR/month)
	gardenItem := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Garten",
		Category:       string(models.BudgetItemCategoryExpense),
		PerChild:       false,
	}
	if err := db.Create(gardenItem).Error; err != nil {
		return err
	}
	gardenEntry := &models.BudgetItemEntry{
		BudgetItemID: gardenItem.ID,
		Period:       models.Period{From: periodStart},
		AmountCents:  100000, // 1000.00 EUR
	}
	if err := db.Create(gardenEntry).Error; err != nil {
		return err
	}
	slog.Info("Created BudgetItem", "name", gardenItem.Name, "amount_eur", "1000.00")

	// Seed budget item: Parent contribution income (90 EUR/month per child)
	elternbeitragItem := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeitrag",
		Category:       string(models.BudgetItemCategoryIncome),
		PerChild:       true,
	}
	if err := db.Create(elternbeitragItem).Error; err != nil {
		return err
	}
	elternbeitragEntry := &models.BudgetItemEntry{
		BudgetItemID: elternbeitragItem.ID,
		Period:       models.Period{From: periodStart},
		AmountCents:  9000, // 90.00 EUR
	}
	if err := db.Create(elternbeitragEntry).Error; err != nil {
		return err
	}
	slog.Info("Created BudgetItem", "name", elternbeitragItem.Name, "amount_eur", "90.00", "per_child", true)

	// Build ChildService for seeding contracts through the service layer
	// (ensures auto-apply funding properties are merged into contracts)
	childStore := store.NewChildStore(db)
	orgStore := store.NewOrganizationStore(db)
	sectionStore := store.NewSectionStore(db)
	govFundingStore := store.NewGovernmentFundingStore(db)
	transactor := store.NewTransactor(db)
	childService := service.NewChildService(childStore, orgStore, govFundingStore, sectionStore, transactor)

	// Seed children with realistic contract histories spanning 3 years
	childCount, contractCount, err := seedChildren(db, childService, org.ID, sections)
	if err != nil {
		return fmt.Errorf("failed to seed children: %w", err)
	}
	slog.Info("Created test children", "children", childCount, "contracts", contractCount)

	// Seed employees with varied scenarios (active, former, upcoming)
	empCount, empContractCount, err := seedEmployees(db, org.ID, sections, defaultSection, payPlan.ID, minijobPayPlan.ID)
	if err != nil {
		return fmt.Errorf("failed to seed employees: %w", err)
	}
	slog.Info("Created test employees", "employees", empCount, "contracts", empContractCount)

	slog.Info("Test data seeding completed",
		"organization", org.Name,
		"users", "superadmin@example.com, admin@example.com, manager@example.com",
		"password", "supersecret",
	)
	return nil
}

// childCohort defines a group of children with similar characteristics.
type childCohort struct {
	count     int
	birthFrom time.Time
	birthTo   time.Time
	joinFrom  time.Time
	joinTo    time.Time
	leftDate  *time.Time // nil = still active
	sectionID uint
}

// seedChildren creates 120 currently active children distributed across sections with 3 years of history.
// Total children including alumni and future is ~200.
//
// The data models a realistic Kita lifecycle:
//   - Children enter Nest (0-2y), progress through Nestflüchter (2-3y) and Große (3-6y)
//   - School starters leave on Jul 31 each year
//   - Most new children join at the start of the Kita year (Aug-Oct), some later
//   - ~20 children have multi-contract histories showing section transitions
//
//nolint:gosec,cyclop // math/rand is fine for test data; complexity is inherent
func seedChildren(db *gorm.DB, childService *service.ChildService, orgID uint, sections []*models.Section) (int, int, error) {
	now := time.Now()
	nest, nestfluechter, grosse := sections[0], sections[1], sections[2]

	// Kita year boundaries (Aug 1 - Jul 31)
	currentKitaYear := kitaYearStartFor(now)
	prevKitaYear := currentKitaYear.AddDate(-1, 0, 0)
	jul := func(year int) time.Time {
		return time.Date(year, time.July, 31, 0, 0, 0, 0, time.UTC)
	}
	midYear2024 := time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC)
	midYear2025 := time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC)
	jul2023 := jul(currentKitaYear.Year() - 2)
	jul2024 := jul(currentKitaYear.Year() - 1)
	jul2025 := jul(currentKitaYear.Year())

	cohorts := []childCohort{
		// --- Currently active children (120 total: 25 Nest + 30 NF + 50 Große single + 15 multi) ---

		// Nest: born 6-24 months ago, joined when 6-12 months old
		{25, now.AddDate(-2, 0, 0), now.AddDate(0, -6, 0),
			prevKitaYear, now.AddDate(0, -1, 0), nil, nest.ID},

		// Nestflüchter: born 24-36 months ago
		{30, now.AddDate(-3, 0, 0), now.AddDate(-2, 0, 0),
			now.AddDate(-2, -6, 0), now.AddDate(0, -3, 0), nil, nestfluechter.ID},

		// Große: born 3-6 years ago, single contract
		{50, now.AddDate(-6, 0, 0), now.AddDate(-3, 0, 0),
			now.AddDate(-4, 0, 0), now.AddDate(0, -6, 0), nil, grosse.ID},

		// --- Children who left for school (Jul 31 each year) ---

		// Left Jul 2023: born ~6 years before that, started ~2020-2021
		{14, jul2023.AddDate(-7, 0, 0), jul2023.AddDate(-6, 0, 0),
			jul2023.AddDate(-4, 0, 0), jul2023.AddDate(-2, 0, 0), &jul2023, grosse.ID},

		// Left Jul 2024: born ~6 years before that, started ~2021-2022
		{15, jul2024.AddDate(-7, 0, 0), jul2024.AddDate(-6, 0, 0),
			jul2024.AddDate(-4, 0, 0), jul2024.AddDate(-2, 0, 0), &jul2024, grosse.ID},

		// Left Jul 2025: born ~6 years before that, started ~2022-2023
		{14, jul2025.AddDate(-7, 0, 0), jul2025.AddDate(-6, 0, 0),
			jul2025.AddDate(-4, 0, 0), jul2025.AddDate(-2, 0, 0), &jul2025, grosse.ID},

		// Left mid-year (family moved) — 2 per year
		{2, midYear2024.AddDate(-3, 0, 0), midYear2024.AddDate(-2, 0, 0),
			midYear2024.AddDate(-2, 0, 0), midYear2024.AddDate(-1, 0, 0), &midYear2024, nestfluechter.ID},
		{2, midYear2025.AddDate(-4, 0, 0), midYear2025.AddDate(-3, 0, 0),
			midYear2025.AddDate(-2, 0, 0), midYear2025.AddDate(-1, 0, 0), &midYear2025, grosse.ID},

		// --- Future children (starting in coming months) ---
		{8, now.AddDate(-1, -6, 0), now.AddDate(0, -6, 0),
			now.AddDate(0, 1, 0), now.AddDate(0, 6, 0), nil, nest.ID},
	}

	ctx := context.Background()
	childCount := 0
	contractCount := 0

	// Generate single-contract children from cohorts
	for _, c := range cohorts {
		for i := 0; i < c.count; i++ {
			child := newChild(orgID, randomDateBetween(c.birthFrom, c.birthTo))
			if err := db.Create(&child).Error; err != nil {
				return 0, 0, err
			}
			joinDate := randomJoinDate(c.joinFrom, c.joinTo)
			// Ensure contract doesn't start before birthdate
			if joinDate.Before(child.Birthdate) {
				joinDate = child.Birthdate
			}
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &models.ChildContractCreateRequest{
				SectionID:  c.sectionID,
				From:       joinDate,
				To:         c.leftDate,
				Properties: propertyCombinations[randInt(len(propertyCombinations))],
			}); err != nil {
				return 0, 0, err
			}
			childCount++
			contractCount++
		}
	}

	// Multi-contract children showing section transitions (Nest → Nestflüchter → Große)
	// Currently active, in Große
	for i := 0; i < 15; i++ {
		birthdate := randomDateBetween(now.AddDate(-5, 0, 0), now.AddDate(-3, -6, 0))
		child := newChild(orgID, birthdate)
		if err := db.Create(&child).Error; err != nil {
			return 0, 0, err
		}

		// Nest contract: started at 8-12 months, ended on a Jul 31
		nestStart := birthdate.AddDate(0, 8+randInt(4), 0)
		nestStart = firstOfMonth(nestStart)
		nestEnd := jul(nestStart.Year() + 1)
		if !nestEnd.After(nestStart.AddDate(0, 4, 0)) {
			nestEnd = nestEnd.AddDate(1, 0, 0)
		}

		// Nestflüchter: Aug 1 after Nest ends, ended on next Jul 31
		nfStart := nestEnd.AddDate(0, 0, 1)
		nfEnd := jul(nfStart.Year() + 1)
		if !nfEnd.After(nfStart.AddDate(0, 4, 0)) {
			nfEnd = nfEnd.AddDate(1, 0, 0)
		}

		// Große: Aug 1 after Nestflüchter ends, ongoing
		grosseStart := nfEnd.AddDate(0, 0, 1)

		reqs := []models.ChildContractCreateRequest{
			{SectionID: nest.ID, From: nestStart, To: &nestEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: nestfluechter.ID, From: nfStart, To: &nfEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: grosse.ID, From: grosseStart, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
		}
		for _, req := range reqs {
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &req); err != nil {
				return 0, 0, err
			}
			contractCount++
		}
		childCount++
	}

	// Multi-contract children who already left for school
	for i := 0; i < 10; i++ {
		exitYear := currentKitaYear.Year() - 1 - randInt(2) // left Jul 2023, 2024, or 2025
		exitDate := jul(exitYear)
		birthdate := exitDate.AddDate(-6, -randInt(6), 0)
		child := newChild(orgID, birthdate)
		if err := db.Create(&child).Error; err != nil {
			return 0, 0, err
		}

		nestStart := birthdate.AddDate(0, 10+randInt(6), 0)
		nestStart = firstOfMonth(nestStart)
		nestEnd := jul(nestStart.Year() + 1)
		if !nestEnd.After(nestStart.AddDate(0, 4, 0)) {
			nestEnd = nestEnd.AddDate(1, 0, 0)
		}
		grosseStart := nestEnd.AddDate(0, 0, 1)

		reqs := []models.ChildContractCreateRequest{
			{SectionID: nest.ID, From: nestStart, To: &nestEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: grosse.ID, From: grosseStart, To: &exitDate, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
		}
		for _, req := range reqs {
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &req); err != nil {
				return 0, 0, err
			}
			contractCount++
		}
		childCount++
	}

	return childCount, contractCount, nil
}

// empDef defines an employee and their contract history.
type empDef struct {
	firstName string
	lastName  string
	birthYear int
	contracts []empContractDef
}

type empContractDef struct {
	staffCategory string
	grade         string
	step          int
	weeklyHours   float64
	from          time.Time
	to            *time.Time
	sectionIdx    int // 0=Nest, 1=Nestflüchter, 2=Große, -1=default
}

// seedEmployees creates ~35 employees with realistic contract scenarios.
// Active employees provide 95-130% staffing coverage depending on time of year.
// Includes former employees (for historical staffing data) and upcoming hires.
//
//nolint:cyclop // complexity is inherent in realistic test data definition
func seedEmployees(db *gorm.DB, orgID uint, namedSections []*models.Section, defaultSection *models.Section, payPlanID uint, minijobPayPlanID uint) (int, int, error) {
	now := time.Now()
	currentKitaYear := kitaYearStartFor(now)

	d := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	tp := func(t time.Time) *time.Time { return &t }

	employees := []empDef{
		// ===== Nest section (8 active) =====
		{"Anna", "Müller", 1988, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2020, 3, 1), nil, 0},
		}},
		{"Thomas", "Schmidt", 1995, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2023, 8, 1), nil, 0},
		}},
		{"Maria", "Weber", 1990, []empContractDef{
			{"supplementary", "S4", 3, 39, d(2022, 1, 15), nil, 0},
		}},
		{"Julia", "Fischer", 1997, []empContractDef{
			{"qualified", "S8a", 1, 30, d(2025, 2, 1), nil, 0},
		}},
		{"Daniela", "Krause", 1993, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2023, 3, 1), nil, 0},
		}},
		{"Sandra", "Schmitz", 1989, []empContractDef{
			{"supplementary", "S4", 2, 30, d(2024, 8, 1), nil, 0},
		}},
		{"Karolin", "Berger", 1994, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2023, 9, 1), nil, 0},
		}},
		{"Stefanie", "Frank", 1997, []empContractDef{
			{"qualified", "S8a", 1, 30, d(2025, 8, 1), nil, 0},
		}},

		// ===== Nestflüchter section (9 active) =====
		{"Stefan", "Meyer", 1980, []empContractDef{
			{"qualified", "S8a", 5, 39, d(2018, 8, 1), nil, 1},
		}},
		{"Sabine", "Wagner", 1991, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2022, 8, 1), nil, 1},
		}},
		{"Martin", "Becker", 1993, []empContractDef{
			{"qualified", "S8b", 2, 39, d(2023, 9, 1), nil, 1},
		}},
		{"Petra", "Schulz", 1986, []empContractDef{
			{"supplementary", "S4", 2, 25, d(2024, 2, 1), nil, 1},
		}},
		{"Heike", "Schäfer", 1984, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2019, 8, 1), nil, 1},
		}},
		{"Robert", "Lange", 1996, []empContractDef{
			{"qualified", "S8b", 1, 30, d(2025, 1, 1), nil, 1},
		}},
		{"Tanja", "Horn", 1990, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2023, 8, 1), nil, 1},
		}},
		{"Dirk", "Winkler", 1993, []empContractDef{
			{"qualified", "S8b", 1, 39, d(2024, 3, 1), nil, 1},
		}},
		{"Silke", "Pohl", 1987, []empContractDef{
			{"supplementary", "S4", 2, 30, d(2025, 8, 1), nil, 1},
		}},

		// ===== Große section (12 active) =====
		{"Andreas", "Hoffmann", 1975, []empContractDef{
			{"qualified", "S8a", 6, 39, d(2015, 8, 1), nil, 2},
		}},
		{"Claudia", "Koch", 1989, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2021, 3, 1), nil, 2},
		}},
		{"Frank", "Richter", 1994, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2023, 8, 1), nil, 2},
		}},
		{"Susanne", "Braun", 1987, []empContractDef{
			{"qualified", "S9", 3, 39, d(2021, 8, 1), nil, 2},
		}},
		{"Christian", "Schröder", 1985, []empContractDef{
			{"supplementary", "S4", 4, 39, d(2020, 1, 1), nil, 2},
		}},
		{"Monika", "Neumann", 1996, []empContractDef{
			{"qualified", "S8b", 1, 30, d(2024, 8, 1), nil, 2},
		}},
		{"Markus", "Schmitt", 1991, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2022, 3, 1), nil, 2},
		}},
		{"Nicole", "Krüger", 1988, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2019, 8, 1), nil, 2},
		}},
		{"Kerstin", "Haas", 1986, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2023, 1, 1), nil, 2},
		}},
		{"Rainer", "Lorenz", 1992, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2024, 8, 1), nil, 2},
		}},
		{"Anke", "Vogel", 1995, []empContractDef{
			{"supplementary", "S4", 1, 39, d(2025, 2, 1), nil, 2},
		}},
		// Deputy/coordinator
		{"Katrin", "Klein", 1982, []empContractDef{
			{"qualified", "S9", 5, 39, d(2016, 8, 1), nil, 2},
		}},

		// ===== Cross-section / support (3 active) =====
		// Non-pedagogical (kitchen)
		{"Birgit", "Wolf", 1978, []empContractDef{
			{"non_pedagogical", "S4", 3, 20, d(2022, 4, 1), nil, -1},
		}},
		// Floater/substitute across sections
		{"Michael", "Hartmann", 1992, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2019, 8, 1), nil, 2},
		}},
		// Cleaning staff
		{"Inge", "Schwarz", 1970, []empContractDef{
			{"non_pedagogical", "S4", 5, 20, d(2018, 1, 1), nil, -1},
		}},
		// Minijob: kitchen helper (10h/week, €556/month)
		{"Gisela", "Peters", 1965, []empContractDef{
			{"non_pedagogical", "Minijob", 1, 10, d(2023, 4, 1), nil, -1},
		}},
		// Minijob: afternoon helper (8h/week, prorated to €444.80/month)
		{"Hanna", "Seidel", 2001, []empContractDef{
			{"supplementary", "Minijob", 1, 8, d(2024, 9, 1), nil, 2},
		}},

		// ===== Former employees (left in last 3 years) =====

		// Left Jan 2025 after 6 years (career change)
		{"Jürgen", "Lang", 1983, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2019, 2, 1), tp(d(2022, 7, 31)), 1},
			{"qualified", "S8a", 4, 39, d(2022, 8, 1), tp(d(2025, 1, 31)), 2},
		}},
		// Left Jul 2024 (moved cities)
		{"Wolfgang", "Krüger", 1990, []empContractDef{
			{"qualified", "S8b", 2, 39, d(2021, 8, 1), tp(d(2024, 7, 31)), 0},
		}},
		// Left Mar 2024 (short stint)
		{"Uwe", "Zimmermann", 1998, []empContractDef{
			{"qualified", "S8a", 1, 39, d(2022, 8, 1), tp(d(2024, 3, 31)), 2},
		}},
		// Left Jul 2023 (retired)
		{"Renate", "Meier", 1963, []empContractDef{
			{"qualified", "S8a", 6, 39, d(2010, 8, 1), tp(d(2023, 7, 31)), 2},
		}},
		// Left Dec 2023 (parental leave, didn't return)
		{"Laura", "Schneider", 1994, []empContractDef{
			{"supplementary", "S4", 2, 30, d(2022, 3, 1), tp(d(2023, 12, 31)), 1},
		}},
		// Left Aug 2024 (burnout)
		{"Heiko", "Baumann", 1986, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2020, 8, 1), tp(d(2024, 8, 31)), 0},
		}},
		// Left Feb 2024 (moved to another Kita)
		{"Christine", "Vogt", 1991, []empContractDef{
			{"qualified", "S8b", 2, 39, d(2021, 3, 1), tp(d(2024, 2, 29)), 2},
		}},

		// ===== Upcoming employees =====

		// Starting next month
		{"Lena", "Hofmann", 1999, []empContractDef{
			{"qualified", "S8a", 1, 39, now.AddDate(0, 1, 0), nil, 0},
		}},
		// Starting in 3 months
		{"Felix", "Werner", 2000, []empContractDef{
			{"supplementary", "S4", 1, 39,
				time.Date(currentKitaYear.Year()+1, time.August, 1, 0, 0, 0, 0, time.UTC),
				nil, 1},
		}},
		// Starting next Kita year
		{"Sophie", "Lehmann", 1998, []empContractDef{
			{"qualified", "S8a", 1, 39,
				time.Date(currentKitaYear.Year()+1, time.August, 1, 0, 0, 0, 0, time.UTC),
				nil, 2},
		}},
	}

	empCount := 0
	contractCount := 0

	for _, e := range employees {
		birthdate := time.Date(e.birthYear, time.Month(3+randInt(9)), 1+randInt(28), 0, 0, 0, 0, time.UTC)
		emp := models.Employee{
			Person: models.Person{
				OrganizationID: orgID,
				FirstName:      e.firstName,
				LastName:       e.lastName,
				Gender:         randomGender(),
				Birthdate:      birthdate,
			},
		}
		if err := db.Create(&emp).Error; err != nil {
			return 0, 0, err
		}
		empCount++

		for _, c := range e.contracts {
			sectionID := defaultSection.ID
			if c.sectionIdx >= 0 && c.sectionIdx < len(namedSections) {
				sectionID = namedSections[c.sectionIdx].ID
			}
			ppID := payPlanID
			if c.grade == "Minijob" {
				ppID = minijobPayPlanID
			}
			if err := createEmployeeContract(db, emp.ID, c.staffCategory, c.grade, c.step, c.weeklyHours, c.from, c.to, ppID, sectionID); err != nil {
				return 0, 0, err
			}
			contractCount++
		}
	}

	return empCount, contractCount, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func intPtr(v int) *int {
	return &v
}

// date creates a UTC date.
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// kitaYearStartFor returns Aug 1 of the Kita year containing the given date.
func kitaYearStartFor(t time.Time) time.Time {
	if t.Month() >= time.August {
		return date(t.Year(), time.August, 1)
	}
	return date(t.Year()-1, time.August, 1)
}

// firstOfMonth returns the first day of the month for a given date.
func firstOfMonth(t time.Time) time.Time {
	return date(t.Year(), t.Month(), 1)
}

// randomDateBetween returns a random date between from and to (inclusive).
//
//nolint:gosec // G404: math/rand is fine for test data generation
func randomDateBetween(from, to time.Time) time.Time {
	if !to.After(from) {
		return from
	}
	days := int(to.Sub(from).Hours() / 24)
	if days <= 0 {
		return from
	}
	return from.AddDate(0, 0, rand.Intn(days)) // #nosec G404
}

// randomJoinDate returns a realistic Kita join date weighted toward Aug-Oct.
//
//nolint:gosec // G404: math/rand is fine for test data generation
func randomJoinDate(from, to time.Time) time.Time {
	t := randomDateBetween(from, to)
	// Snap to 1st of month (contracts typically start on the 1st)
	return firstOfMonth(t)
}

func newChild(orgID uint, birthdate time.Time) models.Child {
	return models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      firstNames[randInt(len(firstNames))],
			LastName:       lastNames[randInt(len(lastNames))],
			Gender:         randomGender(),
			Birthdate:      birthdate,
		},
	}
}

func createEmployeeContract(db *gorm.DB, employeeID uint, staffCategory, grade string, step int, weeklyHours float64, from time.Time, to *time.Time, payPlanID uint, sectionID uint) error {
	contract := models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: from, To: to},
			SectionID: sectionID,
		},
		StaffCategory: staffCategory,
		Grade:         grade,
		Step:          step,
		WeeklyHours:   weeklyHours,
		PayPlanID:     payPlanID,
	}
	return db.Create(&contract).Error
}

package seed

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// rng is the single source for every generated value below.
//
// Deterministic on purpose. Go's global rand has been auto-seeded per process
// since 1.20, so seeding twice produced different children -- different names,
// genders and ages -- and nothing built on top of a seeded database could be
// reproduced. That broke the visual baselines outright: a screenshot of the
// attendance roster or the dashboard's staffing figures is a picture of this
// data, so the committed PNG matched or did not depending on which children CI
// happened to invent that morning. Three consecutive CI runs demonstrated it --
// one produced a baseline, the next matched it, the third did not.
//
// Set SEED_RANDOM_SEED to vary it deliberately; leave it alone to get the same
// database locally and in CI, which is what makes a baseline meaningful.
//
// Not safe for concurrent use, and does not need to be: seeding is sequential.
//
//nolint:gosec // G404: not security-sensitive, and determinism is the point
var rng = rand.New(rand.NewSource(testDataSeed())) // #nosec G404

// defaultTestDataSeed is arbitrary. What matters is that it never changes --
// every committed visual baseline is a picture of the data it produces.
const defaultTestDataSeed int64 = 20260101

func testDataSeed() int64 {
	if raw := os.Getenv("SEED_RANDOM_SEED"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return n
		}
		slog.Warn("SEED_RANDOM_SEED is not an integer; using the default",
			"value", raw, "default", defaultTestDataSeed)
	}
	return defaultTestDataSeed
}

// randInt returns a random integer in [0, n) for test data generation.
func randInt(n int) int {
	return rng.Intn(n)
}

// randomGender returns a random gender for test data.
// Distribution: ~49% male, ~49% female, ~2% diverse
func randomGender() string {
	r := rng.Intn(100)
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
func SeedAdmin(cfg *config.Config, userStore *store.UserStore, userOrgStore *store.UserOrganizationStore) error {
	ctx := context.Background()
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminPassword == "" {
		slog.Info("Admin seeding skipped: SEED_ADMIN_EMAIL or SEED_ADMIN_PASSWORD not set")
		return nil
	}

	// Canonicalize: the unique index is case-insensitive (migration 000009)
	// and all stored rows are lowercase. Lookup and insert must match that
	// form, otherwise a mixed-case SEED_ADMIN_EMAIL would fail the index on
	// second startup.
	seedEmail := strings.ToLower(strings.TrimSpace(cfg.SeedAdminEmail))

	// Check if user already exists
	existingUser, err := userStore.FindByEmail(ctx, seedEmail)
	if err == nil && existingUser != nil {
		slog.Info("Admin user already exists", "email", seedEmail)

		// Ensure superadmin is set in database
		if !existingUser.IsSuperAdmin {
			if err := userOrgStore.SetSuperAdmin(ctx, existingUser.ID, true); err != nil {
				slog.Warn("Failed to ensure superadmin status in database", "error", err)
			} else {
				slog.Info("Superadmin status set in database", "userId", existingUser.ID)
			}
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
		Email:        seedEmail,
		Password:     string(hashedPassword),
		Active:       true,
		IsSuperAdmin: true,
		CreatedBy:    "system",
	}

	if err := userStore.Create(ctx, user); err != nil {
		return err
	}

	slog.Info("Admin user created", "email", seedEmail, "id", user.ID)
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

// seededChild tracks a child created during seeding along with its contract details,
// so we can generate matching ISBJ billing data afterwards.
type seededChild struct {
	child      *models.Child
	contracts  []seededContract
	voucherNum string
}

// seededContract tracks a single contract for a seeded child.
type seededContract struct {
	from       time.Time
	to         *time.Time
	sectionID  uint
	properties models.ContractProperties
}

// SeedTestData creates realistic test data for development:
// - Berlin government funding plan
// - Organization "Kita Sonnenschein" with Berlin funding assigned
// - Test users with different roles (all with password "supersecret")
// - ~60 children across 3 sections with contract histories
// - ~20 employees (active, former, and upcoming)
// - ISBJ billing data for the last 6 months matching the seeded children
// - Voucher numbers for all children with active contracts
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

	// Seed children with realistic contract histories
	seededChildren, err := seedChildren(db, childService, org.ID, sections)
	if err != nil {
		return fmt.Errorf("failed to seed children: %w", err)
	}
	slog.Info("Created test children", "children", len(seededChildren))

	// Seed vouchers for children with active contracts
	voucherCount, err := seedVouchers(db, seededChildren)
	if err != nil {
		return fmt.Errorf("failed to seed vouchers: %w", err)
	}
	slog.Info("Created vouchers", "count", voucherCount)

	// Seed employees with varied scenarios (active, former, upcoming)
	empCount, empContractCount, err := seedEmployees(db, org.ID, sections, defaultSection, payPlan.ID, minijobPayPlan.ID)
	if err != nil {
		return fmt.Errorf("failed to seed employees: %w", err)
	}
	slog.Info("Created test employees", "employees", empCount, "contracts", empContractCount)

	// Seed ISBJ billing data for the last 6 months, matching the seeded children
	billCount, err := seedISBJBillingData(db, govFundingStore, org.ID, seededChildren)
	if err != nil {
		return fmt.Errorf("failed to seed ISBJ billing data: %w", err)
	}
	slog.Info("Created ISBJ billing periods", "count", billCount)

	// Password is intentionally not logged. It is constant ("supersecret"),
	// documented in DEVELOPMENT.md, and only available behind the
	// SEED_TEST_DATA + non-production gate enforced in config.Validate,
	// but emitting it widens the surface that logs sensitive-shaped
	// strings — best to keep credential material out of structured logs
	// as a policy.
	slog.Info("Test data seeding completed",
		"organization", org.Name,
		"users", "superadmin@example.com, admin@example.com, manager@example.com",
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

// seedChildren creates ~60 children distributed across sections.
// Returns the list of seeded children with their contract details for ISBJ billing generation.
//
//nolint:gosec,cyclop // math/rand is fine for test data; complexity is inherent
func seedChildren(db *gorm.DB, childService *service.ChildService, orgID uint, sections []*models.Section) ([]seededChild, error) {
	now := time.Now()
	nest, nestfluechter, grosse := sections[0], sections[1], sections[2]

	currentKitaYear := kitaYearStartFor(now)
	prevKitaYear := currentKitaYear.AddDate(-1, 0, 0)
	jul := func(year int) time.Time {
		return time.Date(year, time.July, 31, 0, 0, 0, 0, time.UTC)
	}
	midYear2024 := time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC)
	jul2024 := jul(currentKitaYear.Year() - 1)
	jul2025 := jul(currentKitaYear.Year())

	cohorts := []childCohort{
		// --- Currently active children (45 total: 8 Nest + 12 NF + 25 Große single) ---
		{8, now.AddDate(-2, 0, 0), now.AddDate(0, -6, 0),
			prevKitaYear, now.AddDate(0, -1, 0), nil, nest.ID},
		{12, now.AddDate(-3, 0, 0), now.AddDate(-2, 0, 0),
			now.AddDate(-2, -6, 0), now.AddDate(0, -3, 0), nil, nestfluechter.ID},
		{25, now.AddDate(-6, 0, 0), now.AddDate(-3, 0, 0),
			now.AddDate(-4, 0, 0), now.AddDate(0, -6, 0), nil, grosse.ID},

		// --- Alumni ---
		{4, jul2024.AddDate(-7, 0, 0), jul2024.AddDate(-6, 0, 0),
			jul2024.AddDate(-4, 0, 0), jul2024.AddDate(-2, 0, 0), &jul2024, grosse.ID},
		{4, jul2025.AddDate(-7, 0, 0), jul2025.AddDate(-6, 0, 0),
			jul2025.AddDate(-4, 0, 0), jul2025.AddDate(-2, 0, 0), &jul2025, grosse.ID},
		{1, midYear2024.AddDate(-3, 0, 0), midYear2024.AddDate(-2, 0, 0),
			midYear2024.AddDate(-2, 0, 0), midYear2024.AddDate(-1, 0, 0), &midYear2024, nestfluechter.ID},

		// --- Future ---
		{3, now.AddDate(-1, -6, 0), now.AddDate(0, -6, 0),
			now.AddDate(0, 1, 0), now.AddDate(0, 6, 0), nil, nest.ID},
	}

	ctx := context.Background()
	var allSeeded []seededChild

	for _, c := range cohorts {
		for range c.count {
			child := newChild(orgID, randomDateBetween(c.birthFrom, c.birthTo))
			if err := db.Create(&child).Error; err != nil {
				return nil, err
			}
			joinDate := randomJoinDate(c.joinFrom, c.joinTo)
			if joinDate.Before(child.Birthdate) {
				joinDate = child.Birthdate
			}
			props := propertyCombinations[randInt(len(propertyCombinations))]
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &models.ChildContractCreateRequest{
				SectionID:  c.sectionID,
				From:       joinDate,
				To:         c.leftDate,
				Properties: props,
			}); err != nil {
				return nil, err
			}
			allSeeded = append(allSeeded, seededChild{
				child: &child,
				contracts: []seededContract{{
					from: joinDate, to: c.leftDate, sectionID: c.sectionID, properties: props,
				}},
			})
		}
	}

	// Multi-contract children (Nest → NF → Große), currently active
	for range 5 {
		birthdate := randomDateBetween(now.AddDate(-5, 0, 0), now.AddDate(-3, -6, 0))
		child := newChild(orgID, birthdate)
		if err := db.Create(&child).Error; err != nil {
			return nil, err
		}
		nestStart := firstOfMonth(birthdate.AddDate(0, 8+randInt(4), 0))
		nestEnd := jul(nestStart.Year() + 1)
		if !nestEnd.After(nestStart.AddDate(0, 4, 0)) {
			nestEnd = nestEnd.AddDate(1, 0, 0)
		}
		nfStart := nestEnd.AddDate(0, 0, 1)
		nfEnd := jul(nfStart.Year() + 1)
		if !nfEnd.After(nfStart.AddDate(0, 4, 0)) {
			nfEnd = nfEnd.AddDate(1, 0, 0)
		}
		grosseStart := nfEnd.AddDate(0, 0, 1)

		reqs := []models.ChildContractCreateRequest{
			{SectionID: nest.ID, From: nestStart, To: &nestEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: nestfluechter.ID, From: nfStart, To: &nfEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: grosse.ID, From: grosseStart, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
		}
		sc := seededChild{child: &child}
		for _, req := range reqs {
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &req); err != nil {
				return nil, err
			}
			sc.contracts = append(sc.contracts, seededContract{
				from: req.From, to: req.To, sectionID: req.SectionID, properties: req.Properties,
			})
		}
		allSeeded = append(allSeeded, sc)
	}

	// Multi-contract alumni
	for range 3 {
		exitYear := currentKitaYear.Year() - 1 - randInt(2)
		exitDate := jul(exitYear)
		birthdate := exitDate.AddDate(-6, -randInt(6), 0)
		child := newChild(orgID, birthdate)
		if err := db.Create(&child).Error; err != nil {
			return nil, err
		}
		nestStart := firstOfMonth(birthdate.AddDate(0, 10+randInt(6), 0))
		nestEnd := jul(nestStart.Year() + 1)
		if !nestEnd.After(nestStart.AddDate(0, 4, 0)) {
			nestEnd = nestEnd.AddDate(1, 0, 0)
		}
		grosseStart := nestEnd.AddDate(0, 0, 1)
		reqs := []models.ChildContractCreateRequest{
			{SectionID: nest.ID, From: nestStart, To: &nestEnd, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
			{SectionID: grosse.ID, From: grosseStart, To: &exitDate, Properties: propertyCombinations[randInt(len(propertyCombinations))]},
		}
		sc := seededChild{child: &child}
		for _, req := range reqs {
			if _, err := childService.CreateContract(ctx, child.ID, orgID, &req); err != nil {
				return nil, err
			}
			sc.contracts = append(sc.contracts, seededContract{
				from: req.From, to: req.To, sectionID: req.SectionID, properties: req.Properties,
			})
		}
		allSeeded = append(allSeeded, sc)
	}

	return allSeeded, nil
}

// seedVouchers assigns voucher numbers to all children's contracts.
func seedVouchers(db *gorm.DB, children []seededChild) (int, error) {
	count := 0
	for i := range children {
		sc := &children[i]
		if len(sc.contracts) == 0 {
			continue
		}
		voucherNum := fmt.Sprintf("GB-%011d-%02d", sc.child.ID+10000000000, 1)
		voucher := models.ChildVoucher{
			ChildID:       sc.child.ID,
			VoucherNumber: voucherNum,
			FirstSeen:     sc.contracts[0].from,
		}
		if err := db.Create(&voucher).Error; err != nil {
			return 0, fmt.Errorf("creating voucher for child %d: %w", sc.child.ID, err)
		}
		sc.voucherNum = voucherNum
		count++
	}
	return count, nil
}

// seedISBJBillingData creates GovernmentFundingBillPeriod records for the last 6 months.
func seedISBJBillingData(db *gorm.DB, fundingStore *store.GovernmentFundingStore, orgID uint, children []seededChild) (int, error) {
	ctx := context.Background()
	funding, err := fundingStore.FindByStateWithDetails(ctx, "berlin", 0, nil)
	if err != nil {
		return 0, fmt.Errorf("loading Berlin funding: %w", err)
	}

	now := time.Now()
	billCount := 0
	for monthsAgo := 6; monthsAgo >= 1; monthsAgo-- {
		billDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -monthsAgo, 0)
		var fundingPeriod *models.GovernmentFundingPeriod
		for i := range funding.Periods {
			if funding.Periods[i].IsActiveOn(billDate) {
				fundingPeriod = &funding.Periods[i]
				break
			}
		}
		if fundingPeriod == nil {
			slog.Warn("No funding period for bill date, skipping", "date", billDate)
			continue
		}
		billPeriod := buildBillPeriod(orgID, billDate, fundingPeriod, children, monthsAgo)
		if billPeriod == nil {
			continue
		}
		if err := db.Create(billPeriod).Error; err != nil {
			return 0, fmt.Errorf("saving bill for %s: %w", billDate.Format("2006-01"), err)
		}
		billCount++
	}
	return billCount, nil
}

// discrepancyType describes what kind of billing discrepancy to introduce.
type discrepancyType int

const (
	discrepancyNone discrepancyType = iota
	discrepancyAmountOff
	discrepancyMissingSurcharge
	discrepancyExtraSurcharge
	discrepancyBillOnly
)

func childDiscrepancy(childIdx int) discrepancyType {
	switch childIdx % 25 {
	case 3:
		return discrepancyAmountOff
	case 7:
		return discrepancyMissingSurcharge
	case 11:
		return discrepancyExtraSurcharge
	case 19:
		return discrepancyBillOnly
	default:
		return discrepancyNone
	}
}

// buildBillPeriod constructs a GovernmentFundingBillPeriod for a single month.
func buildBillPeriod(orgID uint, billDate time.Time, fundingPeriod *models.GovernmentFundingPeriod, children []seededChild, monthsAgo int) *models.GovernmentFundingBillPeriod {
	billEndDate := billDate.AddDate(0, 1, -1)
	var billChildren []models.GovernmentFundingBillChild
	facilityTotal := 0
	correctionTotal := 0

	for childIdx, sc := range children {
		if sc.voucherNum == "" {
			continue
		}
		var activeContract *seededContract
		for i := range sc.contracts {
			c := &sc.contracts[i]
			if !billDate.Before(c.from) && (c.to == nil || !billDate.After(*c.to)) {
				activeContract = c
				break
			}
		}
		if activeContract == nil {
			continue
		}

		childAge := validation.CalculateAgeOnDate(sc.child.Birthdate, billDate)
		disc := childDiscrepancy(childIdx)
		voucherNum := sc.voucherNum
		if disc == discrepancyBillOnly {
			voucherNum = fmt.Sprintf("GB-%011d-%02d", 99900000000+childIdx, 1)
		}

		var payments []models.GovernmentFundingBillPayment
		rowTotal := 0
		for pi := range fundingPeriod.Properties {
			fp := &fundingPeriod.Properties[pi]
			if !fp.MatchesAge(childAge) || !activeContract.properties.HasValue(fp.Key, fp.Value) {
				continue
			}
			amount := fp.Payment
			switch disc {
			case discrepancyAmountOff:
				if fp.Key == "care_type" {
					amount += 347
				}
			case discrepancyMissingSurcharge:
				if fp.Key == "ndh" {
					continue
				}
			}
			payments = append(payments, models.GovernmentFundingBillPayment{
				Key: fp.Key, Value: fp.Value, Amount: amount, RowIndex: 0,
			})
			rowTotal += amount
		}
		if disc == discrepancyExtraSurcharge {
			for pi := range fundingPeriod.Properties {
				fp := &fundingPeriod.Properties[pi]
				if fp.Key == "qm/mss" && fp.MatchesAge(childAge) {
					payments = append(payments, models.GovernmentFundingBillPayment{
						Key: fp.Key, Value: fp.Value, Amount: fp.Payment, RowIndex: 0,
					})
					rowTotal += fp.Payment
					break
				}
			}
		}
		if len(payments) == 0 {
			continue
		}
		billChildren = append(billChildren, models.GovernmentFundingBillChild{
			VoucherNumber: voucherNum,
			ChildName:     sc.child.LastName + ", " + sc.child.FirstName,
			BirthDate:     sc.child.Birthdate.Format("01.06"),
			District:      int64(1 + childIdx%12),
			Payments:      payments,
		})
		facilityTotal += rowTotal
	}

	// Correction rows for the 2 most recent bills
	if monthsAgo <= 2 && len(children) > 10 {
		corrChild := children[10]
		if corrChild.voucherNum != "" {
			corrAmount := -1500
			billChildren = append(billChildren, models.GovernmentFundingBillChild{
				VoucherNumber: corrChild.voucherNum,
				ChildName:     corrChild.child.LastName + ", " + corrChild.child.FirstName,
				BirthDate:     corrChild.child.Birthdate.Format("01.06"),
				District:      3,
				Payments: []models.GovernmentFundingBillPayment{
					{Key: "care_type", Value: "ganztag", Amount: corrAmount, RowIndex: 1},
				},
			})
			correctionTotal += corrAmount
		}
	}

	if len(billChildren) == 0 {
		return nil
	}

	hashInput := fmt.Sprintf("seed-isbj-%s-%d", billDate.Format("2006-01"), orgID)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))
	return &models.GovernmentFundingBillPeriod{
		OrganizationID:    orgID,
		Period:            models.Period{From: billDate, To: &billEndDate},
		FileName:          fmt.Sprintf("Senatsabrechnung_%s.xlsx", billDate.Format("2006-01")),
		FileSha256:        hash,
		FacilityName:      "Kita Sonnenschein",
		FacilityTotal:     facilityTotal + correctionTotal,
		ContractBooking:   facilityTotal - correctionTotal,
		CorrectionBooking: correctionTotal,
		CreatedBy:         models.UintPtr(1),
		Children:          billChildren,
	}
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

// seedEmployees creates ~20 employees proportional to a ~60-child Kita.
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
		// ===== Nest section (3 active) =====
		{"Anna", "Müller", 1988, []empContractDef{
			{"qualified", "S8a", 4, 39, d(2020, 3, 1), nil, 0},
		}},
		{"Thomas", "Schmidt", 1995, []empContractDef{
			{"qualified", "S8a", 2, 39, d(2023, 8, 1), nil, 0},
		}},
		{"Maria", "Weber", 1990, []empContractDef{
			{"supplementary", "S4", 3, 39, d(2022, 1, 15), nil, 0},
		}},

		// ===== Nestflüchter section (4 active) =====
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

		// ===== Große section (6 active) =====
		{"Andreas", "Hoffmann", 1975, []empContractDef{
			{"qualified", "S8a", 6, 39, d(2015, 8, 1), nil, 2},
		}},
		{"Claudia", "Koch", 1989, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2021, 3, 1), nil, 2},
		}},
		{"Susanne", "Braun", 1987, []empContractDef{
			{"qualified", "S9", 3, 39, d(2021, 8, 1), nil, 2},
		}},
		{"Christian", "Schröder", 1985, []empContractDef{
			{"supplementary", "S4", 4, 39, d(2020, 1, 1), nil, 2},
		}},
		{"Markus", "Schmitt", 1991, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2022, 3, 1), nil, 2},
		}},
		// Deputy/coordinator
		{"Katrin", "Klein", 1982, []empContractDef{
			{"qualified", "S9", 5, 39, d(2016, 8, 1), nil, 2},
		}},

		// ===== Cross-section / support (3 active) =====
		{"Birgit", "Wolf", 1978, []empContractDef{
			{"non_pedagogical", "S4", 3, 20, d(2022, 4, 1), nil, -1},
		}},
		{"Inge", "Schwarz", 1970, []empContractDef{
			{"non_pedagogical", "S4", 5, 20, d(2018, 1, 1), nil, -1},
		}},
		{"Gisela", "Peters", 1965, []empContractDef{
			{"non_pedagogical", "Minijob", 1, 10, d(2023, 4, 1), nil, -1},
		}},

		// ===== Former employees =====
		{"Jürgen", "Lang", 1983, []empContractDef{
			{"qualified", "S8a", 3, 39, d(2019, 2, 1), tp(d(2022, 7, 31)), 1},
			{"qualified", "S8a", 4, 39, d(2022, 8, 1), tp(d(2025, 1, 31)), 2},
		}},
		{"Wolfgang", "Krüger", 1990, []empContractDef{
			{"qualified", "S8b", 2, 39, d(2021, 8, 1), tp(d(2024, 7, 31)), 0},
		}},
		{"Renate", "Meier", 1963, []empContractDef{
			{"qualified", "S8a", 6, 39, d(2010, 8, 1), tp(d(2023, 7, 31)), 2},
		}},

		// ===== Upcoming employees =====
		{"Lena", "Hofmann", 1999, []empContractDef{
			{"qualified", "S8a", 1, 39, now.AddDate(0, 1, 0), nil, 0},
		}},
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
	return from.AddDate(0, 0, rng.Intn(days))
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

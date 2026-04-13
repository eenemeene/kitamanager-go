package database

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestMigrationsEmbedded(t *testing.T) {
	// Verify migration files are properly embedded
	source, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)
	defer source.Close()

	// Should have at least version 1
	version, err := source.First()
	require.NoError(t, err)
	assert.Equal(t, uint(1), version)
}

func TestBuildDSN(t *testing.T) {
	cfg := &testConfig{
		host:     "localhost",
		port:     "5432",
		user:     "testuser",
		password: "testpass",
		dbName:   "testdb",
		sslMode:  "require",
	}

	dsn := BuildDSN(cfg.toConfig())
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=testpass")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=require")
}

func TestBuildDSN_DefaultSSLMode(t *testing.T) {
	cfg := &testConfig{
		host:     "localhost",
		port:     "5432",
		user:     "testuser",
		password: "testpass",
		dbName:   "testdb",
	}

	dsn := BuildDSN(cfg.toConfig())
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestBuildMigrateURL(t *testing.T) {
	cfg := &testConfig{
		host:     "localhost",
		port:     "5432",
		user:     "testuser",
		password: "testpass",
		dbName:   "testdb",
		sslMode:  "disable",
	}

	url := BuildMigrateURL(cfg.toConfig())
	assert.Equal(t, "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable", url)
}

func TestBuildMigrateURL_VerifyFull(t *testing.T) {
	cfg := &testConfig{
		host:     "db.example.com",
		port:     "5432",
		user:     "produser",
		password: "prodpass",
		dbName:   "proddb",
		sslMode:  "verify-full",
	}

	url := BuildMigrateURL(cfg.toConfig())
	assert.Equal(t, "postgres://produser:prodpass@db.example.com:5432/proddb?sslmode=verify-full", url)
}

// TestMigrationsRoundTrip verifies all migrations work in both directions:
// up (apply all) → down (revert all) → up (re-apply all).
func TestMigrationsRoundTrip(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("migrate_roundtrip_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	source, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	require.NoError(t, err)
	defer m.Close()

	// Step 1: Apply all migrations (up)
	err = m.Up()
	require.NoError(t, err, "migrations up failed")
	version, dirty, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirty, "database should not be dirty after up")
	t.Logf("after up: version=%d", version)

	// Step 2: Revert all migrations (down)
	err = m.Down()
	require.NoError(t, err, "migrations down failed")

	// Step 3: Re-apply all migrations (up again)
	err = m.Up()
	require.NoError(t, err, "migrations up (second pass) failed")
	version2, dirty2, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirty2, "database should not be dirty after second up")
	assert.Equal(t, version, version2, "version should match after round-trip")
	t.Logf("after round-trip: version=%d", version2)
}

// testConfig is a helper for constructing config.Config in tests.
type testConfig struct {
	host, port, user, password, dbName, sslMode string
}

func (tc *testConfig) toConfig() *config.Config {
	return &config.Config{
		DBHost:     tc.host,
		DBPort:     tc.port,
		DBUser:     tc.user,
		DBPassword: tc.password,
		DBName:     tc.dbName,
		DBSSLMode:  tc.sslMode,
	}
}

// startTestPostgres starts a PostgreSQL 18-alpine testcontainer and returns the connection string.
func startTestPostgres(t *testing.T, dbName string) string {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return connStr
}

// newMigrateInstance creates a new golang-migrate instance using the embedded migration files.
func newMigrateInstance(t *testing.T, connStr string) *migrate.Migrate {
	t.Helper()
	source, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)
	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	require.NoError(t, err)
	return m
}

// openTestGormDB opens a GORM connection to the given database URL.
func openTestGormDB(t *testing.T, connStr string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormPostgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

// columnInfo represents a column from information_schema.columns.
type columnInfo struct {
	TableName  string `gorm:"column:table_name"`
	ColumnName string `gorm:"column:column_name"`
	DataType   string `gorm:"column:data_type"`
	IsNullable string `gorm:"column:is_nullable"`
}

// snapshotSchema returns all columns from the public schema, excluding schema_migrations.
func snapshotSchema(t *testing.T, db *gorm.DB) []columnInfo {
	t.Helper()
	var columns []columnInfo
	err := db.Raw(`
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name != 'schema_migrations'
		ORDER BY table_name, column_name
	`).Scan(&columns).Error
	require.NoError(t, err)
	return columns
}

// normalizeDataType maps SQL-specific types to what GORM would generate.
// SQL migrations intentionally use more specific types (integer, double precision, jsonb)
// while GORM maps Go types differently (int→bigint, float64→numeric, serializer:json→text).
// Normalizing lets the drift test focus on structural issues (missing columns, nullability).
func normalizeDataType(dataType string) string {
	switch dataType {
	case "integer":
		return "bigint"
	case "double precision":
		return "numeric"
	case "jsonb":
		return "text"
	default:
		return dataType
	}
}

// allModels returns every GORM model that maps to a database table.
func allModels() []any {
	return []any{
		&models.Organization{},
		&models.User{},
		&models.UserOrganization{},
		&models.Section{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
		&models.ChildAttendance{},
		&models.GovernmentFunding{},
		&models.GovernmentFundingPeriod{},
		&models.GovernmentFundingProperty{},
		&models.PayPlan{},
		&models.PayPlanPeriod{},
		&models.PayPlanEntry{},
		&models.BudgetItem{},
		&models.BudgetItemEntry{},
		&models.AuditLog{},
		&models.RevokedToken{},
		&models.GovernmentFundingBillPeriod{},
		&models.GovernmentFundingBillChild{},
		&models.GovernmentFundingBillPayment{},
	}
}

// TestMigrationSchemaMatchesModels detects drift between SQL migrations and GORM models.
// It applies all SQL migrations, snapshots the schema, runs GORM AutoMigrate, and verifies
// AutoMigrate made no changes — proving the SQL migrations fully satisfy the model definitions.
func TestMigrationSchemaMatchesModels(t *testing.T) {
	connStr := startTestPostgres(t, "schema_drift_test")

	// Apply all SQL migrations
	m := newMigrateInstance(t, connStr)
	require.NoError(t, m.Up(), "SQL migrations failed")
	m.Close()

	// Open GORM connection
	db := openTestGormDB(t, connStr)

	// Snapshot schema before AutoMigrate
	before := snapshotSchema(t, db)
	require.NotEmpty(t, before, "schema should have columns after migrations")

	// Verify allModels() covers every table created by migrations.
	// Catches the case where a new migration adds a table but allModels() isn't updated.
	dbTables := make(map[string]bool)
	for _, col := range before {
		dbTables[col.TableName] = true
	}
	modelTables := make(map[string]bool)
	for _, model := range allModels() {
		stmt := &gorm.Statement{DB: db}
		require.NoError(t, stmt.Parse(model))
		modelTables[stmt.Schema.Table] = true
	}
	for table := range dbTables {
		if !modelTables[table] {
			t.Errorf("table %q exists in database but has no model in allModels()", table)
		}
	}

	// Run AutoMigrate with all models
	err := db.AutoMigrate(allModels()...)
	require.NoError(t, err, "AutoMigrate failed")

	// Snapshot schema after AutoMigrate
	after := snapshotSchema(t, db)

	// Normalize the before snapshot to account for known GORM-vs-SQL type mapping differences.
	// SQL migrations intentionally use integer/double precision/jsonb, while GORM maps
	// Go types to bigint/numeric/text. Normalizing avoids false positives from these
	// expected differences while still catching missing columns and nullability changes.
	normalized := make([]columnInfo, len(before))
	for i, col := range before {
		normalized[i] = col
		normalized[i].DataType = normalizeDataType(col.DataType)
	}

	// Compare — if AutoMigrate changed anything beyond known type mappings, SQL migrations need updating
	if !assert.Equal(t, normalized, after, "AutoMigrate changed the schema — SQL migrations are missing something") {
		beforeMap := make(map[string]columnInfo)
		for _, c := range normalized {
			beforeMap[c.TableName+"."+c.ColumnName] = c
		}
		afterMap := make(map[string]columnInfo)
		for _, c := range after {
			afterMap[c.TableName+"."+c.ColumnName] = c
		}

		for key, col := range afterMap {
			if _, exists := beforeMap[key]; !exists {
				t.Errorf("  added column: %s (type=%s, nullable=%s)", key, col.DataType, col.IsNullable)
			}
		}
		for key := range beforeMap {
			if _, exists := afterMap[key]; !exists {
				t.Errorf("  removed column: %s", key)
			}
		}
		for key, b := range beforeMap {
			if a, exists := afterMap[key]; exists && b != a {
				t.Errorf("  changed column %s: before{type=%s,nullable=%s} after{type=%s,nullable=%s}",
					key, b.DataType, b.IsNullable, a.DataType, a.IsNullable)
			}
		}
	}
}

// TestMigrationsWithData applies the migration, inserts representative data into
// every table, and verifies data can be inserted correctly.
func TestMigrationsWithData(t *testing.T) {
	connStr := startTestPostgres(t, "populated_data_test")

	m := newMigrateInstance(t, connStr)
	defer m.Close()

	// Step 1: Apply all migrations
	require.NoError(t, m.Up(), "migrations failed")
	version, dirty, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirty)
	t.Logf("applied migrations: version=%d", version)

	// Step 2: Insert representative data into every table.
	db := openTestGormDB(t, connStr)

	inserts := []string{
		`INSERT INTO organizations (id, name, active, state, created_at, updated_at)
		 VALUES (1, 'Test Org', true, 'berlin', NOW(), NOW())`,

		`INSERT INTO users (id, name, email, password, active, is_superadmin, created_at, updated_at)
		 VALUES (1, 'Test User', 'test@example.com', 'hashed', true, false, NOW(), NOW())`,

		`INSERT INTO user_organizations (user_id, organization_id, role, created_at)
		 VALUES (1, 1, 'admin', NOW())`,

		`INSERT INTO sections (id, organization_id, name, is_default, created_at, updated_at)
		 VALUES (1, 1, 'Kita', true, NOW(), NOW())`,

		`INSERT INTO employees (id, organization_id, first_name, last_name, gender, birthdate, created_at, updated_at)
		 VALUES (1, 1, 'Jane', 'Doe', 'female', '1990-01-01', NOW(), NOW())`,

		`INSERT INTO pay_plans (id, organization_id, name, created_at, updated_at)
		 VALUES (1, 1, 'TVöD-SuE', NOW(), NOW())`,

		`INSERT INTO employee_contracts (id, employee_id, from_date, section_id, staff_category, pay_plan_id, created_at, updated_at)
		 VALUES (1, 1, '2024-01-01', 1, 'qualified', 1, NOW(), NOW())`,

		`INSERT INTO children (id, organization_id, first_name, last_name, gender, birthdate, created_at, updated_at)
		 VALUES (1, 1, 'Max', 'Schmidt', 'male', '2020-06-15', NOW(), NOW())`,

		`INSERT INTO child_contracts (id, child_id, from_date, section_id, created_at, updated_at)
		 VALUES (1, 1, '2024-01-01', 1, NOW(), NOW())`,

		`INSERT INTO government_fundings (id, name, state, created_at, updated_at)
		 VALUES (1, 'Berlin Funding', 'berlin', NOW(), NOW())`,

		`INSERT INTO government_funding_periods (id, government_funding_id, from_date, full_time_weekly_hours, created_at)
		 VALUES (1, 1, '2024-01-01', 39.0, NOW())`,

		`INSERT INTO government_funding_properties (id, period_id, key, value, label, payment, requirement, created_at)
		 VALUES (1, 1, 'care_type', 'ganztag', 'Ganztag', 166847, 0.261, NOW())`,

		`INSERT INTO pay_plan_periods (id, pay_plan_id, from_date, weekly_hours, created_at, updated_at)
		 VALUES (1, 1, '2024-01-01', 39.0, NOW(), NOW())`,

		`INSERT INTO pay_plan_entries (id, period_id, grade, step, monthly_amount, created_at, updated_at)
		 VALUES (1, 1, 'S8a', 3, 350000, NOW(), NOW())`,

		`INSERT INTO child_attendances (id, child_id, organization_id, date, status, recorded_by, created_at, updated_at)
		 VALUES (1, 1, 1, '2024-06-15', 'present', 1, NOW(), NOW())`,

		`INSERT INTO budget_items (id, organization_id, name, category, per_child, created_at, updated_at)
		 VALUES (1, 1, 'Rent', 'expense', false, NOW(), NOW())`,

		`INSERT INTO budget_item_entries (id, budget_item_id, from_date, amount_cents, created_at, updated_at)
		 VALUES (1, 1, '2024-01-01', 50000, NOW(), NOW())`,

		`INSERT INTO audit_logs (id, timestamp, action, success)
		 VALUES (1, NOW(), 'login', true)`,

		`INSERT INTO revoked_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (1, 1, 'abc123def456abc123def456abc123def456abc123def456abc123def456abcd', NOW() + INTERVAL '1 hour', NOW())`,

		`INSERT INTO government_funding_bill_periods (id, organization_id, from_date, file_name, file_sha256, facility_name, facility_total, contract_booking, correction_booking, created_by, created_at)
		 VALUES (1, 1, '2024-01-01', 'test.xlsx', 'abc123', 'Test Kita', 100000, 90000, 10000, 1, NOW())`,

		`INSERT INTO government_funding_bill_children (id, period_id, voucher_number, child_name, birth_date, district)
		 VALUES (1, 1, 'V-12345', 'Max Schmidt', '2020-06-15', 1)`,

		`INSERT INTO government_funding_bill_payments (id, child_id, key, value, amount)
		 VALUES (1, 1, 'care_type', 'ganztag', 166847)`,
	}

	for _, insert := range inserts {
		require.NoError(t, db.Exec(insert).Error, "insert failed: %.60s", insert)
	}
	t.Log("inserted test data into all tables")

	// Step 3: Verify all data was inserted correctly
	tables := []string{
		"organizations", "users", "user_organizations", "sections",
		"employees", "employee_contracts", "children", "child_contracts",
		"child_attendances", "government_fundings", "government_funding_periods",
		"government_funding_properties", "pay_plans", "pay_plan_periods",
		"pay_plan_entries", "budget_items", "budget_item_entries",
		"audit_logs", "revoked_tokens",
		"government_funding_bill_periods", "government_funding_bill_children",
		"government_funding_bill_payments",
	}

	for _, table := range tables {
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "table %s should have 1 row", table)
	}
	t.Log("all row counts verified")
}

//go:build integration

package scripts_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eenemeene/kitamanager-go/internal/database"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// containerInfo holds connection details for a test Postgres container.
type containerInfo struct {
	container   *postgres.PostgresContainer
	containerID string
	connStr     string
	host        string
	port        string
	user        string
	password    string
	dbName      string
}

func startPostgres(t *testing.T, ctx context.Context, dbName string) *containerInfo {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
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
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	mappedPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	containerID := pgContainer.GetContainerID()

	return &containerInfo{
		container:   pgContainer,
		containerID: containerID,
		connStr:     connStr,
		host:        host,
		port:        mappedPort.Port(),
		user:        "test",
		password:    "test",
		dbName:      dbName,
	}
}

func openDB(t *testing.T, connStr string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open gorm connection: %v", err)
	}
	return db
}

// seedData inserts representative data covering the main tables.
func seedData(t *testing.T, db *gorm.DB) {
	t.Helper()

	org := &models.Organization{Name: "Test Kita", Active: true, State: string(models.StateBerlin)}
	must(t, db.Create(org).Error)

	section := &models.Section{OrganizationID: org.ID, Name: "Group A", IsDefault: true, CreatedBy: "test"}
	must(t, db.Create(section).Error)

	user := &models.User{Name: "Admin", Email: "admin@test.com", Password: "hashed", Active: true, IsSuperAdmin: true}
	must(t, db.Create(user).Error)

	uo := &models.UserOrganization{UserID: user.ID, OrganizationID: org.ID, Role: models.RoleAdmin, CreatedBy: "test"}
	must(t, db.Create(uo).Error)

	child := &models.Child{Person: models.Person{
		OrganizationID: org.ID,
		FirstName:      "Max",
		LastName:       "Mustermann",
		Birthdate:      time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
	}}
	must(t, db.Create(child).Error)

	childContract := &models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
			SectionID:  section.ID,
			Properties: models.ContractProperties{"care_type": "ganztag", "age_group": "over3"},
		},
	}
	must(t, db.Create(childContract).Error)

	emp := &models.Employee{Person: models.Person{
		OrganizationID: org.ID,
		FirstName:      "Erika",
		LastName:       "Musterfrau",
		Birthdate:      time.Date(1990, 3, 20, 0, 0, 0, 0, time.UTC),
	}}
	must(t, db.Create(emp).Error)

	payPlan := &models.PayPlan{OrganizationID: org.ID, Name: "TV-L"}
	must(t, db.Create(payPlan).Error)

	empContract := &models.EmployeeContract{
		EmployeeID: emp.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC),
			},
			SectionID: section.ID,
		},
		WeeklyHours:   40,
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          3,
		PayPlanID:     payPlan.ID,
	}
	must(t, db.Create(empContract).Error)

	payPlanPeriod := &models.PayPlanPeriod{
		PayPlanID: payPlan.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   timePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
	}
	must(t, db.Create(payPlanPeriod).Error)

	payPlanEntry := &models.PayPlanEntry{
		PeriodID:      payPlanPeriod.ID,
		Grade:         "S8a",
		Step:          3,
		MonthlyAmount: 350000, // cents
	}
	must(t, db.Create(payPlanEntry).Error)

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: string(models.StateBerlin)}
	must(t, db.Create(funding).Error)

	fundingPeriod := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   timePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		FullTimeWeeklyHours: 39.0,
	}
	must(t, db.Create(fundingPeriod).Error)

	fundingProp := &models.GovernmentFundingProperty{
		PeriodID:    fundingPeriod.ID,
		Key:         "care_type",
		Value:       "ganztag",
		Label:       "Ganztag",
		Payment:     83424, // cents
		Requirement: 0.261,
	}
	must(t, db.Create(fundingProp).Error)

	budgetItem := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Office Supplies",
		Category:       "expense",
	}
	must(t, db.Create(budgetItem).Error)

	budgetEntry := &models.BudgetItemEntry{
		BudgetItemID: budgetItem.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   timePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		AmountCents: 50000, // cents
	}
	must(t, db.Create(budgetEntry).Error)
}

// tableRowCount returns the number of rows in a table.
func tableRowCount(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count).Error; err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}
	return count
}

// tableChecksum returns an MD5 checksum of all rows in a table cast to text, providing
// a content-level comparison without needing to scan into Go types.
func tableChecksum(t *testing.T, db *gorm.DB, table string) string {
	t.Helper()
	var checksum string
	err := db.Raw(fmt.Sprintf(
		"SELECT COALESCE(md5(string_agg(row_text, '' ORDER BY row_text)), '') FROM (SELECT %s::text AS row_text FROM %s) sub",
		table, table,
	)).Scan(&checksum).Error
	if err != nil {
		t.Fatalf("failed to checksum table %s: %v", table, err)
	}
	return checksum
}

// userTables returns all non-system tables in the database.
func userTables(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var tables []string
	err := db.Raw(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		  AND table_name != 'schema_migrations'
		ORDER BY table_name
	`).Scan(&tables).Error
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}
	sort.Strings(tables)
	return tables
}

// dockerExec runs a command inside a container and returns the combined output.
func dockerExec(t *testing.T, containerID string, env []string, args ...string) []byte {
	t.Helper()

	cmdArgs := []string{"exec"}
	for _, e := range env {
		cmdArgs = append(cmdArgs, "-e", e)
	}
	cmdArgs = append(cmdArgs, containerID)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker exec %v failed: %v\n%s", args, err, out)
	}
	return out
}

func TestBackupAndRestore(t *testing.T) {
	ctx := context.Background()

	// Start source database and seed data.
	src := startPostgres(t, ctx, "backup_src")
	if err := database.RunMigrationsWithURL(src.connStr); err != nil {
		t.Fatalf("failed to run migrations on source: %v", err)
	}
	srcDB := openDB(t, src.connStr)
	seedData(t, srcDB)

	// Run pg_dump inside the source container to avoid host pg_dump version mismatches.
	backupFile := filepath.Join(t.TempDir(), "backup.sql.gz")

	dockerExec(t, src.containerID,
		[]string{"PGPASSWORD=" + src.password},
		"pg_dump",
		"--host=localhost",
		"--username="+src.user,
		"--dbname="+src.dbName,
		"--format=plain",
		"--no-owner",
		"--no-privileges",
		"--file=/tmp/backup.sql",
	)

	// Copy dump out of container.
	sqlFile := filepath.Join(t.TempDir(), "backup.sql")
	cpCmd := exec.Command("docker", "cp", src.containerID+":/tmp/backup.sql", sqlFile)
	if out, err := cpCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker cp failed: %v\n%s", err, out)
	}

	// Gzip to match the script's output format.
	gzCmd := exec.Command("gzip", "-c", sqlFile)
	gzOut, err := os.Create(backupFile)
	if err != nil {
		t.Fatalf("failed to create backup file: %v", err)
	}
	gzCmd.Stdout = gzOut
	if err := gzCmd.Run(); err != nil {
		gzOut.Close()
		t.Fatalf("gzip failed: %v", err)
	}
	gzOut.Close()

	// Verify the backup file exists and is non-empty.
	info, err := os.Stat(backupFile)
	if err != nil {
		t.Fatalf("backup file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("backup file is empty")
	}

	// Start destination database (no migrations — restore will create schema).
	dst := startPostgres(t, ctx, "backup_dst")

	// Restore: gunzip locally, copy into dst container, run psql inside the container.
	restoreFile := filepath.Join(t.TempDir(), "restore.sql")
	gunzipCmd := exec.Command("gunzip", "-c", backupFile)
	sqlOut, err := os.Create(restoreFile)
	if err != nil {
		t.Fatalf("failed to create restore sql file: %v", err)
	}
	gunzipCmd.Stdout = sqlOut
	if err := gunzipCmd.Run(); err != nil {
		sqlOut.Close()
		t.Fatalf("gunzip failed: %v", err)
	}
	sqlOut.Close()

	// Copy SQL into destination container.
	cpDstCmd := exec.Command("docker", "cp", restoreFile, dst.containerID+":/tmp/restore.sql")
	if out, cpErr := cpDstCmd.CombinedOutput(); cpErr != nil {
		t.Fatalf("docker cp to dst failed: %v\n%s", cpErr, out)
	}

	// Run psql inside the destination container.
	dockerExec(t, dst.containerID,
		[]string{"PGPASSWORD=" + dst.password},
		"psql",
		"--host=localhost",
		"--username="+dst.user,
		"--dbname="+dst.dbName,
		"--file=/tmp/restore.sql",
	)

	dstDB := openDB(t, dst.connStr)

	// Compare tables.
	srcTables := userTables(t, srcDB)
	dstTables := userTables(t, dstDB)

	if strings.Join(srcTables, ",") != strings.Join(dstTables, ",") {
		t.Fatalf("table mismatch:\n  src: %v\n  dst: %v", srcTables, dstTables)
	}

	for _, table := range srcTables {
		srcCount := tableRowCount(t, srcDB, table)
		dstCount := tableRowCount(t, dstDB, table)
		if srcCount != dstCount {
			t.Errorf("table %s: row count mismatch (src=%d, dst=%d)", table, srcCount, dstCount)
			continue
		}

		srcSum := tableChecksum(t, srcDB, table)
		dstSum := tableChecksum(t, dstDB, table)
		if srcSum != dstSum {
			t.Errorf("table %s: checksum mismatch (src=%s, dst=%s)", table, srcSum, dstSum)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

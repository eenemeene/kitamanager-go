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

	"github.com/eenemeene/kitamanager-go/internal/config"
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

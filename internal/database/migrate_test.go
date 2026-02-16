package database

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

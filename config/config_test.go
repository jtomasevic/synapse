package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// DSN
// =========================================================================

func TestDatabaseConfig_DSN(t *testing.T) {
	db := DatabaseConfig{
		Host:     "db.example.com",
		Port:     5433,
		User:     "admin",
		Password: "s3cret",
		DBName:   "mydb",
		SSLMode:  "require",
	}
	want := "postgres://admin:s3cret@db.example.com:5433/mydb?sslmode=require"
	assert.Equal(t, want, db.DSN())
}

func TestDatabaseConfig_DSN_Defaults(t *testing.T) {
	db := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "synapse",
		Password: "synapse",
		DBName:   "synapse",
		SSLMode:  "disable",
	}
	want := "postgres://synapse:synapse@localhost:5432/synapse?sslmode=disable"
	assert.Equal(t, want, db.DSN())
}

// =========================================================================
// Load
// =========================================================================

const validYAML = `server:
  addr: ":9090"
database:
  host: "pg.local"
  port: 5433
  user: "testuser"
  password: "testpass"
  dbname: "testdb"
  sslmode: "require"
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLoad_ValidFile(t *testing.T) {
	path := writeTemp(t, validYAML)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.Server.Addr)
	assert.Equal(t, "pg.local", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.DBName)
	assert.Equal(t, "require", cfg.Database.SSLMode)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeTemp(t, "server:\n  addr: [[[invalid")

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestLoad_PartialYAML(t *testing.T) {
	path := writeTemp(t, `server:
  addr: ":3000"
`)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, ":3000", cfg.Server.Addr)
	// Database fields remain zero-valued.
	assert.Equal(t, "", cfg.Database.Host)
	assert.Equal(t, 0, cfg.Database.Port)
}

// =========================================================================
// ApplyEnv
// =========================================================================

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestApplyEnv_OverridesAll(t *testing.T) {
	cfg := Config{}

	setEnv(t, "SYNAPSE_ADDR", ":7070")
	setEnv(t, "SYNAPSE_DB_HOST", "override-host")
	setEnv(t, "SYNAPSE_DB_PORT", "5555")
	setEnv(t, "SYNAPSE_DB_USER", "override-user")
	setEnv(t, "SYNAPSE_DB_PASSWORD", "override-pass")
	setEnv(t, "SYNAPSE_DB_NAME", "override-db")
	setEnv(t, "SYNAPSE_DB_SSLMODE", "verify-full")

	cfg.ApplyEnv()

	assert.Equal(t, ":7070", cfg.Server.Addr)
	assert.Equal(t, "override-host", cfg.Database.Host)
	assert.Equal(t, 5555, cfg.Database.Port)
	assert.Equal(t, "override-user", cfg.Database.User)
	assert.Equal(t, "override-pass", cfg.Database.Password)
	assert.Equal(t, "override-db", cfg.Database.DBName)
	assert.Equal(t, "verify-full", cfg.Database.SSLMode)
}

func TestApplyEnv_NoOverrideWhenEmpty(t *testing.T) {
	cfg := Config{
		Server:   ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{Host: "original", Port: 5432},
	}

	// All env vars are unset by default in t.Setenv scope — nothing changes.
	cfg.ApplyEnv()

	assert.Equal(t, ":8080", cfg.Server.Addr)
	assert.Equal(t, "original", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestApplyEnv_InvalidPortIgnored(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{Port: 5432},
	}

	setEnv(t, "SYNAPSE_DB_PORT", "not-a-number")
	cfg.ApplyEnv()

	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestApplyEnv_PartialOverride(t *testing.T) {
	cfg := Config{
		Server: ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "synapse",
			Password: "synapse",
			DBName:   "synapse",
			SSLMode:  "disable",
		},
	}

	setEnv(t, "SYNAPSE_DB_PASSWORD", "prod-secret")

	cfg.ApplyEnv()

	assert.Equal(t, ":8080", cfg.Server.Addr)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "prod-secret", cfg.Database.Password)
	assert.Equal(t, "synapse", cfg.Database.User)
}

// =========================================================================
// Integration: Load → ApplyEnv → DSN
// =========================================================================

func TestLoadAndApplyEnv_Integration(t *testing.T) {
	path := writeTemp(t, validYAML)
	setEnv(t, "SYNAPSE_DB_PASSWORD", "env-override")

	cfg, err := Load(path)
	require.NoError(t, err)
	cfg.ApplyEnv()

	want := "postgres://testuser:env-override@pg.local:5433/testdb?sslmode=require"
	assert.Equal(t, want, cfg.Database.DSN())
}

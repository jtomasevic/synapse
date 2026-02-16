// Package config provides typed, file-based configuration for the Synapse
// application.
//
// Configuration is loaded from a YAML file and can be selectively overridden
// by environment variables. The resolution order (lowest → highest priority):
//
//  1. Defaults embedded in the struct tags / zero values.
//  2. Values read from the YAML file via [Load].
//  3. Environment variable overrides applied by [Config.ApplyEnv].
//  4. CLI flags (handled by the caller, e.g. cmd/synapse-api).
package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr string `yaml:"addr"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// DSN builds a PostgreSQL connection string from the individual fields.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// Load reads and parses a YAML configuration file at the given path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// ApplyEnv overrides configuration values with environment variables when
// they are set. Supported variables:
//
//	SYNAPSE_ADDR        → Server.Addr
//	SYNAPSE_DB_HOST     → Database.Host
//	SYNAPSE_DB_PORT     → Database.Port
//	SYNAPSE_DB_USER     → Database.User
//	SYNAPSE_DB_PASSWORD → Database.Password
//	SYNAPSE_DB_NAME     → Database.DBName
//	SYNAPSE_DB_SSLMODE  → Database.SSLMode
func (c *Config) ApplyEnv() {
	if v := os.Getenv("SYNAPSE_ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv("SYNAPSE_DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("SYNAPSE_DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Database.Port = p
		}
	}
	if v := os.Getenv("SYNAPSE_DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("SYNAPSE_DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("SYNAPSE_DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("SYNAPSE_DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}
}

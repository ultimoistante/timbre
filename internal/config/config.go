// Package config loads server configuration from environment variables and
// flags, with sensible defaults for a self-hosted single-binary deployment.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all runtime configuration.
type Config struct {
	Host string
	Port int

	// DataDir is the root for all server state: database, per-user media
	// roots (<DataDir>/users/<id>) and the JWT secret file.
	DataDir string

	// DBDriver is "sqlite" (default) or "postgres".
	DBDriver string
	// DBDSN is the connection string. For sqlite, empty means
	// <DataDir>/timbre.db.
	DBDSN string

	JWTSecret  []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Load builds a Config from the environment, applying defaults and ensuring
// the data directory and JWT secret exist.
func Load() (*Config, error) {
	c := &Config{
		Host:       env("TIMBRE_HOST", "0.0.0.0"),
		Port:       envInt("TIMBRE_PORT", 8080),
		DataDir:    env("TIMBRE_DATA_DIR", "./data"),
		DBDriver:   env("TIMBRE_DB_DRIVER", "sqlite"),
		DBDSN:      env("TIMBRE_DB_DSN", ""),
		AccessTTL:  time.Duration(envInt("TIMBRE_ACCESS_TTL_MIN", 30)) * time.Minute,
		RefreshTTL: time.Duration(envInt("TIMBRE_REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
	}

	abs, err := filepath.Abs(c.DataDir)
	if err != nil {
		return nil, err
	}
	c.DataDir = abs

	if err := os.MkdirAll(c.UsersDir(), 0o755); err != nil {
		return nil, err
	}

	if c.DBDriver == "sqlite" && c.DBDSN == "" {
		c.DBDSN = filepath.Join(c.DataDir, "timbre.db")
	}

	secret, err := loadOrCreateSecret(filepath.Join(c.DataDir, "jwt.secret"))
	if err != nil {
		return nil, err
	}
	c.JWTSecret = secret

	return c, nil
}

// UsersDir is the parent directory holding every user's isolated media root.
func (c *Config) UsersDir() string {
	return filepath.Join(c.DataDir, "users")
}

func loadOrCreateSecret(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err == nil && len(b) >= 32 {
		decoded, derr := hex.DecodeString(string(b))
		if derr == nil && len(decoded) >= 32 {
			return decoded, nil
		}
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(secret)), 0o600); err != nil {
		return nil, err
	}
	return secret, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

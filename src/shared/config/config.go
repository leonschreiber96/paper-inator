// Package config loads runtime configuration from command-line flags with
// environment-variable fallbacks. Keeping configuration in one small struct
// keeps the entrypoint simple and makes the worker and server easy to test.
package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings for the unified binary.
type Config struct {
	DBPath        string        // SQLite database file path
	Addr          string        // HTTP listen address, e.g. ":8080"
	FetchInterval time.Duration // default poll interval when a feed has none set

	// SMTP settings are read for the (deferred) email-summary feature. They are
	// loaded now so the configuration surface is stable.
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

// Load parses flags (with env fallbacks) and returns the resulting Config.
// Precedence: explicit flag > environment variable > built-in default.
func Load() *Config {
	c := &Config{}

	flag.StringVar(&c.DBPath, "db", env("PAPERINATOR_DB", "./paper-inator.db"), "path to the SQLite database file")
	flag.StringVar(&c.Addr, "addr", env("PAPERINATOR_ADDR", ":8080"), "HTTP listen address")
	interval := flag.Duration("fetch-interval", envDuration("PAPERINATOR_FETCH_INTERVAL", 15*time.Minute), "default feed poll interval")
	flag.Parse()
	c.FetchInterval = *interval

	// SMTP comes from the environment only (secrets do not belong on the command line).
	c.SMTPHost = env("PAPERINATOR_SMTP_HOST", "")
	c.SMTPPort = envInt("PAPERINATOR_SMTP_PORT", 587)
	c.SMTPUser = env("PAPERINATOR_SMTP_USER", "")
	c.SMTPPass = env("PAPERINATOR_SMTP_PASS", "")
	c.SMTPFrom = env("PAPERINATOR_SMTP_FROM", "")

	return c
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

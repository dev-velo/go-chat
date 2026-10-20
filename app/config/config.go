// Package config loads runtime settings from the environment.
//
// Variables come in two kinds:
//
//   - Optional — a safe default exists, so the app runs without them.
//   - Required — no sane default exists. A missing one is a hard startup
//     failure: Load returns an error and the process must not serve traffic.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Backends accepted by STORE_BACKEND.
const (
	// StoreMemory keeps exchanges in process memory
	StoreMemory = "memory"
	// StoreS3 persists exchanges to S3.
	StoreS3 = "s3"
)

// Config is the resolved runtime configuration.
type Config struct {
	Port         string // Port is the TCP port the server listens on. Optional.
	GinMode      string // GinMode is the gin framework mode. Optional.
	StoreBackend string // StoreBackend is the persistence backend. Required.
}

// Error reports the required variables that were missing. All listed together
type Error struct {
	Missing []string
}

func (e *Error) Error() string {
	return fmt.Sprintf("missing required environment variable(s): %s",
		strings.Join(e.Missing, ", "))
}

// Load reads configuration from the environment. A non-nil error means the
// caller must exit rather than start the server.
func Load() (Config, error) {
	var l loader

	cfg := Config{
		Port:         l.optional("PORT", "8080"),
		GinMode:      l.optional("GIN_MODE", "release"),
		StoreBackend: l.required("STORE_BACKEND"),
	}
	if len(l.missing) > 0 {
		return Config{}, &Error{Missing: l.missing}
	}
	return cfg, nil
}

// loader reads variables and collects missing ones instead of failing on the
// first.
type loader struct {
	missing []string
}

// optional returns the variable's value, or fallback when unset or blank.
func (l *loader) optional(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// required returns the variable's value and records it as missing when unset or blank.
func (l *loader) required(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		l.missing = append(l.missing, key)
	}
	return v
}

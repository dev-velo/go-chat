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
// options:
// in-memory persistence
// s3 persistence
const (
	StoreMemory = "memory"
	StoreS3     = "s3"
)

// Providers accepted by LLM_PROVIDER.
// options:
// gpt
// claude
const (
	ProviderFake      = "fake"
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
)

// Config is the resolved runtime configuration.
type Config struct {
	Port           string // Port is the TCP port the server listens on. Optional.
	GinMode        string // GinMode is the gin framework mode. Optional.
	StoreBackend   string // StoreBackend is the persistence backend. Required.
	S3BucketName   string
	S3BucketPath   string
	LLMProvider    string // LLMProvider is the model backend. Required.
	LLMModel       string // LLMModel is the provider-specific model id. Required.
	LLMAPIKey      string // LLMAPIKey is the provider-specific API key.
	SSMKeyName     string // Parameter name used when SSM fetching is enabled.
	AppUserName    string
	AppUserPass    string
	SSMUserKeyName string
	SSMPassKeyName string
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
		S3BucketName: l.optional("S3_BUCKET_NAME", ""),
		S3BucketPath: l.optional("S3_BUCKET_PATH", ""),
		LLMProvider:  l.required("LLM_PROVIDER"),
		LLMModel:     l.required("LLM_MODEL"),
	}
	if cfg.StoreBackend == StoreS3 {
		cfg.S3BucketName = l.required("S3_BUCKET_NAME")
		cfg.SSMKeyName = l.required("SSM_KEY_NAME")
		cfg.SSMUserKeyName = l.required("SSM_USER_KEY_NAME")
		cfg.SSMPassKeyName = l.required("SSM_PASS_KEY_NAME")
	} else if cfg.StoreBackend == StoreMemory || cfg.StoreBackend == "" {
		cfg.LLMAPIKey = l.required("LLM_API_KEY")
		cfg.AppUserName = l.required("APP_USER_NAME")
		cfg.AppUserPass = l.required("APP_USER_PASS")
	} else {
		return Config{}, fmt.Errorf("unknown STORE_BACKEND %q (want memory or s3)", cfg.StoreBackend)
	}
	if len(l.missing) > 0 {
		return Config{}, &Error{Missing: l.missing}
	}
	if strings.Contains(cfg.AppUserName, ":") {
		return Config{}, fmt.Errorf("APP_USER_NAME must not contain a colon")
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

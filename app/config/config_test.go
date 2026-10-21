package config

import (
	"strings"
	"testing"
)

// clearEnv blanks every variable Load reads so the ambient environment cannot
// influence a test. t.Setenv restores the previous values when the test ends.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT", "GIN_MODE", "STORE_BACKEND", "S3_BUCKET_NAME", "S3_BUCKET_PATH", "LLM_PROVIDER", "LLM_MODEL", "LLM_API_KEY", "SSM_KEY_NAME", "SSM_USER_KEY_NAME", "SSM_PASS_KEY_NAME", "APP_USER_NAME", "APP_USER_PASS",
	} {
		t.Setenv(key, "")
	}

	// Every test that expects Load to succeed needs the required variables
	// set. Tests about their absence clear them again.
	t.Setenv("STORE_BACKEND", StoreMemory)
	t.Setenv("LLM_PROVIDER", ProviderFake)
	t.Setenv("LLM_MODEL", "a-model")
	t.Setenv("LLM_API_KEY", "sk-test-value")
	t.Setenv("APP_USER_NAME", "tester")
	t.Setenv("APP_USER_PASS", "test-password")
	t.Setenv("SSM_KEY_NAME", "/llm")
	t.Setenv("SSM_USER_KEY_NAME", "/user")
	t.Setenv("SSM_PASS_KEY_NAME", "/pass")
}

func TestLoadAcceptsSSMParameter(t *testing.T) {
	clearEnv(t)
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("STORE_BACKEND", StoreS3)
	t.Setenv("S3_BUCKET_NAME", "bucket")
	t.Setenv("SSM_KEY_NAME", " /go-chat/dev/llm-api-key ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMAPIKey != "" || cfg.AppUserName != "" || cfg.AppUserPass != "" || cfg.SSMKeyName != "/go-chat/dev/llm-api-key" {
		t.Fatal("expected only the trimmed SSM parameter name")
	}
}

func TestLoadSSMOverridesDirectKey(t *testing.T) {
	clearEnv(t)
	t.Setenv("STORE_BACKEND", StoreS3)
	t.Setenv("S3_BUCKET_NAME", "bucket")
	t.Setenv("SSM_KEY_NAME", "/myapp/llm_api_key")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMAPIKey != "" || cfg.AppUserName != "" || cfg.AppUserPass != "" {
		t.Fatal("SSM must replace the direct key when enabled")
	}
}

func TestLoadMemoryUsesDirectSecrets(t *testing.T) {
	for _, missing := range []string{"LLM_API_KEY", "APP_USER_NAME", "APP_USER_PASS"} {
		clearEnv(t)
		t.Setenv("SSM_KEY_NAME", "/myapp/llm_api_key")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.LLMAPIKey != "sk-test-value" || cfg.AppUserName != "tester" || cfg.AppUserPass != "test-password" {
			t.Fatal("disabled SSM must use the direct key")
		}
		t.Setenv(missing, "")
		if _, err := Load(); err == nil {
			t.Fatal("disabled SSM must require a direct key")
		}
	}
}

func TestLoadSSMRequiresParameterName(t *testing.T) {
	clearEnv(t)
	t.Setenv("STORE_BACKEND", StoreS3)
	t.Setenv("S3_BUCKET_NAME", "bucket")
	for _, key := range []string{"SSM_KEY_NAME", "SSM_USER_KEY_NAME", "SSM_PASS_KEY_NAME"} {
		t.Setenv(key, "")
	}
	_, err := Load()
	for _, key := range []string{"SSM_KEY_NAME", "SSM_USER_KEY_NAME", "SSM_PASS_KEY_NAME"} {
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Fatalf("missing parameter must be reported: %s", key)
		}
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with an empty environment must succeed, got: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.GinMode != "release" {
		t.Errorf("GinMode = %q, want %q", cfg.GinMode, "release")
	}
}

func TestLoadRequiresStoreBackend(t *testing.T) {
	clearEnv(t)
	t.Setenv("STORE_BACKEND", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load without STORE_BACKEND must fail")
	}
	if !strings.Contains(err.Error(), "STORE_BACKEND") {
		t.Errorf("error %q does not name the missing variable", err)
	}
}

func TestLoadRequiresEachRequiredVar(t *testing.T) {
	for _, key := range []string{"STORE_BACKEND", "LLM_PROVIDER", "LLM_MODEL", "LLM_API_KEY"} {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(key, "")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load without %s must fail", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error %q does not name the missing variable", err)
			}
		})
	}
}

func TestLoadReportsEveryMissingRequiredVar(t *testing.T) {
	// One startup attempt should reveal the whole list, not one per restart.
	clearEnv(t)
	required := []string{"STORE_BACKEND", "LLM_PROVIDER", "LLM_MODEL", "LLM_API_KEY"}
	for _, key := range required {
		t.Setenv(key, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load must fail when the required variables are missing")
	}
	for _, want := range required {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}
}

func TestLoadKeepsLLMSettings(t *testing.T) {
	clearEnv(t)
	t.Setenv("LLM_PROVIDER", ProviderAnthropic)
	t.Setenv("LLM_MODEL", "claude-opus-5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLMProvider != ProviderAnthropic {
		t.Errorf("LLMProvider = %q", cfg.LLMProvider)
	}
	if cfg.LLMModel != "claude-opus-5" {
		t.Errorf("LLMModel = %q", cfg.LLMModel)
	}
	if cfg.LLMAPIKey != "sk-test-value" {
		t.Errorf("LLMAPIKey was not read from the environment")
	}
}

func TestLoadKeepsStoreBackend(t *testing.T) {
	clearEnv(t)
	t.Setenv("STORE_BACKEND", StoreS3)
	t.Setenv("S3_BUCKET_NAME", " test-bucket ")
	t.Setenv("S3_BUCKET_PATH", " exchanges/ ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.StoreBackend != StoreS3 {
		t.Errorf("StoreBackend = %q, want %q", cfg.StoreBackend, StoreS3)
	}
	if cfg.S3BucketName != "test-bucket" || cfg.S3BucketPath != "exchanges/" {
		t.Fatal("S3 settings must be loaded and trimmed")
	}
}

func TestLoadS3RequiresBucket(t *testing.T) {
	clearEnv(t)
	t.Setenv("STORE_BACKEND", StoreS3)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "S3_BUCKET_NAME") {
		t.Fatal("S3 store must require a bucket name")
	}
}

func TestLoadPrefersEnvironment(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("GIN_MODE", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.GinMode != "debug" {
		t.Errorf("GinMode = %q, want %q", cfg.GinMode, "debug")
	}
}

func TestOptionalTreatsBlankAsUnset(t *testing.T) {
	// A variable that interpolates to whitespace during a deploy must fall
	// back to the default rather than being passed on as a blank value.
	clearEnv(t)
	t.Setenv("PORT", "   ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want the default %q", cfg.Port, "8080")
	}
}

func TestOptionalTrimsSurroundingWhitespace(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "  9090  ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want it trimmed to %q", cfg.Port, "9090")
	}
}

// The loader tests below exercise required() directly. Load does not call it
// yet -- the first caller arrives with the S3 and provider settings -- so this
// is the only thing keeping its behaviour honest until then.

func TestRequiredReturnsValue(t *testing.T) {
	t.Setenv("SOME_KEY", "  a-value  ")

	var l loader
	got := l.required("SOME_KEY")

	if got != "a-value" {
		t.Errorf("required = %q, want it trimmed to %q", got, "a-value")
	}
	if len(l.missing) != 0 {
		t.Errorf("missing = %v, want empty", l.missing)
	}
}

func TestRequiredRecordsUnsetAndBlank(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
	}{
		{name: "unset", set: false},
		{name: "empty", value: "", set: true},
		{name: "whitespace only", value: "   \t ", set: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("SOME_KEY", tc.value)
			} else {
				t.Setenv("SOME_KEY", "")
			}

			var l loader
			if got := l.required("SOME_KEY"); got != "" {
				t.Errorf("required = %q, want empty", got)
			}
			if len(l.missing) != 1 || l.missing[0] != "SOME_KEY" {
				t.Errorf("missing = %v, want [SOME_KEY]", l.missing)
			}
		})
	}
}

func TestRequiredCollectsEveryMissingKey(t *testing.T) {
	// The point of collecting rather than failing fast: one startup attempt
	// reveals the full set, instead of one restart per missing variable.
	t.Setenv("KEY_ONE", "")
	t.Setenv("KEY_TWO", "")
	t.Setenv("KEY_THREE", "a-value")

	var l loader
	l.required("KEY_ONE")
	l.required("KEY_TWO")
	l.required("KEY_THREE")

	if len(l.missing) != 2 {
		t.Fatalf("missing = %v, want 2 entries", l.missing)
	}

	err := &Error{Missing: l.missing}
	for _, want := range []string{"KEY_ONE", "KEY_TWO"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
	if strings.Contains(err.Error(), "KEY_THREE") {
		t.Errorf("error %q mentions the key that was present", err)
	}
}

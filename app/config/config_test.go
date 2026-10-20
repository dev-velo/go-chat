package config

import (
	"strings"
	"testing"
)

// clearEnv blanks every variable Load reads so the ambient environment cannot
// influence a test. t.Setenv restores the previous values when the test ends.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PORT", "GIN_MODE"} {
		t.Setenv(key, "")
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

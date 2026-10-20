package llm

import (
	"errors"
	"testing"
)

func TestNewOpenAIRequiresAKey(t *testing.T) {
	for _, key := range []string{"", "   "} {
		if _, err := NewOpenAI(key, "gpt-4o-mini"); !errors.Is(err, ErrNoAPIKey) {
			t.Errorf("NewOpenAI(%q) error = %v, want ErrNoAPIKey", key, err)
		}
	}
}

func TestNewOpenAIRequiresAModel(t *testing.T) {
	// Unlike Anthropic there is no default model: choosing one would silently
	// pick the operator's cost and capability tier, so it must be stated.
	for _, model := range []string{"", "  "} {
		if _, err := NewOpenAI("sk-test-value", model); !errors.Is(err, ErrNoModel) {
			t.Errorf("NewOpenAI(model=%q) error = %v, want ErrNoModel", model, err)
		}
	}
}

func TestNewOpenAIKeepsTheModel(t *testing.T) {
	client, err := NewOpenAI("sk-test-value", "  gpt-4o-mini  ")
	if err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}
	if client.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want it trimmed to the configured value", client.model)
	}
}

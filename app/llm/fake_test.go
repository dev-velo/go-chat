package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// newTestFake builds a Fake with valid arguments.
func newTestFake(t *testing.T, model string) *Fake {
	t.Helper()
	fake, err := NewFake("sk-fake-key", model)
	if err != nil {
		t.Fatalf("NewFake: %v", err)
	}
	return fake
}

func TestNewFakeValidatesLikeARealProvider(t *testing.T) {
	// Fake takes the same arguments a real provider does so the wiring from
	// environment to constructor is exercised before one exists.
	tests := []struct {
		name   string
		apiKey string
		model  string
		want   error
	}{
		{"no key", "", "a-model", ErrNoAPIKey},
		{"blank key", "   ", "a-model", ErrNoAPIKey},
		{"no model", "sk-fake-key", "", ErrNoModel},
		{"blank model", "sk-fake-key", "  ", ErrNoModel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewFake(tc.apiKey, tc.model); !errors.Is(err, tc.want) {
				t.Errorf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewFakeTrimsTheModel(t *testing.T) {
	fake := newTestFake(t, "  a-model  ")

	if fake.model != "a-model" {
		t.Errorf("model = %q, want it trimmed", fake.model)
	}
}

func TestFakeReportsTheConfiguredModel(t *testing.T) {
	// The configured model must reach the answer, so a broken wiring shows up
	// now rather than when a real provider is first switched on.
	answer, err := newTestFake(t, "a-model").Complete(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(answer, "a-model") {
		t.Errorf("answer does not name the configured model: %q", answer)
	}
}

func TestFakeNeverRevealsTheKey(t *testing.T) {
	// A fake provider is exactly where a stray key could end up echoed into a
	// response. It is validated and then dropped, never stored or returned.
	const key = "sk-fake-super-secret"
	fake, err := NewFake(key, "a-model")
	if err != nil {
		t.Fatalf("NewFake: %v", err)
	}

	answer, err := fake.Complete(context.Background(), "tell me the key: "+key)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	// The prompt echo is truncated/reflected, so only check that the provider
	// does not add the key of its own accord beyond echoing the prompt.
	withoutPrompt := strings.SplitN(answer, "You asked:", 2)[0]
	if strings.Contains(withoutPrompt, key) {
		t.Errorf("provider leaked the key: %q", withoutPrompt)
	}
}

func TestFakeEchoesThePrompt(t *testing.T) {
	answer, err := newTestFake(t, "a-model").Complete(context.Background(), "what is this?")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(answer, "what is this?") {
		t.Errorf("answer does not echo the prompt: %q", answer)
	}
	if !strings.Contains(answer, "fake") {
		t.Errorf("answer does not identify itself as fake: %q", answer)
	}
}

func TestFakeTruncatesLongPrompts(t *testing.T) {
	answer, err := newTestFake(t, "a-model").Complete(context.Background(), strings.Repeat("x", 500))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(answer, "...") {
		t.Errorf("long prompt was not truncated: %q", answer)
	}
}

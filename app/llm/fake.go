package llm

import (
	"context"
	"fmt"
	"strings"
)

// Fake answers without calling a model, testing the wiring
type Fake struct {
	model string
}

var _ Client = (*Fake)(nil)

// NewFake builds a provider. The signature matches a real one deliberately.
func NewFake(apiKey, model string) (*Fake, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("fake: %w", ErrNoAPIKey)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("fake: %w", ErrNoModel)
	}
	return &Fake{model: model}, nil
}

// Name identifies the provider.
func (f *Fake) Name() string { return "fake" }

// Complete returns a stand in answer.
func (f *Fake) Complete(_ context.Context, prompt string) (string, error) {
	summary := strings.Join(strings.Fields(prompt), " ")
	if len(summary) > 120 {
		summary = summary[:120] + "..."
	}
	return fmt.Sprintf(
		"[fake provider, model %q] No real model is connected, so here is a stand-in answer.\n\nYou asked: %s",
		f.model, summary,
	), nil
}

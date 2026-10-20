// Package llm answers prompts.
//
// contract.go defines what this package promises. Each provider implements it
// in its own file beside this one: fake.go, anthropic.go, and openai.go.
package llm

import (
	"context"
	"errors"
)

var ErrNoAPIKey = errors.New("provider needs an API key")
var ErrNoModel = errors.New("provider needs LLM_MODEL set")
var maxTokens = 4096 // maxTokens caps one answer.

// Client turns a prompt into an answer.
// complete returns an answer from a prompt or err
// name returns the provider name, e.g. "anthropic" or "openai".
type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
	Name() string
}

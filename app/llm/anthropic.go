package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic answers using the Claude Messages API.
type Anthropic struct {
	client anthropic.Client
	model  string
}

var _ Client = (*Anthropic)(nil)

// NewAnthropic builds a client.
func NewAnthropic(apiKey, model string) (*Anthropic, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic: %w", ErrNoAPIKey)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("anthropic: %w", ErrNoModel)
	}
	return &Anthropic{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}, nil
}

// Name identifies the provider.
func (a *Anthropic) Name() string { return "anthropic" }

// Complete sends the prompt and returns the text of the reply.
func (a *Anthropic) Complete(ctx context.Context, prompt string) (string, error) {
	response, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic messages: %w", err)
	}

	return responseFromAnthropic(response)
}

// responseFromAnthropic reads the reply out of a message.
// It does filtering and refusal checks, and returns an error if the response is empty.
// no anthropic key to test below filtering
func responseFromAnthropic(response *anthropic.Message) (string, error) {
	if response == nil {
		return "", errors.New("anthropic returned no message")
	}
	if response.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("anthropic declined to answer: %s",
			describeRefusal(response.StopDetails))
	}

	// no anthropic key to test below filtering
	// pulled from https://github.com/anthropics/anthropic-sdk-go/blob/v1.82.0/message.go#L6750
	var answer strings.Builder
	for _, block := range response.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			answer.WriteString(variant.Text)
		}
	}

	if answer.Len() == 0 {
		return "", fmt.Errorf("anthropic returned no text (stop reason %q)", response.StopReason)
	}
	return answer.String(), nil
}

// describeRefusal renders whatever the API chose to say about a decline. Both
// fields are optional, so neither can be relied on alone.
func describeRefusal(details anthropic.RefusalStopDetails) string {
	category := string(details.Category)
	if category == "" {
		category = "unspecified"
	}
	if explanation := strings.TrimSpace(details.Explanation); explanation != "" {
		return fmt.Sprintf("%s (%s)", explanation, category)
	}
	return category
}

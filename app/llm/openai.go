package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAI answers using the Chat Completions API.
type OpenAI struct {
	client openai.Client
	model  string
}

var _ Client = (*OpenAI)(nil)

// NewOpenAI builds a client.
func NewOpenAI(apiKey, model string) (*OpenAI, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("openai: %w", ErrNoAPIKey)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("openai: %w", ErrNoModel)
	}
	return &OpenAI{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}, nil
}

// Name identifies the provider.
func (o *OpenAI) Name() string { return "openai" }

// Complete sends the prompt and returns the text of the reply.
func (o *OpenAI) Complete(ctx context.Context, prompt string) (string, error) {
	response, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:               o.model,
		MaxCompletionTokens: openai.Int(maxTokens),
		N:                   openai.Int(1), // pin to 1 answer
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai chat completions: %w", err)
	}

	return answerFromOpenAI(response)
}

// answerFromOpenAI reads the reply out of a response.
// does filtering and refusal checks, and returns an error if the response is empty.
func answerFromOpenAI(response *openai.ChatCompletion) (string, error) {
	if response == nil || len(response.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}
	choice := response.Choices[0] // pinned choices to 1 above

	if refusal := strings.TrimSpace(choice.Message.Refusal); refusal != "" {
		return "", fmt.Errorf("openai declined to answer: %s", refusal)
	}
	if choice.FinishReason == "content_filter" {
		return "", errors.New("openai filtered the response")
	}
	if choice.Message.Content == "" {
		return "", fmt.Errorf("openai returned no text (finish reason %q)", choice.FinishReason)
	}
	return choice.Message.Content, nil
}

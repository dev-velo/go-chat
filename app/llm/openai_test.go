package llm

import (
	"errors"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
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

// completion builds a response with one choice.
func completion(content, refusal, finishReason string) *openai.ChatCompletion {
	return &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{{
			Message:      openai.ChatCompletionMessage{Content: content, Refusal: refusal},
			FinishReason: finishReason,
		}},
	}
}

func TestAnswerFromOpenAIReturnsContent(t *testing.T) {
	answer, err := answerFromOpenAI(completion("the answer", "", "stop"))
	if err != nil {
		t.Fatalf("answerFromOpenAI: %v", err)
	}
	if answer != "the answer" {
		t.Errorf("answer = %q, want %q", answer, "the answer")
	}
}

func TestAnswerFromOpenAISurfacesRefusal(t *testing.T) {
	// A refusal is a 200 with Content empty and the explanation in its own
	// field. Reading Content first would discard the reason.
	_, err := answerFromOpenAI(completion("", "I can't help with that.", "stop"))
	if err == nil {
		t.Fatal("a refusal must be an error, not an empty answer")
	}
	if !strings.Contains(err.Error(), "I can't help with that.") {
		t.Errorf("error %q does not carry the model's explanation", err)
	}
}

func TestAnswerFromOpenAIReportsContentFilter(t *testing.T) {
	_, err := answerFromOpenAI(completion("", "", "content_filter"))
	if err == nil {
		t.Fatal("a filtered response must be an error")
	}
	if !strings.Contains(err.Error(), "filtered") {
		t.Errorf("error %q does not say the response was filtered", err)
	}
}

func TestAnswerFromOpenAIReportsEmptyResponses(t *testing.T) {
	tests := []struct {
		name     string
		response *openai.ChatCompletion
	}{
		{"no choices", &openai.ChatCompletion{}},
		{"nil response", nil},
		{"empty content", completion("", "", "length")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := answerFromOpenAI(tc.response); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

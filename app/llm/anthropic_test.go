package llm

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewAnthropicRequiresAKey(t *testing.T) {
	for _, key := range []string{"", "   "} {
		if _, err := NewAnthropic(key, "claude-opus-5"); !errors.Is(err, ErrNoAPIKey) {
			t.Errorf("NewAnthropic(%q) error = %v, want ErrNoAPIKey", key, err)
		}
	}
}

func TestNewAnthropicRequiresAModel(t *testing.T) {
	// No default: a stale default model id fails at request time with a 404,
	// which is far harder to read than a refusal to start.
	for _, model := range []string{"", "  "} {
		if _, err := NewAnthropic("sk-test-value", model); !errors.Is(err, ErrNoModel) {
			t.Errorf("NewAnthropic(model=%q) error = %v, want ErrNoModel", model, err)
		}
	}
}

func TestNewAnthropicKeepsTheModel(t *testing.T) {
	client, err := NewAnthropic("sk-test-value", "  claude-opus-5  ")
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}
	if client.model != "claude-opus-5" {
		t.Errorf("model = %q, want it trimmed to the configured value", client.model)
	}
}

// decodeMessage builds a response from real API JSON.
//
// The SDK's content blocks are unions whose AsText() re-unmarshals the raw
// JSON captured at decode time, and that raw field is unexported. A
// hand-built struct therefore yields an empty block, so these tests go
// through the same decode path production does.
func decodeMessage(t *testing.T, raw string) *anthropic.Message {
	t.Helper()
	var message anthropic.Message
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		t.Fatalf("decoding test message: %v", err)
	}
	return &message
}

func TestResponseFromAnthropicReturnsText(t *testing.T) {
	answer, err := responseFromAnthropic(decodeMessage(t, `{
		"stop_reason": "end_turn",
		"content": [{"type": "text", "text": "the answer"}]
	}`))
	if err != nil {
		t.Fatalf("responseFromAnthropic: %v", err)
	}
	if answer != "the answer" {
		t.Errorf("answer = %q, want %q", answer, "the answer")
	}
}

func TestResponseFromAnthropicJoinsTextBlocks(t *testing.T) {
	// Blocks are parts of one answer, not alternatives, so they concatenate.
	answer, err := responseFromAnthropic(decodeMessage(t, `{
		"stop_reason": "end_turn",
		"content": [{"type": "text", "text": "first "}, {"type": "text", "text": "second"}]
	}`))
	if err != nil {
		t.Fatalf("responseFromAnthropic: %v", err)
	}
	if answer != "first second" {
		t.Errorf("answer = %q, want the blocks joined", answer)
	}
}

func TestResponseFromAnthropicReadsTextAfterThinking(t *testing.T) {
	// The case that rules out reading content[0]: a model that thinks by
	// default opens with a ThinkingBlock and puts the prose after it.
	answer, err := responseFromAnthropic(decodeMessage(t, `{
		"stop_reason": "end_turn",
		"content": [
			{"type": "thinking", "thinking": "internal reasoning", "signature": "sig"},
			{"type": "text", "text": "the answer"}
		]
	}`))
	if err != nil {
		t.Fatalf("responseFromAnthropic: %v", err)
	}
	if answer != "the answer" {
		t.Errorf("answer = %q, want the text block that follows the thinking block", answer)
	}
	if strings.Contains(answer, "internal reasoning") {
		t.Error("thinking content leaked into the answer")
	}
}

func TestResponseFromAnthropicSurfacesRefusal(t *testing.T) {
	_, err := responseFromAnthropic(decodeMessage(t, `{
		"stop_reason": "refusal",
		"stop_details": {"type": "refusal", "category": "cyber", "explanation": "I can't help with that."},
		"content": []
	}`))
	if err == nil {
		t.Fatal("a refusal must be an error, not an empty answer")
	}
	for _, want := range []string{"I can't help with that.", "cyber"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not carry %q", err, want)
		}
	}
}

func TestResponseFromAnthropicHandlesBareRefusal(t *testing.T) {
	// Both detail fields are optional; a refusal with neither must still read
	// as a refusal rather than as an empty response.
	_, err := responseFromAnthropic(decodeMessage(t, `{"stop_reason": "refusal", "content": []}`))
	if err == nil {
		t.Fatal("a refusal must be an error")
	}
	if !strings.Contains(err.Error(), "declined") {
		t.Errorf("error %q does not say the model declined", err)
	}
}

func TestResponseFromAnthropicReportsEmptyResponses(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"no blocks", `{"stop_reason": "end_turn", "content": []}`},
		{"no text blocks", `{"stop_reason": "max_tokens", "content": [{"type": "thinking", "thinking": "only reasoning"}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := responseFromAnthropic(decodeMessage(t, tc.raw)); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}

	t.Run("nil message", func(t *testing.T) {
		if _, err := responseFromAnthropic(nil); err == nil {
			t.Error("expected an error, got none")
		}
	})
}

package store

import (
	"encoding/json"
	"testing"
	"time"
)

// TestExchangeJSONContract pins the wire format the frontend reads. The page
// is off-limits, so these three field names and the timestamp format cannot
// be changed here -- a rename would render blank history rows rather than
// failing anything else in this repo.
func TestExchangeJSONContract(t *testing.T) {
	e := Exchange{
		Prompt:    "a question",
		Response:  "an answer",
		Timestamp: time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC),
	}

	body, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"prompt":"a question","response":"an answer","timestamp":"2026-10-19T12:30:00Z"}`
	if got := string(body); got != want {
		t.Errorf("JSON contract changed:\n got %s\nwant %s", got, want)
	}
}

// TestExchangeTimestampParsesInBrowsers guards the format the page passes to
// new Date(): RFC 3339, which is what encoding/json emits for a time.Time.
func TestExchangeTimestampParsesInBrowsers(t *testing.T) {
	var out struct {
		Timestamp string `json:"timestamp"`
	}
	body, err := json.Marshal(Exchange{Timestamp: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, out.Timestamp); err != nil {
		t.Errorf("timestamp %q is not RFC 3339: %v", out.Timestamp, err)
	}
}

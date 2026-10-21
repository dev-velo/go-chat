package api

import (
	"strings"
	"testing"
	"time"
)

func TestSessionRoundTrip(t *testing.T) {
	key := sessionKey("user", "password")
	now := time.Now()
	value := issueSession(key, now.Add(time.Hour))

	if !validSession(key, value, now) {
		t.Fatal("freshly issued session was rejected")
	}
	if validSession(key, value, now.Add(2*time.Hour)) {
		t.Fatal("expired session was accepted")
	}
	if validSession(sessionKey("user", "rotated"), value, now) {
		t.Fatal("session survived a password change")
	}
	if validSession(sessionKey("other", "password"), value, now) {
		t.Fatal("session survived a username change")
	}
}

func TestSessionRejectsTampering(t *testing.T) {
	key := sessionKey("user", "password")
	now := time.Now()
	value := issueSession(key, now.Add(time.Hour))
	deadline, signature, _ := strings.Cut(value, ".")

	// A holder who pushes their own expiry out keeps the old signature, which
	// no longer covers the deadline they wrote.
	far := strings.Replace(deadline, deadline, "99999999999", 1) + "." + signature

	for name, bad := range map[string]string{
		"empty":            "",
		"no separator":     deadline + signature,
		"blank signature":  deadline + ".",
		"forged signature": deadline + "." + sign(sessionKey("user", "guess"), deadline),
		"extended expiry":  far,
		"unparsable":       "not-a-number." + sign(key, "not-a-number"),
	} {
		if validSession(key, bad, now) {
			t.Errorf("%s: accepted %q", name, bad)
		}
	}
}

package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// sessionCookie is the name of the signed session cookie.
const sessionCookie = "go_chat_session"

// sessionTTL is how long one login lasts. It is deliberately long: index.html
// is off-limits, so it cannot be taught to re-authenticate when a request
// comes back 401. A week keeps that from being a routine experience.
const sessionTTL = 7 * 24 * time.Hour

// sessionKey derives the cookie signing key from the credentials.
//
// Deriving beats storing: there is no second secret to put in SSM, every
// Lambda execution environment arrives at the same key without coordinating,
// and rotating the password invalidates every outstanding session for free.
func sessionKey(username, password string) []byte {
	mac := hmac.New(sha256.New, []byte(username+"\x00"+password))
	mac.Write([]byte("go-chat session v1"))
	return mac.Sum(nil)
}

// issueSession returns a cookie value that proves a login until expiry. The
// value is "<expiry>.<signature>" and carries its own deadline, so sessions
// need no server-side state to survive a cold start.
func issueSession(key []byte, expiry time.Time) string {
	deadline := strconv.FormatInt(expiry.Unix(), 10)
	return deadline + "." + sign(key, deadline)
}

// validSession reports whether value carries an unexpired signature from key.
func validSession(key []byte, value string, now time.Time) bool {
	deadline, signature, ok := strings.Cut(value, ".")
	if !ok {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(signature), []byte(sign(key, deadline))) != 1 {
		return false
	}
	// Read only once the signature has been trusted. Parsing first would let
	// the holder pick their own expiry by editing the cookie.
	seconds, err := strconv.ParseInt(deadline, 10, 64)
	return err == nil && now.Before(time.Unix(seconds, 0))
}

// sign returns the URL-safe HMAC of message under key.
func sign(key []byte, message string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

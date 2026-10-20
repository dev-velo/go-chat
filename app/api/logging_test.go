package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// logged runs one request through the middleware and returns every log line
// it produced, decoded.
func logged(t *testing.T, method, target string, register func(*gin.Engine)) []map[string]any {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router := gin.New()
	router.Use(Logging(), Recovery())
	register(router)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, target, nil))

	var lines []map[string]any
	for _, raw := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(raw) == 0 {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal(raw, &line); err != nil {
			t.Fatalf("log line is not JSON: %s", raw)
		}
		lines = append(lines, line)
	}
	return lines
}

func TestLoggingLevelFollowsStatus(t *testing.T) {
	tests := []struct {
		name      string
		handler   handler
		wantLevel string
	}{
		{"success", func(c *gin.Context) error { c.JSON(http.StatusOK, gin.H{}); return nil }, "INFO"},
		{"client error", func(*gin.Context) error { return badRequest("nope") }, "WARN"},
		{"server error", func(*gin.Context) error { return errors.New("broke") }, "ERROR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := logged(t, http.MethodGet, "/x", func(r *gin.Engine) {
				r.GET("/x", wrap(tc.handler))
			})

			if len(lines) != 1 {
				t.Fatalf("got %d log lines, want exactly 1: %v", len(lines), lines)
			}
			if lines[0]["level"] != tc.wantLevel {
				t.Errorf("level = %v, want %s", lines[0]["level"], tc.wantLevel)
			}
		})
	}
}

func TestLoggingIncludesServerErrorDetail(t *testing.T) {
	lines := logged(t, http.MethodGet, "/x", func(r *gin.Engine) {
		r.GET("/x", wrap(func(*gin.Context) error { return errors.New("bucket denied PutObject") }))
	})

	if got := lines[0]["error"]; got != "bucket denied PutObject" {
		t.Errorf("error = %v, want the underlying detail", got)
	}
}

func TestLoggingSkipsHealthyHealthChecks(t *testing.T) {
	lines := logged(t, http.MethodGet, healthPath, func(r *gin.Engine) {
		r.GET(healthPath, wrap(func(c *gin.Context) error {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return nil
		}))
	})

	if len(lines) != 0 {
		t.Errorf("a healthy check logged %d lines, want 0: %v", len(lines), lines)
	}
}

func TestLoggingUsesRawPathForUnmatchedRoutes(t *testing.T) {
	lines := logged(t, http.MethodGet, "/no-such-route", func(*gin.Engine) {})

	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1", len(lines))
	}
	if lines[0]["path"] != "/no-such-route" {
		t.Errorf("path = %v, want the raw URL", lines[0]["path"])
	}
}

func TestRecoveryLogsPanicOnceAndReturns500(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router := gin.New()
	router.Use(Logging(), Recovery())
	router.GET("/x", func(*gin.Context) { panic("handler exploded") })

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if n := bytes.Count(bytes.TrimSpace(buf.Bytes()), []byte("\n")) + 1; n != 1 {
		t.Errorf("panic produced %d log lines, want 1:\n%s", n, buf.String())
	}

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	if line["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", line["level"])
	}
	if detail, _ := line["error"].(string); !bytes.Contains([]byte(detail), []byte("handler exploded")) {
		t.Errorf("error = %q, want it to name the panic", detail)
	}
	if body := rec.Body.String(); bytes.Contains([]byte(body), []byte("handler exploded")) {
		t.Errorf("response leaked the panic: %s", body)
	}
}

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// run mounts h through wrap and performs one request against it.
func run(t *testing.T, h handler) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.GET("/x", wrap(h))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec
}

func TestWrapPassesThroughSuccess(t *testing.T) {
	rec := run(t, func(c *gin.Context) error {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return nil
	})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWrapUsesClientErrorStatusAndMessage(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"bad request", badRequest("prompt is required"), http.StatusBadRequest, "prompt is required"},
		{"too large", tooLarge("prompt is too long"), http.StatusRequestEntityTooLarge, "prompt is too long"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := run(t, func(*gin.Context) error { return tc.err })

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("Unmarshal %q: %v", rec.Body.String(), err)
			}
			if body["error"] != tc.wantMsg {
				t.Errorf("error = %q, want %q", body["error"], tc.wantMsg)
			}
		})
	}
}

func TestWrapHidesServerErrorDetail(t *testing.T) {
	// The caller must not learn what broke; the log carries that.
	rec := run(t, func(*gin.Context) error {
		return errors.New("bucket go-chat-prod-exchanges denied PutObject")
	})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); strings.Contains(body, "bucket") || strings.Contains(body, "PutObject") {
		t.Errorf("response leaked internal detail: %s", body)
	}
}

func TestWrapFindsWrappedClientError(t *testing.T) {
	// errors.As must see through fmt.Errorf wrapping.
	rec := run(t, func(*gin.Context) error {
		return fmt.Errorf("decoding body: %w", badRequest("prompt is required"))
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

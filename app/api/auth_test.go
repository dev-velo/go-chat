package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBasicAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BasicAuth("user", "password"))
	for _, path := range []string{"/", "/history", "/chat", "/healthz"} {
		router.Any(path, func(c *gin.Context) { c.Status(http.StatusOK) })
	}
	for _, path := range []string{"/", "/history", "/chat", "/healthz", "/missing"} {
		for _, creds := range [][2]string{{"", ""}, {"user", "wrong"}, {"wrong", "password"}, {"user", "password"}} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if creds[0] != "" {
				req.SetBasicAuth(creds[0], creds[1])
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			want := http.StatusUnauthorized
			if creds[0] == "user" && creds[1] == "password" {
				want = http.StatusOK
				if path == "/missing" {
					want = http.StatusNotFound
				}
			}
			if w.Code != want || (want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") == "") {
				t.Fatalf("%s: got %d want %d", path, w.Code, want)
			}
		}
	}
	for _, origin := range []string{"https://evil.example", "null", "http://example.com"} {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
		req.SetBasicAuth("user", "password")
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		want := http.StatusForbidden
		if origin == "http://example.com" {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Fatalf("origin %s: got %d", origin, w.Code)
		}
	}
}

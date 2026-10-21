package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-chat/app/store"

	"github.com/gin-gonic/gin"
)

func TestRouterProtectsPageAndAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A nil model will panic if an unauthenticated chat reaches the handler.
	router := newRouter(store.NewMemory(), nil, "user", "password")
	for _, path := range []string{"/", "/config.js", "/history", "/healthz", "/chat", "/missing"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("unprotected route %s: %d", path, w.Code)
		}
		if path != "/chat" && path != "/missing" {
			req = httptest.NewRequest(method, path, nil)
			req.SetBasicAuth("user", "password")
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("authorized route %s: %d", path, w.Code)
			}
		}
	}
}

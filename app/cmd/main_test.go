package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
	}

	session := signIn(t, router)
	for _, path := range []string{"/", "/config.js", "/history", "/healthz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(session)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("authorized route %s: %d", path, w.Code)
		}
	}
}

// The login form is the only way in, so it has to be reachable without one.
func TestLoginPageIsReachableUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(store.NewMemory(), nil, "user", "password")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login page: %d", w.Code)
	}
}

// signIn posts the credentials and returns the session cookie they earn.
func signIn(t *testing.T, router *gin.Engine) *http.Cookie {
	t.Helper()
	form := url.Values{"username": {"user"}, "password": {"password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "go_chat_session" && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("login did not issue a session: %d", w.Code)
	return nil
}

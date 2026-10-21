package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// guarded builds a router shaped like the real one: the guard in front, the
// auth routes and a stand-in for every protected endpoint behind it.
func guarded() (*gin.Engine, *Auth) {
	gin.SetMode(gin.TestMode)
	auth := NewAuth("user", "password")
	router := gin.New()
	router.Use(auth.Middleware())
	auth.RegisterAuthRoutes(router)
	for _, path := range []string{"/", "/config.js", "/history", "/chat", "/healthz"} {
		router.Any(path, func(c *gin.Context) { c.Status(http.StatusOK) })
	}
	return router, auth
}

// signIn posts the credentials and returns the session cookie they earn.
func signIn(t *testing.T, router *gin.Engine, username, password string) *http.Cookie {
	t.Helper()
	form := url.Values{"username": {username}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == sessionCookie && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("no session cookie issued: %d", w.Code)
	return nil
}

func TestGuardBlocksEveryRouteWithoutASession(t *testing.T) {
	router, _ := guarded()
	for _, path := range []string{"/", "/config.js", "/history", "/chat", "/healthz", "/missing"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: got %d want 401", path, w.Code)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: unauthenticated response is cacheable", path)
		}
	}
}

// A navigation gets the form; a fetch gets a status it can report. Redirecting
// a fetch would hand index.html login HTML where it expects JSON.
func TestGuardRedirectsNavigationsAndRejectsFetches(t *testing.T) {
	router, _ := guarded()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("navigation: got %d want 303", w.Code)
	}
	if location := w.Header().Get("Location"); location != "/login?next=%2F" {
		t.Fatalf("navigation sent to %q", location)
	}

	req = httptest.NewRequest(http.MethodGet, "/history", nil)
	req.Header.Set("Accept", "*/*")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("fetch: got %d want 401", w.Code)
	}
}

func TestLoginIssuesASessionThatOpensEveryRoute(t *testing.T) {
	router, _ := guarded()

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `name="password"`) {
		t.Fatalf("login form unreachable: %d", w.Code)
	}

	cookie := signIn(t, router, "user", "password")
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("weak cookie attributes: %+v", cookie)
	}

	for _, path := range []string{"/", "/config.js", "/history", "/chat", "/healthz"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: got %d want 200", path, w.Code)
		}
	}
}

func TestLoginRejectsWrongCredentials(t *testing.T) {
	router, _ := guarded()
	for _, creds := range [][2]string{{"user", "wrong"}, {"wrong", "password"}, {"", ""}} {
		form := url.Values{"username": {creds[0]}, "password": {creds[1]}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%v: got %d want 401", creds, w.Code)
		}
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == sessionCookie && cookie.Value != "" {
				t.Errorf("%v: issued a session anyway", creds)
			}
		}
	}
}

// With no credentials configured the app must not be open to everyone.
func TestUnconfiguredCredentialsLockEveryoneOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := NewAuth("", "")
	router := gin.New()
	router.Use(auth.Middleware())
	auth.RegisterAuthRoutes(router)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	form := url.Values{"username": {""}, "password": {""}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("blank credentials signed in: %d", w.Code)
	}
}

func TestGuardRejectsForgedAndExpiredSessions(t *testing.T) {
	router, auth := guarded()
	valid := signIn(t, router, "user", "password")

	for name, value := range map[string]string{
		"forged":  issueSession(sessionKey("user", "guess"), time.Now().Add(time.Hour)),
		"expired": issueSession(auth.key, time.Now().Add(-time.Minute)),
		"garbage": valid.Value + "x",
	} {
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: value})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s session: got %d want 401", name, w.Code)
		}
	}
}

// SameSite=Lax covers most of this; the check is what is left over.
func TestGuardRejectsCrossOriginWrites(t *testing.T) {
	router, _ := guarded()
	cookie := signIn(t, router, "user", "password")

	for _, origin := range []string{"https://evil.example", "null", "http://example.com"} {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
		req.AddCookie(cookie)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		want := http.StatusForbidden
		if origin == "http://example.com" {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Errorf("origin %s: got %d want %d", origin, w.Code, want)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
	req.AddCookie(cookie)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("cross-site fetch: got %d want 403", w.Code)
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	router, _ := guarded()
	cookie := signIn(t, router, "user", "password")

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
		t.Fatalf("logout: %d %q", w.Code, w.Header().Get("Location"))
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && (c.Value != "" || c.MaxAge >= 0) {
			t.Fatalf("session cookie not cleared: %+v", c)
		}
	}
}

// The post-login destination comes from the query string, so it must never be
// able to point off this host.
func TestDestinationRefusesOffsiteRedirects(t *testing.T) {
	for next, want := range map[string]string{
		"/history":              "/history",
		"/?a=b":                 "/?a=b",
		"":                      "/",
		"//evil.example":        "/",
		`/\evil.example`:        "/",
		"https://evil.example":  "/",
		"http://evil.example/x": "/",
		"evil.example":          "/",
		"/login":                "/",
		"/logout":               "/",
		"/login?next=%2Flogin":  "/",
	} {
		if got := destination(next); got != want {
			t.Errorf("destination(%q) = %q want %q", next, got, want)
		}
	}
}

func TestCookieIsSecureOffLocalhost(t *testing.T) {
	for host, wantSecure := range map[string]bool{
		"localhost:8080":                  false,
		"127.0.0.1:8080":                  false,
		"abc.lambda-url.us-east-2.on.aws": true,
		"example.com":                     true,
	} {
		gin.SetMode(gin.TestMode)
		auth := NewAuth("user", "password")
		router := gin.New()
		auth.RegisterAuthRoutes(router)

		form := url.Values{"username": {"user"}, "password": {"password"}}
		req := httptest.NewRequest(http.MethodPost, "http://"+host+"/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == sessionCookie && cookie.Secure != wantSecure {
				t.Errorf("%s: Secure=%v want %v", host, cookie.Secure, wantSecure)
			}
		}
	}
}

package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Auth guards the page and the API with a signed session cookie.
//
// Basic Auth cannot be used behind a Lambda function URL: the URL rewrites a
// WWW-Authenticate response header to x-amzn-Remapped-www-authenticate, so the
// browser never sees the challenge and shows an error page instead of a login
// prompt. A form and a cookie keep the whole exchange inside headers AWS
// leaves alone.
type Auth struct {
	user, pass [32]byte
	key        []byte
	configured bool
}

// NewAuth prepares the guard for one credential pair. Credentials are hashed
// up front so comparing them never leaks their length through timing.
func NewAuth(username, password string) *Auth {
	return &Auth{
		user:       sha256.Sum256([]byte(username)),
		pass:       sha256.Sum256([]byte(password)),
		key:        sessionKey(username, password),
		configured: strings.TrimSpace(username) != "" && strings.TrimSpace(password) != "",
	}
}

// Middleware rejects every request that does not carry a valid session.
func (a *Auth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Nothing behind this guard is cacheable, including the redirect to
		// the login form.
		c.Header("Cache-Control", "no-store")

		if publicPath(c.FullPath()) {
			c.Next()
			return
		}
		cookie, err := c.Cookie(sessionCookie)
		if err != nil || !a.configured || !validSession(a.key, cookie, time.Now()) {
			a.challenge(c)
			return
		}
		// SameSite=Lax already withholds the cookie from cross-site form
		// posts. This rejects what browsers still consider top-level rather
		// than letting another site spend LLM credits.
		if !safeMethod(c.Request.Method) && crossOrigin(c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// matches reports whether the submitted credentials are the configured ones.
func (a *Auth) matches(username, password string) bool {
	user, pass := sha256.Sum256([]byte(username)), sha256.Sum256([]byte(password))
	ok := subtle.ConstantTimeCompare(user[:], a.user[:]) &
		subtle.ConstantTimeCompare(pass[:], a.pass[:])
	return a.configured && ok == 1
}

// challenge sends an unauthenticated caller somewhere it can act on.
//
// A browser navigating to a page gets the login form. A fetch from index.html
// gets a 401 it can report: redirecting it would hand JSON.parse a page of
// login HTML and surface the failure as a parse error instead of an expired
// session.
func (a *Auth) challenge(c *gin.Context) {
	if c.Request.Method == http.MethodGet && strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.Redirect(http.StatusSeeOther, loginPath+"?next="+url.QueryEscape(c.Request.URL.RequestURI()))
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
}

// safeMethod reports whether the method is one browsers send freely.
func safeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// crossOrigin reports whether the request came from another site.
func crossOrigin(c *gin.Context) bool {
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		return true
	}
	origin := c.GetHeader("Origin")
	return origin != "" && origin != "https://"+c.Request.Host && origin != "http://"+c.Request.Host
}

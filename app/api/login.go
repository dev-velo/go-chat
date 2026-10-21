package api

import (
	_ "embed"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Routes the session guard lets through unauthenticated. They are the only
// way in, so guarding them would lock everyone out.
const (
	loginPath  = "/login"
	logoutPath = "/logout"
)

//go:embed login.html
var loginHTML string

var loginPage = template.Must(template.New("login").Parse(loginHTML))

// RegisterAuthRoutes mounts the sign-in form and the sign-out link.
func (a *Auth) RegisterAuthRoutes(router *gin.Engine) {
	router.GET(loginPath, a.loginForm)
	router.POST(loginPath, a.login)
	router.GET(logoutPath, a.logout)
}

// publicPath reports whether a matched route needs no session.
func publicPath(route string) bool {
	return route == loginPath || route == logoutPath
}

// loginForm renders the sign-in page.
func (a *Auth) loginForm(c *gin.Context) {
	a.render(c, http.StatusOK, destination(c.Query("next")), "")
}

// login checks submitted credentials and starts a session.
func (a *Auth) login(c *gin.Context) {
	next := destination(c.PostForm("next"))
	if !a.matches(c.PostForm("username"), c.PostForm("password")) {
		// One message for both a wrong username and a wrong password: naming
		// which half was wrong confirms the other half to someone guessing.
		a.render(c, http.StatusUnauthorized, next, "Incorrect username or password.")
		return
	}
	a.setSession(c, issueSession(a.key, time.Now().Add(sessionTTL)), int(sessionTTL.Seconds()))
	c.Redirect(http.StatusSeeOther, next)
}

// logout clears the session cookie. It answers GET because index.html is
// off-limits and so cannot grow a sign-out button to post from; the worst a
// forged call can do is sign the user out.
func (a *Auth) logout(c *gin.Context) {
	a.setSession(c, "", -1)
	c.Redirect(http.StatusSeeOther, loginPath)
}

// render writes the form, preserving where the caller was headed.
func (a *Auth) render(c *gin.Context, status int, next, message string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	_ = loginPage.Execute(c.Writer, map[string]string{"Next": next, "Error": message})
}

// setSession writes the session cookie.
//
// Secure is dropped only for local development, where the server speaks plain
// http and browsers would discard the cookie. Every deployed host is HTTPS and
// gets the flag.
func (a *Auth) setSession(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !localhost(c.Request.Host),
		SameSite: http.SameSiteLaxMode,
	})
}

// destination sanitises where to send someone after signing in. Anything that
// is not a plain path on this host becomes "/", so the form cannot be turned
// into an open redirect, and the auth pages themselves are excluded so a
// finished login never lands back on the form.
func destination(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, `/\`) {
		return "/"
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || publicPath(parsed.Path) {
		return "/"
	}
	return next
}

// localhost reports whether host names this machine.
func localhost(host string) bool {
	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

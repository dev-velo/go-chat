package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BasicAuth protects the page and API with a browser-native login prompt.
// Hashing before comparison avoids exposing credential lengths through timing.
func BasicAuth(username, password string) gin.HandlerFunc {
	wantUser, wantPass := sha256.Sum256([]byte(username)), sha256.Sum256([]byte(password))
	configured := strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		gotUser, gotPass := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(pass))
		userMatch := subtle.ConstantTimeCompare(gotUser[:], wantUser[:])
		passMatch := subtle.ConstantTimeCompare(gotPass[:], wantPass[:])
		if !ok || !configured || userMatch&passMatch != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Go Chat", charset="UTF-8"`)
			c.Header("Cache-Control", "no-store")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Browsers attach Basic credentials automatically. Reject cross-origin
		// unsafe requests rather than letting another site spend LLM credits.
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
			origin := c.GetHeader("Origin")
			if c.GetHeader("Sec-Fetch-Site") == "cross-site" ||
				(origin != "" && origin != "https://"+c.Request.Host && origin != "http://"+c.Request.Host) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

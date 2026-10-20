package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging logs one JSON line per request.
//
// handlers attach failures with c.Error, which Logging reports. It does not log successful health checks.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			// Caller mistakes are routine. Logging them at ERROR would mean
			// alarming every time a bot sends junk.
			level = slog.LevelWarn
		}

		// Health checks run every few seconds forever. Keep the successful
		// ones out of the stream; failures still log.
		if c.FullPath() == healthPath && level == slog.LevelInfo {
			return
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", path(c),
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds()) / 1000,
			"ip", c.ClientIP(),
		}
		if err := c.Errors.Last(); err != nil {
			attrs = append(attrs, "error", err.Err)
		}
		// Set by CloudFront and the ALB; the thread to pull when correlating
		// a user report with a log line.
		if trace := c.GetHeader("X-Amzn-Trace-Id"); trace != "" {
			attrs = append(attrs, "trace_id", trace)
		}

		slog.Log(c.Request.Context(), level, "request", attrs...)
	}
}

// path returns the matched route, falling back to the raw URL so unmatched
// requests are still identifiable. Using the route keeps cardinality low once
// paths carry parameters.
func path(c *gin.Context) string {
	if matched := c.FullPath(); matched != "" {
		return matched
	}
	return c.Request.URL.Path
}

// Recovery turns a panic into a 500 and attaches it for Logging to report.
// It does not log: that would produce a second ERROR line for one failure.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				_ = c.Error(fmt.Errorf("panic: %v\n%s", r, debug.Stack()))
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					gin.H{"error": "something went wrong"})
			}
		}()

		c.Next()
	}
}

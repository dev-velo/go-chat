// Package api exposes the HTTP contract the frontend depends on:
//
//	POST /chat     {"prompt": "..."} -> {"prompt", "response", "timestamp"}
//	GET  /history  -> [{"prompt", "response", "timestamp"}, ...] newest first
//
// routes.go is the routing table. Each endpoint lives in its own file.
package api

import (
	"net/http"

	"go-chat/app/llm"
	"go-chat/app/store"

	"github.com/gin-gonic/gin"
)

// healthPath is excluded from successful access logging.
const healthPath = "/healthz"

// api holds what the handlers need. Handlers are methods on it rather than
// closures, so each one can live in its own file.
type api struct {
	exchanges store.Store
	model     llm.Client
}

// RegisterRoutes mounts the API on router, backed by exchanges.
func RegisterRoutes(router *gin.Engine, exchanges store.Store, model llm.Client) {
	a := &api{exchanges: exchanges, model: model}

	router.GET(healthPath, wrap(a.health))
	router.POST("/chat", wrap(a.chat))
	router.GET("/history", wrap(a.history))
}

// health reports that the process is up.
func (a *api) health(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	return nil
}

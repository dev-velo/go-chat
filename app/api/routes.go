package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/chat", func(c *gin.Context) {
		// TODO: Generate and persist a response to the submitted prompt.
		c.JSON(http.StatusOK, gin.H{"response": "pong"})
	})

	router.GET("/history", func(c *gin.Context) {
		// TODO: Return persisted exchanges with prompt, response, and timestamp.
		c.JSON(http.StatusOK, []gin.H{
			{
				"prompt":    "ping",
				"response":  "pong",
				"timestamp": "2026-01-01T00:00:00Z",
			},
		})
	})
}

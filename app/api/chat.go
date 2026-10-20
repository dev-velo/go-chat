package api

import (
	"fmt"
	"strings"
	"time"

	"go-chat/app/store"

	"github.com/gin-gonic/gin"
	"net/http"
)

// maxPromptBytes bounds what we accept, so one caller cannot push an
// unbounded string into the store.
const maxPromptBytes = 8 * 1024

type chatRequest struct {
	Prompt string `json:"prompt"`
}

// chat answers a prompt and persists the exchange.
func (a *api) chat(c *gin.Context) error {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return badRequest("body must be JSON with a prompt field")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return badRequest("prompt is required")
	}
	if len(prompt) > maxPromptBytes {
		return tooLarge("prompt is too long")
	}

	exchange := store.Exchange{
		Prompt: prompt,
		// TODO: replace with a real model response once a provider is wired.
		Response:  "[no model connected yet] You asked: " + prompt,
		Timestamp: time.Now().UTC(),
	}
	if err := a.exchanges.Save(c.Request.Context(), exchange); err != nil {
		return fmt.Errorf("saving exchange: %w", err)
	}

	c.JSON(http.StatusOK, exchange)
	return nil
}

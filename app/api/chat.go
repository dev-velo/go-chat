package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-chat/app/store"

	"github.com/gin-gonic/gin"
)

// maxPromptBytes bounds what we accept, so one caller cannot push an
// unbounded string into the store.
const maxPromptBytes = 8 * 1024

// maxBodyBytes caps the request body itself, leaving room above
// maxPromptBytes for JSON framing and escaping.
//
// This has to be enforced while reading, not after parsing: without it a
// caller can make the process allocate the whole body before we ever look at
// the prompt length, so a handful of large requests can exhaust a container's
// memory even though every one of them is rejected.
const maxBodyBytes = 64 * 1024

type chatRequest struct {
	Prompt string `json:"prompt"`
}

// chat answers a prompt and persists the exchange.
func (a *api) chat(c *gin.Context) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return tooLarge("request body is too large")
		}
		return badRequest("body must be JSON with a prompt field")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return badRequest("prompt is required")
	}
	if len(prompt) > maxPromptBytes {
		return tooLarge("prompt is too long")
	}

	answer, err := a.model.Complete(c.Request.Context(), prompt)
	if err != nil {
		return fmt.Errorf("completing prompt: %w", err)
	}

	exchange := store.Exchange{
		Prompt:    prompt,
		Response:  answer,
		Timestamp: time.Now().UTC(),
	}
	if err := a.exchanges.Save(c.Request.Context(), exchange); err != nil {
		return fmt.Errorf("saving exchange: %w", err)
	}

	c.JSON(http.StatusOK, exchange)
	return nil
}

package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// historyLimit is how many exchanges /history returns.
const historyLimit = 50

// history returns recent exchanges, newest first.
func (a *api) history(c *gin.Context) error {
	found, err := a.exchanges.List(c.Request.Context(), historyLimit)
	if err != nil {
		return fmt.Errorf("listing exchanges: %w", err)
	}

	c.JSON(http.StatusOK, found)
	return nil
}

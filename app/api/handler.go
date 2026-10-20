package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handler is a gin handler that can fail. Returning an error instead of
// responding keeps the 500 body and the error log in exactly one place, which
// is what lets a single CloudWatch metric filter catch every failure.
type handler func(*gin.Context) error

// clientError is a failure the caller caused. It carries the status and the
// message to show them, and is never logged as a server error.
type clientError struct {
	status  int
	message string
}

func (e *clientError) Error() string { return e.message }

// badRequest reports a malformed or missing input.
func badRequest(message string) error {
	return &clientError{status: http.StatusBadRequest, message: message}
}

// tooLarge reports input that exceeded a limit.
func tooLarge(message string) error {
	return &clientError{status: http.StatusRequestEntityTooLarge, message: message}
}

// wrap adapts a handler to gin, turning its error into a response.
//
// Caller errors return their own status and are not logged. Server errors are logged and return a generic 500 body.
func wrap(h handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := h(c)
		if err == nil {
			return
		}

		var clientErr *clientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.status, gin.H{"error": clientErr.message})
			return
		}

		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
	}
}

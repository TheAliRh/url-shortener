package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandlerFunc is a Gin handler that reports failure by returning an error.
type HandlerFunc func(c *gin.Context) error

// handle adapts a HandlerFunc to a gin.HandlerFunc. It is the only place that
// logs request errors and decides what the client sees.
func handle(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := h(c)
		if err == nil {
			return
		}

		status, message := statusFor(err)

		// Client errors (4xx) are expected, so only log server failures
		if status >= http.StatusInternalServerError {
			log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		}

		if c.Writer.Written() {
			return // the handler already started a response, too late to change it
		}

		c.AbortWithStatusJSON(status, gin.H{"error": message})
	}
}

// statusFor maps an error to an HTTP status and a client-safe message.
func statusFor(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "Short URL not found"
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest, "Invalid request"
	default:
		// Anything unrecognized is a server problem: never leak its details
		return http.StatusInternalServerError, "Internal server error"
	}
}

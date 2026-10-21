package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	maxAPIBody = 8 << 10 // 8 KiB: a long URL (max 2048 chars) plus JSON overhead fits easily
	// maxAuthBody (4 KiB) is defined in auth-handlers.go
)

// bodyLimit rejects request bodies larger than max bytes with 413.
//
// Two layers:
//   - a declared Content-Length over the limit is refused immediately, before
//     any of the body is read;
//   - http.MaxBytesReader caps what is actually read, which also covers chunked
//     uploads and clients that lie about Content-Length. When it trips, the
//     server also closes the connection after the response.
func bodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > max {
			c.Header("Connection", "close")
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}

// bindJSON binds the request body into dst. On failure it writes the response
// itself and returns false: 413 if the body hit the size limit, otherwise 400
// with badRequestMsg.
func bindJSON(c *gin.Context, dst any, badRequestMsg string) bool {
	err := c.ShouldBindJSON(dst)
	if err == nil {
		return true
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
		return false
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": badRequestMsg})
	return false
}

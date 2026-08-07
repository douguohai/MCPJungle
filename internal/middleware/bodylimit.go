package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DefaultMaxBodySize is 10 MB — generous for JSON payloads but prevents memory exhaustion.
const DefaultMaxBodySize int64 = 10 << 20 // 10 MB

// BodySizeLimitMiddleware returns a gin middleware that limits the request body size.
// Requests exceeding maxSize bytes receive a 413 Request Entity Too Large response.
func BodySizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	if maxSize <= 0 {
		maxSize = DefaultMaxBodySize
	}
	return func(c *gin.Context) {
		// Only limit bodies for methods that carry one.
		if c.Request.Body == nil {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()

		// After handler: if the body was too large, gin/http will set this error.
		if c.Request.ContentLength > maxSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("request body too large (max %d bytes)", maxSize),
			})
		}
	}
}

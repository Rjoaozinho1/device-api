package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth returns a Gin middleware that validates the API key sent
// in the X-API-Key header against the API_KEY environment variable.
// Requests without a valid key receive a 401 Unauthorized response.
func APIKeyAuth() gin.HandlerFunc {
	expected := os.Getenv("API_KEY")

	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "API_KEY is not configured on the server",
			})
			return
		}

		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing X-API-Key header",
			})
			return
		}

		if key != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Next()
	}
}

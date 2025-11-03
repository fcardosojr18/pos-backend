package middleware

import (
	
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: parse/verify JWT from Authorization header
		// If invalid:
		// c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		// return
		c.Next()
	}
}

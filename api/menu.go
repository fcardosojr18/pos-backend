package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterMenuRoutes wires the /v1/menu endpoints.
func RegisterMenuRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/menu")
	r.GET("", getMenuItems)
	r.POST("", createMenuItem)
	// r.PUT("/:id", updateMenuItem)
	// r.DELETE("/:id", deleteMenuItem)
}

// ===== Handlers =====
// TIP: Paste your *existing* menu handler logic from main.go into these.

func getMenuItems(c *gin.Context) {
	// REPLACE with your current GET logic from main.go
	// Example placeholder:
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}

func createMenuItem(c *gin.Context) {
	// REPLACE with your current POST logic from main.go
	// Example placeholder:
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// func updateMenuItem(c *gin.Context) { /* paste your PUT logic */ }
// func deleteMenuItem(c *gin.Context) { /* paste your DELETE logic */ }

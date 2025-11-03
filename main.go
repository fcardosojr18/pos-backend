package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"pos-backend/api"
	"pos-backend/internal/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// Middlewares
	r.Use(middleware.CORS())

	// Health check route
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ----- API Routes -----
	v1 := r.Group("/v1")
	api.RegisterMenuRoutes(v1)
	api.RegisterOrderRoutes(v1)
	// api.RegisterAuthRoutes(v1) // can add later

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"pos-backend/api"
	"pos-backend/internal/db"
	"pos-backend/internal/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	// DB
	pool := db.Connect(context.Background())
	defer db.Close()

	r := gin.Default()
	registerKDS(r)

	r.Run(":" + port)
	r.Use(middleware.CORS())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	v1 := r.Group("/v1")
	api.RegisterMenuRoutes(v1, pool)
	api.RegisterOrderRoutes(v1, pool)
	api.RegisterAuthRoutes(v1) // harmless stub
	api.RegisterKitchenRoutes(v1, pool)
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil { log.Fatal(err) }
}


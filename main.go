package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pos-backend/api" // your module
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// 👇 add this so we SEE what DB we're using
	fmt.Println("KDS USING DATABASE_URL:", dbURL)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	r := gin.Default()

	api.RegisterOrderRoutes(r.Group("/v1"), pool)
	registerKDS(r, pool)

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

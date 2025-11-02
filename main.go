package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type MenuItem struct {
	ID         int64  `json:"id"`
	Category   string `json:"category"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env %s", key)
	}
	return v
}

func main() {
	dsn := mustGetEnv("DATABASE_URL")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Simple migration
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS menu_items (
		id SERIAL PRIMARY KEY,
		category TEXT NOT NULL,
		name TEXT NOT NULL,
		price_cents INTEGER NOT NULL CHECK (price_cents >= 0)
	);
	`)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	_ = r.SetTrustedProxies([]string{"127.0.0.1"}) // only trust localhost
	r.Use(gin.Logger(), gin.Recovery())


	// Health
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// List all menu items
	r.GET("/v1/menu", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id, category, name, price_cents FROM menu_items ORDER BY id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_query"})
			return
		}
		defer rows.Close()

		var items []MenuItem
		for rows.Next() {
			var m MenuItem
			if err := rows.Scan(&m.ID, &m.Category, &m.Name, &m.PriceCents); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan"})
				return
			}
			items = append(items, m)
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	// Create a menu item
	r.POST("/v1/menu", func(c *gin.Context) {
		var in MenuItem
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_json"})
			return
		}
		if in.Category == "" || in.Name == "" || in.PriceCents < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_fields"})
			return
		}
		err := db.QueryRow(
			`INSERT INTO menu_items (category, name, price_cents) VALUES ($1,$2,$3) RETURNING id`,
			in.Category, in.Name, in.PriceCents,
		).Scan(&in.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "insert"})
			return
		}
		c.JSON(http.StatusCreated, in)
	})

	// Update a menu item
	r.PUT("/v1/menu/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_id"})
			return
		}
		var in MenuItem
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_json"})
			return
		}
		if in.Category == "" || in.Name == "" || in.PriceCents < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_fields"})
			return
		}
		res, err := db.Exec(
			`UPDATE menu_items SET category=$1, name=$2, price_cents=$3 WHERE id=$4`,
			in.Category, in.Name, in.PriceCents, id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update"})
			return
		}
		aff, _ := res.RowsAffected()
		if aff == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu item not found"})
			return
		}
		in.ID = id
		c.JSON(http.StatusOK, in)
	})

	// Delete a menu item
	r.DELETE("/v1/menu/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_id"})
			return
		}
		res, err := db.Exec(`DELETE FROM menu_items WHERE id=$1`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete"})
			return
		}
		aff, _ := res.RowsAffected()
		if aff == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu item not found"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}


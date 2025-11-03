package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type createMenuInput struct {
	Category   string `json:"category" binding:"required"`
	Name       string `json:"name" binding:"required,min=2"`
	PriceCents int    `json:"price_cents" binding:"required,gte=0"`
}

func RegisterMenuRoutes(rg *gin.RouterGroup, pool *pgxpool.Pool) {
	r := rg.Group("/menu")
	r.GET("", func(c *gin.Context) {
		rows, err := pool.Query(c, `SELECT id, category, name, price_cents FROM menu_items WHERE is_active = TRUE ORDER BY category, name`)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_query"}); return }
		type item struct {
			ID         int64  `json:"id"`
			Category   string `json:"category"`
			Name       string `json:"name"`
			PriceCents int    `json:"price_cents"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.ID, &it.Category, &it.Name, &it.PriceCents); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error":"scan"}); return
			}
			items = append(items, it)
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	r.POST("", func(c *gin.Context) {
		var in createMenuInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error":"bad_request","message":err.Error()}); return
		}
		var id int64
		err := pool.QueryRow(c,
			`INSERT INTO menu_items (category, name, price_cents) VALUES ($1,$2,$3) RETURNING id`,
			in.Category, in.Name, in.PriceCents).Scan(&id)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_insert"}); return }
		c.JSON(http.StatusCreated, gin.H{"id": id})
	})

	// Optional: soft-delete
	r.DELETE("/:id", func(c *gin.Context) {
		if _, err := pool.Exec(c, `UPDATE menu_items SET is_active = FALSE WHERE id = $1`, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error":"db_update"}); return
		}
		c.Status(http.StatusNoContent)
	})
	_ = context.Background() // silence unused if needed
}

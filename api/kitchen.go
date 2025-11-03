package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterKitchenRoutes(rg *gin.RouterGroup, pool *pgxpool.Pool) {
	k := rg.Group("/kitchen")

	// GET /v1/kitchen/tickets  → list active items for the KDS
	k.GET("/tickets", func(c *gin.Context) {
		rows, err := pool.Query(c, `
			SELECT
				oi.id, oi.order_id, mi.name, mi.category,
				oi.quantity, COALESCE(oi.note,''), oi.kitchen_status,
				o.created_at, oi.created_at
			FROM order_items oi
			JOIN orders o     ON o.id = oi.order_id
			JOIN menu_items mi ON mi.id = oi.menu_item_id
			WHERE o.status = 'open'
			  AND oi.kitchen_status IN ('queued','prepping')
			ORDER BY oi.created_at ASC`)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_query"}); return }

		type ticket struct {
			ID            int64     `json:"id"`
			OrderID       int64     `json:"order_id"`
			ItemName      string    `json:"item_name"`
			Category      string    `json:"category"`
			Quantity      int       `json:"quantity"`
			Note          string    `json:"note"`
			KitchenStatus string    `json:"kitchen_status"`
			OrderCreated  time.Time `json:"order_created_at"`
			ItemCreated   time.Time `json:"item_created_at"`
		}
		out := []ticket{}
		for rows.Next() {
			var t ticket
			if err := rows.Scan(&t.ID, &t.OrderID, &t.ItemName, &t.Category, &t.Quantity, &t.Note, &t.KitchenStatus, &t.OrderCreated, &t.ItemCreated); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error":"scan"}); return
			}
			out = append(out, t)
		}
		c.JSON(http.StatusOK, gin.H{"tickets": out})
	})

	// POST /v1/kitchen/items/:id/status  → update item’s kitchen_status
	k.POST("/items/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		var payload struct {
			Status string `json:"status" binding:"required"` // queued | prepping | ready | served
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error":"bad_request","message":err.Error()}); return
		}
		status := strings.ToLower(payload.Status)
		switch status {
		case "queued", "prepping", "ready", "served":
			// ok
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_status"}); return
		}

		ct, err := pool.Exec(c, `UPDATE order_items SET kitchen_status=$1 WHERE id=$2`, status, id)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_update"}); return }
		if ct.RowsAffected() == 0 { c.JSON(http.StatusNotFound, gin.H{"error":"not_found"}); return }
		c.JSON(http.StatusOK, gin.H{"id": id, "kitchen_status": status})
	})
}

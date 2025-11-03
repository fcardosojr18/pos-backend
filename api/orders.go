package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItemInput struct {
	MenuItemID int64  `json:"menu_item_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
	Note       string `json:"note"`
}
type CreateOrderInput struct {
	Items    []OrderItemInput `json:"items" binding:"required,min=1"`
	TipCents int              `json:"tip_cents"`
}

func RegisterOrderRoutes(rg *gin.RouterGroup, pool *pgxpool.Pool) {
	r := rg.Group("/orders")

	r.GET("", func(c *gin.Context) {
		rows, err := pool.Query(c, `SELECT id, subtotal_cents, tax_cents, tip_cents, total_cents, status, created_at FROM orders ORDER BY id DESC LIMIT 100`)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_query"}); return }
		type order struct {
			ID            int64     `json:"id"`
			SubtotalCents int       `json:"subtotal_cents"`
			TaxCents      int       `json:"tax_cents"`
			TipCents      int       `json:"tip_cents"`
			TotalCents    int       `json:"total_cents"`
			Status        string    `json:"status"`
			CreatedAt     time.Time `json:"created_at"`
		}
		out := []order{}
		for rows.Next() {
			var o order
			if err := rows.Scan(&o.ID, &o.SubtotalCents, &o.TaxCents, &o.TipCents, &o.TotalCents, &o.Status, &o.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error":"scan"}); return
			}
			out = append(out, o)
		}
		c.JSON(http.StatusOK, gin.H{"orders": out})
	})

	r.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var o struct {
			ID, Subtotal, Tax, Tip, Total int
			Status                        string
			CreatedAt                     time.Time
		}
		err := pool.QueryRow(c, `SELECT id, subtotal_cents, tax_cents, tip_cents, total_cents, status, created_at FROM orders WHERE id=$1`, id).
			Scan(&o.ID, &o.Subtotal, &o.Tax, &o.Tip, &o.Total, &o.Status, &o.CreatedAt)
		if err != nil {
			if err == pgx.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error":"not_found"}); return }
			c.JSON(http.StatusInternalServerError, gin.H{"error":"db_query"}); return
		}
		rows, err := pool.Query(c, `SELECT id, menu_item_id, quantity, price_cents, note, kitchen_status, created_at FROM order_items WHERE order_id=$1 ORDER BY id`, id)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_query_items"}); return }
		type item struct {
			ID         int64     `json:"id"`
			MenuItemID int64     `json:"menu_item_id"`
			Quantity   int       `json:"quantity"`
			PriceCents int       `json:"price_cents"`
			Note       string    `json:"note"`
			Status     string    `json:"kitchen_status"`
			CreatedAt  time.Time `json:"created_at"`
		}
		items := []item{}
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.ID, &it.MenuItemID, &it.Quantity, &it.PriceCents, &it.Note, &it.Status, &it.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error":"scan_items"}); return
			}
			items = append(items, it)
		}
		c.JSON(http.StatusOK, gin.H{
			"id":              o.ID,
			"subtotal_cents":  o.Subtotal,
			"tax_cents":       o.Tax,
			"tip_cents":       o.Tip,
			"total_cents":     o.Total,
			"status":          o.Status,
			"created_at":      o.CreatedAt,
			"order_lineitems": items,
		})
	})

	r.POST("", func(c *gin.Context) {
		var in CreateOrderInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error":"bad_request","message":err.Error()}); return
		}

		// gather unique menu_item_ids
		idSet := map[int64]struct{}{}
		for _, it := range in.Items { idSet[it.MenuItemID] = struct{}{} }
		ids := make([]int64, 0, len(idSet))
		for k := range idSet { ids = append(ids, k) }

		// fetch prices
		priceMap := map[int64]int{}
		rows, err := pool.Query(c, `SELECT id, price_cents FROM menu_items WHERE id = ANY($1)`, ids)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"price_query"}); return }
		for rows.Next() {
			var id int64; var price int
			if err := rows.Scan(&id, &price); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"scan_price"}); return }
			priceMap[id] = price
		}
		// validate all items exist
		for _, it := range in.Items {
			if _, ok := priceMap[it.MenuItemID]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error":"unknown_menu_item","menu_item_id":it.MenuItemID}); return
			}
		}

		// compute totals
		subtotal := 0
		for _, it := range in.Items {
			subtotal += priceMap[it.MenuItemID] * it.Quantity
		}
		tax := int(math.Round(float64(subtotal) * 0.0625)) // 6.25%
		total := subtotal + tax + in.TipCents

		// transaction
		tx, err := pool.Begin(c)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"tx_begin"}); return }
		defer tx.Rollback(c) // safe even if committed

		var orderID int64
		if err := tx.QueryRow(c,
			`INSERT INTO orders (subtotal_cents, tax_cents, tip_cents, total_cents, status) VALUES ($1,$2,$3,$4,'open') RETURNING id`,
			subtotal, tax, in.TipCents, total).Scan(&orderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error":"insert_order"}); return
		}

		batch := &pgx.Batch{}
		for _, it := range in.Items {
			batch.Queue(
				`INSERT INTO order_items (order_id, menu_item_id, quantity, price_cents, note, kitchen_status)
				 VALUES ($1,$2,$3,$4,$5,'queued')`,
				orderID, it.MenuItemID, it.Quantity, priceMap[it.MenuItemID], it.Note,
			)
		}
		br := tx.SendBatch(c, batch)
		if err := br.Close(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"insert_items"}); return }

		if err := tx.Commit(c); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"tx_commit"}); return }

		c.JSON(http.StatusCreated, gin.H{"id": orderID, "subtotal_cents": subtotal, "tax_cents": tax, "tip_cents": in.TipCents, "total_cents": total, "status": "open"})
	})

	r.POST("/:id/pay", func(c *gin.Context) {
		id := c.Param("id")
		ct, err := pool.Exec(c, `UPDATE orders SET status='paid' WHERE id=$1 AND status='open'`, id)
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db_update"}); return }
		if ct.RowsAffected() == 0 { c.JSON(http.StatusConflict, gin.H{"error":"not_open_or_not_found"}); return }
		c.JSON(http.StatusOK, gin.H{"id": id, "status": "paid"})
	})
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

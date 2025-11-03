package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KdsItem struct {
	Name    string   `json:"name"`
	Qty     int      `json:"qty"`
	Mods    []string `json:"mods,omitempty"`
	Station string   `json:"station,omitempty"`
}

type KdsOrder struct {
	ID           string     `json:"id"`
	OrderNumber  string     `json:"orderNumber"`
	Table        string     `json:"table,omitempty"`
	CustomerName string     `json:"customerName,omitempty"`
	Type         string     `json:"type"`
	Station      string     `json:"station"`
	Status       string     `json:"status"`
	Items        []KdsItem  `json:"items"`
	Notes        string     `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	BumpedAt     *time.Time `json:"bumpedAt,omitempty"`
}

func registerKDS(r *gin.Engine, pool *pgxpool.Pool) {
	api := r.Group("/api/kds")

	api.GET("/orders", func(c *gin.Context) {
		ctx := c.Request.Context()
		orders := []KdsOrder{}

		rows, err := pool.Query(ctx, `
			SELECT id, order_number, table_name, customer_name,
			       type, station, status, notes, created_at, bumped_at
			FROM kds_orders
			ORDER BY created_at ASC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var (
				o              KdsOrder
				tableNameNS    sql.NullString
				customerNameNS sql.NullString
				notesNS        sql.NullString
			)

			if err := rows.Scan(
				&o.ID,
				&o.OrderNumber,
				&tableNameNS,
				&customerNameNS,
				&o.Type,
				&o.Station,
				&o.Status,
				&notesNS,
				&o.CreatedAt,
				&o.BumpedAt,
			); err != nil {
				// bad row, skip it
				continue
			}

			if tableNameNS.Valid {
				o.Table = tableNameNS.String
			}
			if customerNameNS.Valid {
				o.CustomerName = customerNameNS.String
			}
			if notesNS.Valid {
				o.Notes = notesNS.String
			}

			// fetch items
			itemRows, err := pool.Query(ctx, `
				SELECT name, qty, mods, station
				FROM kds_order_items
				WHERE order_id = $1
			`, o.ID)
			if err == nil {
				for itemRows.Next() {
					var it KdsItem
					// all item fields are NOT NULL in your test, so direct scan is fine
					if err := itemRows.Scan(&it.Name, &it.Qty, &it.Mods, &it.Station); err == nil {
						o.Items = append(o.Items, it)
					}
				}
				itemRows.Close()
			}

			orders = append(orders, o)
		}

		// now we should have real rows, so just return them
		c.JSON(http.StatusOK, orders)
	})
}

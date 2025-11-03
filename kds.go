package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type KdsItem struct {
	Name    string   `json:"name"`
	Qty     int      `json:"qty"`
	Mods    []string `json:"mods,omitempty"`
	Station string   `json:"station,omitempty"`
}

type KdsOrder struct {
	ID           string    `json:"id"`
	OrderNumber  string    `json:"orderNumber"`
	Table        string    `json:"table,omitempty"`
	CustomerName string    `json:"customerName,omitempty"`
	Type         string    `json:"type"`    // DINE_IN | TAKEOUT | DELIVERY
	Station      string    `json:"station"` // Grill | Fry | Cold | ...
	Status       string    `json:"status"`  // NEW | COOKING | READY
	Items        []KdsItem `json:"items"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	BumpedAt     *time.Time `json:"bumpedAt,omitempty"`
}

// temp in-memory data
var kdsOrders = []KdsOrder{
	{
		ID:          "401",
		OrderNumber: "401",
		Table:       "12",
		Type:        "DINE_IN",
		Station:     "Grill",
		Status:      "NEW",
		CreatedAt:   time.Now().Add(-1 * time.Minute),
		Items: []KdsItem{
			{Name: "Bacon Cheeseburger", Qty: 1, Mods: []string{"no onion"}, Station: "Grill"},
			{Name: "Fries", Qty: 1, Station: "Fry"},
		},
		Notes: "Allergy: shellfish",
	},
	{
		ID:          "402",
		OrderNumber: "402",
		Table:       "Bar1",
		Type:        "DINE_IN",
		Station:     "Fry",
		Status:      "COOKING",
		CreatedAt:   time.Now().Add(-5 * time.Minute),
		Items: []KdsItem{
			{Name: "Buffalo Wings (8)", Qty: 1, Mods: []string{"extra sauce"}, Station: "Fry"},
			{Name: "Side Ranch", Qty: 1},
		},
	},
	{
		ID:          "403",
		OrderNumber: "403",
		Type:        "TAKEOUT",
		Station:     "Cold",
		Status:      "READY",
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		Items: []KdsItem{
			{Name: "Chicken Caesar Wrap", Qty: 1, Mods: []string{"no tomato"}, Station: "Cold"},
			{Name: "Chips", Qty: 1},
		},
	},
	{
		ID:          "404",
		OrderNumber: "404",
		Type:        "DELIVERY",
		Station:     "Grill",
		Status:      "NEW",
		CreatedAt:   time.Now().Add(-3 * time.Minute),
		Items: []KdsItem{
			{Name: "BBQ Burger", Qty: 2, Mods: []string{"no pickle"}, Station: "Grill"},
			{Name: "Onion Rings", Qty: 1, Station: "Fry"},
		},
		Notes: "DoorDash",
	},
	{
		ID:          "405",
		OrderNumber: "405",
		Table:       " Patio-3",
		Type:        "DINE_IN",
		Station:     "Cold",
		Status:      "COOKING",
		CreatedAt:   time.Now().Add(-7 * time.Minute),
		Items: []KdsItem{
			{Name: "House Salad", Qty: 1, Mods: []string{"dressing on side"}, Station: "Cold"},
			{Name: "Clam Chowder", Qty: 1},
		},
	},
}

func registerKDS(r *gin.Engine) {
	api := r.Group("/api/kds")
	api.GET("/orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, kdsOrders)
	})
	api.PATCH("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		var body struct {
			Status string `json:"status"`
		}
		if err := c.BindJSON(&body); err != nil || body.Status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
			return
		}
		now := time.Now()
		for i := range kdsOrders {
			if kdsOrders[i].ID == id {
				kdsOrders[i].Status = body.Status
				kdsOrders[i].BumpedAt = &now
				c.JSON(http.StatusOK, gin.H{"ok": true})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})
}

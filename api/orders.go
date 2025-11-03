package api

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ----- Demo data & storage (in-memory) -----

// Simple demo price list (in cents) matching your seed menu:
// 1: Cheeseburger 1299, 2: Veggie 1199, 3: Fries 399
var demoPrices = map[int64]int{
	1: 1299,
	2: 1199,
	3: 399,
}

var (
	orderMu     sync.Mutex
	orders      = make(map[int64]*Order)
	nextOrderID int64 = 1
)

// ----- Types -----

type OrderItemInput struct {
	ItemID   int64 `json:"item_id" binding:"required"`
	Quantity int   `json:"quantity" binding:"required,gt=0"`
}

type CreateOrderInput struct {
	Items    []OrderItemInput `json:"items" binding:"required,min=1"`
	TipCents int              `json:"tip_cents"`
}

type OrderItem struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	// price captured at time of sale would normally go here
}

type Order struct {
	ID            int64       `json:"id"`
	Items         []OrderItem `json:"items"`
	SubtotalCents int         `json:"subtotal_cents"`
	TipCents      int         `json:"tip_cents"`
	TaxCents      int         `json:"tax_cents"`
	TotalCents    int         `json:"total_cents"`
	Status        string      `json:"status"`     // "open", "paid", "void"
	CreatedAt     time.Time   `json:"created_at"` // ISO8601
}

// ----- Routes -----

func RegisterOrderRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/orders")
	r.GET("", listOrders)
	r.GET("/:id", getOrder)
	r.POST("", createOrder)
	r.POST("/:id/pay", payOrder)
}

// ----- Handlers -----

func listOrders(c *gin.Context) {
	orderMu.Lock()
	defer orderMu.Unlock()

	out := make([]*Order, 0, len(orders))
	for _, o := range orders {
		out = append(out, o)
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}

func getOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	orderMu.Lock()
	defer orderMu.Unlock()

	o, ok := orders[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, o)
}

func createOrder(c *gin.Context) {
	var in CreateOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	subtotal := 0
	items := make([]OrderItem, 0, len(in.Items))
	for _, it := range in.Items {
		price, ok := demoPrices[it.ItemID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_item", "item_id": it.ItemID})
			return
		}
		subtotal += price * it.Quantity
		items = append(items, OrderItem{ItemID: int(it.ItemID), Quantity: it.Quantity})
	}

	// 6.25% MA sales tax (demo)
	tax := int(math.Round(float64(subtotal) * 0.0625))
	total := subtotal + tax + in.TipCents

	orderMu.Lock()
	id := nextOrderID
	nextOrderID++
	o := &Order{
		ID:            id,
		Items:         items,
		SubtotalCents: subtotal,
		TipCents:      in.TipCents,
		TaxCents:      tax,
		TotalCents:    total,
		Status:        "open",
		CreatedAt:     time.Now().UTC(),
	}
	orders[id] = o
	orderMu.Unlock()

	c.JSON(http.StatusCreated, o)
}

func payOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	orderMu.Lock()
	defer orderMu.Unlock()

	o, ok := orders[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	if o.Status == "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": "already_paid"})
		return
	}
	o.Status = "paid"
	c.JSON(http.StatusOK, o)
}

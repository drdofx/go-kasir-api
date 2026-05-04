package model

import "time"

// Transaction represents a checkout transaction.
type Transaction struct {
	ID          int                 `json:"id"`
	TotalAmount int                 `json:"total_amount"`
	CreatedAt   time.Time           `json:"created_at"`
	Details     []TransactionDetail `json:"details"`
}

// TransactionDetail represents transaction line items.
type TransactionDetail struct {
	ID            int    `json:"id"`
	TransactionID int    `json:"transaction_id"`
	ProductID     int    `json:"product_id"`
	ProductName   string `json:"product_name,omitempty"`
	Quantity      int    `json:"quantity"`
	Subtotal      int    `json:"subtotal"`
}

// ProductSnapshot is used for batch-fetching products during checkout.
type ProductSnapshot struct {
	ID    int
	Name  string
	Price int
	Stock int
}

// CheckoutItem represents a product and quantity pair.
type CheckoutItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// CheckoutRequest is the payload for checkout.
type CheckoutRequest struct {
	Items []CheckoutItem `json:"items"`
}

// SalesSummary represents revenue summary.
type SalesSummary struct {
	TotalRevenue      int         `json:"total_revenue"`
	TotalTransactions int         `json:"total_transactions"`
	TopProduct        *TopProduct `json:"top_product,omitempty"`
}

// TopProduct represents top-selling product summary.
type TopProduct struct {
	Name    string `json:"name"`
	QtySold int    `json:"qty_sold"`
}

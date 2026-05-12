package models

import "time"

type CartItem struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	ProductID string    `json:"productId"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
	Currency  string    `json:"currency"`
	IsDeleted bool      `json:"isDeleted"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"userId"`
	TotalAmount float64     `json:"totalAmount"`
	Currency    string      `json:"currency"`
	Status      string      `json:"status"`
	Items       []OrderItem `json:"items,omitempty"`
	IsDeleted   bool        `json:"isDeleted"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type OrderItem struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"orderId"`
	ProductID string    `json:"productId"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"createdAt"`
}

type CheckoutRequest struct {
	UserID string `json:"userId"`
}

type CheckoutResponse struct {
	OrderID     string  `json:"orderId"`
	TotalAmount float64 `json:"totalAmount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
}

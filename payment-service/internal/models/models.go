package models

import "time"

type Payment struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	OrderID       string    `json:"orderId"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"paymentMethod"`
	TransactionID string    `json:"transactionId"`
	IsDeleted     bool      `json:"isDeleted"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

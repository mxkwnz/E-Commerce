package models

import "time"

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PhotoURL    string    `json:"photoUrl,omitempty"`
	Description string    `json:"description,omitempty"`
	Brand       string    `json:"brand,omitempty"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category,omitempty"`
	Stock       int       `json:"stock,omitempty"`
	IsDeleted   bool      `json:"isDeleted"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Inventory struct {
	ID        string    `json:"id"`
	ProductID string    `json:"productId"`
	Quantity  int       `json:"quantity"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	IsDeleted bool      `json:"isDeleted"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

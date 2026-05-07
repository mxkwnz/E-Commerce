package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type cartItem struct {
	ID        string  `json:"id"`
	Quantity  int     `json:"quantity"`
	UserID    string  `json:"userId"`
	ProductID string  `json:"productId"`
	UnitPrice float64 `json:"unitPrice"`
	Currency  string  `json:"currency"`
	IsDeleted bool    `json:"isDeleted"`
}

type order struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Items       []cartItem `json:"items"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	IsDeleted   bool       `json:"isDeleted"`
}

type server struct {
	mu    sync.RWMutex
	carts map[string]cartItem
	order map[string]order
}

func main() {
	s := &server{
		carts: map[string]cartItem{},
		order: map[string]order{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order-service"})
	})

	log.Println("order-service listening on :8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

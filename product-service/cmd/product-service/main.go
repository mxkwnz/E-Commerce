package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	PhotoURL    string  `json:"photoUrl"`
	Description string  `json:"description"`
	Brand       string  `json:"brand"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	IsDeleted   bool    `json:"isDeleted"`
}

type server struct {
	mu       sync.RWMutex
	products map[string]product
}

func main() {
	s := &server{
		products: map[string]product{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "product-service"})
	})

	log.Println("product-service listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

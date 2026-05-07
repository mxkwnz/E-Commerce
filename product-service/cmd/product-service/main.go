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

type inventory struct {
	ID        string `json:"id"`
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	IsDeleted bool   `json:"isDeleted"`
}

type server struct {
	mu          sync.RWMutex
	products    map[string]product
	inventories map[string]inventory
}

func main() {
	s := &server{
		products:    map[string]product{},
		inventories: map[string]inventory{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /products", s.getProducts)
	mux.HandleFunc("GET /products/{id}", s.getProductByID)
	mux.HandleFunc("POST /products", s.createProduct)
	mux.HandleFunc("PUT /products/{id}", s.updateProduct)
	mux.HandleFunc("DELETE /products/{id}", s.deleteProduct)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "product-service"})
	})

	log.Println("product-service listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

func (s *server) getProducts(w http.ResponseWriter, r *http.Request) {
	brand := r.URL.Query().Get("brand")
	query := strings.ToLower(r.URL.Query().Get("q"))
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []product{}
	for _, p := range s.products {
		if p.IsDeleted {
			continue
		}
		if brand != "" && !strings.EqualFold(p.Brand, brand) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(p.Name+" "+p.Description), query) {
			continue
		}
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	p, ok := s.products[id]
	s.mu.RUnlock()
	if !ok || p.IsDeleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) createProduct(w http.ResponseWriter, r *http.Request) {
	var in product
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	if in.Currency == "" {
		in.Currency = "USD"
	}
	s.mu.Lock()
	s.products[in.ID] = in
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) updateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in product
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	p, ok := s.products[id]
	if ok && !p.IsDeleted {
		if in.Name != "" {
			p.Name = in.Name
		}
		if in.PhotoURL != "" {
			p.PhotoURL = in.PhotoURL
		}
		if in.Description != "" {
			p.Description = in.Description
		}
		if in.Brand != "" {
			p.Brand = in.Brand
		}
		if in.Price > 0 {
			p.Price = in.Price
		}
		if in.Currency != "" {
			p.Currency = in.Currency
		}
		s.products[id] = p
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	p, ok := s.products[id]
	if ok {
		p.IsDeleted = true
		s.products[id] = p
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

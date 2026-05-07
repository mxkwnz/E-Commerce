package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
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

	mux.HandleFunc("GET /cart-items", s.getCartByUserID)
	mux.HandleFunc("GET /cart-items/{id}", s.getCartByID)
	mux.HandleFunc("POST /cart-items", s.createCartItem)
	mux.HandleFunc("PUT /cart-items/{id}", s.updateCartItem)
	mux.HandleFunc("DELETE /cart-items/{id}", s.deleteCartItem)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order-service"})
	})

	log.Println("order-service listening on :8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}

func (s *server) getCartByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []cartItem{}
	for _, i := range s.carts {
		if i.IsDeleted {
			continue
		}
		if userID != "" && i.UserID != userID {
			continue
		}
		out = append(out, i)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getCartByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	item, ok := s.carts[id]
	s.mu.RUnlock()
	if !ok || item.IsDeleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) createCartItem(w http.ResponseWriter, r *http.Request) {
	var in cartItem
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" || in.ProductID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	s.mu.Lock()
	s.carts[in.ID] = in
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) updateCartItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in cartItem
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	item, ok := s.carts[id]
	if ok && !item.IsDeleted {
		if in.Quantity > 0 {
			item.Quantity = in.Quantity
		}
		s.carts[id] = item
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) deleteCartItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	item, ok := s.carts[id]
	if ok {
		item.IsDeleted = true
		s.carts[id] = item
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

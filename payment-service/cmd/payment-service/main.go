package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type payment struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	OrderID   string  `json:"orderId"`
	Status    string  `json:"status"`
	IsDeleted bool    `json:"isDeleted"`
}

type server struct {
	mu       sync.RWMutex
	payments map[string]payment
}

func main() {
	s := &server{
		payments: map[string]payment{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /payments", s.getPayments)
	mux.HandleFunc("GET /payments/{id}", s.getPaymentByID)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment-service"})
	})

	log.Println("payment-service listening on :8084")
	log.Fatal(http.ListenAndServe(":8084", mux))
}

func (s *server) createPayment(w http.ResponseWriter, r *http.Request) {
	var in payment
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	in.Status = "pending"
	s.mu.Lock()
	s.payments[in.ID] = in
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) getPayments(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("orderId")
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []payment{}
	for _, p := range s.payments {
		if p.IsDeleted {
			continue
		}
		if orderID != "" && p.OrderID != orderID {
			continue
		}
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getPaymentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	p, ok := s.payments[id]
	s.mu.RUnlock()
	if !ok || p.IsDeleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

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

type review struct {
	ID        string `json:"id"`
	Rating    int    `json:"rating"`
	UserID    string `json:"userId"`
	ProductID string `json:"productId"`
	Comment   string `json:"comment"`
	IsDeleted bool   `json:"isDeleted"`
}

type server struct {
	mu       sync.RWMutex
	payments map[string]payment
	reviews  map[string]review
}

func main() {
	s := &server{
		payments: map[string]payment{},
		reviews:  map[string]review{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments/pay", s.pay)
	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /payments", s.getPayments)
	mux.HandleFunc("GET /payments/{id}", s.getPaymentByID)

	mux.HandleFunc("GET /reviews", s.getReviews)
	mux.HandleFunc("GET /reviews/{id}", s.getReviewByID)
	mux.HandleFunc("POST /reviews", s.createReview)

	mux.HandleFunc("PUT /payments/{id}", s.updatePayment)
	mux.HandleFunc("DELETE /payments/{id}", s.deletePayment)
	mux.HandleFunc("GET /payments/user/{userId}", s.getPaymentsByUserID)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment-service"})
	})

	log.Println("payment-service listening on :8084")
	log.Fatal(http.ListenAndServe(":8084", mux))
}

func (s *server) pay(w http.ResponseWriter, r *http.Request) {
	var in payment
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" || in.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	in.Status = "paid"
	s.mu.Lock()
	s.payments[in.ID] = in
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) createPayment(w http.ResponseWriter, r *http.Request) {
	s.pay(w, r)
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

func (s *server) getReviews(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("productId")
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []review{}
	for _, rv := range s.reviews {
		if rv.IsDeleted {
			continue
		}
		if productID != "" && rv.ProductID != productID {
			continue
		}
		out = append(out, rv)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getReviewByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	rv, ok := s.reviews[id]
	s.mu.RUnlock()
	if !ok || rv.IsDeleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, rv)
}

func (s *server) createReview(w http.ResponseWriter, r *http.Request) {
	var in review
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" || in.ProductID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	s.mu.Lock()
	s.reviews[in.ID] = in
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) updatePayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in payment
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	p, ok := s.payments[id]
	if ok && !p.IsDeleted {
		if in.Status != "" {
			p.Status = in.Status
		}
		s.payments[id] = p
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) deletePayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	p, ok := s.payments[id]
	if ok {
		p.IsDeleted = true
		s.payments[id] = p
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (s *server) getPaymentsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []payment{}
	for _, p := range s.payments {
		if !p.IsDeleted && p.UserID == userID {
			out = append(out, p)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

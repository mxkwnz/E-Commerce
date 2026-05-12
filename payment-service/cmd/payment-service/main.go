package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type payment struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	OrderID         string  `json:"orderId"`
	Status          string  `json:"status"`
	PaymentMethod   string  `json:"paymentMethod,omitempty"`
	TransactionID   string  `json:"transactionId,omitempty"`
	IsDeleted       bool    `json:"isDeleted"`
}

type server struct {
	mu       sync.RWMutex
	payments map[string]payment
}

func main() {
	addr := getenv("HTTP_ADDR", ":8084")

	s := &server{
		payments: map[string]payment{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments/pay", s.pay)
	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /payments", s.getPayments)
	mux.HandleFunc("GET /payments/user/{userId}", s.getPaymentsByUserID)
	mux.HandleFunc("GET /payments/{id}", s.getPaymentByID)
	mux.HandleFunc("PUT /payments/{id}", s.updatePayment)
	mux.HandleFunc("DELETE /payments/{id}", s.deletePayment)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment-service"})
	})

	httpSrv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("payment-service listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
}

func (s *server) pay(w http.ResponseWriter, r *http.Request) {
	var in payment
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if in.UserID == "" || in.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId and orderId required"})
		return
	}
	if in.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}
	if in.Currency == "" {
		in.Currency = "USD"
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

func (s *server) updatePayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in payment
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	s.mu.Lock()
	p, ok := s.payments[id]
	if !ok || p.IsDeleted {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if in.Status != "" {
		p.Status = in.Status
	}
	if in.TransactionID != "" {
		p.TransactionID = in.TransactionID
	}
	s.payments[id] = p
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, p)
}

func (s *server) deletePayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	p, ok := s.payments[id]
	if !ok || p.IsDeleted {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	p.IsDeleted = true
	s.payments[id] = p
	s.mu.Unlock()
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

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

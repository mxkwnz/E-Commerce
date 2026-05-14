package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-service/internal/database"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	"payment-service/internal/service"

	"github.com/nats-io/nats.go"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8084")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	var nc *nats.Conn
	var err error
	for i := 0; i < 5; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("[NATS] attempt %d failed: %v — retrying in 2s", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[NATS] could not connect: %v (continuing without NATS)", err)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("[NATS] connected")
	}

	repo := repository.NewPaymentRepository()
	paymentSvc := service.NewPaymentService(repo)

	if nc != nil {
		sub := messaging.NewSubscriber(nc, paymentSvc)
		if err := sub.Subscribe(); err != nil {
			log.Printf("[NATS] subscribe error: %v", err)
		} else {
			defer sub.Drain()
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			UserID        string  `json:"userId"`
			OrderID       string  `json:"orderId"`
			Amount        float64 `json:"amount"`
			Currency      string  `json:"currency"`
			PaymentMethod string  `json:"paymentMethod"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		p, err := paymentSvc.CreatePayment(in.UserID, in.OrderID, in.Amount, in.Currency, in.PaymentMethod)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, p)
	})

	mux.HandleFunc("GET /payments", func(w http.ResponseWriter, r *http.Request) {
		orderID := r.URL.Query().Get("orderId")
		writeJSON(w, http.StatusOK, paymentSvc.GetAll(orderID))
	})

	mux.HandleFunc("GET /payments/user/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		writeJSON(w, http.StatusOK, paymentSvc.GetByUser(userID))
	})

	mux.HandleFunc("GET /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		p, err := paymentSvc.GetPayment(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("PUT /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var in struct {
			Status        string `json:"status"`
			TransactionID string `json:"transactionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		p, err := paymentSvc.UpdateStatus(id, in.Status, in.TransactionID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("DELETE /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := paymentSvc.Delete(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"service": "payment-service",
			"nats":    nc != nil,
		})
	})

	httpSrv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("[HTTP] payment-service listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"payment-service/internal/repository"
	"syscall"
	"time"

	paymentgrpc "payment-service/internal/grpc"
	"payment-service/internal/messaging"
	"payment-service/internal/service"
	paymentpb "payment-service/proto"

	"github.com/nats-io/nats.go"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8084")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")

	var nc *nats.Conn
	var err error
	for i := 0; i < 5; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("[NATS] attempt %d failed — retrying in 2s", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[NATS] unavailable: %v", err)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("[NATS] connected")
	}

	paymentRepo := repository.NewPaymentRepository()
	paymentSvc := service.NewPaymentService(paymentRepo)

	if nc != nil {
		sub := messaging.NewSubscriber(nc, paymentSvc)
		if err := sub.Subscribe(); err != nil {
			log.Printf("[NATS] subscribe error: %v", err)
		} else {
			defer sub.Drain()
		}
	}

	go func() {
		lis, err := net.Listen("tcp", ":50054")
		if err != nil {
			log.Fatalf("[gRPC] payment-service listen :50054: %v", err)
		}
		s := googlegrpc.NewServer()
		paymentpb.RegisterPaymentServiceServer(s, paymentgrpc.NewPaymentServer(paymentSvc))
		log.Println("[gRPC] payment-service listening on :50054")
		if err := s.Serve(lis); err != nil {
			log.Printf("[gRPC] stopped: %v", err)
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments/pay", func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusOK, paymentSvc.GetAll(r.URL.Query().Get("orderId")))
	})

	mux.HandleFunc("GET /payments/user/{userId}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, paymentSvc.GetByUser(r.PathValue("userId")))
	})

	mux.HandleFunc("GET /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := paymentSvc.GetPayment(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("PUT /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Status        string `json:"status"`
			TransactionID string `json:"transactionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		p, err := paymentSvc.UpdateStatus(r.PathValue("id"), in.Status, in.TransactionID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("DELETE /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := paymentSvc.Delete(r.PathValue("id")); err != nil {
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

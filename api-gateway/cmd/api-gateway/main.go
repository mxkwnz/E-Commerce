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

	"api-gateway/internal/checkout"
	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"api-gateway/internal/swagger"
	"api-gateway/internal/topup"

	"github.com/final-ap2-course2/telemetry"
)

func main() {
	log.Println("=================================================")
	log.Println("Starting API Gateway")
	log.Println("=================================================")

	ctx := context.Background()
	shutdown := telemetry.Init(ctx, "api-gateway")
	defer shutdown(ctx)

	cfg := config.Load()
	authMiddleware := middleware.NewAuthMiddleware(cfg.AuthServiceURL)
	p := proxy.NewProxy(cfg)

	var checkoutHandler http.Handler = http.HandlerFunc(p.Order)
	if queue, err := checkout.NewQueue(cfg.NATSURL); err != nil {
		log.Printf("[NATS] checkout queue disabled: %v", err)
	} else {
		defer queue.Close()
		checkoutHandler = checkout.NewHandler(cfg.OrderServiceURL, queue)
		log.Println("[NATS] checkout queue enabled (JetStream checkout.pending)")
	}

	var topUpHandler http.Handler = http.HandlerFunc(p.Auth)
	if topUpQueue, err := topup.NewQueue(cfg.NATSURL); err != nil {
		log.Printf("[NATS] top-up queue disabled: %v", err)
	} else {
		defer topUpQueue.Close()
		topUpHandler = topup.NewHandler(cfg.AuthServiceURL, topUpQueue)
		log.Println("[NATS] top-up queue enabled (JetStream balance.topup.pending)")
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"service": "api-gateway",
			"port":    cfg.Port,
		})
	})

	mux.HandleFunc("POST /auth/register", p.Auth)
	mux.HandleFunc("POST /auth/login", p.Auth)
	mux.HandleFunc("POST /auth/validate", p.Auth)
	mux.HandleFunc("POST /auth/forgot-password", p.Auth)
	mux.HandleFunc("POST /auth/reset-password", p.Auth)

	mux.Handle("POST /auth/change-password/send-code",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("POST /auth/change-password/confirm",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))

	mux.Handle("POST /auth/logout",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("GET /users",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("GET /users/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("PUT /users/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("DELETE /users/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))

	mux.Handle("POST /users",
		authMiddleware.Require(http.HandlerFunc(p.Auth)))
	mux.Handle("POST /users/{id}/balance/top-up",
		authMiddleware.RequireAllowAuthDown(topUpHandler))

	mux.HandleFunc("GET /products", p.Product)
	mux.HandleFunc("GET /inventory", p.Product)
	mux.Handle("POST /inventory",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("DELETE /inventory/{productId}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.HandleFunc("GET /products/{id}", p.Product)
	mux.HandleFunc("GET /inventory/{productId}", p.Product)
	mux.Handle("POST /products",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("PUT /products/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("DELETE /products/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("PUT /inventory/{productId}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))

	mux.Handle("POST /products/{id}/favorite",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("DELETE /products/{id}/favorite",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("GET /favorites",
		authMiddleware.Require(http.HandlerFunc(p.Product)))

	mux.HandleFunc("GET /reviews", p.Product)
	mux.HandleFunc("GET /reviews/{id}", p.Product)

	mux.Handle("POST /reviews",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("PUT /reviews/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.Handle("DELETE /reviews/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Product)))
	mux.HandleFunc("GET /products/{productId}/reviews", p.Product)
	mux.Handle("GET /users/{userId}/reviews",
		authMiddleware.Require(http.HandlerFunc(p.Product)))

	mux.Handle("GET /cart-items",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("GET /cart-items/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("GET /users/{userId}/cart-items",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("POST /cart-items",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("PUT /cart-items/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("DELETE /cart-items/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))

	mux.Handle("POST /orders/checkout",
		authMiddleware.Require(checkoutHandler))
	mux.Handle("POST /orders",
		authMiddleware.Require(checkoutHandler))
	mux.Handle("GET /orders",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("GET /users/{userId}/orders",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("GET /orders/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("PUT /orders/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("DELETE /orders/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("POST /orders/{id}/confirm",
		authMiddleware.Require(http.HandlerFunc(p.Order)))
	mux.Handle("POST /orders/{id}/cancel",
		authMiddleware.Require(http.HandlerFunc(p.Order)))

	mux.Handle("POST /payments/pay",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("POST /payments",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("GET /payments",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("GET /payments/user/{userId}",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("GET /payments/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("PUT /payments/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))
	mux.Handle("DELETE /payments/{id}",
		authMiddleware.Require(http.HandlerFunc(p.Payment)))

	swagger.Register(mux)

	mux.Handle("/", http.FileServer(http.Dir("./static")))

	srv := &http.Server{

		Addr:         ":" + cfg.Port,
		Handler: telemetry.WrapHTTP(
			middleware.Logger(middleware.CORS(mux)), "api-gateway"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API Gateway listening on :%s", cfg.Port)
		log.Println("=================================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("API Gateway stopped")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

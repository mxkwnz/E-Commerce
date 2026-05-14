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

	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
)

func main() {
	log.Println("=================================================")
	log.Println("Starting API Gateway")
	log.Println("=================================================")

	cfg := config.Load()
	authMiddleware := middleware.NewAuthMiddleware(cfg.AuthServiceURL)
	p := proxy.NewProxy(cfg)

	mux := http.NewServeMux()

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

	mux.HandleFunc("GET /products", p.Product)
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

	// NOTE: Mukhammedali will add order and payment routes here

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      middleware.Logger(middleware.CORS(mux)),
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

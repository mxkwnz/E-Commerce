package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/database"
	grpcserver "order-service/internal/grpc"
	"order-service/internal/handler"
	"order-service/internal/repository"
	"order-service/internal/service"
	pb "order-service/proto"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	log.Println("=================================================")
	log.Println("Starting Order Service")
	log.Println("=================================================")

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Database connected")

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Migrations completed")

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("⚠ NATS connection failed: %v (continuing without NATS)", err)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("✓ NATS connected")
	}

	grpcAddr := getenv("GRPC_ADDR", ":50051")
	httpAddr := getenv("HTTP_ADDR", ":8083")

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("gRPC listen %s: %v", grpcAddr, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)
	pb.RegisterOrderServiceServer(grpcSrv, grpcserver.NewOrderServer(nc))

	go func() {
		log.Println("=================================================")
		log.Printf("✓ gRPC server listening on %s", grpcAddr)
		log.Println("=================================================")
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	cartRepo := repository.NewCartRepository()
	orderRepo := repository.NewOrderRepository()
	cartService := service.NewCartService(cartRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, nc)

	cartHandler := handler.NewCartHandler(cartService)
	orderHandler := handler.NewOrderHandler(orderService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /cart-items", cartHandler.GetCart)
	mux.HandleFunc("POST /cart-items", cartHandler.AddToCart)
	mux.HandleFunc("PUT /cart-items/{id}", cartHandler.UpdateCartItem)
	mux.HandleFunc("DELETE /cart-items/{id}", cartHandler.DeleteCartItem)

	mux.HandleFunc("POST /orders/checkout", orderHandler.Checkout)
	mux.HandleFunc("GET /orders", orderHandler.GetUserOrders)
	mux.HandleFunc("GET /orders/{id}", orderHandler.GetOrder)
	mux.HandleFunc("POST /orders/{id}/confirm", orderHandler.ConfirmOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", orderHandler.CancelOrder)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"service":   "order-service",
			"http_addr": httpAddr,
			"grpc_addr": grpcAddr,
			"protocols": []string{"HTTP/REST", "gRPC"},
		})
	})

	httpSrv := &http.Server{Addr: httpAddr, Handler: mux}

	go func() {
		log.Println("=================================================")
		log.Printf("✓ HTTP server listening on %s", httpAddr)
		log.Println("=================================================")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	grpcSrv.GracefulStop()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
	log.Println("Shutdown complete")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

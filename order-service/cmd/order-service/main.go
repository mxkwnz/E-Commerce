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
	"order-service/internal/messaging"
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

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	natsURL := getenv("NATS_URL", "nats://localhost:4222")
	var nc *nats.Conn
	var err error
	for i := 0; i < 10; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("[NATS] connection attempt %d failed: %v — retrying in 2s", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[NATS] could not connect after retries: %v (continuing without NATS)", err)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("[NATS] connected")
	}


	orderRepo := repository.NewOrderRepository()
	cartRepo := repository.NewCartRepository()
	orderSvc := service.NewOrderService(orderRepo, cartRepo, nc)

	if nc != nil {
		sub := messaging.NewSubscriber(nc, orderSvc)
		if err := sub.Subscribe(); err != nil {
			log.Printf("[NATS] subscribe error: %v", err)
		} else {
			defer sub.Drain()
		}
		if js, err := nc.JetStream(); err != nil {
			log.Printf("[NATS] JetStream error: %v", err)
		} else if _, err := messaging.StartCheckoutConsumer(js, orderSvc); err != nil {
			log.Printf("[NATS] checkout consumer error: %v", err)
		}
	}

	grpcAddr := getenv("GRPC_ADDR", ":50051")
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
		log.Printf("[gRPC] listening on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("[gRPC] stopped: %v", err)
		}
	}()

	httpAddr := getenv("HTTP_ADDR", ":8083")
	cartSvc := service.NewCartService(cartRepo)
	mux := http.NewServeMux()

	cartHandler := handler.NewCartHandler(cartSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	mux.HandleFunc("GET /cart-items", cartHandler.GetCart)
	mux.HandleFunc("GET /cart-items/{id}", cartHandler.GetCartItemByID)
	mux.HandleFunc("GET /users/{userId}/cart-items", cartHandler.ListCartForPathUser)
	mux.HandleFunc("POST /cart-items", cartHandler.AddToCart)
	mux.HandleFunc("PUT /cart-items/{id}", cartHandler.UpdateCartItem)
	mux.HandleFunc("DELETE /cart-items/{id}", cartHandler.DeleteCartItem)
	mux.HandleFunc("POST /orders/checkout", orderHandler.Checkout)
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetUserOrders)
	mux.HandleFunc("GET /users/{userId}/orders", orderHandler.GetOrdersForPathUser)
	mux.HandleFunc("GET /orders/{id}", orderHandler.GetOrder)
	mux.HandleFunc("PUT /orders/{id}", orderHandler.UpdateOrder)
	mux.HandleFunc("DELETE /orders/{id}", orderHandler.DeleteOrder)
	mux.HandleFunc("POST /orders/{id}/confirm", orderHandler.ConfirmOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", orderHandler.CancelOrder)


	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "order-service",
			"nats":    nc != nil,
		})
	})

	httpSrv := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		log.Printf("[HTTP] listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[HTTP] %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	grpcSrv.GracefulStop()
	_ = httpSrv.Shutdown(ctx)
	log.Println("Shutdown complete")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

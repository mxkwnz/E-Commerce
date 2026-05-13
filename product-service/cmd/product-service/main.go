package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/final-ap2-course2/product-service/internal/cache"
	"github.com/final-ap2-course2/product-service/internal/database"
	grpcserver "github.com/final-ap2-course2/product-service/internal/grpc"
	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
	"github.com/final-ap2-course2/product-service/internal/service"
	pb "github.com/final-ap2-course2/product-service/proto"

	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting Product Service")

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := cache.Connect(); err != nil {
		log.Printf("⚠ Redis connection failed: %v (continuing without cache)", err)
	} else {
		defer cache.Close()
	}

	go startGRPCServer()
	startHTTPServer()
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen on port 50052: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	productServer := grpcserver.NewProductServer()
	pb.RegisterProductServiceServer(grpcServer, productServer)

	log.Println("✓ gRPC server listening on :50052")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPServer() {
	productRepo := repository.NewProductRepository()
	invRepo := repository.NewInventoryRepository()
	productService := service.NewProductService(productRepo, invRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		brand := r.URL.Query().Get("brand")
		category := r.URL.Query().Get("category")
		query := r.URL.Query().Get("q")

		var products []models.Product
		var err error

		if query != "" {
			products, _, err = productService.SearchProducts(query, 1, 100)
		} else if brand != "" {
			products, _, err = productService.GetProductsByBrand(brand, 1, 100)
		} else if category != "" {
			products, _, err = productService.GetProductsByCategory(category, 1, 100)
		} else {
			products, _, err = productService.ListProducts(1, 100)
		}

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, products)
	})

	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		product, err := productService.GetProduct(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, product)
	})

	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var product models.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productService.CreateProduct(&product); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, product)
	})

	mux.HandleFunc("PUT /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var updates models.Product
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productService.UpdateProduct(id, &updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		product, _ := productService.GetProduct(id)
		writeJSON(w, http.StatusOK, product)
	})

	mux.HandleFunc("DELETE /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := productService.DeleteProduct(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("GET /inventory/{productId}", func(w http.ResponseWriter, r *http.Request) {
		productID := r.PathValue("productId")
		inventory, err := productService.GetInventory(productID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, inventory)
	})

	mux.HandleFunc("PUT /inventory/{productId}", func(w http.ResponseWriter, r *http.Request) {
		productID := r.PathValue("productId")
		var req struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productService.UpdateInventory(productID, req.Quantity); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		inventory, _ := productService.GetInventory(productID)
		writeJSON(w, http.StatusOK, inventory)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"service": "product-service",
			"ports":   map[string]string{"http": "8082", "grpc": "50052"},
		})
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("✓ HTTP server listening on :8082")
		if err := http.ListenAndServe(":8082", mux); err != nil {
			log.Fatal(err)
		}
	}()

	<-stop
	log.Println("Shutting down gracefully...")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

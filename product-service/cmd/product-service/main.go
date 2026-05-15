package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/final-ap2-course2/product-service/internal/cache"
	"github.com/final-ap2-course2/product-service/internal/database"
	grpcserver "github.com/final-ap2-course2/product-service/internal/grpc"
	"github.com/final-ap2-course2/product-service/internal/handler"
	"github.com/final-ap2-course2/product-service/internal/messaging"
	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
	"github.com/final-ap2-course2/product-service/internal/service"
	pb "github.com/final-ap2-course2/product-service/proto"

	"github.com/nats-io/nats.go"
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
		log.Printf("Redis unavailable: %v (continuing without cache)", err)
	} else {
		defer cache.Close()
	}

	natsURL := getenv("NATS_URL", "nats://localhost:4222")
	var nc *nats.Conn
	var err error
	for i := 0; i < 10; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("[NATS] attempt %d failed — retrying in 2s", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[NATS] unavailable: %v (continuing without NATS)", err)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("[NATS] connected")
	}


	productRepo := repository.NewProductRepository()
	invRepo := repository.NewInventoryRepository()
	favRepo := repository.NewFavoriteRepository()
	reviewRepo := repository.NewReviewRepository()

	productSvc := service.NewProductService(productRepo, invRepo, favRepo)
	reviewSvc := service.NewReviewService(reviewRepo, productRepo)

	if nc != nil {
		sub := messaging.NewSubscriber(nc, productSvc)
		if err := sub.Subscribe(); err != nil {
			log.Printf("[NATS] subscribe error: %v", err)
		} else {
			defer sub.Drain()
		}
	}

	go startGRPCServer(productSvc)
	startHTTPServer(productSvc, reviewSvc)
}

func startGRPCServer(productSvc *service.ProductService) {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("gRPC listen :50052: %v", err)
	}
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)
	pb.RegisterProductServiceServer(s, grpcserver.NewProductServer())
	log.Println("[gRPC] product-service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC serve: %v", err)
	}
}

func startHTTPServer(productSvc *service.ProductService, reviewSvc *service.ReviewService) {
	mux := http.NewServeMux()

	reviewHandler := handler.NewReviewHandler(reviewSvc)

	mux.HandleFunc("GET /favorites", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ids, err := productSvc.ListFavoriteProductIDs(userID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if ids == nil {
			ids = []string{}
		}
		writeJSON(w, http.StatusOK, map[string][]string{"productIds": ids})
	})

	mux.HandleFunc("GET /reviews", reviewHandler.ListReviews)
	mux.HandleFunc("GET /reviews/{id}", reviewHandler.GetReviewByID)
	mux.HandleFunc("POST /reviews", reviewHandler.CreateReview)
	mux.HandleFunc("PUT /reviews/{id}", reviewHandler.UpdateReview)
	mux.HandleFunc("DELETE /reviews/{id}", reviewHandler.DeleteReview)
	mux.HandleFunc("GET /products/{productId}/reviews", reviewHandler.GetProductReviews)
	mux.HandleFunc("GET /users/{userId}/reviews", reviewHandler.GetUserReviews)

	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		filter := models.ProductFilter{
			Brand:    r.URL.Query().Get("brand"),
			Category: r.URL.Query().Get("category"),
			Gender:   r.URL.Query().Get("gender"),
			Search:   r.URL.Query().Get("q"),
		}
		
		products, _, err := productSvc.ListProductsWithFilter(filter, 1, 100)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if products == nil {
			products = []models.Product{}
		}
		writeJSON(w, http.StatusOK, products)
	})

	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := productSvc.GetProduct(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var product models.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productSvc.CreateProduct(&product); err != nil {
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
		if err := productSvc.UpdateProduct(id, &updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		p, _ := productSvc.GetProduct(id)
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("DELETE /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := productSvc.DeleteProduct(r.PathValue("id")); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("POST /products/{id}/favorite", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err := productSvc.AddFavorite(userID, r.PathValue("id")); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"message": "added to favorites"})
	})

	mux.HandleFunc("DELETE /products/{id}/favorite", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err := productSvc.RemoveFavorite(userID, r.PathValue("id")); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "removed from favorites"})
	})

	mux.HandleFunc("GET /inventory", func(w http.ResponseWriter, r *http.Request) {
		brand := r.URL.Query().Get("brand")
		productID := r.URL.Query().Get("productId")
		list, err := productSvc.ListInventory(productID, brand)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if list == nil {
			list = []models.Inventory{}
		}
		writeJSON(w, http.StatusOK, list)
	})

	mux.HandleFunc("POST /inventory", func(w http.ResponseWriter, r *http.Request) {
		var inv models.Inventory
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productSvc.CreateInventoryRecord(&inv); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		out, _ := productSvc.GetInventory(inv.ProductID)
		writeJSON(w, http.StatusCreated, out)
	})

	mux.HandleFunc("GET /inventory/{productId}", func(w http.ResponseWriter, r *http.Request) {
		inv, err := productSvc.GetInventory(r.PathValue("productId"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, inv)
	})

	mux.HandleFunc("PUT /inventory/{productId}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := productSvc.UpdateInventory(r.PathValue("productId"), req.Quantity); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		inv, _ := productSvc.GetInventory(r.PathValue("productId"))
		writeJSON(w, http.StatusOK, inv)
	})

	mux.HandleFunc("DELETE /inventory/{productId}", func(w http.ResponseWriter, r *http.Request) {
		if err := productSvc.DeleteInventory(r.PathValue("productId")); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
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
		log.Println("[HTTP] product-service listening on :8082")
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

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

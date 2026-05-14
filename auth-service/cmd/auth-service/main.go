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

	"github.com/final-ap2-course2/auth-service/internal/database"
	grpcserver "github.com/final-ap2-course2/auth-service/internal/grpc"
	"github.com/final-ap2-course2/auth-service/internal/models"
	"github.com/final-ap2-course2/auth-service/internal/repository"
	"github.com/final-ap2-course2/auth-service/internal/service"
	pb "github.com/final-ap2-course2/auth-service/proto"

	"github.com/nats-io/nats.go"
	grpclib "google.golang.org/grpc"
)

func main() {
	log.Println("=================================================")
	log.Println("Starting Auth Service")
	log.Println("=================================================")

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Database connected")

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed")

	natsURL := getenv("NATS_URL", "nats://localhost:4222")
	var nc *nats.Conn
	var natsErr error
	for i := 0; i < 5; i++ {
		nc, natsErr = nats.Connect(natsURL)
		if natsErr == nil {
			break
		}
		log.Printf("[NATS] attempt %d failed — retrying in 2s: %v", i+1, natsErr)
		time.Sleep(2 * time.Second)
	}
	if natsErr != nil {
		log.Printf("[NATS] unavailable: %v (continuing without NATS)", natsErr)
		nc = nil
	} else {
		defer nc.Close()
		log.Println("[NATS] auth-service connected")
	}

	go startGRPCServer(nc)

	startHTTPServer(nc)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func startGRPCServer(nc *nats.Conn) {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}

	grpcServer := grpclib.NewServer(
		grpclib.MaxRecvMsgSize(10*1024*1024),
		grpclib.MaxSendMsgSize(10*1024*1024),
	)

	authServer := grpcserver.NewAuthServer()
	authServer.SetNATS(nc)
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	log.Println("=================================================")
	log.Println("gRPC server listening on :50053")
	log.Println("  Endpoints available: 12")
	log.Println("  - Authentication: 4 endpoints")
	log.Println("  - User Management: 5 endpoints")
	log.Println("  - Password Recovery: 3 endpoints")
	log.Println("=================================================")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPServer(nc *nats.Conn) {
	userRepo := repository.NewUserRepository()
	sessionRepo := repository.NewSessionRepository()
	resetTokenRepo := repository.NewResetTokenRepository()

	authService := service.NewAuthService(userRepo, sessionRepo, resetTokenRepo)
	authService.SetPublisher(nc)
	userService := service.NewUserService(userRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		var req models.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		resp, err := authService.Register(&req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		resp, err := authService.Login(&req)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /auth/validate", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token required"})
			return
		}

		user, err := authService.ValidateToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		writeJSON(w, http.StatusOK, user)
	})

	mux.HandleFunc("POST /auth/logout", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
			return
		}

		if err := authService.Logout(token); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		user, err := userService.GetUser(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusOK, user)
	})

	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		users, _, err := userService.ListUsers(1, 100, "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, users)
	})

	mux.HandleFunc("PUT /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var updates models.User
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		user, err := userService.UpdateUser(id, &updates)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, user)
	})

	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		user, err := userService.GetUser(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err := userService.DeleteUser(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		authService.NotifyUserDeleted(user.ID, user.Email, user.Username, user.Role)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	})

	mux.HandleFunc("POST /auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		token, _ := authService.ForgotPassword(req.Email)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "reset email sent",
			"token":   token,
		})
	})

	mux.HandleFunc("POST /auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token       string `json:"token"`
			NewPassword string `json:"newPassword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		if err := authService.ResetPassword(req.Token, req.NewPassword); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "password reset"})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"service":   "auth-service",
			"http_port": "8081",
			"grpc_port": "50053",
			"protocols": []string{"HTTP/REST", "gRPC"},
			"features":  []string{"bcrypt", "Session Tokens", "Password Reset", "SMTP"},
			"endpoints": map[string]int{
				"http": 10,
				"grpc": 12,
			},
		})
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("=================================================")
		log.Println("HTTP server listening on :8081")
		log.Println("  Endpoints available: 10")
		log.Println("  - Auth: 4 endpoints")
		log.Println("  - Users: 4 endpoints")
		log.Println("  - Password Recovery: 2 endpoints")
		log.Println("=================================================")
		log.Println("Auth Service is ready!")
		log.Println("=================================================")

		if err := http.ListenAndServe(":8081", mux); err != nil {
			log.Fatal(err)
		}
	}()

	<-stop
	log.Println("\n=================================================")
	log.Println("Shutting down gracefully...")
	log.Println("=================================================")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

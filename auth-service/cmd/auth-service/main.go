package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type user struct {
	ID          string  `json:"id"`
	Password    string  `json:"password,omitempty"`
	Username    string  `json:"username"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phoneNumber"`
	Balance     float64 `json:"balance"`
	IsDeleted   bool    `json:"isDeleted"`
}

type server struct {
	mu    sync.RWMutex
	users map[string]user
}

func main() {
	s := &server{
		users: map[string]user{},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth-service"})
	})

	log.Println("auth-service listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

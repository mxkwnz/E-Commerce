package main

import (
	"crypto/sha256"
	"encoding/hex"
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

	mux.HandleFunc("POST /auth/register", s.register)
	mux.HandleFunc("POST /auth/login", s.login)
	mux.HandleFunc("GET /users", s.getUsers)
	mux.HandleFunc("GET /users/{id}", s.getUserByID)
	mux.HandleFunc("POST /users", s.createUser)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth-service"})
	})

	log.Println("auth-service listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	var in user
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" || in.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	in.ID = newID()
	s.mu.Lock()
	s.users[in.ID] = in
	s.mu.Unlock()
	token := tokenFor(in.ID, in.Email)
	writeJSON(w, http.StatusCreated, map[string]any{"userId": in.ID, "accessToken": token})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == in.Email && u.Password == in.Password && !u.IsDeleted {
			writeJSON(w, http.StatusOK, map[string]any{"userId": u.ID, "accessToken": tokenFor(u.ID, u.Email)})
			return
		}
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
}

func (s *server) getUsers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]user, 0, len(s.users))
	for _, u := range s.users {
		if !u.IsDeleted {
			u.Password = ""
			out = append(out, u)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) getUserByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	u, ok := s.users[id]
	s.mu.RUnlock()
	if !ok || u.IsDeleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	u.Password = ""
	writeJSON(w, http.StatusOK, u)
}

func (s *server) createUser(w http.ResponseWriter, r *http.Request) {
	s.register(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

func tokenFor(id, email string) string {
	sum := sha256.Sum256([]byte(id + ":" + email + ":" + time.Now().String()))
	return hex.EncodeToString(sum[:])
}

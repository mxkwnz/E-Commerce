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

	"github.com/final-ap2-course2/auth-service/internal/usecase"
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
	mux.HandleFunc("POST /auth/forgot-password", s.forgotPassword)
	mux.HandleFunc("POST /auth/reset-password", s.resetPassword)

	mux.HandleFunc("GET /users", s.getUsers)
	mux.HandleFunc("POST /users", s.createUser)
	mux.HandleFunc("GET /users/{id}", s.getUserByID)
	mux.HandleFunc("PUT /users/{id}", s.updateUser)
	mux.HandleFunc("DELETE /users/{id}", s.deleteUser)

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

func (s *server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	_ = usecase.SendResetEmail(in.Email)
	writeJSON(w, http.StatusOK, map[string]string{"message": "reset token sent"})
}

func (s *server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID      string `json:"userId"`
		NewPassword string `json:"newPassword"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" || in.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	u, ok := s.users[in.UserID]
	if ok {
		u.Password = in.NewPassword
		s.users[in.UserID] = u
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
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

func (s *server) createUser(w http.ResponseWriter, r *http.Request) {
	s.register(w, r)
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

func (s *server) updateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in user
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	u, ok := s.users[id]
	if ok && !u.IsDeleted {
		if in.Username != "" {
			u.Username = in.Username
		}
		if in.FirstName != "" {
			u.FirstName = in.FirstName
		}
		if in.LastName != "" {
			u.LastName = in.LastName
		}
		if in.Email != "" {
			u.Email = in.Email
		}
		if in.PhoneNumber != "" {
			u.PhoneNumber = in.PhoneNumber
		}
		s.users[id] = u
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	u.Password = ""
	writeJSON(w, http.StatusOK, u)
}

func (s *server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	u, ok := s.users[id]
	if ok {
		u.IsDeleted = true
		s.users[id] = u
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
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

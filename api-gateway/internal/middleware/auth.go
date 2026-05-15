package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var ErrAuthUnreachable = errors.New("auth service unreachable")

type AuthMiddleware struct {
	authURL    string
	httpClient *http.Client
}

func NewAuthMiddleware(authServiceURL string) *AuthMiddleware {
	return &AuthMiddleware{
		authURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "authorization token required",
			})
			return
		}

		user, err := m.validate(token)
		if err != nil {
			log.Printf("[AUTH] validation error: %v", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid or expired token",
			})
			return
		}

		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-User-Email", user.Email)
		r.Header.Set("X-User-Role", user.Role)

		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) RequireAllowAuthDown(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "authorization token required",
			})
			return
		}

		user, err := m.validate(token)
		if err == nil {
			r.Header.Set("X-User-ID", user.ID)
			r.Header.Set("X-User-Email", user.Email)
			r.Header.Set("X-User-Role", user.Role)
			next.ServeHTTP(w, r)
			return
		}

		if errors.Is(err, ErrAuthUnreachable) {
			log.Printf("[AUTH] auth-service unreachable, deferring auth for request: %v", err)
			r.Header.Set("X-Auth-Unavailable", "true")
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("[AUTH] validation error: %v", err)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid or expired token",
		})
	})
}

func (m *AuthMiddleware) validate(token string) (*userInfo, error) {
	req, err := http.NewRequest(http.MethodPost, m.authURL+"/auth/validate", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthUnreachable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadGateway {
		return nil, ErrAuthUnreachable
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errorFromBody(body)
	}

	var u userInfo
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

type userInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func ExtractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return h
}

func errorFromBody(body []byte) error {
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &e); err == nil && e.Error != "" {
		return errors.New(e.Error)
	}
	return errors.New("authentication failed")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

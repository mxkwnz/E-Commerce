package topup

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"api-gateway/internal/middleware"
)

type Handler struct {
	authURL string
	queue   *Queue
	client  *http.Client
}

func NewHandler(authURL string, queue *Queue) *Handler {
	return &Handler{
		authURL: authURL,
		queue:   queue,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if r.Header.Get("X-Auth-Unavailable") == "true" {
		h.respondQueued(w, r, targetID, bodyBytes)
		return
	}

	target := strings.TrimSuffix(h.authURL, "/") + r.URL.Path
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(bodyBytes))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to forward request"})
		return
	}
	outReq.Header = r.Header.Clone()
	if outReq.Header.Get("Content-Type") == "" {
		outReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(outReq)
	if err == nil && resp.StatusCode < http.StatusBadGateway {
		defer resp.Body.Close()
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}
	if resp != nil {
		resp.Body.Close()
	}

	h.respondQueued(w, r, targetID, bodyBytes)
}

func (h *Handler) respondQueued(w http.ResponseWriter, r *http.Request, targetID string, bodyBytes []byte) {
	if h.queue == nil || targetID == "" {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}

	token := middleware.ExtractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization token required"})
		return
	}

	var body struct {
		Amount float64 `json:"amount"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil || body.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
		return
	}

	actorID := r.Header.Get("X-User-ID")
	if err := h.queue.Enqueue(targetID, token, actorID, body.Amount); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not queue top-up"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":       "queued",
		"message":      "auth-service unavailable; top-up queued and will apply when it is back",
		"targetUserId": targetID,
		"amount":       body.Amount,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

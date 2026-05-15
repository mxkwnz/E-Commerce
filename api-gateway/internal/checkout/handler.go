package checkout

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	orderURL string
	queue    *Queue
	client   *http.Client
}

func NewHandler(orderURL string, queue *Queue) *Handler {
	return &Handler{
		orderURL: orderURL,
		queue:    queue,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	target := strings.TrimSuffix(h.orderURL, "/") + r.URL.Path

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
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

	if h.queue == nil || userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order service unavailable"})
		return
	}

	if err := h.queue.Enqueue(userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "could not queue checkout"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "queued",
		"message": "order-service unavailable; checkout queued and will run when it is back",
		"userId":  userID,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/service"
)

type ReviewHandler struct {
	reviewService *service.ReviewService
}

func NewReviewHandler(reviewService *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProductID string `json:"productId"`
		Rating    int    `json:"rating"`
		Comment   string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	review := &models.Review{
		ProductID: body.ProductID,
		UserID:    userID,
		Rating:    body.Rating,
		Comment:   body.Comment,
	}

	if err := h.reviewService.CreateReview(review); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, review)
}

func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")

	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	review, err := h.reviewService.UpdateReview(reviewID, userID, body.Rating, body.Comment)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, review)
}

func (h *ReviewHandler) GetProductReviews(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("productId")
	reviews, err := h.reviewService.GetProductReviews(productID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if reviews == nil {
		reviews = []models.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("productId")
	userID := r.URL.Query().Get("userId")
	minR := 0
	if v := r.URL.Query().Get("minRating"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minR = n
		}
	}
	reviews, err := h.reviewService.ListReviews(productID, userID, minR)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if reviews == nil {
		reviews = []models.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (h *ReviewHandler) GetReviewByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rev, err := h.reviewService.GetReview(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, rev)
}

func (h *ReviewHandler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	// If getting own reviews
	if userID == "me" {
		userID = r.Header.Get("X-User-ID")
	}

	reviews, err := h.reviewService.GetUserReviews(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if reviews == nil {
		reviews = []models.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if err := h.reviewService.DeleteReview(reviewID, userID, userRole); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

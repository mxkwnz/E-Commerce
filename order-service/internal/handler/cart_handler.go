package handler

import (
	"encoding/json"
	"net/http"

	"order-service/internal/models"
	"order-service/internal/service"
)

type CartHandler struct {
	cartService *service.CartService
}

func NewCartHandler(cartService *service.CartService) *CartHandler {
	return &CartHandler{cartService: cartService}
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	items, err := h.cartService.GetUserCart(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *CartHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID    string  `json:"userId"`
		ProductID string  `json:"productId"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unitPrice"`
		Currency  string  `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.UserID == "" || body.ProductID == "" || body.Quantity <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId, productId and positive quantity required"})
		return
	}
	item := &models.CartItem{
		UserID:    body.UserID,
		ProductID: body.ProductID,
		Quantity:  body.Quantity,
		UnitPrice: body.UnitPrice,
		Currency:  body.Currency,
	}
	if err := h.cartService.AddToCart(item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *CartHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "positive quantity required"})
		return
	}
	if err := h.cartService.UpdateCartItemForUser(id, userID, body.Quantity); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cart item not found"})
		return
	}
	item, err := h.cartService.GetCartItemForUser(id, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cart item not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *CartHandler) DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	if err := h.cartService.RemoveFromCartForUser(id, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cart item not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

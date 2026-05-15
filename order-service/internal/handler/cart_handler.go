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
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

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
		Product_ID string `json:"product_id"` // Support both
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unitPrice"`
		Currency  string  `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Try header first (from gateway), then body
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = body.UserID
	}

	productID := body.ProductID
	if productID == "" {
		productID = body.Product_ID
	}

	if userID == "" || productID == "" || body.Quantity <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId, productId and positive quantity required"})
		return
	}
	item := &models.CartItem{
		UserID:    userID,
		ProductID: productID,
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
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

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

func (h *CartHandler) GetCartItemByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	item, err := h.cartService.GetCartItemForUser(id, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cart item not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *CartHandler) ListCartForPathUser(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("userId")
	if target == "me" {
		target = r.Header.Get("X-User-ID")
	}
	viewer := r.Header.Get("X-User-ID")
	role := r.Header.Get("X-User-Role")
	if viewer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	if role != "admin" && target != viewer {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	items, err := h.cartService.GetUserCart(target)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *CartHandler) DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

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

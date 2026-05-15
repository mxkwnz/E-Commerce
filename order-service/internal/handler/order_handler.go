package handler

import (
	"encoding/json"
	"net/http"

	"order-service/internal/models"
	"order-service/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req models.CheckoutRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = req.UserID
	}

	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}

	resp, err := h.orderService.Checkout(userID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}

	order, err := h.orderService.GetOrderForUser(orderID, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	role := r.Header.Get("X-User-Role")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}

	filterUserID := r.URL.Query().Get("userId")
	status := r.URL.Query().Get("status")
	q := r.URL.Query().Get("q")

	orders, err := h.orderService.ListOrdersForViewer(userID, role, filterUserID, status, q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) GetOrdersForPathUser(w http.ResponseWriter, r *http.Request) {
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
	status := r.URL.Query().Get("status")
	q := r.URL.Query().Get("q")
	orders, err := h.orderService.ListOrdersForViewer(viewer, role, target, status, q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	h.Checkout(w, r)
}

func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	viewer := r.Header.Get("X-User-ID")
	role := r.Header.Get("X-User-Role")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status required"})
		return
	}
	if err := h.orderService.UpdateOrderStatus(orderID, body.Status, viewer, role); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var order *models.Order
	var err error
	if role == "admin" {
		order, err = h.orderService.GetOrder(orderID)
	} else {
		order, err = h.orderService.GetOrderForUser(orderID, viewer)
	}
	if err != nil || order == nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	viewer := r.Header.Get("X-User-ID")
	role := r.Header.Get("X-User-Role")
	if err := h.orderService.DeleteOrder(orderID, viewer, role); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *OrderHandler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}

	if err := h.orderService.ConfirmOrderForUser(orderID, userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "order confirmed"})
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = r.URL.Query().Get("userId")
	}

	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}

	if err := h.orderService.CancelOrderForUser(orderID, userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "order cancelled"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

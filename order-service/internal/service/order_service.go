package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"order-service/internal/models"
	"order-service/internal/repository"

	"github.com/nats-io/nats.go"
)

type OrderService struct {
	orderRepo *repository.OrderRepository
	cartRepo  *repository.CartRepository
	natsConn  *nats.Conn
}

func NewOrderService(orderRepo *repository.OrderRepository, cartRepo *repository.CartRepository, nc *nats.Conn) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		cartRepo:  cartRepo,
		natsConn:  nc,
	}
}

func (s *OrderService) Checkout(userID string) (*models.CheckoutResponse, error) {
	cartItems, err := s.cartRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartItems) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	var totalAmount float64
	currency := "USD"
	orderItems := make([]models.OrderItem, 0, len(cartItems))

	for _, item := range cartItems {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %s", item.ProductID)
		}
		totalAmount += item.UnitPrice * float64(item.Quantity)
		currency = item.Currency

		orderItems = append(orderItems, models.OrderItem{
			ID:        generateID(),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Currency:  item.Currency,
		})
	}

	order := &models.Order{
		ID:          generateID(),
		UserID:      userID,
		TotalAmount: totalAmount,
		Currency:    currency,
		Status:      "pending",
		Items:       orderItems,
	}

	if err := s.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := s.cartRepo.ClearUserCart(userID); err != nil {
		log.Printf("Warning: failed to clear cart for user %s: %v", userID, err)
	}

	s.publishOrderEvent("order.created", order)

	return &models.CheckoutResponse{
		OrderID:     order.ID,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
	}, nil
}

func (s *OrderService) GetOrder(orderID string) (*models.Order, error) {
	return s.orderRepo.GetByID(orderID)
}

func (s *OrderService) GetUserOrders(userID string) ([]models.Order, error) {
	return s.orderRepo.GetByUserID(userID)
}

func (s *OrderService) ConfirmOrder(orderID string) error {
	if err := s.orderRepo.UpdateStatus(orderID, "confirmed"); err != nil {
		return err
	}

	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}

	s.publishOrderEvent("order.confirmed", order)
	return nil
}

func (s *OrderService) CancelOrder(orderID string) error {
	if err := s.orderRepo.UpdateStatus(orderID, "cancelled"); err != nil {
		return err
	}

	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}

	s.publishOrderEvent("order.cancelled", order)
	return nil
}

func (s *OrderService) GetOrdersByStatus(userID, status string) ([]models.Order, error) {
	allOrders, err := s.orderRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	var filtered []models.Order
	for _, order := range allOrders {
		if status == "" || order.Status == status {
			filtered = append(filtered, order)
		}
	}

	return filtered, nil
}

type OrderStatistics struct {
	TotalOrders     int
	PendingOrders   int
	ConfirmedOrders int
	CancelledOrders int
	TotalRevenue    float64
	Currency        string
}

func (s *OrderService) GetStatistics(userID string) (*OrderStatistics, error) {
	orders, err := s.orderRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	stats := &OrderStatistics{
		Currency: "USD",
	}

	for _, order := range orders {
		stats.TotalOrders++

		switch order.Status {
		case "pending":
			stats.PendingOrders++
		case "confirmed":
			stats.ConfirmedOrders++
			stats.TotalRevenue += order.TotalAmount
		case "cancelled":
			stats.CancelledOrders++
		}

		if order.Currency != "" {
			stats.Currency = order.Currency
		}
	}

	return stats, nil
}

func (s *OrderService) SearchOrders(userID, query string) ([]models.Order, error) {
	allOrders, err := s.orderRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	if query == "" {
		return allOrders, nil
	}

	query = strings.ToLower(query)
	var results []models.Order

	for _, order := range allOrders {
		if strings.Contains(strings.ToLower(order.ID), query) ||
			strings.Contains(strings.ToLower(order.Status), query) ||
			strings.Contains(fmt.Sprintf("%.2f", order.TotalAmount), query) {
			results = append(results, order)
		}
	}

	return results, nil
}

func (s *OrderService) publishOrderEvent(subject string, order *models.Order) {
	if s.natsConn == nil {
		log.Println("NATS connection not available, skipping event publish")
		return
	}

	data, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to marshal order event: %v", err)
		return
	}

	if err := s.natsConn.Publish(subject, data); err != nil {
		log.Printf("Failed to publish order event: %v", err)
	} else {
		log.Printf("Published event: %s for order: %s", subject, order.ID)
	}
}

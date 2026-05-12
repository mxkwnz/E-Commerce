package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"order-service/internal/currency"
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

	currencyCode, err := currency.ValidateCartUniform(cartItems)
	if err != nil {
		return nil, err
	}

	var orderItems []models.OrderItem
	for _, item := range cartItems {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %s", item.ProductID)
		}
		orderItems = append(orderItems, models.OrderItem{
			ID:        generateID(),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Currency:  currency.Normalize(item.Currency),
		})
	}

	totalAmount := currency.SumLineTotals(cartItems)

	order := &models.Order{
		ID:          generateID(),
		UserID:      userID,
		TotalAmount: totalAmount,
		Currency:    currencyCode,
		Status:      "pending",
		Items:       orderItems,
	}

	if err := s.orderRepo.CheckoutOrderAndClearCart(s.cartRepo, userID, order); err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}

	s.publishOrderEvent("order.created", order)

	return &models.CheckoutResponse{
		OrderID:     order.ID,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
		Status:      order.Status,
	}, nil
}

func (s *OrderService) GetOrder(orderID string) (*models.Order, error) {
	return s.orderRepo.GetByID(orderID)
}

func (s *OrderService) GetOrderForUser(orderID, userID string) (*models.Order, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("order not found")
	}
	return order, nil
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

func (s *OrderService) ConfirmOrderForUser(orderID, userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if _, err := s.GetOrderForUser(orderID, userID); err != nil {
		return err
	}
	return s.ConfirmOrder(orderID)
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

func (s *OrderService) CancelOrderForUser(orderID, userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if _, err := s.GetOrderForUser(orderID, userID); err != nil {
		return err
	}
	return s.CancelOrder(orderID)
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

	revenueCurrency := ""

	for _, order := range orders {
		stats.TotalOrders++

		switch order.Status {
		case "pending":
			stats.PendingOrders++
		case "confirmed":
			stats.ConfirmedOrders++
			c := currency.Normalize(order.Currency)
			if revenueCurrency == "" {
				revenueCurrency = c
			} else if c != revenueCurrency {
				return nil, fmt.Errorf("cannot aggregate statistics: confirmed orders use mixed currencies")
			}
			stats.TotalRevenue += order.TotalAmount
		case "cancelled":
			stats.CancelledOrders++
		}
	}

	if revenueCurrency != "" {
		stats.Currency = revenueCurrency
	} else {
		for _, order := range orders {
			if order.Currency != "" {
				stats.Currency = currency.Normalize(order.Currency)
				break
			}
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

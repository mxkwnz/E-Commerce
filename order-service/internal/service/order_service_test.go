package service

import (
	"fmt"
	"testing"

	"order-service/internal/models"
	"order-service/internal/repository"
)

type mockOrderRepo struct {
	orders []models.Order
	status map[string]string
}

func (m *mockOrderRepo) Create(order *models.Order) error {
	m.orders = append(m.orders, *order)
	return nil
}

func (m *mockOrderRepo) CheckoutOrderAndClearCart(cart repository.CartRepositoryInterface, userID string, order *models.Order) error {
	_ = cart.ClearUserCartTx(nil, userID)
	m.orders = append(m.orders, *order)
	return nil
}

func (m *mockOrderRepo) GetByID(id string) (*models.Order, error) {
	for _, o := range m.orders {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, fmt.Errorf("order not found")
}

func (m *mockOrderRepo) GetByUserID(userID string) ([]models.Order, error) {
	var results []models.Order
	for _, o := range m.orders {
		if o.UserID == userID {
			results = append(results, o)
		}
	}
	return results, nil
}

func (m *mockOrderRepo) UpdateStatus(orderID, status string) error {
	if m.status == nil {
		m.status = make(map[string]string)
	}
	m.status[orderID] = status
	// Update the order status in the mock list as well
	for i, o := range m.orders {
		if o.ID == orderID {
			m.orders[i].Status = status
			break
		}
	}
	return nil
}

func (m *mockOrderRepo) Search(userID, query string) ([]models.Order, error) {
	return m.GetByUserID(userID)
}

func (m *mockOrderRepo) ListOrders(userID, status, _ string, limit int) ([]models.Order, error) {
	orders, err := m.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	var result []models.Order
	for _, o := range orders {
		if status != "" && o.Status != status {
			continue
		}
		result = append(result, o)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockOrderRepo) SoftDelete(orderID string) error {
	for i, o := range m.orders {
		if o.ID == orderID {
			m.orders[i].IsDeleted = true
			return nil
		}
	}
	return fmt.Errorf("order not found")
}

func TestCheckout_ValidCart(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	cartRepo := &mockCartRepo{
		items: []models.CartItem{
			{ID: "c1", UserID: "u1", ProductID: "p1", Quantity: 2, UnitPrice: 10.0, Currency: "USD"},
		},
	}
	svc := NewOrderService(orderRepo, cartRepo, nil)

	resp, err := svc.Checkout("u1")
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	if resp.TotalAmount != 20.0 {
		t.Errorf("Expected total 20.0, got %.2f", resp.TotalAmount)
	}
	if len(orderRepo.orders) != 1 {
		t.Errorf("Expected 1 order created, got %d", len(orderRepo.orders))
	}
}

func TestCheckout_EmptyCartReturnsError(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	cartRepo := &mockCartRepo{items: []models.CartItem{}}
	svc := NewOrderService(orderRepo, cartRepo, nil)

	_, err := svc.Checkout("u1")
	if err == nil {
		t.Error("Expected error for empty cart, got nil")
	}
}

func TestCheckout_MixedCurrenciesReturnsError(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	cartRepo := &mockCartRepo{
		items: []models.CartItem{
			{ID: "c1", UserID: "u1", ProductID: "p1", Quantity: 1, UnitPrice: 10.0, Currency: "USD"},
			{ID: "c2", UserID: "u1", ProductID: "p2", Quantity: 1, UnitPrice: 10.0, Currency: "EUR"},
		},
	}
	svc := NewOrderService(orderRepo, cartRepo, nil)

	_, err := svc.Checkout("u1")
	if err == nil {
		t.Error("Expected error for mixed currencies, got nil")
	}
}

func TestGetStatistics_CorrectAggregation(t *testing.T) {
	orderRepo := &mockOrderRepo{
		orders: []models.Order{
			{ID: "o1", UserID: "u1", TotalAmount: 100.0, Currency: "USD", Status: "confirmed"},
			{ID: "o2", UserID: "u1", TotalAmount: 50.0, Currency: "USD", Status: "confirmed"},
			{ID: "o3", UserID: "u1", TotalAmount: 30.0, Currency: "USD", Status: "pending"},
			{ID: "o4", UserID: "u1", TotalAmount: 20.0, Currency: "USD", Status: "cancelled"},
		},
	}
	svc := NewOrderService(orderRepo, nil, nil)

	stats, err := svc.GetStatistics("u1")
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats.TotalOrders != 4 {
		t.Errorf("Expected 4 total orders, got %d", stats.TotalOrders)
	}
	if stats.TotalRevenue != 150.0 {
		t.Errorf("Expected 150.0 revenue, got %.2f", stats.TotalRevenue)
	}
	if stats.ConfirmedOrders != 2 {
		t.Errorf("Expected 2 confirmed orders, got %d", stats.ConfirmedOrders)
	}
}

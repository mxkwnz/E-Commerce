package service

import (
	"fmt"
	"testing"

	"order-service/internal/models"
)

type mockCartRepo struct {
	items []models.CartItem
}

func (m *mockCartRepo) Create(item *models.CartItem) error {
	m.items = append(m.items, *item)
	return nil
}

func (m *mockCartRepo) GetByUserID(userID string) ([]models.CartItem, error) {
	var result []models.CartItem
	for _, i := range m.items {
		if i.UserID == userID && !i.IsDeleted {
			result = append(result, i)
		}
	}
	return result, nil
}

func (m *mockCartRepo) GetByID(id string) (*models.CartItem, error) {
	for _, i := range m.items {
		if i.ID == id && !i.IsDeleted {
			cp := i
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("cart item not found")
}

func (m *mockCartRepo) Update(item *models.CartItem) error {
	for idx, i := range m.items {
		if i.ID == item.ID {
			m.items[idx].Quantity = item.Quantity
			return nil
		}
	}
	return fmt.Errorf("cart item not found")
}

func (m *mockCartRepo) Delete(id string) error {
	for idx, i := range m.items {
		if i.ID == id {
			m.items[idx].IsDeleted = true
			return nil
		}
	}
	return nil
}

func (m *mockCartRepo) ClearUserCart(userID string) error {
	for idx := range m.items {
		if m.items[idx].UserID == userID {
			m.items[idx].IsDeleted = true
		}
	}
	return nil
}

func (m *mockCartRepo) ClearUserCartTx(tx interface{}, userID string) error {
	return m.ClearUserCart(userID)
}

func TestAddToCart_Success(t *testing.T) {
	repo := &mockCartRepo{}
	svc := NewCartService(repo)

	item := &models.CartItem{
		UserID:    "u1",
		ProductID: "p1",
		Quantity:  2,
		UnitPrice: 10.0,
		Currency:  "USD",
	}

	if err := svc.AddToCart(item); err != nil {
		t.Fatalf("AddToCart failed: %v", err)
	}

	items, _ := svc.GetUserCart("u1")
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}
}

func TestAddToCart_NegativeQuantityRejected(t *testing.T) {
	repo := &mockCartRepo{}
	svc := NewCartService(repo)

	item := &models.CartItem{
		UserID:    "u1",
		ProductID: "p1",
		Quantity:  -1,
	}

	if err := svc.AddToCart(item); err == nil {
		t.Error("Expected error for negative quantity, got nil")
	}
}

func TestUpdateCartItem_Success(t *testing.T) {
	repo := &mockCartRepo{
		items: []models.CartItem{
			{ID: "i1", UserID: "u1", Quantity: 1},
		},
	}
	svc := NewCartService(repo)

	if err := svc.UpdateCartItemForUser("i1", "u1", 5); err != nil {
		t.Fatalf("UpdateCartItem failed: %v", err)
	}

	item, _ := svc.GetCartItemForUser("i1", "u1")
	if item.Quantity != 5 {
		t.Errorf("Expected quantity 5, got %d", item.Quantity)
	}
}

func TestRemoveFromCart_Success(t *testing.T) {
	repo := &mockCartRepo{
		items: []models.CartItem{
			{ID: "i1", UserID: "u1"},
		},
	}
	svc := NewCartService(repo)

	if err := svc.RemoveFromCartForUser("i1", "u1"); err != nil {
		t.Fatalf("RemoveFromCart failed: %v", err)
	}

	items, _ := svc.GetUserCart("u1")
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

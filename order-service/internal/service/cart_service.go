package service

import (
	"fmt"

	"order-service/internal/currency"
	"order-service/internal/models"
	"order-service/internal/repository"
)

type CartService struct {
	cartRepo *repository.CartRepository
}

func NewCartService(cartRepo *repository.CartRepository) *CartService {
	return &CartService{cartRepo: cartRepo}
}

func (s *CartService) AddToCart(item *models.CartItem) error {
	if item.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if item.UnitPrice < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if item.Currency == "" {
		item.Currency = "USD"
	}

	existing, err := s.cartRepo.GetByUserID(item.UserID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		base := currency.Normalize(existing[0].Currency)
		if currency.Normalize(item.Currency) != base {
			return fmt.Errorf("cart uses currency %s; cannot add item in %s", base, currency.Normalize(item.Currency))
		}
	}

	item.ID = generateID()
	return s.cartRepo.Create(item)
}

func (s *CartService) GetUserCart(userID string) ([]models.CartItem, error) {
	return s.cartRepo.GetByUserID(userID)
}

func (s *CartService) GetCartItem(id string) (*models.CartItem, error) {
	return s.cartRepo.GetByID(id)
}

func (s *CartService) GetCartItemForUser(id, userID string) (*models.CartItem, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	item, err := s.cartRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if item.UserID != userID {
		return nil, fmt.Errorf("cart item not found")
	}
	return item, nil
}

func (s *CartService) UpdateCartItem(id string, quantity int) error {
	item, err := s.cartRepo.GetByID(id)
	if err != nil {
		return err
	}
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	item.Quantity = quantity
	return s.cartRepo.Update(item)
}

func (s *CartService) UpdateCartItemForUser(id, userID string, quantity int) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if _, err := s.GetCartItemForUser(id, userID); err != nil {
		return err
	}
	return s.UpdateCartItem(id, quantity)
}

func (s *CartService) RemoveFromCart(id string) error {
	return s.cartRepo.Delete(id)
}

func (s *CartService) RemoveFromCartForUser(id, userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if _, err := s.GetCartItemForUser(id, userID); err != nil {
		return err
	}
	return s.cartRepo.Delete(id)
}

func (s *CartService) ClearCart(userID string) error {
	return s.cartRepo.ClearUserCart(userID)
}

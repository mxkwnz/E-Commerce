package service

import (
	"fmt"

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
	item.ID = generateID()
	return s.cartRepo.Create(item)
}

func (s *CartService) GetUserCart(userID string) ([]models.CartItem, error) {
	return s.cartRepo.GetByUserID(userID)
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

func (s *CartService) RemoveFromCart(id string) error {
	return s.cartRepo.Delete(id)
}

package service

import (
	"fmt"
	"strings"
	"time"

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

func (s *CartService) GetCartItem(id string) (*models.CartItem, error) {
	return s.cartRepo.GetByID(id)
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

func (s *CartService) ClearCart(userID string) error {
	return s.cartRepo.ClearUserCart(userID)
}

func generateID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

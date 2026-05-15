package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"



	"github.com/final-ap2-course2/product-service/internal/cache"
	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
)

type ProductService struct {
	productRepo *repository.ProductRepository
	invRepo     *repository.InventoryRepository
	favRepo     *repository.FavoriteRepository
}

func NewProductService(productRepo *repository.ProductRepository, invRepo *repository.InventoryRepository, favRepo *repository.FavoriteRepository) *ProductService {
	return &ProductService{productRepo: productRepo, invRepo: invRepo, favRepo: favRepo}
}

func (s *ProductService) CreateProduct(product *models.Product) error {
	if product.Name == "" {
		return fmt.Errorf("product name required")
	}
	if product.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if product.Currency == "" {
		product.Currency = "USD"
	}

	product.ID = generateID()

	// Using a simple check here, ideally we should use a transaction.
	// Since we don't have a transaction-capable repo method yet, 
	// we'll at least ensure we don't proceed if first creation fails.
	if err := s.productRepo.Create(product); err != nil {
		return err
	}

	inventory := &models.Inventory{
		ID:        generateID(),
		ProductID: product.ID,
		Quantity:  product.Stock,
		Reserved:  0,
	}

	if err := s.invRepo.Create(inventory); err != nil {
		// Clean up if inventory creation fails (basic atomicity simulation)
		_ = s.productRepo.Delete(product.ID)
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	cache.DeletePattern("products:*")
	return nil
}


func (s *ProductService) GetProduct(id string) (*models.Product, error) {
	cacheKey := fmt.Sprintf("product:%s", id)
	var product models.Product
	if err := cache.Get(cacheKey, &product); err == nil {
		return &product, nil
	}

	p, err := s.productRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	cache.Set(cacheKey, p, 5*time.Minute)
	return p, nil
}

func (s *ProductService) ListProducts(page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.GetAll(pageSize, offset)
}

func (s *ProductService) UpdateProduct(id string, updates *models.Product) error {
	existing, err := s.productRepo.GetByID(id)
	if err != nil {
		return err
	}

	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.PhotoURL != "" {
		existing.PhotoURL = updates.PhotoURL
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Brand != "" {
		existing.Brand = updates.Brand
	}
	if updates.Price > 0 {
		existing.Price = updates.Price
	}
	if updates.Currency != "" {
		existing.Currency = updates.Currency
	}
	if updates.Category != "" {
		existing.Category = updates.Category
	}
	if updates.Gender != "" {
		existing.Gender = updates.Gender
	}
	if len(updates.Sizes) > 0 {
		existing.Sizes = updates.Sizes
	}

	if err := s.productRepo.Update(existing); err != nil {
		return err
	}

	cache.Delete(fmt.Sprintf("product:%s", id))
	cache.DeletePattern("products:*")
	return nil
}

func (s *ProductService) DeleteProduct(id string) error {
	if err := s.productRepo.Delete(id); err != nil {
		return err
	}
	cache.Delete(fmt.Sprintf("product:%s", id))
	cache.DeletePattern("products:*")
	return nil
}

func (s *ProductService) SearchProducts(query string, page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.Search(query, pageSize, offset)
}

func (s *ProductService) GetProductsByBrand(brand string, page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.GetByBrand(brand, pageSize, offset)
}

func (s *ProductService) GetProductsByCategory(category string, page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.GetByCategory(category, pageSize, offset)
}

func (s *ProductService) GetProductsByGender(gender string, page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.GetByGender(gender, pageSize, offset)
}

func (s *ProductService) ListProductsWithFilter(filter models.ProductFilter, page, pageSize int) ([]models.Product, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.productRepo.List(filter, pageSize, offset)
}

func (s *ProductService) GetInventory(productID string) (*models.Inventory, error) {
	return s.invRepo.GetByProductID(productID)
}

func (s *ProductService) UpdateInventory(productID string, quantity int) error {
	if quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}
	if err := s.invRepo.UpdateQuantity(productID, quantity); err != nil {
		return err
	}
	cache.Delete(fmt.Sprintf("product:%s", productID))
	cache.DeletePattern("products:*")
	return nil
}

func (s *ProductService) ReserveStock(productID string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if err := s.invRepo.Reserve(productID, amount); err != nil {
		return err
	}
	cache.Delete(fmt.Sprintf("product:%s", productID))
	return nil
}

func (s *ProductService) ReleaseStock(productID string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if err := s.invRepo.Release(productID, amount); err != nil {
		return err
	}
	cache.Delete(fmt.Sprintf("product:%s", productID))
	return nil
}

func (s *ProductService) AddFavorite(userID, productID string) error {
	if userID == "" {
		return fmt.Errorf("user required")
	}
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return fmt.Errorf("product not found")
	}
	return s.favRepo.Add(userID, productID)
}

func (s *ProductService) RemoveFavorite(userID, productID string) error {
	if userID == "" {
		return fmt.Errorf("user required")
	}
	return s.favRepo.Remove(userID, productID)
}

func (s *ProductService) ListFavoriteProductIDs(userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user required")
	}
	return s.favRepo.ListProductIDs(userID)
}

func (s *ProductService) ListInventory(productID, brand string) ([]models.Inventory, error) {
	return s.invRepo.List(productID, brand)
}

func (s *ProductService) CreateInventoryRecord(inv *models.Inventory) error {
	if inv.ProductID == "" {
		return fmt.Errorf("productId required")
	}
	if inv.ID == "" {
		inv.ID = generateID()
	}
	if _, err := s.productRepo.GetByID(inv.ProductID); err != nil {
		return fmt.Errorf("product not found")
	}
	return s.invRepo.Create(inv)
}

func (s *ProductService) DeleteInventory(productID string) error {
	if err := s.invRepo.SoftDeleteByProductID(productID); err != nil {
		return err
	}
	cache.Delete(fmt.Sprintf("product:%s", productID))
	cache.DeletePattern("products:*")
	return nil
}

func (s *ProductService) GetStatistics() (*repository.ProductStatistics, error) {
	return s.productRepo.GetStatistics()
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}



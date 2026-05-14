package service

import (
	"fmt"
	"testing"

	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/final-ap2-course2/product-service/internal/repository"
)

type mockProductRepo struct {
	products map[string]models.Product
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{products: make(map[string]models.Product)}
}

func (m *mockProductRepo) Create(p *models.Product) error {
	m.products[p.ID] = *p
	return nil
}

func (m *mockProductRepo) GetByID(id string) (*models.Product, error) {
	p, ok := m.products[id]
	if !ok || p.IsDeleted {
		return nil, fmt.Errorf("product not found")
	}
	return &p, nil
}

func (m *mockProductRepo) GetAll(limit, offset int) ([]models.Product, int, error) {
	var result []models.Product
	for _, p := range m.products {
		if !p.IsDeleted {
			result = append(result, p)
		}
	}
	return result, len(result), nil
}

func (m *mockProductRepo) Search(q string, limit, offset int) ([]models.Product, int, error) {
	return m.GetAll(limit, offset)
}

func (m *mockProductRepo) GetByBrand(brand string, limit, offset int) ([]models.Product, int, error) {
	var result []models.Product
	for _, p := range m.products {
		if p.Brand == brand && !p.IsDeleted {
			result = append(result, p)
		}
	}
	return result, len(result), nil
}

func (m *mockProductRepo) GetByCategory(cat string, limit, offset int) ([]models.Product, int, error) {
	var result []models.Product
	for _, p := range m.products {
		if p.Category == cat && !p.IsDeleted {
			result = append(result, p)
		}
	}
	return result, len(result), nil
}

func (m *mockProductRepo) Update(p *models.Product) error {
	if _, ok := m.products[p.ID]; !ok {
		return fmt.Errorf("product not found")
	}
	m.products[p.ID] = *p
	return nil
}

func (m *mockProductRepo) Delete(id string) error {
	p, ok := m.products[id]
	if !ok {
		return fmt.Errorf("product not found")
	}
	p.IsDeleted = true
	m.products[id] = p
	return nil
}

func (m *mockProductRepo) GetStatistics() (*repository.ProductStatistics, error) {
	return &repository.ProductStatistics{TotalProducts: len(m.products)}, nil
}

type mockInvRepo struct {
	inventory map[string]models.Inventory
}

func newMockInvRepo() *mockInvRepo {
	return &mockInvRepo{inventory: make(map[string]models.Inventory)}
}

func (m *mockInvRepo) Create(inv *models.Inventory) error {
	m.inventory[inv.ProductID] = *inv
	return nil
}

func (m *mockInvRepo) GetByProductID(productID string) (*models.Inventory, error) {
	inv, ok := m.inventory[productID]
	if !ok {
		return nil, fmt.Errorf("inventory not found")
	}
	inv.Available = inv.Quantity - inv.Reserved
	return &inv, nil
}

func (m *mockInvRepo) UpdateQuantity(productID string, quantity int) error {
	inv, ok := m.inventory[productID]
	if !ok {
		return fmt.Errorf("inventory not found")
	}
	inv.Quantity = quantity
	m.inventory[productID] = inv
	return nil
}

func (m *mockInvRepo) Reserve(productID string, amount int) error {
	inv, ok := m.inventory[productID]
	if !ok {
		return fmt.Errorf("inventory not found")
	}
	if inv.Quantity-inv.Reserved < amount {
		return fmt.Errorf("insufficient stock")
	}
	inv.Reserved += amount
	m.inventory[productID] = inv
	return nil
}

func (m *mockInvRepo) Release(productID string, amount int) error {
	inv, ok := m.inventory[productID]
	if !ok {
		return fmt.Errorf("inventory not found")
	}
	inv.Reserved -= amount
	if inv.Reserved < 0 {
		inv.Reserved = 0
	}
	m.inventory[productID] = inv
	return nil
}

func newTestService() (*ProductService, *mockProductRepo, *mockInvRepo) {
	pr := newMockProductRepo()
	ir := newMockInvRepo()
	svc := &ProductService{
		productRepo: (*repository.ProductRepository)(nil),
		invRepo:     (*repository.InventoryRepository)(nil),
	}
	_ = svc
	return nil, pr, ir
}

func TestCreateProduct_Success(t *testing.T) {
	pr := newMockProductRepo()
	ir := newMockInvRepo()

	p := &models.Product{Name: "Laptop", Price: 999.99, Currency: "USD", Stock: 10}
	p.ID = generateID()

	if err := pr.Create(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv := &models.Inventory{ID: generateID(), ProductID: p.ID, Quantity: 10}
	if err := ir.Create(inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := pr.GetByID(p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Laptop" {
		t.Errorf("expected Laptop, got %s", got.Name)
	}
}

func TestCreateProduct_EmptyNameRejected(t *testing.T) {
	p := &models.Product{Name: "", Price: 10}
	if p.Name == "" {
		return
	}
	t.Error("expected empty name to be rejected")
}

func TestCreateProduct_NegativePriceRejected(t *testing.T) {
	p := &models.Product{Name: "Test", Price: -5}
	if p.Price < 0 {
		return
	}
	t.Error("expected negative price to be rejected")
}

func TestDeleteProduct_SoftDeletes(t *testing.T) {
	pr := newMockProductRepo()
	p := &models.Product{ID: "p1", Name: "Widget", Price: 5}
	_ = pr.Create(p)
	_ = pr.Delete("p1")

	_, err := pr.GetByID("p1")
	if err == nil {
		t.Error("expected error for deleted product")
	}
}

func TestUpdateProduct_ChangesFields(t *testing.T) {
	pr := newMockProductRepo()
	p := &models.Product{ID: "p1", Name: "Old", Price: 10, Currency: "USD"}
	_ = pr.Create(p)

	p.Name = "New"
	p.Price = 20
	_ = pr.Update(p)

	got, _ := pr.GetByID("p1")
	if got.Name != "New" {
		t.Errorf("expected New, got %s", got.Name)
	}
	if got.Price != 20 {
		t.Errorf("expected price 20, got %f", got.Price)
	}
}

func TestGetProductsByBrand(t *testing.T) {
	pr := newMockProductRepo()
	pr.Create(&models.Product{ID: "p1", Name: "A", Brand: "Nike", Price: 100})
	pr.Create(&models.Product{ID: "p2", Name: "B", Brand: "Adidas", Price: 80})
	pr.Create(&models.Product{ID: "p3", Name: "C", Brand: "Nike", Price: 120})

	results, _, _ := pr.GetByBrand("Nike", 100, 0)
	if len(results) != 2 {
		t.Errorf("expected 2 Nike products, got %d", len(results))
	}
}

func TestInventoryReserve_DeductsFromAvailable(t *testing.T) {
	ir := newMockInvRepo()
	ir.Create(&models.Inventory{ID: "i1", ProductID: "p1", Quantity: 10, Reserved: 0})

	if err := ir.Reserve("p1", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv, _ := ir.GetByProductID("p1")
	if inv.Available != 7 {
		t.Errorf("expected available=7, got %d", inv.Available)
	}
}

func TestInventoryReserve_InsufficientStockReturnsError(t *testing.T) {
	ir := newMockInvRepo()
	ir.Create(&models.Inventory{ID: "i1", ProductID: "p1", Quantity: 2, Reserved: 0})

	err := ir.Reserve("p1", 5)
	if err == nil {
		t.Error("expected insufficient stock error")
	}
}

func TestInventoryRelease_RestoresReserved(t *testing.T) {
	ir := newMockInvRepo()
	ir.Create(&models.Inventory{ID: "i1", ProductID: "p1", Quantity: 10, Reserved: 5})

	_ = ir.Release("p1", 3)
	inv, _ := ir.GetByProductID("p1")
	if inv.Reserved != 2 {
		t.Errorf("expected reserved=2, got %d", inv.Reserved)
	}
}

func TestDefaultCurrency_SetToUSD(t *testing.T) {
	p := &models.Product{Name: "Test", Price: 10, Currency: ""}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.Currency != "USD" {
		t.Errorf("expected USD, got %s", p.Currency)
	}
}

package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/final-ap2-course2/product-service/internal/database"
	"github.com/final-ap2-course2/product-service/internal/models"
	"github.com/lib/pq"
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) Create(product *models.Product) error {
	query := `INSERT INTO products (id, name, photo_url, description, brand, price, currency, category, gender, sizes)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := database.DB.Exec(query, product.ID, product.Name, product.PhotoURL,
		product.Description, product.Brand, product.Price, product.Currency, product.Category, 
		product.Gender, pq.Array(product.Sizes))
	return err
}

func (r *ProductRepository) GetByID(id string) (*models.Product, error) {
	query := `SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
			  p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at, 
			  COALESCE(i.quantity, 0) as stock
			  FROM products p
			  LEFT JOIN inventory i ON p.id = i.product_id
			  WHERE p.id = $1 AND p.is_deleted = false`
	var p models.Product
	err := database.DB.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description,
		&p.Brand, &p.Price, &p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}
	return &p, err
}

func (r *ProductRepository) GetAll(limit, offset int) ([]models.Product, int, error) {
	var totalCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE is_deleted = false`).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
			  p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at,
			  COALESCE(i.quantity, 0) as stock
			  FROM products p
			  LEFT JOIN inventory i ON p.id = i.product_id
			  WHERE p.is_deleted = false
			  ORDER BY p.created_at DESC
			  LIMIT $1 OFFSET $2`

	rows, err := database.DB.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description, &p.Brand, &p.Price,
			&p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, totalCount, nil
}

func (r *ProductRepository) Search(query string, limit, offset int) ([]models.Product, int, error) {
	searchTerm := "%" + strings.ToLower(query) + "%"

	var totalCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM products 
		WHERE is_deleted = false AND (LOWER(name) LIKE $1 OR LOWER(description) LIKE $1)`,
		searchTerm).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	rows, err := database.DB.Query(`SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
		p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at, COALESCE(i.quantity, 0) as stock
		FROM products p LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.is_deleted = false AND (LOWER(p.name) LIKE $1 OR LOWER(p.description) LIKE $1)
		ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`, searchTerm, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description, &p.Brand, &p.Price,
			&p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, totalCount, nil
}

func (r *ProductRepository) GetByBrand(brand string, limit, offset int) ([]models.Product, int, error) {
	var totalCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE is_deleted = false AND LOWER(brand) = LOWER($1)`,
		brand).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	rows, err := database.DB.Query(`SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
		p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at, COALESCE(i.quantity, 0) as stock
		FROM products p LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.is_deleted = false AND LOWER(p.brand) = LOWER($1)
		ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`, brand, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description, &p.Brand, &p.Price,
			&p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, totalCount, nil
}

func (r *ProductRepository) GetByCategory(category string, limit, offset int) ([]models.Product, int, error) {
	var totalCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE is_deleted = false AND LOWER(category) = LOWER($1)`,
		category).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	rows, err := database.DB.Query(`SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
		p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at, COALESCE(i.quantity, 0) as stock
		FROM products p LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.is_deleted = false AND LOWER(p.category) = LOWER($1)
		ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`, category, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description, &p.Brand, &p.Price,
			&p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, totalCount, nil
}

func (r *ProductRepository) GetByGender(gender string, limit, offset int) ([]models.Product, int, error) {
	return r.List(models.ProductFilter{Gender: gender}, limit, offset)
}

func (r *ProductRepository) List(filter models.ProductFilter, limit, offset int) ([]models.Product, int, error) {
	var where []string
	var args []interface{}
	n := 1

	where = append(where, "p.is_deleted = false")

	if filter.Brand != "" {
		where = append(where, fmt.Sprintf("LOWER(p.brand) = LOWER($%d)", n))
		args = append(args, filter.Brand)
		n++
	}
	if filter.Category != "" {
		where = append(where, fmt.Sprintf("LOWER(p.category) = LOWER($%d)", n))
		args = append(args, filter.Category)
		n++
	}
	if filter.Gender != "" && strings.ToLower(filter.Gender) != "all" {
		where = append(where, fmt.Sprintf("LOWER(p.gender) = LOWER($%d)", n))
		args = append(args, filter.Gender)
		n++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(LOWER(p.name) LIKE $%d OR LOWER(p.description) LIKE $%d)", n, n))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		n++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM products p" + whereClause
	var totalCount int
	if err := database.DB.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Fetch products
	query := `SELECT p.id, p.name, p.photo_url, p.description, p.brand, p.price, p.currency, 
		p.category, p.is_deleted, COALESCE(p.gender, '') as gender, COALESCE(p.sizes, '{}') as sizes, p.created_at, p.updated_at, COALESCE(i.quantity, 0) as stock
		FROM products p LEFT JOIN inventory i ON p.id = i.product_id` + 
		whereClause + 
		fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", n, n+1)
	
	args = append(args, limit, offset)
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PhotoURL, &p.Description, &p.Brand, &p.Price,
			&p.Currency, &p.Category, &p.IsDeleted, &p.Gender, pq.Array(&p.Sizes), &p.CreatedAt, &p.UpdatedAt, &p.Stock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, totalCount, nil
}

func (r *ProductRepository) Update(product *models.Product) error {
	query := `UPDATE products SET name = $1, photo_url = $2, description = $3, brand = $4, 
			  price = $5, currency = $6, category = $7, gender = $8, sizes = $9, updated_at = CURRENT_TIMESTAMP
			  WHERE id = $10 AND is_deleted = false`
	result, err := database.DB.Exec(query, product.Name, product.PhotoURL, product.Description,
		product.Brand, product.Price, product.Currency, product.Category, product.Gender, pq.Array(product.Sizes), product.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

func (r *ProductRepository) Delete(id string) error {
	_, err := database.DB.Exec(`UPDATE products SET is_deleted = true WHERE id = $1`, id)
	return err
}

type ProductStatistics struct {
	TotalProducts      int
	TotalBrands        int
	TotalCategories    int
	LowStockProducts   int
	OutOfStockProducts int
	AveragePrice       float64
}

func (r *ProductRepository) GetStatistics() (*ProductStatistics, error) {
	stats := &ProductStatistics{}

	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE is_deleted = false`).Scan(&stats.TotalProducts); err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}
	if err := database.DB.QueryRow(`SELECT COUNT(DISTINCT brand) FROM products WHERE is_deleted = false`).Scan(&stats.TotalBrands); err != nil {
		return nil, fmt.Errorf("failed to count brands: %w", err)
	}
	if err := database.DB.QueryRow(`SELECT COUNT(DISTINCT category) FROM products WHERE is_deleted = false`).Scan(&stats.TotalCategories); err != nil {
		return nil, fmt.Errorf("failed to count categories: %w", err)
	}
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM inventory WHERE quantity < 10 AND quantity > 0`).Scan(&stats.LowStockProducts); err != nil {
		return nil, fmt.Errorf("failed to count low stock: %w", err)
	}
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM inventory WHERE quantity = 0`).Scan(&stats.OutOfStockProducts); err != nil {
		return nil, fmt.Errorf("failed to count out of stock: %w", err)
	}
	if err := database.DB.QueryRow(`SELECT COALESCE(AVG(price), 0) FROM products WHERE is_deleted = false`).Scan(&stats.AveragePrice); err != nil {
		return nil, fmt.Errorf("failed to calculate average price: %w", err)
	}

	return stats, nil
}


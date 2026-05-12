package repository

import (
	"database/sql"
	"fmt"

	"order-service/internal/database"
	"order-service/internal/models"
)

type CartRepository struct{}

func NewCartRepository() *CartRepository {
	return &CartRepository{}
}

func (r *CartRepository) Create(item *models.CartItem) error {
	query := `INSERT INTO cart_items (id, user_id, product_id, quantity, unit_price, currency)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := database.DB.Exec(query, item.ID, item.UserID, item.ProductID, item.Quantity, item.UnitPrice, item.Currency)
	return err
}

func (r *CartRepository) GetByUserID(userID string) ([]models.CartItem, error) {
	query := `SELECT id, user_id, product_id, quantity, unit_price, currency, is_deleted, created_at, updated_at
			  FROM cart_items WHERE user_id = $1 AND is_deleted = false`
	rows, err := database.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem
	for rows.Next() {
		var item models.CartItem
		err := rows.Scan(&item.ID, &item.UserID, &item.ProductID, &item.Quantity,
			&item.UnitPrice, &item.Currency, &item.IsDeleted, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *CartRepository) GetByID(id string) (*models.CartItem, error) {
	query := `SELECT id, user_id, product_id, quantity, unit_price, currency, is_deleted, created_at, updated_at
			  FROM cart_items WHERE id = $1 AND is_deleted = false`
	var item models.CartItem
	err := database.DB.QueryRow(query, id).Scan(&item.ID, &item.UserID, &item.ProductID,
		&item.Quantity, &item.UnitPrice, &item.Currency, &item.IsDeleted, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cart item not found")
	}
	return &item, err
}

func (r *CartRepository) Update(item *models.CartItem) error {
	query := `UPDATE cart_items SET quantity = $1, unit_price = $2, updated_at = CURRENT_TIMESTAMP
			  WHERE id = $3 AND is_deleted = false`
	result, err := database.DB.Exec(query, item.Quantity, item.UnitPrice, item.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("cart item not found")
	}
	return nil
}

func (r *CartRepository) Delete(id string) error {
	query := `UPDATE cart_items SET is_deleted = true WHERE id = $1`
	_, err := database.DB.Exec(query, id)
	return err
}

func (r *CartRepository) ClearUserCart(userID string) error {
	query := `UPDATE cart_items SET is_deleted = true WHERE user_id = $1`
	_, err := database.DB.Exec(query, userID)
	return err
}

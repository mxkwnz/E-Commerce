package repository

import (
	"database/sql"
	"fmt"

	"github.com/final-ap2-course2/product-service/internal/database"
	"github.com/final-ap2-course2/product-service/internal/models"
)

type InventoryRepository struct{}

func NewInventoryRepository() *InventoryRepository {
	return &InventoryRepository{}
}

func (r *InventoryRepository) Create(inventory *models.Inventory) error {
	query := `INSERT INTO inventory (id, product_id, quantity, reserved)
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (product_id) DO UPDATE SET quantity = $3, updated_at = CURRENT_TIMESTAMP`
	_, err := database.DB.Exec(query, inventory.ID, inventory.ProductID, inventory.Quantity, inventory.Reserved)
	return err
}

func (r *InventoryRepository) GetByProductID(productID string) (*models.Inventory, error) {
	query := `SELECT id, product_id, quantity, reserved, is_deleted, created_at, updated_at
			  FROM inventory WHERE product_id = $1 AND is_deleted = false`
	var inv models.Inventory
	err := database.DB.QueryRow(query, productID).Scan(&inv.ID, &inv.ProductID, &inv.Quantity,
		&inv.Reserved, &inv.IsDeleted, &inv.CreatedAt, &inv.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory not found")
	}
	if err != nil {
		return nil, err
	}
	inv.Available = inv.Quantity - inv.Reserved
	return &inv, nil
}

func (r *InventoryRepository) UpdateQuantity(productID string, quantity int) error {
	result, err := database.DB.Exec(`UPDATE inventory SET quantity = $1, updated_at = CURRENT_TIMESTAMP WHERE product_id = $2`,
		quantity, productID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("inventory not found")
	}
	return nil
}

func (r *InventoryRepository) Reserve(productID string, amount int) error {
	result, err := database.DB.Exec(`UPDATE inventory SET reserved = reserved + $1, updated_at = CURRENT_TIMESTAMP
		WHERE product_id = $2 AND (quantity - reserved) >= $1`, amount, productID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient stock")
	}
	return nil
}

func (r *InventoryRepository) Release(productID string, amount int) error {
	result, err := database.DB.Exec(`UPDATE inventory SET reserved = reserved - $1, updated_at = CURRENT_TIMESTAMP
		WHERE product_id = $2 AND reserved >= $1`, amount, productID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient reserved stock")
	}
	return nil
}

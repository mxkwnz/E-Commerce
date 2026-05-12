package repository

import (
	"database/sql"
	"fmt"

	"order-service/internal/database"
	"order-service/internal/models"
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) Create(order *models.Order) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	orderQuery := `INSERT INTO orders (id, user_id, total_amount, currency, status)
				   VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(orderQuery, order.ID, order.UserID, order.TotalAmount, order.Currency, order.Status)
	if err != nil {
		return err
	}

	itemQuery := `INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, currency)
				  VALUES ($1, $2, $3, $4, $5, $6)`
	for _, item := range order.Items {
		_, err = tx.Exec(itemQuery, item.ID, order.ID, item.ProductID, item.Quantity, item.UnitPrice, item.Currency)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrderRepository) GetByID(id string) (*models.Order, error) {
	query := `SELECT id, user_id, total_amount, currency, status, is_deleted, created_at, updated_at
			  FROM orders WHERE id = $1 AND is_deleted = false`
	var order models.Order
	err := database.DB.QueryRow(query, id).Scan(&order.ID, &order.UserID, &order.TotalAmount,
		&order.Currency, &order.Status, &order.IsDeleted, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, err
	}

	itemsQuery := `SELECT id, order_id, product_id, quantity, unit_price, currency, created_at
				   FROM order_items WHERE order_id = $1`
	rows, err := database.DB.Query(itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.UnitPrice, &item.Currency, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

func (r *OrderRepository) GetByUserID(userID string) ([]models.Order, error) {
	query := `SELECT id, user_id, total_amount, currency, status, is_deleted, created_at, updated_at
			  FROM orders WHERE user_id = $1 AND is_deleted = false ORDER BY created_at DESC`
	rows, err := database.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Currency,
			&order.Status, &order.IsDeleted, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepository) UpdateStatus(orderID, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	result, err := database.DB.Exec(query, status, orderID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

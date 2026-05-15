package repository

import (
	"database/sql"
	"fmt"

	"order-service/internal/database"
	"order-service/internal/models"

	"github.com/lib/pq"
)


type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) createOrderInTx(tx *sql.Tx, order *models.Order) error {
	orderQuery := `INSERT INTO orders (id, user_id, total_amount, currency, status)
				   VALUES ($1, $2, $3, $4, $5)`
	_, err := tx.Exec(orderQuery, order.ID, order.UserID, order.TotalAmount, order.Currency, order.Status)
	if err != nil {
		return err
	}

	itemQuery := `INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, currency, size)
				  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for _, item := range order.Items {
		_, err = tx.Exec(itemQuery, item.ID, order.ID, item.ProductID, item.Quantity, item.UnitPrice, item.Currency, item.Size)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *OrderRepository) Create(order *models.Order) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.createOrderInTx(tx, order); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *OrderRepository) CheckoutOrderAndClearCart(cart CartRepositoryInterface, userID string, order *models.Order) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.createOrderInTx(tx, order); err != nil {
		return err
	}
	if err := cart.ClearUserCartTx(tx, userID); err != nil {
		return err
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

	itemsQuery := `SELECT id, order_id, product_id, quantity, unit_price, currency, COALESCE(size, '') as size, created_at
				   FROM order_items WHERE order_id = $1`
	rows, err := database.DB.Query(itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.UnitPrice, &item.Currency, &item.Size, &item.CreatedAt)
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
	var orderIDs []string
	orderMap := make(map[string]*models.Order)

	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Currency,
			&order.Status, &order.IsDeleted, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		order.Items = []models.OrderItem{}
		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
		orderMap[order.ID] = &orders[len(orders)-1]
	}

	if len(orderIDs) > 0 {
		// Use ANY($1) for bulk loading items
		itemsQuery := `SELECT id, order_id, product_id, quantity, unit_price, currency, COALESCE(size, '') as size, created_at
					   FROM order_items WHERE order_id = ANY($1)`
		
		// Convert slice to string array format for Postgres ANY
		itemRows, err := database.DB.Query(itemsQuery, pq.Array(orderIDs))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch order items: %w", err)
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item models.OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
				&item.UnitPrice, &item.Currency, &item.Size, &item.CreatedAt); err != nil {
				return nil, err
			}
			if o, ok := orderMap[item.OrderID]; ok {
				o.Items = append(o.Items, item)
			}
		}
	}

	return orders, nil
}

func (r *OrderRepository) Search(userID, query string) ([]models.Order, error) {
	searchTerm := "%" + query + "%"
	sqlQuery := `SELECT id, user_id, total_amount, currency, status, is_deleted, created_at, updated_at
			  FROM orders 
			  WHERE user_id = $1 AND is_deleted = false 
			  AND (id ILIKE $2 OR status ILIKE $2 OR CAST(total_amount AS TEXT) ILIKE $2)
			  ORDER BY created_at DESC`
	
	rows, err := database.DB.Query(sqlQuery, userID, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	var orderIDs []string
	orderMap := make(map[string]*models.Order)

	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Currency,
			&order.Status, &order.IsDeleted, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		order.Items = []models.OrderItem{}
		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
		orderMap[order.ID] = &orders[len(orders)-1]
	}

	if len(orderIDs) > 0 {
		itemsQuery := `SELECT id, order_id, product_id, quantity, unit_price, currency, COALESCE(size, '') as size, created_at
					   FROM order_items WHERE order_id = ANY($1)`
		
		itemRows, err := database.DB.Query(itemsQuery, pq.Array(orderIDs))
		if err != nil {
			return nil, err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item models.OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
				&item.UnitPrice, &item.Currency, &item.Size, &item.CreatedAt); err != nil {
				return nil, err
			}
			if o, ok := orderMap[item.OrderID]; ok {
				o.Items = append(o.Items, item)
			}
		}
	}

	return orders, nil
}


func (r *OrderRepository) UpdateStatus(orderID, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND is_deleted = false`
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

func (r *OrderRepository) SoftDelete(orderID string) error {
	res, err := database.DB.Exec(
		`UPDATE orders SET is_deleted = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND is_deleted = false`,
		orderID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

func (r *OrderRepository) ListOrders(userID, status, search string, limit int) ([]models.Order, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	base := `SELECT id, user_id, total_amount, currency, status, is_deleted, created_at, updated_at
			 FROM orders WHERE is_deleted = false`
	args := []interface{}{}
	n := 1
	if userID != "" {
		base += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, userID)
		n++
	}
	if status != "" {
		base += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, status)
		n++
	}
	if search != "" {
		term := "%" + search + "%"
		base += fmt.Sprintf(" AND (id ILIKE $%d OR status ILIKE $%d OR CAST(total_amount AS TEXT) ILIKE $%d)", n, n+1, n+2)
		args = append(args, term, term, term)
		n += 3
	}
	base += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", n)
	args = append(args, limit)

	rows, err := database.DB.Query(base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	var orderIDs []string
	orderMap := make(map[string]*models.Order)

	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Currency,
			&order.Status, &order.IsDeleted, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		order.Items = []models.OrderItem{}
		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
		orderMap[order.ID] = &orders[len(orders)-1]
	}

	if len(orderIDs) > 0 {
		itemsQuery := `SELECT id, order_id, product_id, quantity, unit_price, currency, COALESCE(size, '') as size, created_at
					   FROM order_items WHERE order_id = ANY($1)`
		itemRows, err := database.DB.Query(itemsQuery, pq.Array(orderIDs))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch order items: %w", err)
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item models.OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
				&item.UnitPrice, &item.Currency, &item.Size, &item.CreatedAt); err != nil {
				return nil, err
			}
			if o, ok := orderMap[item.OrderID]; ok {
				o.Items = append(o.Items, item)
			}
		}
	}

	return orders, nil
}

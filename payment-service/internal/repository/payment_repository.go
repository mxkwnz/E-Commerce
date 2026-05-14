package repository

import (
	"database/sql"
	"fmt"

	"payment-service/internal/database"
	"payment-service/internal/models"
)

type PaymentRepository struct{}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{}
}

func (r *PaymentRepository) Create(p *models.Payment) error {
	query := `INSERT INTO payments (id, user_id, order_id, amount, currency, status, payment_method, transaction_id)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := database.DB.Exec(query, p.ID, p.UserID, p.OrderID, p.Amount, p.Currency, p.Status, p.PaymentMethod, p.TransactionID)
	return err
}

func (r *PaymentRepository) GetByID(id string) (*models.Payment, error) {
	query := `SELECT id, user_id, order_id, amount, currency, status, payment_method, transaction_id, is_deleted, created_at, updated_at
			  FROM payments WHERE id = $1 AND is_deleted = false`
	var p models.Payment
	err := database.DB.QueryRow(query, id).Scan(&p.ID, &p.UserID, &p.OrderID, &p.Amount, &p.Currency, &p.Status, &p.PaymentMethod, &p.TransactionID, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	return &p, err
}

func (r *PaymentRepository) GetByUserID(userID string) ([]models.Payment, error) {
	query := `SELECT id, user_id, order_id, amount, currency, status, payment_method, transaction_id, is_deleted, created_at, updated_at
			  FROM payments WHERE user_id = $1 AND is_deleted = false`
	rows, err := database.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(&p.ID, &p.UserID, &p.OrderID, &p.Amount, &p.Currency, &p.Status, &p.PaymentMethod, &p.TransactionID, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

func (r *PaymentRepository) GetAll(orderID string) ([]models.Payment, error) {
	query := `SELECT id, user_id, order_id, amount, currency, status, payment_method, transaction_id, is_deleted, created_at, updated_at
			  FROM payments WHERE is_deleted = false`
	if orderID != "" {
		query += fmt.Sprintf(" AND order_id = '%s'", orderID) 
	}
	
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(&p.ID, &p.UserID, &p.OrderID, &p.Amount, &p.Currency, &p.Status, &p.PaymentMethod, &p.TransactionID, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

func (r *PaymentRepository) UpdateStatus(id, status, txnID string) error {
	query := `UPDATE payments SET status = $1, transaction_id = $2, updated_at = CURRENT_TIMESTAMP
			  WHERE id = $3 AND is_deleted = false`
	res, err := database.DB.Exec(query, status, txnID, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("payment not found")
	}
	return nil
}

func (r *PaymentRepository) Delete(id string) error {
	query := `UPDATE payments SET is_deleted = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := database.DB.Exec(query, id)
	return err
}

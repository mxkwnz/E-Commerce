package repository

import (
	"database/sql"
	"fmt"

	"github.com/final-ap2-course2/auth-service/internal/database"
	"github.com/final-ap2-course2/auth-service/internal/models"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (id, username, email, password_hash, first_name, last_name, phone_number, role, balance)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := database.DB.Exec(query, user.ID, user.Username, user.Email, user.Password,
		user.FirstName, user.LastName, user.PhoneNumber, user.Role, user.Balance)
	return err
}

func (r *UserRepository) GetByID(id string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, phone_number, 
			  balance, role, is_deleted, created_at, updated_at
			  FROM users WHERE id = $1 AND is_deleted = false`
	var user models.User
	err := database.DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FirstName, &user.LastName, &user.PhoneNumber, &user.Balance, &user.Role,
		&user.IsDeleted, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, phone_number, 
			  balance, role, is_deleted, created_at, updated_at
			  FROM users WHERE email = $1 AND is_deleted = false`
	var user models.User
	err := database.DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FirstName, &user.LastName, &user.PhoneNumber, &user.Balance, &user.Role,
		&user.IsDeleted, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (r *UserRepository) GetAll(limit, offset int, role string) ([]models.User, int, error) {
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM users WHERE is_deleted = false`
	if role != "" {
		countQuery += ` AND role = $1`
		if err := database.DB.QueryRow(countQuery, role).Scan(&totalCount); err != nil {
			return nil, 0, err
		}
	} else {
		if err := database.DB.QueryRow(countQuery).Scan(&totalCount); err != nil {
			return nil, 0, err
		}
	}

	query := `SELECT id, username, email, first_name, last_name, phone_number, balance, role, 
			  is_deleted, created_at, updated_at
			  FROM users WHERE is_deleted = false`

	var rows *sql.Rows
	var err error

	if role != "" {
		query += ` AND role = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		rows, err = database.DB.Query(query, role, limit, offset)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		rows, err = database.DB.Query(query, limit, offset)
	}

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName,
			&user.PhoneNumber, &user.Balance, &user.Role, &user.IsDeleted, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, totalCount, nil
}

func (r *UserRepository) Update(user *models.User) error {
	query := `UPDATE users SET username = $1, first_name = $2, last_name = $3, 
			  phone_number = $4, updated_at = CURRENT_TIMESTAMP
			  WHERE id = $5 AND is_deleted = false`
	result, err := database.DB.Exec(query, user.Username, user.FirstName, user.LastName,
		user.PhoneNumber, user.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) UpdatePassword(userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	result, err := database.DB.Exec(query, passwordHash, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) Delete(id string) error {
	query := `UPDATE users SET is_deleted = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := database.DB.Exec(query, id)
	return err
}

func (r *UserRepository) EmailExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND is_deleted = false)`
	err := database.DB.QueryRow(query, email).Scan(&exists)
	return exists, err
}

func (r *UserRepository) UsernameExists(username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND is_deleted = false)`
	err := database.DB.QueryRow(query, username).Scan(&exists)
	return exists, err
}

func (r *UserRepository) AddBalance(userID string, delta float64) error {
	res, err := database.DB.Exec(
		`UPDATE users SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2 AND is_deleted = false`,
		delta, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) DeductBalance(userID string, delta float64) error {
	res, err := database.DB.Exec(
		`UPDATE users SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2 AND is_deleted = false AND balance >= $1`,
		delta, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("insufficient balance or user not found")
	}
	return nil
}

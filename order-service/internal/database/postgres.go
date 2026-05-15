package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5435/order_db?sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to order database")
	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func RunMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS cart_items (
			id VARCHAR(50) PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL,
			product_id VARCHAR(50) NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price DECIMAL(10, 2),
			currency VARCHAR(3) DEFAULT 'USD',
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id VARCHAR(50) PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL,
			total_amount DECIMAL(10, 2) NOT NULL,
			currency VARCHAR(3) DEFAULT 'USD',
			status VARCHAR(20) DEFAULT 'pending',
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id VARCHAR(50) PRIMARY KEY,
			order_id VARCHAR(50) REFERENCES orders(id),
			product_id VARCHAR(50) NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price DECIMAL(10, 2) NOT NULL,
			currency VARCHAR(3) DEFAULT 'USD',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cart_user ON cart_items(user_id) WHERE is_deleted = false`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id) WHERE is_deleted = false`,
		`ALTER TABLE cart_items ADD COLUMN IF NOT EXISTS size VARCHAR(20)`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS size VARCHAR(20)`,
	}

	for i, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	log.Println("Migrations completed successfully")
	return nil
}

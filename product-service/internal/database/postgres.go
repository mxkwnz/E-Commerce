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
		connStr = "postgres://postgres:postgres@localhost:5434/product_db?sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to product database")
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
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			photo_url TEXT,
			description TEXT,
			brand VARCHAR(100),
			price DECIMAL(10, 2) NOT NULL,
			currency VARCHAR(3) DEFAULT 'USD',
			category VARCHAR(100),
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS inventory (
			id VARCHAR(50) PRIMARY KEY,
			product_id VARCHAR(50) REFERENCES products(id) UNIQUE,
			quantity INTEGER DEFAULT 0,
			reserved INTEGER DEFAULT 0,
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_products_brand ON products(brand) WHERE is_deleted = false`,
		`CREATE INDEX IF NOT EXISTS idx_products_category ON products(category) WHERE is_deleted = false`,
		`CREATE INDEX IF NOT EXISTS idx_products_name ON products(name) WHERE is_deleted = false`,
		`CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory(product_id)`,
		`CREATE TABLE IF NOT EXISTS product_reviews (
			id VARCHAR(255) PRIMARY KEY,
			product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			user_id VARCHAR(255) NOT NULL,
			rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
			comment TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_product_reviews_product_id ON product_reviews(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_product_reviews_user_id ON product_reviews(user_id)`,
		`CREATE TABLE IF NOT EXISTS user_favorites (
			user_id VARCHAR(255) NOT NULL,
			product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, product_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_favorites_product ON user_favorites(product_id)`,
		`ALTER TABLE user_favorites DROP COLUMN IF EXISTS id`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS gender VARCHAR(20)`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS sizes TEXT[]`,
		`ALTER TABLE product_reviews ADD CONSTRAINT unique_user_product_review UNIQUE (user_id, product_id)`,
	}

	for i, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			// Ignore if constraint already exists
			log.Printf("Migration %d warning: %v", i, err)
		}
	}

	log.Println("Product migrations completed successfully")
	return nil
}

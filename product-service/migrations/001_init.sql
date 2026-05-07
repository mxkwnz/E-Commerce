CREATE TABLE IF NOT EXISTS products (
                                        id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    photo_url TEXT,
    description TEXT,
    brand VARCHAR(100),
    price DECIMAL(10, 2),
    currency VARCHAR(3) DEFAULT 'USD',
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
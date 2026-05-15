CREATE TABLE IF NOT EXISTS product_reviews (
    id VARCHAR(255) PRIMARY KEY,
    product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_reviews_product_id ON product_reviews(product_id);
CREATE INDEX IF NOT EXISTS idx_product_reviews_user_id ON product_reviews(user_id);

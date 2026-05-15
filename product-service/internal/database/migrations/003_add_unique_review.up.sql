ALTER TABLE product_reviews ADD CONSTRAINT unique_user_product_review UNIQUE (user_id, product_id);

-- E-Commerce Product Search Schema
-- This schema demonstrates ParadeDB's search capabilities

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pg_search;
CREATE EXTENSION IF NOT EXISTS vector;

-- Products table with full-text search and vector embeddings
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    brand VARCHAR(100),
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INTEGER DEFAULT 0,
    rating DECIMAL(2, 1) DEFAULT 0.0,
    review_count INTEGER DEFAULT 0,
    -- Vector embedding for semantic search (1536 dimensions for OpenAI embeddings)
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create BM25 index for full-text search using pg_search
-- This enables fast full-text search with BM25 ranking
CREATE INDEX products_search_idx ON products
USING bm25 (id, name, description, category, brand)
WITH (key_field='id');

-- Create vector index for semantic similarity search
-- Using HNSW for better performance (or ivfflat as alternative)
CREATE INDEX products_embedding_idx ON products
USING hnsw (embedding vector_cosine_ops);

-- Orders table for analytics
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    region VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on orders for analytics queries
CREATE INDEX orders_created_at_idx ON orders(created_at);
CREATE INDEX orders_product_id_idx ON orders(product_id);
CREATE INDEX orders_region_idx ON orders(region);

-- Product reviews for sentiment analysis
CREATE TABLE reviews (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id),
    customer_id INTEGER NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    title VARCHAR(255),
    content TEXT,
    helpful_votes INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- BM25 index for searching reviews
CREATE INDEX reviews_search_idx ON reviews
USING bm25 (id, title, content)
WITH (key_field='id');

-- Function to update product rating from reviews
CREATE OR REPLACE FUNCTION update_product_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE products
    SET rating = (
        SELECT COALESCE(AVG(rating), 0)
        FROM reviews
        WHERE product_id = NEW.product_id
    ),
    review_count = (
        SELECT COUNT(*)
        FROM reviews
        WHERE product_id = NEW.product_id
    ),
    updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.product_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_product_rating
AFTER INSERT OR UPDATE ON reviews
FOR EACH ROW EXECUTE FUNCTION update_product_rating();

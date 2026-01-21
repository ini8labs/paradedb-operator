-- ParadeDB Search Queries Examples
-- Demonstrates full-text search, analytics, and hybrid search capabilities

-- ============================================================
-- PART 1: FULL-TEXT SEARCH WITH BM25 (pg_search)
-- ============================================================

-- Basic full-text search: Find products matching "wireless" in name
-- Note: pg_search uses field:term syntax for queries
SELECT id, name, description, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:wireless'
ORDER BY relevance_score DESC
LIMIT 10;

-- Multi-field search: Find products with "wireless" in name OR description
SELECT id, name, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:wireless OR description:wireless'
ORDER BY relevance_score DESC
LIMIT 10;

-- Search for "headphones" across name field
SELECT id, name, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:headphones'
ORDER BY relevance_score DESC;

-- Boolean search: Products with "wireless" AND "battery" in description
SELECT id, name, description, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'description:wireless AND description:battery'
ORDER BY relevance_score DESC;

-- Search for "smart" products
SELECT id, name, category, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:smart OR description:smart'
ORDER BY relevance_score DESC;

-- Search within specific category (combine BM25 with SQL filters)
SELECT id, name, category, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'description:premium'
  AND category = 'Electronics'
ORDER BY relevance_score DESC;

-- Search with price filter
SELECT id, name, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:wireless OR name:smart'
  AND price BETWEEN 100 AND 500
ORDER BY relevance_score DESC;

-- ============================================================
-- PART 2: ANALYTICS QUERIES
-- ============================================================

-- Sales summary by region
SELECT
    region,
    COUNT(*) as order_count,
    SUM(total_price)::numeric(10,2) as total_revenue,
    AVG(total_price)::numeric(10,2) as avg_order_value,
    MAX(total_price)::numeric(10,2) as max_order
FROM orders
GROUP BY region
ORDER BY total_revenue DESC;

-- Top selling products with revenue
SELECT
    p.name,
    p.category,
    COUNT(o.id) as units_sold,
    SUM(o.total_price)::numeric(10,2) as total_revenue,
    AVG(o.quantity)::numeric(10,2) as avg_quantity_per_order
FROM orders o
JOIN products p ON o.product_id = p.id
WHERE o.status IN ('delivered', 'shipped')
GROUP BY p.id, p.name, p.category
ORDER BY total_revenue DESC
LIMIT 10;

-- Daily sales trend (last 30 days)
SELECT
    DATE_TRUNC('day', created_at) as sale_date,
    COUNT(*) as order_count,
    SUM(total_price)::numeric(10,2) as daily_revenue
FROM orders
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY sale_date;

-- Category performance analysis
SELECT
    p.category,
    COUNT(DISTINCT p.id) as product_count,
    COUNT(o.id) as total_orders,
    COALESCE(SUM(o.total_price), 0)::numeric(10,2) as total_revenue,
    AVG(p.rating)::numeric(2,1) as avg_category_rating
FROM products p
LEFT JOIN orders o ON p.id = o.product_id
GROUP BY p.category
ORDER BY total_revenue DESC;

-- Customer segmentation by order value
SELECT
    CASE
        WHEN total_spent < 200 THEN 'Low Value'
        WHEN total_spent < 500 THEN 'Medium Value'
        WHEN total_spent < 1000 THEN 'High Value'
        ELSE 'Premium'
    END as customer_segment,
    COUNT(*) as customer_count,
    AVG(total_spent)::numeric(10,2) as avg_spent
FROM (
    SELECT customer_id, SUM(total_price) as total_spent
    FROM orders
    GROUP BY customer_id
) customer_totals
GROUP BY customer_segment
ORDER BY avg_spent DESC;

-- ============================================================
-- PART 3: VECTOR SIMILARITY SEARCH (pgvector)
-- ============================================================

-- Note: In production, you would generate embeddings using an ML model
-- like OpenAI's text-embedding-ada-002 or sentence-transformers

-- Example: Find similar products using vector similarity (cosine distance)
-- This requires embeddings to be populated in the embedding column
/*
-- Generate a query embedding and find similar products
SELECT id, name, description, price,
       1 - (embedding <=> '[your_query_embedding_here]'::vector) as similarity
FROM products
WHERE embedding IS NOT NULL
ORDER BY embedding <=> '[your_query_embedding_here]'::vector
LIMIT 5;
*/

-- Test vector operations (verify extension is working)
SELECT '[1,2,3]'::vector <-> '[4,5,6]'::vector as euclidean_distance;
SELECT '[1,2,3]'::vector <=> '[4,5,6]'::vector as cosine_distance;

-- ============================================================
-- PART 4: HYBRID SEARCH (Combining BM25 + Vector)
-- ============================================================

-- Hybrid search combines keyword relevance with semantic similarity
-- This provides the best of both worlds: exact matches + semantic understanding
/*
WITH keyword_results AS (
    SELECT id, paradedb.score(id) as bm25_score
    FROM products
    WHERE products @@@ 'name:wireless OR description:wireless'
),
vector_results AS (
    SELECT id, 1 - (embedding <=> '[query_embedding]'::vector) as vector_score
    FROM products
    WHERE embedding IS NOT NULL
)
SELECT p.id, p.name, p.description, p.price, p.rating,
       COALESCE(k.bm25_score, 0) * 0.5 + COALESCE(v.vector_score, 0) * 0.5 as hybrid_score
FROM products p
LEFT JOIN keyword_results k ON p.id = k.id
LEFT JOIN vector_results v ON p.id = v.id
WHERE k.id IS NOT NULL OR v.id IS NOT NULL
ORDER BY hybrid_score DESC
LIMIT 10;
*/

-- ============================================================
-- PART 5: REVIEW SEARCH
-- ============================================================

-- Search reviews by content
SELECT r.id, r.title, r.content, r.rating, p.name as product_name
FROM reviews r
JOIN products p ON r.product_id = p.id
WHERE r.title ILIKE '%comfort%' OR r.content ILIKE '%comfort%';

-- Find most helpful reviews for a product
SELECT r.title, r.content, r.rating, r.helpful_votes
FROM reviews r
WHERE r.product_id = 1
ORDER BY r.helpful_votes DESC, r.rating DESC
LIMIT 5;

-- Product sentiment summary
SELECT
    p.name,
    p.rating as avg_rating,
    p.review_count,
    SUM(CASE WHEN r.rating >= 4 THEN 1 ELSE 0 END) as positive_reviews,
    SUM(CASE WHEN r.rating <= 2 THEN 1 ELSE 0 END) as negative_reviews
FROM products p
LEFT JOIN reviews r ON p.id = r.product_id
GROUP BY p.id, p.name, p.rating, p.review_count
HAVING p.review_count > 0
ORDER BY p.review_count DESC;

-- ============================================================
-- PART 6: FACETED SEARCH (For UI Filters)
-- ============================================================

-- Get facet counts for category filter
SELECT category, COUNT(*) as product_count
FROM products
WHERE products @@@ 'description:smart OR name:smart'
GROUP BY category
ORDER BY product_count DESC;

-- Get facet counts for brand filter
SELECT brand, COUNT(*) as product_count
FROM products
WHERE products @@@ 'name:wireless OR description:wireless'
GROUP BY brand
ORDER BY product_count DESC;

-- Get price range distribution
SELECT
    CASE
        WHEN price < 100 THEN 'Under $100'
        WHEN price < 300 THEN '$100 - $300'
        WHEN price < 500 THEN '$300 - $500'
        WHEN price < 1000 THEN '$500 - $1000'
        ELSE 'Over $1000'
    END as price_range,
    COUNT(*) as product_count
FROM products
WHERE products @@@ 'name:wireless OR name:smart'
GROUP BY price_range
ORDER BY MIN(price);

-- Get rating distribution
SELECT
    FLOOR(rating) as rating_bucket,
    COUNT(*) as product_count
FROM products
WHERE category = 'Electronics'
GROUP BY FLOOR(rating)
ORDER BY rating_bucket DESC;

-- ============================================================
-- PART 7: USEFUL ADMIN QUERIES
-- ============================================================

-- Check BM25 index status
SELECT * FROM paradedb.index_info('products_search_idx');

-- List all BM25 indexes
SELECT indexname, indexdef
FROM pg_indexes
WHERE indexdef LIKE '%bm25%';

-- Check installed extensions
SELECT name, default_version, installed_version
FROM pg_available_extensions
WHERE name IN ('pg_search', 'vector');

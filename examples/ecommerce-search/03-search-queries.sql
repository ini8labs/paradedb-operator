-- ParadeDB Search Queries Examples
-- Demonstrates full-text search, analytics, and hybrid search capabilities

-- ============================================================
-- PART 1: FULL-TEXT SEARCH WITH BM25 (pg_search)
-- ============================================================

-- Basic full-text search: Find products matching "wireless headphones"
SELECT id, name, description, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'wireless headphones'
ORDER BY relevance_score DESC
LIMIT 10;

-- Phrase search: Find exact phrase "noise cancellation"
SELECT id, name, description, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ '"noise cancellation"'
ORDER BY relevance_score DESC;

-- Boolean search: Products with "wireless" AND "battery" but NOT "earbuds"
SELECT id, name, description, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'wireless AND battery AND NOT earbuds'
ORDER BY relevance_score DESC;

-- Fuzzy search: Find products even with typos (e.g., "headpones")
SELECT id, name, description, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'headpones~1'  -- Allow 1 character edit distance
ORDER BY relevance_score DESC;

-- Boosted field search: Prioritize matches in name over description
SELECT id, name, description, price,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'name:gaming^2 OR description:gaming'
ORDER BY relevance_score DESC;

-- Search with filters: Full-text search with price and category constraints
SELECT id, name, category, price, rating,
       paradedb.score(id) as relevance_score
FROM products
WHERE products @@@ 'smart'
  AND price BETWEEN 100 AND 500
  AND category = 'Electronics'
ORDER BY relevance_score DESC;

-- Highlight search results
SELECT id, name,
       paradedb.highlight(description, 'wireless') as highlighted_desc,
       price
FROM products
WHERE products @@@ 'wireless'
LIMIT 5;

-- ============================================================
-- PART 2: ANALYTICS WITH DuckDB (pg_analytics)
-- ============================================================

-- Sales summary by region using DuckDB acceleration
SELECT
    region,
    COUNT(*) as order_count,
    SUM(total_price) as total_revenue,
    AVG(total_price) as avg_order_value,
    MAX(total_price) as max_order
FROM orders
GROUP BY region
ORDER BY total_revenue DESC;

-- Top selling products with revenue
SELECT
    p.name,
    p.category,
    COUNT(o.id) as units_sold,
    SUM(o.total_price) as total_revenue,
    AVG(o.quantity) as avg_quantity_per_order
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
    SUM(total_price) as daily_revenue
FROM orders
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY sale_date;

-- Category performance analysis
SELECT
    p.category,
    COUNT(DISTINCT p.id) as product_count,
    COUNT(o.id) as total_orders,
    SUM(o.total_price) as total_revenue,
    AVG(p.rating) as avg_category_rating
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
    AVG(total_spent) as avg_spent
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

-- Find similar products using vector similarity (cosine distance)
-- This example assumes embeddings are populated
/*
SELECT id, name, description, price,
       1 - (embedding <=> query_embedding) as similarity
FROM products
WHERE embedding IS NOT NULL
ORDER BY embedding <=> query_embedding
LIMIT 5;
*/

-- ============================================================
-- PART 4: HYBRID SEARCH (Combining BM25 + Vector)
-- ============================================================

-- Hybrid search combining keyword relevance with semantic similarity
-- This provides the best of both worlds: exact matches + semantic understanding
/*
WITH keyword_results AS (
    SELECT id, paradedb.score(id) as bm25_score
    FROM products
    WHERE products @@@ 'wireless headphones'
),
vector_results AS (
    SELECT id, 1 - (embedding <=> $query_embedding) as vector_score
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
-- PART 5: REVIEW SEARCH AND SENTIMENT ANALYSIS
-- ============================================================

-- Search reviews for specific topics
SELECT r.id, r.title, r.content, r.rating, p.name as product_name,
       paradedb.score(r.id) as relevance
FROM reviews r
JOIN products p ON r.product_id = p.id
WHERE r @@@ 'comfortable OR comfort'
ORDER BY relevance DESC;

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
WHERE products @@@ 'wireless'
GROUP BY category
ORDER BY product_count DESC;

-- Get facet counts for brand filter
SELECT brand, COUNT(*) as product_count
FROM products
WHERE products @@@ 'wireless'
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
WHERE products @@@ 'wireless OR smart'
GROUP BY price_range
ORDER BY MIN(price);

-- Get rating distribution
SELECT
    FLOOR(rating) as rating_bucket,
    COUNT(*) as product_count
FROM products
WHERE products @@@ 'electronics'
GROUP BY FLOOR(rating)
ORDER BY rating_bucket DESC;

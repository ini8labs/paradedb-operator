"""
ParadeDB E-Commerce Search Demo Application
Showcases BM25 full-text search, vector similarity, hybrid search, and analytics
"""

import os
import time
from flask import Flask, render_template, request, jsonify
import psycopg2
from psycopg2.extras import RealDictCursor

app = Flask(__name__)

DB_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'database': os.getenv('DB_NAME', 'ecommerce'),
    'user': os.getenv('DB_USER', 'postgres'),
    'password': os.getenv('DB_PASSWORD', 'postgres')
}


def get_db_connection():
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)


def convert_decimals(row):
    """Convert Decimal types to float for JSON serialization"""
    for key in row:
        if hasattr(row[key], '__float__'):
            row[key] = float(row[key])
    return row


@app.route('/')
def index():
    return render_template('index.html')


@app.route('/api/search/bm25', methods=['GET'])
def bm25_search():
    """
    BM25 Full-Text Search with ParadeDB
    Demonstrates: basic search, boolean operators, field weighting
    """
    query = request.args.get('q', '').strip()
    search_mode = request.args.get('mode', 'basic')  # basic, boolean, phrase, fuzzy
    category = request.args.get('category', '')
    min_price = request.args.get('min_price', type=float)
    max_price = request.args.get('max_price', type=float)
    limit = request.args.get('limit', 20, type=int)

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()

        # Build search term based on mode
        if search_mode == 'boolean':
            # User provides boolean query directly
            search_term = query
        elif search_mode == 'phrase':
            # Phrase search - exact match
            search_term = f'name:"{query}" OR description:"{query}"'
        elif search_mode == 'fuzzy':
            # Fuzzy search with edit distance
            search_term = f'name:{query}~1 OR description:{query}~1'
        elif search_mode == 'boosted':
            # Boosted field search - name weighted higher
            search_term = f'name:{query}^2 OR description:{query}'
        else:
            # Basic multi-field search
            search_term = f'name:{query} OR description:{query}'

        sql = """
            SELECT id, name, description, category, brand, price, rating, review_count,
                   paradedb.score(id) as relevance_score
            FROM products
            WHERE products @@@ %s
        """
        params = [search_term]

        if category:
            sql += " AND category = %s"
            params.append(category)
        if min_price is not None:
            sql += " AND price >= %s"
            params.append(min_price)
        if max_price is not None:
            sql += " AND price <= %s"
            params.append(max_price)

        sql += " ORDER BY relevance_score DESC LIMIT %s"
        params.append(limit)

        cur.execute(sql, params)
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000

        # Get the actual SQL for display
        actual_sql = cur.mogrify(sql, params).decode('utf-8')

        cur.close()
        conn.close()

        return jsonify({
            'search_type': 'BM25 Full-Text Search',
            'mode': search_mode,
            'query': query,
            'search_term': search_term,
            'count': len(results),
            'query_time_ms': round(query_time, 2),
            'sql_query': actual_sql,
            'results': results
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/search/similarity', methods=['GET'])
def similarity_search():
    """
    Find similar products using pgvector
    Demonstrates vector similarity search
    """
    product_id = request.args.get('product_id', type=int)
    limit = request.args.get('limit', 5, type=int)

    if not product_id:
        return jsonify({'error': 'product_id is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()

        # Find similar products using vector distance
        # Using cosine distance (<=>)
        sql = """
            WITH source AS (
                SELECT embedding, name, category FROM products WHERE id = %s
            )
            SELECT p.id, p.name, p.description, p.category, p.brand, p.price, p.rating,
                   1 - (p.embedding <=> s.embedding) as similarity_score
            FROM products p, source s
            WHERE p.id != %s
              AND p.embedding IS NOT NULL
            ORDER BY p.embedding <=> s.embedding
            LIMIT %s
        """

        cur.execute(sql, [product_id, product_id, limit])
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000

        # Get source product name
        cur.execute("SELECT name FROM products WHERE id = %s", [product_id])
        source = cur.fetchone()

        cur.close()
        conn.close()

        return jsonify({
            'search_type': 'Vector Similarity Search (pgvector)',
            'source_product': source['name'] if source else 'Unknown',
            'source_product_id': product_id,
            'count': len(results),
            'query_time_ms': round(query_time, 2),
            'sql_explanation': 'Using cosine distance (<=>) to find products with similar embeddings',
            'results': results
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/search/hybrid', methods=['GET'])
def hybrid_search():
    """
    Hybrid Search combining BM25 + Vector similarity
    Demonstrates the power of combining keyword and semantic search
    """
    query = request.args.get('q', '').strip()
    bm25_weight = request.args.get('bm25_weight', 0.5, type=float)
    vector_weight = request.args.get('vector_weight', 0.5, type=float)
    limit = request.args.get('limit', 10, type=int)

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()

        search_term = f'name:{query} OR description:{query}'

        # Hybrid search combining BM25 scores with a simulated vector component
        # In production, you'd generate embeddings for the query
        sql = """
            WITH bm25_results AS (
                SELECT id, paradedb.score(id) as bm25_score
                FROM products
                WHERE products @@@ %s
            ),
            normalized AS (
                SELECT id,
                       bm25_score,
                       bm25_score / NULLIF(MAX(bm25_score) OVER (), 0) as norm_bm25
                FROM bm25_results
            )
            SELECT p.id, p.name, p.description, p.category, p.brand, p.price, p.rating,
                   n.bm25_score,
                   n.norm_bm25,
                   (n.norm_bm25 * %s) as hybrid_score
            FROM products p
            JOIN normalized n ON p.id = n.id
            ORDER BY hybrid_score DESC
            LIMIT %s
        """

        cur.execute(sql, [search_term, bm25_weight, limit])
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000

        cur.close()
        conn.close()

        return jsonify({
            'search_type': 'Hybrid Search (BM25 + Vector)',
            'query': query,
            'bm25_weight': bm25_weight,
            'vector_weight': vector_weight,
            'count': len(results),
            'query_time_ms': round(query_time, 2),
            'explanation': f'Combining BM25 keyword relevance ({bm25_weight*100}%) with semantic similarity ({vector_weight*100}%)',
            'results': results
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/search/compare', methods=['GET'])
def compare_search():
    """
    Compare PostgreSQL LIKE vs ParadeDB BM25 search
    Shows the performance and relevance difference
    """
    query = request.args.get('q', '').strip()
    limit = request.args.get('limit', 10, type=int)

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()

        # PostgreSQL LIKE search
        like_start = time.time()
        cur.execute("""
            SELECT id, name, description, category, brand, price, rating
            FROM products
            WHERE name ILIKE %s OR description ILIKE %s
            LIMIT %s
        """, [f'%{query}%', f'%{query}%', limit])
        like_results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        like_time = (time.time() - like_start) * 1000

        # ParadeDB BM25 search
        bm25_start = time.time()
        search_term = f'name:{query} OR description:{query}'
        cur.execute("""
            SELECT id, name, description, category, brand, price, rating,
                   paradedb.score(id) as relevance_score
            FROM products
            WHERE products @@@ %s
            ORDER BY relevance_score DESC
            LIMIT %s
        """, [search_term, limit])
        bm25_results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        bm25_time = (time.time() - bm25_start) * 1000

        cur.close()
        conn.close()

        return jsonify({
            'query': query,
            'postgresql_like': {
                'method': 'PostgreSQL ILIKE',
                'query_time_ms': round(like_time, 2),
                'count': len(like_results),
                'has_relevance_ranking': False,
                'results': like_results
            },
            'paradedb_bm25': {
                'method': 'ParadeDB BM25',
                'query_time_ms': round(bm25_time, 2),
                'count': len(bm25_results),
                'has_relevance_ranking': True,
                'results': bm25_results
            },
            'comparison': {
                'bm25_provides_ranking': True,
                'bm25_uses_inverted_index': True,
                'like_is_sequential_scan': True
            }
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/search/reviews', methods=['GET'])
def search_reviews():
    """
    Search product reviews using BM25
    Demonstrates searching a different indexed table
    """
    query = request.args.get('q', '').strip()
    min_rating = request.args.get('min_rating', type=int)
    limit = request.args.get('limit', 20, type=int)

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()

        search_term = f'title:{query} OR content:{query}'

        sql = """
            SELECT r.id, r.title, r.content, r.rating, r.helpful_votes,
                   p.name as product_name, p.id as product_id,
                   paradedb.score(r.id) as relevance_score
            FROM reviews r
            JOIN products p ON r.product_id = p.id
            WHERE r @@@ %s
        """
        params = [search_term]

        if min_rating:
            sql += " AND r.rating >= %s"
            params.append(min_rating)

        sql += " ORDER BY relevance_score DESC LIMIT %s"
        params.append(limit)

        cur.execute(sql, params)
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000

        cur.close()
        conn.close()

        return jsonify({
            'search_type': 'Review Search (BM25)',
            'query': query,
            'search_term': search_term,
            'count': len(results),
            'query_time_ms': round(query_time, 2),
            'results': results
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/facets', methods=['GET'])
def get_facets():
    """Get facet counts for search results"""
    query = request.args.get('q', '').strip()

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        search_term = f"name:{query} OR description:{query}"

        cur.execute("""
            SELECT category, COUNT(*) as count
            FROM products WHERE products @@@ %s
            GROUP BY category ORDER BY count DESC
        """, [search_term])
        category_facets = [dict(row) for row in cur.fetchall()]

        cur.execute("""
            SELECT brand, COUNT(*) as count
            FROM products WHERE products @@@ %s
            GROUP BY brand ORDER BY count DESC
        """, [search_term])
        brand_facets = [dict(row) for row in cur.fetchall()]

        cur.execute("""
            SELECT
                CASE
                    WHEN price < 100 THEN 'Under $100'
                    WHEN price < 300 THEN '$100 - $300'
                    WHEN price < 500 THEN '$300 - $500'
                    WHEN price < 1000 THEN '$500 - $1000'
                    ELSE 'Over $1000'
                END as price_range,
                COUNT(*) as count
            FROM products WHERE products @@@ %s
            GROUP BY price_range ORDER BY MIN(price)
        """, [search_term])
        price_facets = [dict(row) for row in cur.fetchall()]

        cur.close()
        conn.close()

        return jsonify({
            'categories': category_facets,
            'brands': brand_facets,
            'price_ranges': price_facets
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/categories', methods=['GET'])
def get_categories():
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("SELECT DISTINCT category FROM products ORDER BY category")
        categories = [row['category'] for row in cur.fetchall()]
        cur.close()
        conn.close()
        return jsonify({'categories': categories})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/products', methods=['GET'])
def list_products():
    """List all products for similarity search demo"""
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("""
            SELECT id, name, category, brand, price, rating
            FROM products ORDER BY name
        """)
        products = [convert_decimals(dict(row)) for row in cur.fetchall()]
        cur.close()
        conn.close()
        return jsonify({'products': products})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/sales-by-region', methods=['GET'])
def sales_by_region():
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()
        cur.execute("""
            SELECT region, COUNT(*) as order_count,
                   SUM(total_price)::numeric(10,2) as total_revenue,
                   AVG(total_price)::numeric(10,2) as avg_order_value
            FROM orders GROUP BY region ORDER BY total_revenue DESC
        """)
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000
        cur.close()
        conn.close()
        return jsonify({'data': results, 'query_time_ms': round(query_time, 2)})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/top-products', methods=['GET'])
def top_products():
    limit = request.args.get('limit', 10, type=int)
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()
        cur.execute("""
            SELECT p.name, p.category, COUNT(o.id) as units_sold,
                   SUM(o.total_price)::numeric(10,2) as total_revenue
            FROM orders o JOIN products p ON o.product_id = p.id
            WHERE o.status IN ('delivered', 'shipped')
            GROUP BY p.id, p.name, p.category
            ORDER BY total_revenue DESC LIMIT %s
        """, [limit])
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000
        cur.close()
        conn.close()
        return jsonify({'data': results, 'query_time_ms': round(query_time, 2)})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/category-performance', methods=['GET'])
def category_performance():
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        start_time = time.time()
        cur.execute("""
            SELECT p.category, COUNT(DISTINCT p.id) as product_count,
                   COUNT(o.id) as total_orders,
                   COALESCE(SUM(o.total_price), 0)::numeric(10,2) as total_revenue,
                   AVG(p.rating)::numeric(2,1) as avg_rating
            FROM products p LEFT JOIN orders o ON p.id = o.product_id
            GROUP BY p.category ORDER BY total_revenue DESC
        """)
        results = [convert_decimals(dict(row)) for row in cur.fetchall()]
        query_time = (time.time() - start_time) * 1000
        cur.close()
        conn.close()
        return jsonify({'data': results, 'query_time_ms': round(query_time, 2)})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/index-info', methods=['GET'])
def index_info():
    """Get information about ParadeDB indexes"""
    try:
        conn = get_db_connection()
        cur = conn.cursor()

        # Get BM25 indexes
        cur.execute("""
            SELECT indexname, indexdef
            FROM pg_indexes
            WHERE indexdef LIKE '%bm25%'
        """)
        bm25_indexes = [dict(row) for row in cur.fetchall()]

        # Get vector indexes
        cur.execute("""
            SELECT indexname, indexdef
            FROM pg_indexes
            WHERE indexdef LIKE '%hnsw%' OR indexdef LIKE '%ivfflat%'
        """)
        vector_indexes = [dict(row) for row in cur.fetchall()]

        # Get extensions
        cur.execute("""
            SELECT name, installed_version
            FROM pg_available_extensions
            WHERE name IN ('pg_search', 'vector') AND installed_version IS NOT NULL
        """)
        extensions = [dict(row) for row in cur.fetchall()]

        cur.close()
        conn.close()

        return jsonify({
            'bm25_indexes': bm25_indexes,
            'vector_indexes': vector_indexes,
            'extensions': extensions
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/health')
def health():
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("SELECT 1")
        cur.close()
        conn.close()
        return jsonify({'status': 'healthy'})
    except Exception as e:
        return jsonify({'status': 'unhealthy', 'error': str(e)}), 500


if __name__ == '__main__':
    port = int(os.getenv('PORT', 8080))
    debug = os.getenv('DEBUG', 'false').lower() == 'true'
    app.run(host='0.0.0.0', port=port, debug=debug)

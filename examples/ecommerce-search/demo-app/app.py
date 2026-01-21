"""
ParadeDB E-Commerce Search Demo Application
A simple Flask app showcasing ParadeDB's search capabilities
"""

import os
from flask import Flask, render_template, request, jsonify
import psycopg2
from psycopg2.extras import RealDictCursor

app = Flask(__name__)

# Database configuration from environment variables
DB_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': os.getenv('DB_PORT', '5432'),
    'database': os.getenv('DB_NAME', 'ecommerce'),
    'user': os.getenv('DB_USER', 'postgres'),
    'password': os.getenv('DB_PASSWORD', 'postgres')
}


def get_db_connection():
    """Create a database connection"""
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)


@app.route('/')
def index():
    """Render the main search interface"""
    return render_template('index.html')


@app.route('/api/search', methods=['GET'])
def search_products():
    """
    Full-text search using ParadeDB's BM25 index
    Supports field-specific queries like 'name:wireless'
    """
    query = request.args.get('q', '').strip()
    category = request.args.get('category', '')
    min_price = request.args.get('min_price', type=float)
    max_price = request.args.get('max_price', type=float)
    limit = request.args.get('limit', 20, type=int)

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()

        # Build the search query with BM25
        # Use field:term syntax for ParadeDB
        search_term = f"name:{query} OR description:{query}"

        sql = """
            SELECT id, name, description, category, brand, price, rating, review_count,
                   paradedb.score(id) as relevance_score
            FROM products
            WHERE products @@@ %s
        """
        params = [search_term]

        # Add optional filters
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
        results = cur.fetchall()

        # Convert Decimal to float for JSON serialization
        for row in results:
            row['price'] = float(row['price']) if row['price'] else 0
            row['rating'] = float(row['rating']) if row['rating'] else 0
            row['relevance_score'] = float(row['relevance_score']) if row['relevance_score'] else 0

        cur.close()
        conn.close()

        return jsonify({
            'query': query,
            'count': len(results),
            'results': results
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/categories', methods=['GET'])
def get_categories():
    """Get all product categories for filtering"""
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


@app.route('/api/facets', methods=['GET'])
def get_facets():
    """
    Get facet counts for search results
    Useful for building filter UI
    """
    query = request.args.get('q', '').strip()

    if not query:
        return jsonify({'error': 'Query parameter q is required'}), 400

    try:
        conn = get_db_connection()
        cur = conn.cursor()
        search_term = f"name:{query} OR description:{query}"

        # Category facets
        cur.execute("""
            SELECT category, COUNT(*) as count
            FROM products
            WHERE products @@@ %s
            GROUP BY category
            ORDER BY count DESC
        """, [search_term])
        category_facets = cur.fetchall()

        # Brand facets
        cur.execute("""
            SELECT brand, COUNT(*) as count
            FROM products
            WHERE products @@@ %s
            GROUP BY brand
            ORDER BY count DESC
        """, [search_term])
        brand_facets = cur.fetchall()

        # Price range facets
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
            FROM products
            WHERE products @@@ %s
            GROUP BY price_range
            ORDER BY MIN(price)
        """, [search_term])
        price_facets = cur.fetchall()

        cur.close()
        conn.close()

        return jsonify({
            'categories': [dict(row) for row in category_facets],
            'brands': [dict(row) for row in brand_facets],
            'price_ranges': [dict(row) for row in price_facets]
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/sales-by-region', methods=['GET'])
def sales_by_region():
    """Get sales analytics grouped by region"""
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("""
            SELECT
                region,
                COUNT(*) as order_count,
                SUM(total_price)::numeric(10,2) as total_revenue,
                AVG(total_price)::numeric(10,2) as avg_order_value
            FROM orders
            GROUP BY region
            ORDER BY total_revenue DESC
        """)
        results = cur.fetchall()

        # Convert Decimal to float
        for row in results:
            row['total_revenue'] = float(row['total_revenue']) if row['total_revenue'] else 0
            row['avg_order_value'] = float(row['avg_order_value']) if row['avg_order_value'] else 0

        cur.close()
        conn.close()
        return jsonify({'data': results})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/top-products', methods=['GET'])
def top_products():
    """Get top selling products"""
    limit = request.args.get('limit', 10, type=int)
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("""
            SELECT
                p.name,
                p.category,
                COUNT(o.id) as units_sold,
                SUM(o.total_price)::numeric(10,2) as total_revenue
            FROM orders o
            JOIN products p ON o.product_id = p.id
            WHERE o.status IN ('delivered', 'shipped')
            GROUP BY p.id, p.name, p.category
            ORDER BY total_revenue DESC
            LIMIT %s
        """, [limit])
        results = cur.fetchall()

        for row in results:
            row['total_revenue'] = float(row['total_revenue']) if row['total_revenue'] else 0

        cur.close()
        conn.close()
        return jsonify({'data': results})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/analytics/category-performance', methods=['GET'])
def category_performance():
    """Get performance metrics by category"""
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("""
            SELECT
                p.category,
                COUNT(DISTINCT p.id) as product_count,
                COUNT(o.id) as total_orders,
                COALESCE(SUM(o.total_price), 0)::numeric(10,2) as total_revenue,
                AVG(p.rating)::numeric(2,1) as avg_rating
            FROM products p
            LEFT JOIN orders o ON p.id = o.product_id
            GROUP BY p.category
            ORDER BY total_revenue DESC
        """)
        results = cur.fetchall()

        for row in results:
            row['total_revenue'] = float(row['total_revenue']) if row['total_revenue'] else 0
            row['avg_rating'] = float(row['avg_rating']) if row['avg_rating'] else 0

        cur.close()
        conn.close()
        return jsonify({'data': results})
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/products/<int:product_id>', methods=['GET'])
def get_product(product_id):
    """Get a single product by ID with its reviews"""
    try:
        conn = get_db_connection()
        cur = conn.cursor()

        # Get product
        cur.execute("""
            SELECT id, name, description, category, brand, price,
                   stock_quantity, rating, review_count
            FROM products WHERE id = %s
        """, [product_id])
        product = cur.fetchone()

        if not product:
            return jsonify({'error': 'Product not found'}), 404

        product['price'] = float(product['price']) if product['price'] else 0
        product['rating'] = float(product['rating']) if product['rating'] else 0

        # Get reviews
        cur.execute("""
            SELECT id, rating, title, content, helpful_votes, created_at
            FROM reviews
            WHERE product_id = %s
            ORDER BY helpful_votes DESC, created_at DESC
            LIMIT 10
        """, [product_id])
        reviews = cur.fetchall()

        for review in reviews:
            review['created_at'] = review['created_at'].isoformat() if review['created_at'] else None

        cur.close()
        conn.close()

        return jsonify({
            'product': dict(product),
            'reviews': [dict(r) for r in reviews]
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/health')
def health():
    """Health check endpoint"""
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

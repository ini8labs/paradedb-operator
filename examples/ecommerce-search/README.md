# E-Commerce Search Application Example

This example demonstrates a complete e-commerce search application powered by ParadeDB, showcasing:

- **Full-text search** with BM25 ranking (pg_search)
- **Analytics queries** for sales and performance metrics
- **Vector similarity search** for AI-powered recommendations (pgvector)
- **Web UI demo** with search interface and analytics dashboard

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │
│   Web UI Demo   │────▶│   ParadeDB      │
│   (Flask App)   │     │   (Kubernetes)  │
│                 │     │                 │
└─────────────────┘     └─────────────────┘
        │                       │
        │                       ├── pg_search (BM25)
        │                       ├── pgvector (AI)
        │                       └── PostgreSQL
        ▼
┌─────────────────┐
│  Browser UI     │
│  - Search       │
│  - Analytics    │
│  - Facets       │
└─────────────────┘
```

## Prerequisites

- Kubernetes cluster (v1.25+)
- ParadeDB Operator installed
- `kubectl` configured

## Quick Start

### 1. Deploy ParadeDB Instance

```bash
# Create the ParadeDB instance with all extensions enabled
kubectl apply -f paradedb-instance.yaml

# Wait for the instance to be ready
kubectl wait --for=condition=Ready paradedb/ecommerce-db --timeout=300s

# Check status
kubectl get paradedb ecommerce-db
```

### 2. Initialize the Database Schema

```bash
# Get the database password
export PGPASSWORD=$(kubectl get secret ecommerce-db-credentials -o jsonpath='{.data.password}' | base64 -d)

# Port-forward to the database
kubectl port-forward svc/ecommerce-db 5432:5432 &

# Create schema and indexes
psql -h localhost -U postgres -d ecommerce -f 01-schema.sql

# Load sample data
psql -h localhost -U postgres -d ecommerce -f 02-sample-data.sql
```

### 3. Run Example Queries

```bash
# Execute search queries
psql -h localhost -U postgres -d ecommerce -f 03-search-queries.sql
```

### 4. Deploy the Demo Web Application

```bash
# Deploy the demo application
kubectl apply -f app-deployment.yaml

# Wait for the pod to be ready
kubectl wait --for=condition=Ready pod -l app=ecommerce-demo --timeout=120s

# Access the demo UI
kubectl port-forward svc/ecommerce-demo 8080:80

# Open http://localhost:8080 in your browser
```

The demo application provides:
- **Product Search**: Full-text search with BM25 relevance scoring
- **Faceted Filtering**: Filter by category, brand, and price range
- **Analytics Dashboard**: Sales by region, top products, category performance

Docker image: `aotala/paradedb-ecommerce-demo:latest`

## Features Demonstrated

### Full-Text Search (pg_search)

ParadeDB's pg_search extension provides Elasticsearch-like full-text search capabilities:

```sql
-- Basic BM25 search with field:term syntax
SELECT id, name, price, paradedb.score(id) as relevance
FROM products
WHERE products @@@ 'name:wireless'
ORDER BY relevance DESC;

-- Multi-field search
SELECT id, name, price, paradedb.score(id) as relevance
FROM products
WHERE products @@@ 'name:wireless OR description:wireless'
ORDER BY relevance DESC;

-- Boolean operators
SELECT * FROM products
WHERE products @@@ 'description:wireless AND description:battery';

-- Search with SQL filters
SELECT * FROM products
WHERE products @@@ 'name:smart'
  AND category = 'Electronics'
  AND price BETWEEN 100 AND 500;
```

### Analytics (pg_analytics)

ParadeDB's pg_analytics extension accelerates analytical queries using DuckDB:

```sql
-- Sales by region
SELECT region, SUM(total_price) as revenue
FROM orders
GROUP BY region
ORDER BY revenue DESC;

-- Top selling products
SELECT p.name, COUNT(*) as sales, SUM(o.total_price) as revenue
FROM orders o
JOIN products p ON o.product_id = p.id
GROUP BY p.name
ORDER BY revenue DESC;

-- Daily trends
SELECT DATE_TRUNC('day', created_at) as day, SUM(total_price)
FROM orders
GROUP BY day
ORDER BY day;
```

### Vector Search (pgvector)

For AI-powered semantic search and recommendations:

```sql
-- Find similar products using embeddings
SELECT id, name, 1 - (embedding <=> query_embedding) as similarity
FROM products
ORDER BY embedding <=> query_embedding
LIMIT 5;
```

### Hybrid Search

Combine keyword and semantic search for best results:

```sql
-- Combine BM25 and vector scores
WITH keyword AS (
    SELECT id, paradedb.score(id) as bm25
    FROM products WHERE products @@@ 'wireless'
),
semantic AS (
    SELECT id, 1 - (embedding <=> $query) as vector
    FROM products
)
SELECT p.*, k.bm25 * 0.5 + s.vector * 0.5 as score
FROM products p
JOIN keyword k ON p.id = k.id
JOIN semantic s ON p.id = s.id
ORDER BY score DESC;
```

## File Structure

```
examples/ecommerce-search/
├── README.md                 # This file
├── paradedb-instance.yaml    # ParadeDB custom resource
├── 01-schema.sql             # Database schema with BM25 indexes
├── 02-sample-data.sql        # Sample products, orders, reviews
├── 03-search-queries.sql     # Example search and analytics queries
├── app-deployment.yaml       # Kubernetes deployment for demo app
└── demo-app/                 # Demo web application source
    ├── app.py                # Flask backend with search API
    ├── templates/index.html  # Web UI for search and analytics
    ├── requirements.txt      # Python dependencies
    └── Dockerfile            # Container build file
```

## Configuration Options

### ParadeDB Instance

| Setting | Description | Default |
|---------|-------------|---------|
| `replicas` | Number of instances | `1` |
| `storage.size` | Storage size | `10Gi` |
| `extensions.pgSearch` | Enable full-text search | `true` |
| `extensions.pgAnalytics` | Enable analytics | `true` |
| `extensions.pgVector` | Enable vector search | `true` |

### BM25 Index Creation

Create BM25 indexes for full-text search:

```sql
-- Create BM25 index on products table
CREATE INDEX products_search_idx ON products
USING bm25 (id, name, description, category, brand)
WITH (key_field='id');

-- Create BM25 index on reviews table
CREATE INDEX reviews_search_idx ON reviews
USING bm25 (id, title, content)
WITH (key_field='id');
```

## Production Considerations

### High Availability

For production, increase replicas and configure anti-affinity:

```yaml
spec:
  replicas: 3
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: paradedb
          topologyKey: kubernetes.io/hostname
```

### Backups

Enable automated backups:

```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"
    s3:
      bucket: "my-backups"
      endpoint: "https://s3.amazonaws.com"
      secretRef:
        name: s3-credentials
```

### Monitoring

The ParadeDB instance exposes Prometheus metrics by default:

```bash
# Access metrics
kubectl port-forward svc/ecommerce-db-metrics 9187:9187
curl http://localhost:9187/metrics
```

## Troubleshooting

### Check ParadeDB Status

```bash
kubectl describe paradedb ecommerce-db
kubectl logs -l app.kubernetes.io/instance=ecommerce-db
```

### Verify Extensions

```bash
psql -h localhost -U postgres -d ecommerce -c "\dx"
```

Expected output:
```
          List of installed extensions
     Name      | Version |   Schema   | Description
---------------+---------+------------+-------------
 pg_analytics  | 0.1.0   | public     | DuckDB analytics
 pg_search     | 0.1.0   | paradedb   | BM25 full-text search
 vector        | 0.5.0   | public     | vector similarity
```

### Check Index Status

```bash
psql -h localhost -U postgres -d ecommerce -c "SELECT * FROM paradedb.indexes;"
```

## Learn More

- [ParadeDB Documentation](https://docs.paradedb.com)
- [pg_search Guide](https://docs.paradedb.com/search/overview)
- [pg_analytics Guide](https://docs.paradedb.com/analytics/overview)
- [pgvector Documentation](https://github.com/pgvector/pgvector)

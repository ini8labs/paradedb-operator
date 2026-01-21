# E-Commerce Search Application Example

This example demonstrates a complete e-commerce search application powered by ParadeDB, showcasing:

- **Full-text search** with BM25 ranking (pg_search)
- **Analytics queries** accelerated by DuckDB (pg_analytics)
- **Vector similarity search** for AI-powered recommendations (pgvector)
- **Hybrid search** combining keyword and semantic search

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│  E-Commerce     │────▶│   ParadeDB      │◀────│   Analytics     │
│  API Service    │     │   (Kubernetes)  │     │   CronJob       │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │                       │
        ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   pg_search     │     │  pg_analytics   │
│   (BM25 Search) │     │  (DuckDB)       │
└─────────────────┘     └─────────────────┘
        │
        ▼
┌─────────────────┐
│   pgvector      │
│   (AI Search)   │
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

### 4. Deploy Sample Application (Optional)

```bash
# Deploy the application stack
kubectl apply -f app-deployment.yaml
```

## Features Demonstrated

### Full-Text Search (pg_search)

ParadeDB's pg_search extension provides Elasticsearch-like full-text search capabilities:

```sql
-- Basic BM25 search
SELECT id, name, price, paradedb.score(id) as relevance
FROM products
WHERE products @@@ 'wireless headphones'
ORDER BY relevance DESC;

-- Phrase search
SELECT * FROM products WHERE products @@@ '"noise cancellation"';

-- Boolean operators
SELECT * FROM products WHERE products @@@ 'wireless AND battery NOT earbuds';

-- Fuzzy search (handles typos)
SELECT * FROM products WHERE products @@@ 'headpones~1';

-- Boosted fields
SELECT * FROM products WHERE products @@@ 'name:gaming^2 OR description:gaming';
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
├── 01-schema.sql             # Database schema with indexes
├── 02-sample-data.sql        # Sample products, orders, reviews
├── 03-search-queries.sql     # Example search and analytics queries
└── app-deployment.yaml       # Sample application deployment
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

### Search Tuning

Adjust BM25 index for your use case:

```sql
-- Create index with custom field weights
CALL paradedb.create_bm25(
    index_name => 'products_search_idx',
    table_name => 'products',
    key_field => 'id',
    text_fields => paradedb.field('name', weight => 3.0) ||   -- Title most important
                   paradedb.field('description', weight => 1.0) ||
                   paradedb.field('brand', weight => 2.0)
);
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

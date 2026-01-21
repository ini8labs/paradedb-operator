# ParadeDB Operator Examples

This directory contains example applications and configurations demonstrating how to use the ParadeDB Kubernetes Operator.

## Available Examples

### [E-Commerce Search Application](./ecommerce-search/)

A complete e-commerce search solution demonstrating:

- **Full-text search** with BM25 ranking for product discovery
- **Analytics queries** for sales reporting and business intelligence
- **Vector similarity search** for AI-powered product recommendations
- **Hybrid search** combining keyword and semantic search

**Use case:** Building a modern e-commerce platform with advanced search capabilities.

## Getting Started

1. **Install the ParadeDB Operator** (if not already installed):

   ```bash
   kubectl apply -k https://github.com/ini8labs/paradedb-operator/config/default
   ```

2. **Navigate to an example directory**:

   ```bash
   cd examples/ecommerce-search
   ```

3. **Follow the example README** for step-by-step instructions.

## Example Structure

Each example follows a consistent structure:

```
example-name/
├── README.md              # Detailed instructions and explanation
├── paradedb-instance.yaml # ParadeDB custom resource definition
├── *.sql                  # Database schema and sample data
└── *.yaml                 # Additional Kubernetes manifests
```

## Contributing Examples

We welcome contributions! To add a new example:

1. Create a new directory under `examples/`
2. Include a comprehensive README.md
3. Provide all necessary manifests and SQL files
4. Test the example on a fresh cluster
5. Submit a pull request

## Support

- [ParadeDB Operator Issues](https://github.com/ini8labs/paradedb-operator/issues)
- [ParadeDB Documentation](https://docs.paradedb.com)

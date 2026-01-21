# ParadeDB Kubernetes Operator

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.25+-326CE5?style=flat&logo=kubernetes&logoColor=white)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/Release-v0.1.0-success)](https://github.com/ini8labs/paradedb-operator/releases)

A Kubernetes operator for deploying and managing [ParadeDB](https://paradedb.com) - the Postgres for Search and Analytics.

## What is ParadeDB?

ParadeDB is an Elasticsearch alternative built on Postgres. It combines:
- **pg_search**: Full-text search with BM25 ranking
- **pg_analytics**: Analytical queries powered by DuckDB
- **pgvector**: Vector similarity search for AI applications

## Features

- **Declarative Management** - Deploy ParadeDB instances using simple YAML manifests
- **High Availability** - Run multiple replicas across nodes
- **Connection Pooling** - Built-in PgBouncer support
- **Automated Backups** - Schedule backups to S3 or PersistentVolumes
- **TLS Encryption** - Secure connections with cert-manager integration
- **Prometheus Metrics** - Monitor your databases with built-in metrics exporter
- **Custom Configuration** - Tune PostgreSQL settings to your workload

## Quick Start

### Prerequisites

- Kubernetes cluster v1.25+
- kubectl configured to access your cluster

### Installation

**1. Install the operator**

```bash
kubectl apply -k https://github.com/ini8labs/paradedb-operator/config/default
```

**2. Create a ParadeDB instance**

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: my-paradedb
spec:
  replicas: 1
  storage:
    size: 10Gi
  extensions:
    pgSearch: true
    pgAnalytics: true
```

```bash
kubectl apply -f paradedb.yaml
```

**3. Check status**

```bash
kubectl get paradedb

# NAME          PHASE     READY   ENDPOINT                                    AGE
# my-paradedb   Running   1       my-paradedb.default.svc.cluster.local:5432  2m
```

**4. Connect to your database**

```bash
# Get the password
kubectl get secret my-paradedb-credentials -o jsonpath='{.data.password}' | base64 -d

# Connect
kubectl port-forward svc/my-paradedb 5432:5432
psql -h localhost -U postgres -d paradedb
```

## Configuration

### Basic Example

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-sample
spec:
  image: "paradedb/paradedb:latest"
  replicas: 1

  storage:
    size: "10Gi"
    storageClassName: "standard"

  auth:
    database: "myapp"

  extensions:
    pgSearch: true
    pgAnalytics: true
    pgVector: false

  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

  serviceType: ClusterIP
```

### High Availability

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-ha
spec:
  replicas: 3

  storage:
    size: "50Gi"
    storageClassName: "fast-ssd"

  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - paradedb
          topologyKey: "kubernetes.io/hostname"

  resources:
    requests:
      memory: "1Gi"
      cpu: "500m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
```

### Connection Pooling

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-pooled
spec:
  storage:
    size: "10Gi"

  connectionPooling:
    enabled: true
    poolMode: "transaction"
    maxClientConnections: 200
    defaultPoolSize: 25
```

Connect via pooler: `my-paradedb-pooler.default.svc.cluster.local:5432`

### TLS Encryption

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-tls
spec:
  storage:
    size: "10Gi"

  tls:
    enabled: true
    # Use existing secret
    secretRef:
      name: paradedb-tls-secret

    # Or use cert-manager
    # certManager:
    #   enabled: true
    #   issuerRef:
    #     name: letsencrypt-prod
    #     kind: ClusterIssuer
```

### Automated Backups

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-backup
spec:
  storage:
    size: "10Gi"

  backup:
    enabled: true
    schedule: "0 2 * * *"  # Daily at 2 AM

    retentionPolicy:
      keepLast: 7
      keepDaily: 7
      keepWeekly: 4

    s3:
      endpoint: "https://s3.amazonaws.com"
      bucket: "my-backups"
      region: "us-east-1"
      path: "paradedb"
      secretRef:
        name: s3-credentials
```

### Prometheus Monitoring

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-monitored
spec:
  storage:
    size: "10Gi"

  monitoring:
    enabled: true
    serviceMonitor:
      enabled: true
      labels:
        release: prometheus
```

### Custom PostgreSQL Settings

```yaml
apiVersion: database.paradedb.io/v1alpha1
kind: ParadeDB
metadata:
  name: paradedb-tuned
spec:
  storage:
    size: "10Gi"

  postgresConfig:
    max_connections: "200"
    shared_buffers: "512MB"
    effective_cache_size: "2GB"
    work_mem: "16MB"

  auth:
    database: "myapp"
    pgHBA:
      - "host all all 10.0.0.0/8 scram-sha-256"
```

## Operations

### Scaling

```bash
# Scale replicas
kubectl patch paradedb my-paradedb --type='merge' -p '{"spec":{"replicas":3}}'

# Expand storage (requires StorageClass with allowVolumeExpansion)
kubectl patch paradedb my-paradedb --type='merge' -p '{"spec":{"storage":{"size":"20Gi"}}}'
```

### Upgrading

```bash
kubectl patch paradedb my-paradedb --type='merge' -p '{"spec":{"image":"paradedb/paradedb:v0.9.0"}}'
```

### Viewing Status

```bash
kubectl get paradedb my-paradedb -o yaml
```

Status fields:
- `phase`: Current state (Pending, Creating, Running, Updating, Failed, Deleting)
- `readyReplicas`: Number of healthy replicas
- `endpoint`: Connection endpoint
- `poolerEndpoint`: Connection pooler endpoint (if enabled)

### Uninstalling

```bash
# Delete instance (PVCs are retained by default)
kubectl delete paradedb my-paradedb

# Remove operator
kubectl delete -k https://github.com/ini8labs/paradedb-operator/config/default
```

## Connecting to ParadeDB

### From within the cluster

```bash
kubectl run psql --rm -it --image=postgres:16 -- \
  psql -h my-paradedb -U postgres -d paradedb
```

### From outside the cluster

```bash
kubectl port-forward svc/my-paradedb 5432:5432
psql -h localhost -U postgres -d paradedb
```

### Connection string format

```
postgresql://postgres:<password>@<endpoint>:5432/<database>
```

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod -l app.kubernetes.io/instance=my-paradedb
kubectl logs -l app.kubernetes.io/instance=my-paradedb
```

### Connection issues

```bash
# Verify service exists
kubectl get svc my-paradedb

# Test connectivity
kubectl run test --rm -it --image=busybox -- nc -zv my-paradedb 5432
```

### Check operator logs

```bash
kubectl logs -n paradedb-operator-system -l control-plane=controller-manager
```

## Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `image` | ParadeDB container image | `paradedb/paradedb:latest` |
| `replicas` | Number of instances (1-10) | `1` |
| `storage.size` | Storage size | Required |
| `storage.storageClassName` | StorageClass to use | Default class |
| `auth.database` | Default database name | `paradedb` |
| `extensions.pgSearch` | Enable full-text search | `true` |
| `extensions.pgAnalytics` | Enable analytics | `true` |
| `extensions.pgVector` | Enable vector search | `false` |
| `connectionPooling.enabled` | Enable PgBouncer | `false` |
| `backup.enabled` | Enable automated backups | `false` |
| `backup.schedule` | Backup cron schedule | `0 2 * * *` |
| `monitoring.enabled` | Enable Prometheus metrics | `true` |
| `tls.enabled` | Enable TLS encryption | `false` |
| `serviceType` | Kubernetes Service type | `ClusterIP` |
| `resources` | CPU/Memory requests and limits | - |
| `postgresConfig` | Custom PostgreSQL parameters | - |

## Version Compatibility

| Operator Version | Kubernetes | ParadeDB |
|-----------------|------------|----------|
| v0.1.x | 1.25+ | 0.8.x+ |

## Links

- [ParadeDB](https://paradedb.com) - The Postgres for Search and Analytics
- [ParadeDB Documentation](https://docs.paradedb.com)
- [GitHub Issues](https://github.com/ini8labs/paradedb-operator/issues)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2024 INI8Labs

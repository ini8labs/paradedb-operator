# ParadeDB Operator Architecture

This document describes the architecture of the ParadeDB Kubernetes Operator.

## High-Level Architecture

```mermaid
flowchart TB
    subgraph K8s["Kubernetes Cluster"]
        subgraph OperatorNS["paradedb-operator-system namespace"]
            CM["Controller Manager<br/>(Deployment)"]
            SA["ServiceAccount<br/>+ RBAC"]
            MS["Metrics Service<br/>:8443"]
        end

        subgraph UserNS["User Namespace"]
            CR["ParadeDB CR<br/>(Custom Resource)"]

            subgraph ManagedResources["Managed Resources"]
                STS["StatefulSet"]
                SVC["Service<br/>:5432"]
                HSVC["Headless Service"]
                MSVC["Metrics Service<br/>:9187"]
                CM_CFG["ConfigMap<br/>(postgresql.conf)"]
                SEC["Secret<br/>(credentials)"]
                PVC["PersistentVolumeClaim"]
            end

            subgraph OptionalResources["Optional Resources"]
                POOLER["PgBouncer<br/>Deployment"]
                PSVC["Pooler Service"]
                BACKUP["Backup CronJob"]
                SMON["ServiceMonitor"]
            end

            subgraph Pods["Database Pods"]
                POD1["Pod: paradedb-0"]
                POD2["Pod: paradedb-1"]
                POD3["Pod: paradedb-n"]
            end
        end

        API["Kubernetes API Server"]
    end

    subgraph External["External Systems"]
        S3["S3 Storage<br/>(Backups)"]
        PROM["Prometheus"]
        CERTM["cert-manager"]
    end

    %% Controller watches and manages
    CM -->|watches| API
    API -->|events| CR
    CM -->|reconciles| ManagedResources
    CM -->|reconciles| OptionalResources

    %% Resource relationships
    STS --> POD1
    STS --> POD2
    STS --> POD3
    POD1 --> PVC
    SVC --> Pods
    HSVC --> Pods
    MSVC --> Pods

    %% Optional connections
    POOLER --> SVC
    PSVC --> POOLER
    BACKUP --> S3
    SMON --> PROM
    CERTM -.->|TLS certs| SEC

    %% Styling
    classDef operator fill:#326CE5,color:white
    classDef resource fill:#00ADD8,color:white
    classDef optional fill:#6B7280,color:white
    classDef external fill:#10B981,color:white

    class CM,SA operator
    class STS,SVC,HSVC,CM_CFG,SEC,PVC,POD1,POD2,POD3 resource
    class POOLER,PSVC,BACKUP,SMON,MSVC optional
    class S3,PROM,CERTM external
```

## Reconciliation Flow

```mermaid
flowchart TD
    START([Watch Event Triggered]) --> FETCH[Fetch ParadeDB Resource]
    FETCH --> DEL{Deletion<br/>Timestamp?}

    DEL -->|Yes| FINALIZE[Run Finalizers<br/>Cleanup Resources]
    FINALIZE --> REMOVE[Remove Finalizer]
    REMOVE --> END1([End])

    DEL -->|No| ADDFIN[Add Finalizer<br/>if Missing]
    ADDFIN --> INIT[Initialize Status<br/>Phase: Pending]

    INIT --> CRED[Reconcile Credentials Secret]
    CRED --> CFG[Reconcile ConfigMap<br/>postgresql.conf, pg_hba.conf, init.sql]
    CFG --> STS[Reconcile StatefulSet]
    STS --> SVC[Reconcile Service]
    SVC --> HSVC[Reconcile Headless Service]

    HSVC --> POOL{Connection<br/>Pooling?}
    POOL -->|Yes| POOLER[Reconcile PgBouncer<br/>Deployment + Service]
    POOL -->|No| MON
    POOLER --> MON

    MON{Monitoring<br/>Enabled?}
    MON -->|Yes| METRICS[Reconcile Metrics<br/>Service + ServiceMonitor]
    MON -->|No| BACKUP_CHK
    METRICS --> BACKUP_CHK

    BACKUP_CHK{Backup<br/>Enabled?}
    BACKUP_CHK -->|Yes| BACKUP[Reconcile Backup CronJob]
    BACKUP_CHK -->|No| STATUS
    BACKUP --> STATUS

    STATUS[Update Status<br/>Phase, Replicas, Endpoint]
    STATUS --> COND[Set Conditions<br/>Ready, Progressing, Degraded]
    COND --> REQUEUE([Requeue After 60s])

    %% Styling
    classDef start fill:#10B981,color:white
    classDef decision fill:#F59E0B,color:white
    classDef action fill:#3B82F6,color:white
    classDef end_node fill:#EF4444,color:white

    class START,REQUEUE start
    class DEL,POOL,MON,BACKUP_CHK decision
    class FETCH,FINALIZE,REMOVE,ADDFIN,INIT,CRED,CFG,STS,SVC,HSVC,POOLER,METRICS,BACKUP,STATUS,COND action
    class END1 end_node
```

## Component Details

### Controller Manager

The Controller Manager runs in the `paradedb-operator-system` namespace and contains the `ParadeDBReconciler`:

```mermaid
flowchart LR
    subgraph ControllerManager["Controller Manager Pod"]
        MAIN["main.go<br/>(entrypoint)"]
        RECONCILER["ParadeDBReconciler"]
        HELPERS["helpers.go<br/>(config builders)"]

        MAIN --> RECONCILER
        RECONCILER --> HELPERS
    end

    subgraph WatchedResources["Watched Resources"]
        PARADEDB["ParadeDB CRs"]
        STS["StatefulSets"]
        DEPLOY["Deployments"]
        SVC["Services"]
    end

    RECONCILER -->|owns| WatchedResources
```

### ParadeDB Pod Architecture

```mermaid
flowchart TB
    subgraph Pod["ParadeDB Pod"]
        subgraph Containers["Containers"]
            PARADEDB["paradedb<br/>:5432<br/>(PostgreSQL + Extensions)"]
            EXPORTER["postgres-exporter<br/>:9187<br/>(optional)"]
        end

        subgraph Volumes["Volume Mounts"]
            DATA["/var/lib/postgresql/data<br/>(PVC)"]
            CONFIG["/etc/postgresql<br/>(ConfigMap)"]
            CREDS["/etc/postgresql/secrets<br/>(Secret)"]
            TLS["/etc/postgresql/tls<br/>(optional)"]
        end
    end

    PARADEDB --> DATA
    PARADEDB --> CONFIG
    PARADEDB --> CREDS
    PARADEDB --> TLS
    EXPORTER --> PARADEDB
```

### ParadeDB Extensions

```mermaid
flowchart LR
    subgraph ParadeDB["ParadeDB Instance"]
        PG["PostgreSQL Core"]

        subgraph Extensions["Extensions"]
            SEARCH["pg_search<br/>(Full-text Search)"]
            ANALYTICS["pg_analytics<br/>(DuckDB Integration)"]
            VECTOR["pgvector<br/>(Vector Search)"]
            STATS["pg_stat_statements"]
            UUID["uuid-ossp"]
        end
    end

    PG --> Extensions

    SEARCH -->|BM25 Ranking| FTS["Full-Text Search"]
    ANALYTICS -->|Columnar| OLAP["Analytics Queries"]
    VECTOR -->|Similarity| AI["AI/ML Applications"]
```

## Resource Ownership

```mermaid
flowchart TB
    CR["ParadeDB CR"] -->|owns| STS["StatefulSet"]
    CR -->|owns| SVC["Service"]
    CR -->|owns| HSVC["Headless Service"]
    CR -->|owns| CFG["ConfigMap"]
    CR -->|owns| SEC["Secret"]
    CR -->|owns| MSVC["Metrics Service"]
    CR -->|owns| POOLER["PgBouncer Deployment"]
    CR -->|owns| PSVC["Pooler Service"]
    CR -->|owns| PCFG["Pooler ConfigMap"]
    CR -->|owns| BACKUP["Backup CronJob"]
    CR -->|owns| SMON["ServiceMonitor"]

    STS -->|owns| PODS["Pods"]
    STS -->|owns| PVC["PVCs"]

    style CR fill:#326CE5,color:white
    style STS fill:#00ADD8,color:white
```

## Network Architecture

```mermaid
flowchart TB
    subgraph Cluster["Kubernetes Cluster"]
        subgraph Services["Services"]
            MAIN["my-paradedb<br/>ClusterIP:5432"]
            HEADLESS["my-paradedb-headless<br/>None (DNS)"]
            METRICS["my-paradedb-metrics<br/>ClusterIP:9187"]
            POOLER["my-paradedb-pooler<br/>ClusterIP:5432"]
        end

        subgraph Pods["StatefulSet Pods"]
            P0["my-paradedb-0<br/>:5432, :9187"]
            P1["my-paradedb-1<br/>:5432, :9187"]
        end

        subgraph PoolerPod["Pooler Pod"]
            PG["PgBouncer<br/>:5432"]
        end
    end

    CLIENT["Client Application"] --> MAIN
    CLIENT --> POOLER
    MAIN --> Pods
    HEADLESS --> P0
    HEADLESS --> P1
    METRICS --> Pods
    POOLER --> PG
    PG --> MAIN

    PROM["Prometheus"] --> METRICS
```

## Backup Architecture

```mermaid
flowchart LR
    subgraph Cluster["Kubernetes Cluster"]
        CRON["CronJob<br/>(scheduled)"]
        JOB["Job<br/>(runs backup)"]
        POD["Backup Pod"]
        PARADEDB["ParadeDB Pod"]
    end

    subgraph Storage["Backup Storage"]
        S3["S3 Bucket"]
        PVC["PVC<br/>(optional)"]
    end

    CRON -->|creates| JOB
    JOB -->|runs| POD
    POD -->|pg_dump| PARADEDB
    POD -->|upload| S3
    POD -->|write| PVC
```

## Status Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Pending: CR Created
    Pending --> Creating: Reconciliation Starts
    Creating --> Running: All Replicas Ready
    Creating --> Failed: Error Occurred
    Running --> Updating: Spec Changed
    Updating --> Running: Update Complete
    Updating --> Failed: Update Error
    Running --> Deleting: CR Deleted
    Failed --> Creating: Error Resolved
    Deleting --> [*]: Finalizers Complete
```

## Data Flow

```mermaid
flowchart TB
    subgraph UserWorkflow["User Workflow"]
        USER["User/Admin"]
        KUBECTL["kubectl apply"]
        YAML["ParadeDB YAML"]
    end

    subgraph ControlPlane["Control Plane"]
        API["API Server"]
        ETCD["etcd"]
        CTRL["Controller Manager"]
    end

    subgraph DataPlane["Data Plane"]
        STS["StatefulSet"]
        POD["ParadeDB Pods"]
        PVC["Persistent Storage"]
    end

    subgraph Clients["Database Clients"]
        APP["Applications"]
        PSQL["psql"]
    end

    USER --> KUBECTL
    KUBECTL --> YAML
    YAML --> API
    API --> ETCD
    CTRL -->|watch| API
    CTRL -->|create| STS
    STS --> POD
    POD --> PVC

    APP --> POD
    PSQL --> POD
```

## Deployment Architecture

```mermaid
flowchart TB
    subgraph K8sCluster["Kubernetes Cluster"]
        subgraph OpNS["paradedb-operator-system"]
            subgraph CtrlDeploy["Controller Deployment"]
                CTRL["controller-manager<br/>distroless image"]
            end
            RBAC["RBAC<br/>ServiceAccount<br/>ClusterRole<br/>ClusterRoleBinding"]
            CRD["CRD<br/>database.paradedb.io"]
        end

        subgraph NS1["Namespace: production"]
            DB1["ParadeDB: prod-db"]
            STS1["StatefulSet (3 replicas)"]
            POOL1["PgBouncer"]
        end

        subgraph NS2["Namespace: staging"]
            DB2["ParadeDB: staging-db"]
            STS2["StatefulSet (1 replica)"]
        end

        subgraph NS3["Namespace: dev"]
            DB3["ParadeDB: dev-db"]
            STS3["StatefulSet (1 replica)"]
        end
    end

    CTRL -->|manages| NS1
    CTRL -->|manages| NS2
    CTRL -->|manages| NS3

    style CTRL fill:#326CE5,color:white
    style DB1,DB2,DB3 fill:#00ADD8,color:white
```

## Key Files

| File | Purpose |
|------|---------|
| `api/v1alpha1/paradedb_types.go` | CRD type definitions (Spec/Status) |
| `internal/controller/paradedb_controller.go` | Main reconciliation logic |
| `internal/controller/helpers.go` | Configuration builders |
| `cmd/main.go` | Controller manager entrypoint |
| `config/crd/` | CRD manifests |
| `config/rbac/` | RBAC definitions |
| `config/manager/` | Controller deployment |

## Technologies

- **Language**: Go 1.25
- **Framework**: controller-runtime v0.23.0
- **Kubernetes**: client-go v0.35.0
- **Build**: Kubebuilder
- **Container**: distroless base image

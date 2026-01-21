/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ParadeDBSpec defines the desired state of ParadeDB
type ParadeDBSpec struct {
	// Image is the ParadeDB container image to use
	// +kubebuilder:default="paradedb/paradedb:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// Replicas is the number of ParadeDB instances (1 for standalone, >1 for HA)
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// PostgresVersion specifies the PostgreSQL version
	// +kubebuilder:default="16"
	// +optional
	PostgresVersion string `json:"postgresVersion,omitempty"`

	// Storage configuration for ParadeDB
	// +required
	Storage StorageSpec `json:"storage"`

	// Resources defines the CPU and memory resources for ParadeDB pods
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Auth contains authentication configuration
	// +optional
	Auth AuthSpec `json:"auth,omitempty"`

	// TLS configuration for encrypted connections
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`

	// ConnectionPooling configuration (PgBouncer)
	// +optional
	ConnectionPooling *ConnectionPoolingSpec `json:"connectionPooling,omitempty"`

	// Backup configuration
	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`

	// Monitoring configuration
	// +optional
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`

	// Extensions to enable in ParadeDB
	// +optional
	Extensions ExtensionsSpec `json:"extensions,omitempty"`

	// PostgresConfig allows custom PostgreSQL configuration parameters
	// +optional
	PostgresConfig map[string]string `json:"postgresConfig,omitempty"`

	// ServiceType specifies the type of Service to create
	// +kubebuilder:default="ClusterIP"
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// NodeSelector for pod scheduling
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations for pod scheduling
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity for pod scheduling
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PodSecurityContext for the ParadeDB pods
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// ContainerSecurityContext for the ParadeDB container
	// +optional
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
}

// StorageSpec defines storage configuration
type StorageSpec struct {
	// Size is the size of the PersistentVolumeClaim
	// +kubebuilder:default="10Gi"
	Size resource.Quantity `json:"size"`

	// StorageClassName is the name of the StorageClass to use
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessModes for the PVC
	// +kubebuilder:default={"ReadWriteOnce"}
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// WalStorage for separate WAL storage
	// +optional
	WalStorage *WalStorageSpec `json:"walStorage,omitempty"`
}

// WalStorageSpec defines separate WAL storage configuration
type WalStorageSpec struct {
	// Size of the WAL storage
	// +kubebuilder:default="5Gi"
	Size resource.Quantity `json:"size"`

	// StorageClassName for WAL storage
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// AuthSpec defines authentication configuration
type AuthSpec struct {
	// SuperuserSecretRef references a Secret containing superuser credentials
	// The secret must contain 'username' and 'password' keys
	// +optional
	SuperuserSecretRef *corev1.SecretReference `json:"superuserSecretRef,omitempty"`

	// Database is the default database to create
	// +kubebuilder:default="paradedb"
	// +optional
	Database string `json:"database,omitempty"`

	// Users defines additional database users to create
	// +optional
	Users []DatabaseUser `json:"users,omitempty"`

	// EnablePgHBA enables custom pg_hba.conf configuration
	// +optional
	PgHBA []string `json:"pgHBA,omitempty"`
}

// DatabaseUser defines a database user
type DatabaseUser struct {
	// Name of the user
	Name string `json:"name"`

	// SecretRef references a Secret containing the user's password
	SecretRef corev1.SecretReference `json:"secretRef"`

	// Databases the user has access to
	// +optional
	Databases []string `json:"databases,omitempty"`

	// Privileges for the user
	// +optional
	Privileges []string `json:"privileges,omitempty"`
}

// TLSSpec defines TLS configuration
type TLSSpec struct {
	// Enabled enables TLS for PostgreSQL connections
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// SecretRef references a Secret containing TLS certificates
	// The secret must contain 'tls.crt', 'tls.key', and optionally 'ca.crt'
	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`

	// CertManager enables automatic certificate management via cert-manager
	// +optional
	CertManager *CertManagerSpec `json:"certManager,omitempty"`
}

// CertManagerSpec defines cert-manager integration
type CertManagerSpec struct {
	// Enabled enables cert-manager integration
	Enabled bool `json:"enabled"`

	// IssuerRef references a cert-manager Issuer or ClusterIssuer
	// +optional
	IssuerRef *CertIssuerRef `json:"issuerRef,omitempty"`
}

// CertIssuerRef references a cert-manager issuer
type CertIssuerRef struct {
	// Name of the issuer
	Name string `json:"name"`

	// Kind of the issuer (Issuer or ClusterIssuer)
	// +kubebuilder:default="Issuer"
	// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
	Kind string `json:"kind"`
}

// ConnectionPoolingSpec defines connection pooling configuration
type ConnectionPoolingSpec struct {
	// Enabled enables PgBouncer connection pooling
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Image is the PgBouncer container image
	// +kubebuilder:default="bitnami/pgbouncer:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// PoolMode specifies the pool mode
	// +kubebuilder:default="transaction"
	// +kubebuilder:validation:Enum=session;transaction;statement
	// +optional
	PoolMode string `json:"poolMode,omitempty"`

	// MaxClientConnections is the maximum number of client connections
	// +kubebuilder:default=100
	// +optional
	MaxClientConnections int32 `json:"maxClientConnections,omitempty"`

	// DefaultPoolSize is the default pool size per user/database pair
	// +kubebuilder:default=20
	// +optional
	DefaultPoolSize int32 `json:"defaultPoolSize,omitempty"`

	// MinPoolSize is the minimum pool size
	// +kubebuilder:default=0
	// +optional
	MinPoolSize int32 `json:"minPoolSize,omitempty"`

	// ReservePoolSize is the number of reserve connections
	// +kubebuilder:default=5
	// +optional
	ReservePoolSize int32 `json:"reservePoolSize,omitempty"`

	// Resources for the PgBouncer container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// BackupSpec defines backup configuration
type BackupSpec struct {
	// Enabled enables automated backups
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Schedule is a cron expression for backup scheduling
	// +kubebuilder:default="0 2 * * *"
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// RetentionPolicy defines how long to keep backups
	// +optional
	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`

	// S3 configuration for storing backups in S3-compatible storage
	// +optional
	S3 *S3BackupSpec `json:"s3,omitempty"`

	// PVC configuration for storing backups on PersistentVolumes
	// +optional
	PVC *PVCBackupSpec `json:"pvc,omitempty"`
}

// RetentionPolicy defines backup retention
type RetentionPolicy struct {
	// KeepLast is the number of recent backups to keep
	// +kubebuilder:default=7
	// +optional
	KeepLast int32 `json:"keepLast,omitempty"`

	// KeepDaily is the number of daily backups to keep
	// +kubebuilder:default=7
	// +optional
	KeepDaily int32 `json:"keepDaily,omitempty"`

	// KeepWeekly is the number of weekly backups to keep
	// +kubebuilder:default=4
	// +optional
	KeepWeekly int32 `json:"keepWeekly,omitempty"`
}

// S3BackupSpec defines S3-compatible backup storage
type S3BackupSpec struct {
	// Endpoint is the S3 endpoint URL
	Endpoint string `json:"endpoint"`

	// Bucket is the S3 bucket name
	Bucket string `json:"bucket"`

	// Region is the S3 region
	// +optional
	Region string `json:"region,omitempty"`

	// SecretRef references a Secret containing S3 credentials
	// The secret must contain 'accessKeyId' and 'secretAccessKey'
	SecretRef corev1.SecretReference `json:"secretRef"`

	// Path prefix for backups in the bucket
	// +optional
	Path string `json:"path,omitempty"`
}

// PVCBackupSpec defines PVC-based backup storage
type PVCBackupSpec struct {
	// Size is the size of the backup PVC
	// +kubebuilder:default="20Gi"
	Size resource.Quantity `json:"size"`

	// StorageClassName for the backup PVC
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// MonitoringSpec defines monitoring configuration
type MonitoringSpec struct {
	// Enabled enables Prometheus metrics exporter
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Image is the postgres_exporter container image
	// +kubebuilder:default="quay.io/prometheuscommunity/postgres-exporter:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// Port for the metrics endpoint
	// +kubebuilder:default=9187
	// +optional
	Port int32 `json:"port,omitempty"`

	// Resources for the exporter container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceMonitor enables creating a ServiceMonitor for Prometheus Operator
	// +optional
	ServiceMonitor *ServiceMonitorSpec `json:"serviceMonitor,omitempty"`

	// CustomQueries allows defining custom metrics queries
	// +optional
	CustomQueries map[string]string `json:"customQueries,omitempty"`
}

// ServiceMonitorSpec defines ServiceMonitor configuration
type ServiceMonitorSpec struct {
	// Enabled enables ServiceMonitor creation
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Labels to add to the ServiceMonitor
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Interval for scraping metrics
	// +kubebuilder:default="30s"
	// +optional
	Interval string `json:"interval,omitempty"`
}

// ExtensionsSpec defines ParadeDB extensions configuration
type ExtensionsSpec struct {
	// PgSearch enables the pg_search extension (full-text search)
	// +kubebuilder:default=true
	PgSearch bool `json:"pgSearch,omitempty"`

	// PgAnalytics enables the pg_analytics extension (DuckDB integration)
	// +kubebuilder:default=true
	PgAnalytics bool `json:"pgAnalytics,omitempty"`

	// PgVector enables the pgvector extension (vector similarity search)
	// +kubebuilder:default=false
	// +optional
	PgVector bool `json:"pgVector,omitempty"`

	// Additional is a list of additional PostgreSQL extensions to enable
	// +optional
	Additional []string `json:"additional,omitempty"`
}

// ParadeDBPhase represents the current phase of the ParadeDB instance
// +kubebuilder:validation:Enum=Pending;Creating;Running;Updating;Failed;Deleting
type ParadeDBPhase string

const (
	ParadeDBPhasePending  ParadeDBPhase = "Pending"
	ParadeDBPhaseCreating ParadeDBPhase = "Creating"
	ParadeDBPhaseRunning  ParadeDBPhase = "Running"
	ParadeDBPhaseUpdating ParadeDBPhase = "Updating"
	ParadeDBPhaseFailed   ParadeDBPhase = "Failed"
	ParadeDBPhaseDeleting ParadeDBPhase = "Deleting"
)

// ParadeDBStatus defines the observed state of ParadeDB
type ParadeDBStatus struct {
	// Phase represents the current phase of the ParadeDB instance
	// +optional
	Phase ParadeDBPhase `json:"phase,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentVersion is the current ParadeDB version running
	// +optional
	CurrentVersion string `json:"currentVersion,omitempty"`

	// Endpoint is the connection endpoint for the database
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// PoolerEndpoint is the connection endpoint for the connection pooler
	// +optional
	PoolerEndpoint string `json:"poolerEndpoint,omitempty"`

	// LastBackup is the timestamp of the last successful backup
	// +optional
	LastBackup *metav1.Time `json:"lastBackup,omitempty"`

	// LastBackupSize is the size of the last backup
	// +optional
	LastBackupSize string `json:"lastBackupSize,omitempty"`

	// Conditions represent the current state of the ParadeDB resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Message provides additional status information
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.currentVersion`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=pdb

// ParadeDB is the Schema for the paradedbs API
type ParadeDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +required
	Spec   ParadeDBSpec   `json:"spec"`
	Status ParadeDBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ParadeDBList contains a list of ParadeDB
type ParadeDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ParadeDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ParadeDB{}, &ParadeDBList{})
}

// Helper functions

// GetReplicas returns the number of replicas
func (p *ParadeDB) GetReplicas() int32 {
	if p.Spec.Replicas == nil {
		return 1
	}
	return *p.Spec.Replicas
}

// IsConnectionPoolingEnabled returns true if connection pooling is enabled
func (p *ParadeDB) IsConnectionPoolingEnabled() bool {
	return p.Spec.ConnectionPooling != nil && p.Spec.ConnectionPooling.Enabled
}

// IsTLSEnabled returns true if TLS is enabled
func (p *ParadeDB) IsTLSEnabled() bool {
	return p.Spec.TLS != nil && p.Spec.TLS.Enabled
}

// IsBackupEnabled returns true if backup is enabled
func (p *ParadeDB) IsBackupEnabled() bool {
	return p.Spec.Backup != nil && p.Spec.Backup.Enabled
}

// IsMonitoringEnabled returns true if monitoring is enabled
func (p *ParadeDB) IsMonitoringEnabled() bool {
	return p.Spec.Monitoring == nil || p.Spec.Monitoring.Enabled
}

// GetImage returns the ParadeDB image to use
func (p *ParadeDB) GetImage() string {
	if p.Spec.Image == "" {
		return "paradedb/paradedb:latest"
	}
	return p.Spec.Image
}

// GetServiceName returns the service name for the ParadeDB instance
func (p *ParadeDB) GetServiceName() string {
	return p.Name
}

// GetStatefulSetName returns the StatefulSet name for the ParadeDB instance
func (p *ParadeDB) GetStatefulSetName() string {
	return p.Name
}

// GetPoolerServiceName returns the pooler service name
func (p *ParadeDB) GetPoolerServiceName() string {
	return p.Name + "-pooler"
}

// GetPoolerDeploymentName returns the pooler deployment name
func (p *ParadeDB) GetPoolerDeploymentName() string {
	return p.Name + "-pooler"
}

// GetMetricsServiceName returns the metrics service name
func (p *ParadeDB) GetMetricsServiceName() string {
	return p.Name + "-metrics"
}

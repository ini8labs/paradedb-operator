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

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1alpha1 "github.com/paradedb/paradedb-operator/api/v1alpha1"
)

const (
	// Finalizer for ParadeDB resources
	paradedbFinalizer = "database.paradedb.io/finalizer"

	// Condition types
	ConditionTypeReady       = "Ready"
	ConditionTypeProgressing = "Progressing"
	ConditionTypeDegraded    = "Degraded"

	// Requeue intervals
	requeueAfterError   = 30 * time.Second
	requeueAfterSuccess = 60 * time.Second
)

// ParadeDBReconciler reconciles a ParadeDB object
type ParadeDBReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=database.paradedb.io,resources=paradedbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.paradedb.io,resources=paradedbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.paradedb.io,resources=paradedbs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop
func (r *ParadeDBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling ParadeDB", "namespace", req.Namespace, "name", req.Name)

	// Fetch the ParadeDB instance
	paradedb := &databasev1alpha1.ParadeDB{}
	err := r.Get(ctx, req.NamespacedName, paradedb)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ParadeDB resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ParadeDB")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if paradedb.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(paradedb, paradedbFinalizer) {
			log.Info("Performing Finalizer Operations for ParadeDB")

			// Update status to Deleting
			paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseDeleting
			if err := r.Status().Update(ctx, paradedb); err != nil {
				log.Error(err, "Failed to update ParadeDB status")
				return ctrl.Result{}, err
			}

			// Perform cleanup operations
			if err := r.finalizeParadeDB(ctx, paradedb); err != nil {
				log.Error(err, "Failed to finalize ParadeDB")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(paradedb, paradedbFinalizer)
			if err := r.Update(ctx, paradedb); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(paradedb, paradedbFinalizer) {
		log.Info("Adding Finalizer for ParadeDB")
		controllerutil.AddFinalizer(paradedb, paradedbFinalizer)
		if err := r.Update(ctx, paradedb); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Initialize status if empty
	if paradedb.Status.Phase == "" {
		paradedb.Status.Phase = databasev1alpha1.ParadeDBPhasePending
		if err := r.Status().Update(ctx, paradedb); err != nil {
			log.Error(err, "Failed to update ParadeDB status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status to Creating if Pending
	if paradedb.Status.Phase == databasev1alpha1.ParadeDBPhasePending {
		paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseCreating
		if err := r.Status().Update(ctx, paradedb); err != nil {
			log.Error(err, "Failed to update ParadeDB status")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(paradedb, corev1.EventTypeNormal, "Creating", "Starting ParadeDB creation")
	}

	// Reconcile credentials secret
	if err := r.reconcileCredentialsSecret(ctx, paradedb); err != nil {
		log.Error(err, "Failed to reconcile credentials secret")
		return r.handleError(ctx, paradedb, err, "Failed to reconcile credentials secret")
	}

	// Reconcile ConfigMap for PostgreSQL configuration
	if err := r.reconcileConfigMap(ctx, paradedb); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		return r.handleError(ctx, paradedb, err, "Failed to reconcile ConfigMap")
	}

	// Reconcile StatefulSet
	if err := r.reconcileStatefulSet(ctx, paradedb); err != nil {
		log.Error(err, "Failed to reconcile StatefulSet")
		return r.handleError(ctx, paradedb, err, "Failed to reconcile StatefulSet")
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, paradedb); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return r.handleError(ctx, paradedb, err, "Failed to reconcile Service")
	}

	// Reconcile Headless Service for StatefulSet
	if err := r.reconcileHeadlessService(ctx, paradedb); err != nil {
		log.Error(err, "Failed to reconcile Headless Service")
		return r.handleError(ctx, paradedb, err, "Failed to reconcile Headless Service")
	}

	// Reconcile Connection Pooler (PgBouncer) if enabled
	if paradedb.IsConnectionPoolingEnabled() {
		if err := r.reconcileConnectionPooler(ctx, paradedb); err != nil {
			log.Error(err, "Failed to reconcile Connection Pooler")
			return r.handleError(ctx, paradedb, err, "Failed to reconcile Connection Pooler")
		}
	}

	// Reconcile Metrics Exporter if monitoring is enabled
	if paradedb.IsMonitoringEnabled() {
		if err := r.reconcileMetricsService(ctx, paradedb); err != nil {
			log.Error(err, "Failed to reconcile Metrics Service")
			return r.handleError(ctx, paradedb, err, "Failed to reconcile Metrics Service")
		}
	}

	// Reconcile Backup CronJob if backup is enabled
	if paradedb.IsBackupEnabled() {
		if err := r.reconcileBackupCronJob(ctx, paradedb); err != nil {
			log.Error(err, "Failed to reconcile Backup CronJob")
			return r.handleError(ctx, paradedb, err, "Failed to reconcile Backup CronJob")
		}
	}

	// Update status based on StatefulSet status
	if err := r.updateStatus(ctx, paradedb); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: requeueAfterError}, err
	}

	log.Info("Successfully reconciled ParadeDB")
	return ctrl.Result{RequeueAfter: requeueAfterSuccess}, nil
}

// handleError handles errors during reconciliation
func (r *ParadeDBReconciler) handleError(ctx context.Context, paradedb *databasev1alpha1.ParadeDB, err error, message string) (ctrl.Result, error) {
	paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseFailed
	paradedb.Status.Message = message + ": " + err.Error()

	meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             metav1.ConditionTrue,
		Reason:             "ReconciliationFailed",
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})

	if updateErr := r.Status().Update(ctx, paradedb); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	r.Recorder.Event(paradedb, corev1.EventTypeWarning, "ReconciliationFailed", message)
	return ctrl.Result{RequeueAfter: requeueAfterError}, err
}

// finalizeParadeDB performs cleanup when ParadeDB is being deleted
func (r *ParadeDBReconciler) finalizeParadeDB(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)
	log.Info("Finalizing ParadeDB", "name", paradedb.Name)

	// Cleanup is handled by Kubernetes garbage collection via OwnerReferences
	// Add any additional cleanup logic here if needed

	r.Recorder.Event(paradedb, corev1.EventTypeNormal, "Deleted", "ParadeDB instance deleted successfully")
	return nil
}

// reconcileCredentialsSecret creates or updates the credentials secret
func (r *ParadeDBReconciler) reconcileCredentialsSecret(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	// Check if user provided a secret reference
	if paradedb.Spec.Auth.SuperuserSecretRef != nil {
		// Verify the secret exists
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      paradedb.Spec.Auth.SuperuserSecretRef.Name,
			Namespace: paradedb.Namespace,
		}, secret)
		if err != nil {
			return fmt.Errorf("failed to get superuser secret: %w", err)
		}
		return nil
	}

	// Create default credentials secret
	secretName := paradedb.Name + "-credentials"
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: paradedb.Namespace}, secret)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating credentials secret", "name", secretName)

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"username": "postgres",
				"password": generateRandomPassword(16),
				"database": paradedb.Spec.Auth.Database,
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, secret, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, secret); err != nil {
			return err
		}

		r.Recorder.Event(paradedb, corev1.EventTypeNormal, "SecretCreated", "Credentials secret created")
	} else if err != nil {
		return err
	}

	return nil
}

// reconcileConfigMap creates or updates the PostgreSQL configuration ConfigMap
func (r *ParadeDBReconciler) reconcileConfigMap(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	configMapName := paradedb.Name + "-config"
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: paradedb.Namespace}, configMap)

	// Build PostgreSQL configuration
	postgresConf := buildPostgresConfig(paradedb)
	pgHBAConf := buildPgHBAConfig(paradedb)
	initScript := buildInitScript(paradedb)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating ConfigMap", "name", configMapName)

		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
			},
			Data: map[string]string{
				"postgresql.conf": postgresConf,
				"pg_hba.conf":     pgHBAConf,
				"init.sql":        initScript,
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, configMap, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, configMap); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update existing ConfigMap
		configMap.Data = map[string]string{
			"postgresql.conf": postgresConf,
			"pg_hba.conf":     pgHBAConf,
			"init.sql":        initScript,
		}
		if err := r.Update(ctx, configMap); err != nil {
			return err
		}
	}

	return nil
}

// reconcileStatefulSet creates or updates the StatefulSet for ParadeDB
func (r *ParadeDBReconciler) reconcileStatefulSet(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: paradedb.GetStatefulSetName(), Namespace: paradedb.Namespace}, statefulSet)

	desired := r.buildStatefulSet(paradedb)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating StatefulSet", "name", desired.Name)

		if err := controllerutil.SetControllerReference(paradedb, desired, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, desired); err != nil {
			return err
		}

		r.Recorder.Event(paradedb, corev1.EventTypeNormal, "StatefulSetCreated", "StatefulSet created successfully")
	} else if err != nil {
		return err
	} else {
		// Update existing StatefulSet
		statefulSet.Spec.Replicas = desired.Spec.Replicas
		statefulSet.Spec.Template = desired.Spec.Template

		if err := r.Update(ctx, statefulSet); err != nil {
			return err
		}
	}

	return nil
}

// reconcileService creates or updates the main Service for ParadeDB
func (r *ParadeDBReconciler) reconcileService(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	service := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: paradedb.GetServiceName(), Namespace: paradedb.Namespace}, service)

	desired := r.buildService(paradedb)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "name", desired.Name)

		if err := controllerutil.SetControllerReference(paradedb, desired, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, desired); err != nil {
			return err
		}

		r.Recorder.Event(paradedb, corev1.EventTypeNormal, "ServiceCreated", "Service created successfully")
	} else if err != nil {
		return err
	} else {
		// Update existing Service (preserve ClusterIP)
		service.Spec.Ports = desired.Spec.Ports
		service.Spec.Type = desired.Spec.Type
		service.Spec.Selector = desired.Spec.Selector

		if err := r.Update(ctx, service); err != nil {
			return err
		}
	}

	return nil
}

// reconcileHeadlessService creates the headless service for StatefulSet
func (r *ParadeDBReconciler) reconcileHeadlessService(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	serviceName := paradedb.GetServiceName() + "-headless"
	service := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: paradedb.Namespace}, service)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Headless Service", "name", serviceName)

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
			},
			Spec: corev1.ServiceSpec{
				Selector:  r.getSelectorLabels(paradedb),
				ClusterIP: "None",
				Ports: []corev1.ServicePort{
					{
						Name:     "postgres",
						Port:     5432,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, service, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, service); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// reconcileConnectionPooler creates or updates the PgBouncer deployment
func (r *ParadeDBReconciler) reconcileConnectionPooler(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	// Create PgBouncer ConfigMap
	if err := r.reconcilePoolerConfigMap(ctx, paradedb); err != nil {
		return err
	}

	// Create PgBouncer Deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: paradedb.GetPoolerDeploymentName(), Namespace: paradedb.Namespace}, deployment)

	desired := r.buildPoolerDeployment(paradedb)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating PgBouncer Deployment", "name", desired.Name)

		if err := controllerutil.SetControllerReference(paradedb, desired, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, desired); err != nil {
			return err
		}

		r.Recorder.Event(paradedb, corev1.EventTypeNormal, "PoolerCreated", "Connection pooler created")
	} else if err != nil {
		return err
	}

	// Create PgBouncer Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: paradedb.GetPoolerServiceName(), Namespace: paradedb.Namespace}, service)

	if err != nil && errors.IsNotFound(err) {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paradedb.GetPoolerServiceName(),
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app.kubernetes.io/name":      "pgbouncer",
					"app.kubernetes.io/instance":  paradedb.Name,
					"app.kubernetes.io/component": "pooler",
				},
				Type: paradedb.Spec.ServiceType,
				Ports: []corev1.ServicePort{
					{
						Name:     "pgbouncer",
						Port:     5432,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, service, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, service); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// reconcilePoolerConfigMap creates the PgBouncer configuration
func (r *ParadeDBReconciler) reconcilePoolerConfigMap(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	configMapName := paradedb.Name + "-pooler-config"
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: paradedb.Namespace}, configMap)

	pooling := paradedb.Spec.ConnectionPooling
	pgbouncerIni := fmt.Sprintf(`[databases]
%s = host=%s port=5432 dbname=%s

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 5432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = %s
max_client_conn = %d
default_pool_size = %d
min_pool_size = %d
reserve_pool_size = %d
admin_users = postgres
stats_users = postgres
`,
		paradedb.Spec.Auth.Database,
		paradedb.GetServiceName(),
		paradedb.Spec.Auth.Database,
		pooling.PoolMode,
		pooling.MaxClientConnections,
		pooling.DefaultPoolSize,
		pooling.MinPoolSize,
		pooling.ReservePoolSize,
	)

	if err != nil && errors.IsNotFound(err) {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
			},
			Data: map[string]string{
				"pgbouncer.ini": pgbouncerIni,
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, configMap, r.Scheme); err != nil {
			return err
		}

		return r.Create(ctx, configMap)
	} else if err != nil {
		return err
	}

	return nil
}

// reconcileMetricsService creates the metrics service for Prometheus
func (r *ParadeDBReconciler) reconcileMetricsService(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	log := logf.FromContext(ctx)

	service := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: paradedb.GetMetricsServiceName(), Namespace: paradedb.Namespace}, service)

	metricsPort := int32(9187)
	if paradedb.Spec.Monitoring != nil && paradedb.Spec.Monitoring.Port != 0 {
		metricsPort = paradedb.Spec.Monitoring.Port
	}

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Metrics Service", "name", paradedb.GetMetricsServiceName())

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      paradedb.GetMetricsServiceName(),
				Namespace: paradedb.Namespace,
				Labels:    r.getLabels(paradedb),
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   fmt.Sprintf("%d", metricsPort),
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: r.getSelectorLabels(paradedb),
				Ports: []corev1.ServicePort{
					{
						Name:     "metrics",
						Port:     metricsPort,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(paradedb, service, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, service); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// reconcileBackupCronJob creates the backup CronJob
func (r *ParadeDBReconciler) reconcileBackupCronJob(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	// Backup implementation would go here
	// For now, we'll skip the actual CronJob creation as it requires additional setup
	return nil
}

// updateStatus updates the ParadeDB status based on the StatefulSet status
func (r *ParadeDBReconciler) updateStatus(ctx context.Context, paradedb *databasev1alpha1.ParadeDB) error {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: paradedb.GetStatefulSetName(), Namespace: paradedb.Namespace}, statefulSet)
	if err != nil {
		return err
	}

	// Update ready replicas
	paradedb.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas
	paradedb.Status.ObservedGeneration = paradedb.Generation
	paradedb.Status.CurrentVersion = paradedb.GetImage()

	// Determine phase based on replica status
	desiredReplicas := paradedb.GetReplicas()
	if statefulSet.Status.ReadyReplicas == desiredReplicas {
		paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseRunning
		paradedb.Status.Message = "ParadeDB is running"

		meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             "AllReplicasReady",
			Message:            fmt.Sprintf("All %d replicas are ready", desiredReplicas),
			LastTransitionTime: metav1.Now(),
		})

		meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeProgressing,
			Status:             metav1.ConditionFalse,
			Reason:             "DeploymentComplete",
			Message:            "Deployment complete",
			LastTransitionTime: metav1.Now(),
		})

		meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDegraded,
			Status:             metav1.ConditionFalse,
			Reason:             "AllReplicasHealthy",
			Message:            "All replicas are healthy",
			LastTransitionTime: metav1.Now(),
		})
	} else if statefulSet.Status.ReadyReplicas > 0 {
		paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseUpdating
		paradedb.Status.Message = fmt.Sprintf("Scaling: %d/%d replicas ready", statefulSet.Status.ReadyReplicas, desiredReplicas)

		meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             "Scaling",
			Message:            paradedb.Status.Message,
			LastTransitionTime: metav1.Now(),
		})
	} else {
		paradedb.Status.Phase = databasev1alpha1.ParadeDBPhaseCreating
		paradedb.Status.Message = "Waiting for replicas to become ready"

		meta.SetStatusCondition(&paradedb.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             "Creating",
			Message:            "Creating ParadeDB pods",
			LastTransitionTime: metav1.Now(),
		})
	}

	// Set endpoint
	paradedb.Status.Endpoint = fmt.Sprintf("%s.%s.svc.cluster.local:5432", paradedb.GetServiceName(), paradedb.Namespace)

	if paradedb.IsConnectionPoolingEnabled() {
		paradedb.Status.PoolerEndpoint = fmt.Sprintf("%s.%s.svc.cluster.local:5432", paradedb.GetPoolerServiceName(), paradedb.Namespace)
	}

	return r.Status().Update(ctx, paradedb)
}

// buildStatefulSet creates the StatefulSet spec for ParadeDB
func (r *ParadeDBReconciler) buildStatefulSet(paradedb *databasev1alpha1.ParadeDB) *appsv1.StatefulSet {
	labels := r.getLabels(paradedb)
	selectorLabels := r.getSelectorLabels(paradedb)
	replicas := paradedb.GetReplicas()

	// Get credentials secret name
	credentialsSecretName := paradedb.Name + "-credentials"
	if paradedb.Spec.Auth.SuperuserSecretRef != nil {
		credentialsSecretName = paradedb.Spec.Auth.SuperuserSecretRef.Name
	}

	// Build containers
	containers := []corev1.Container{
		{
			Name:  "paradedb",
			Image: paradedb.GetImage(),
			Ports: []corev1.ContainerPort{
				{
					Name:          "postgres",
					ContainerPort: 5432,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: []corev1.EnvVar{
				{
					Name: "POSTGRES_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
							Key:                  "username",
						},
					},
				},
				{
					Name: "POSTGRES_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
							Key:                  "password",
						},
					},
				},
				{
					Name:  "POSTGRES_DB",
					Value: paradedb.Spec.Auth.Database,
				},
				{
					Name:  "PGDATA",
					Value: "/var/lib/postgresql/data/pgdata",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/var/lib/postgresql/data",
				},
				{
					Name:      "config",
					MountPath: "/docker-entrypoint-initdb.d",
				},
			},
			Resources: paradedb.Spec.Resources,
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{"pg_isready", "-U", "postgres"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
				TimeoutSeconds:      5,
				FailureThreshold:    6,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{"pg_isready", "-U", "postgres"},
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				FailureThreshold:    3,
			},
		},
	}

	// Add metrics exporter sidecar if monitoring is enabled
	if paradedb.IsMonitoringEnabled() {
		metricsImage := "quay.io/prometheuscommunity/postgres-exporter:latest"
		metricsPort := int32(9187)
		if paradedb.Spec.Monitoring != nil {
			if paradedb.Spec.Monitoring.Image != "" {
				metricsImage = paradedb.Spec.Monitoring.Image
			}
			if paradedb.Spec.Monitoring.Port != 0 {
				metricsPort = paradedb.Spec.Monitoring.Port
			}
		}

		exporterContainer := corev1.Container{
			Name:  "postgres-exporter",
			Image: metricsImage,
			Ports: []corev1.ContainerPort{
				{
					Name:          "metrics",
					ContainerPort: metricsPort,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "DATA_SOURCE_URI",
					Value: "localhost:5432/" + paradedb.Spec.Auth.Database + "?sslmode=disable",
				},
				{
					Name: "DATA_SOURCE_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
							Key:                  "username",
						},
					},
				},
				{
					Name: "DATA_SOURCE_PASS",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
							Key:                  "password",
						},
					},
				},
			},
		}

		if paradedb.Spec.Monitoring != nil {
			exporterContainer.Resources = paradedb.Spec.Monitoring.Resources
		}

		containers = append(containers, exporterContainer)
	}

	// Apply container security context
	if paradedb.Spec.ContainerSecurityContext != nil {
		containers[0].SecurityContext = paradedb.Spec.ContainerSecurityContext
	}

	// Build PVC template
	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	if len(paradedb.Spec.Storage.AccessModes) > 0 {
		accessModes = paradedb.Spec.Storage.AccessModes
	}

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "data",
				Labels: labels,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: paradedb.Spec.Storage.Size,
					},
				},
				StorageClassName: paradedb.Spec.Storage.StorageClassName,
			},
		},
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paradedb.GetStatefulSetName(),
			Namespace: paradedb.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: paradedb.GetServiceName() + "-headless",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9187",
					},
				},
				Spec: corev1.PodSpec{
					Containers:       containers,
					NodeSelector:     paradedb.Spec.NodeSelector,
					Tolerations:      paradedb.Spec.Tolerations,
					Affinity:         paradedb.Spec.Affinity,
					SecurityContext:  paradedb.Spec.PodSecurityContext,
					ImagePullSecrets: []corev1.LocalObjectReference{},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: paradedb.Name + "-config",
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}

	return statefulSet
}

// buildService creates the Service spec for ParadeDB
func (r *ParadeDBReconciler) buildService(paradedb *databasev1alpha1.ParadeDB) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paradedb.GetServiceName(),
			Namespace: paradedb.Namespace,
			Labels:    r.getLabels(paradedb),
		},
		Spec: corev1.ServiceSpec{
			Selector: r.getSelectorLabels(paradedb),
			Type:     paradedb.Spec.ServiceType,
			Ports: []corev1.ServicePort{
				{
					Name:     "postgres",
					Port:     5432,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

// buildPoolerDeployment creates the PgBouncer Deployment spec
func (r *ParadeDBReconciler) buildPoolerDeployment(paradedb *databasev1alpha1.ParadeDB) *appsv1.Deployment {
	pooling := paradedb.Spec.ConnectionPooling
	image := "bitnami/pgbouncer:latest"
	if pooling.Image != "" {
		image = pooling.Image
	}

	credentialsSecretName := paradedb.Name + "-credentials"
	if paradedb.Spec.Auth.SuperuserSecretRef != nil {
		credentialsSecretName = paradedb.Spec.Auth.SuperuserSecretRef.Name
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "pgbouncer",
		"app.kubernetes.io/instance":   paradedb.Name,
		"app.kubernetes.io/component":  "pooler",
		"app.kubernetes.io/managed-by": "paradedb-operator",
	}

	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      paradedb.GetPoolerDeploymentName(),
			Namespace: paradedb.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pgbouncer",
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "pgbouncer",
									ContainerPort: 5432,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PGBOUNCER_DATABASE",
									Value: paradedb.Spec.Auth.Database,
								},
								{
									Name:  "POSTGRESQL_HOST",
									Value: paradedb.GetServiceName(),
								},
								{
									Name: "POSTGRESQL_USERNAME",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
											Key:                  "username",
										},
									},
								},
								{
									Name: "POSTGRESQL_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecretName},
											Key:                  "password",
										},
									},
								},
								{
									Name:  "PGBOUNCER_POOL_MODE",
									Value: pooling.PoolMode,
								},
								{
									Name:  "PGBOUNCER_MAX_CLIENT_CONN",
									Value: fmt.Sprintf("%d", pooling.MaxClientConnections),
								},
								{
									Name:  "PGBOUNCER_DEFAULT_POOL_SIZE",
									Value: fmt.Sprintf("%d", pooling.DefaultPoolSize),
								},
							},
							Resources: pooling.Resources,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5432),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5432),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		},
	}
}

// getLabels returns labels for ParadeDB resources
func (r *ParadeDBReconciler) getLabels(paradedb *databasev1alpha1.ParadeDB) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "paradedb",
		"app.kubernetes.io/instance":   paradedb.Name,
		"app.kubernetes.io/version":    paradedb.Spec.PostgresVersion,
		"app.kubernetes.io/component":  "database",
		"app.kubernetes.io/managed-by": "paradedb-operator",
	}
}

// getSelectorLabels returns selector labels for ParadeDB
func (r *ParadeDBReconciler) getSelectorLabels(paradedb *databasev1alpha1.ParadeDB) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "paradedb",
		"app.kubernetes.io/instance": paradedb.Name,
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *ParadeDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.ParadeDB{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Named("paradedb").
		Complete(r)
}

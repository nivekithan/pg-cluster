/*
Copyright 2025.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

// PostgresReconciler reconciles a Postgres object
type PostgresReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Postgres object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PostgresReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var postgres databasev1.Postgres

	if err := r.Get(ctx, req.NamespacedName, &postgres); err != nil {
		log.Error(err, "unable to fetch postgres")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Found Postgres instance", "postgres", postgres)

	// Validate secret exists and contains required keys
	_, err := r.fetchSecret(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to validate secret")
		return ctrl.Result{}, err
	}

	postgresPVC, err := r.ensurePVC(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure PVC")
		return ctrl.Result{}, err
	}

	log.Info("Ensured PVC", "pvcName", postgresPVC.Name)

	postgresDeployment, err := r.ensureDeployment(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Ensured Deployment", "deploymentName", postgresDeployment.Name)

	postgresService, err := r.ensureService(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure Service")
		return ctrl.Result{}, err
	}

	log.Info("Ensured Service", "serviceName", postgresService.Name)

	return ctrl.Result{}, nil
}

func (r *PostgresReconciler) ensureDeployment(ctx context.Context, postgres databasev1.Postgres) (*appsv1.Deployment, error) {
	log := logf.FromContext(ctx)
	deploymentName := fmt.Sprintf("%s-deployment", postgres.Name)

	var postgresDeployment appsv1.Deployment

	if err := r.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: deploymentName}, &postgresDeployment); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Deployment")
			return nil, err
		}

		log.Info("Deployment not found, creating new one")
	}

	if postgresDeployment.Name != "" {
		log.Info("Deployment already exists", "deploymentName", postgresDeployment.Name)
		return &postgresDeployment, nil
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
	}

	postgresDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: postgres.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
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
							Name:  "postgres",
							Image: "nivekithan/postgres-pgbackrest:latest",
							Args:  []string{"postgres"},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: postgres.Spec.PostgresSecretRef,
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Name:          "postgres",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "postgres-storage",
									MountPath: "/var/lib/postgresql/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "postgres-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-pvc", postgres.Name),
								},
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference for Deployment
	if err := controllerutil.SetControllerReference(&postgres, &postgresDeployment, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for Deployment")
		return nil, err
	}

	// Create Deployment
	if err := r.Client.Create(ctx, &postgresDeployment); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create Deployment")
			return nil, err
		}
	}

	return &postgresDeployment, nil
}

func (r *PostgresReconciler) ensureService(ctx context.Context, postgres databasev1.Postgres) (*corev1.Service, error) {
	log := logf.FromContext(ctx)
	serviceName := fmt.Sprintf("%s-service", postgres.Name)

	var postgresService corev1.Service

	if err := r.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: serviceName}, &postgresService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Service")
			return nil, err
		}

		log.Info("Service not found, creating new one")
	}

	if postgresService.Name != "" {
		log.Info("Service already exists", "serviceName", postgresService.Name)
		return &postgresService, nil
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
	}

	postgresService = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: postgres.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
					NodePort:   30432,
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Set owner reference for Service
	if err := controllerutil.SetControllerReference(&postgres, &postgresService, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for Service")
		return nil, err
	}

	// Create Service
	if err := r.Client.Create(ctx, &postgresService); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create Service")
			return nil, err
		}
	}

	return &postgresService, nil
}

func (r *PostgresReconciler) ensurePVC(ctx context.Context, postgres databasev1.Postgres) (*corev1.PersistentVolumeClaim, error) {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-pvc", postgres.Name)

	var postgresPVC corev1.PersistentVolumeClaim

	if err := r.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: pvcName}, &postgresPVC); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get PVC")
			return nil, err
		}

		log.Info("PVC not found, creating new one")
	}

	if postgresPVC.Name != "" {
		log.Info("PVC already exists", "pvcName", postgresPVC.Name)
		return &postgresPVC, nil
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
	}

	postgresPVC = corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: postgres.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: postgres.Spec.Size,
				},
			},
		},
	}

	// Set owner reference for PVC
	if err := controllerutil.SetControllerReference(&postgres, &postgresPVC, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for PVC")
		return nil, err
	}

	// Create PVC
	if err := r.Client.Create(ctx, &postgresPVC); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create PVC")
			return nil, err
		}
	}

	return &postgresPVC, nil
}

func (r *PostgresReconciler) fetchSecret(ctx context.Context, postgres databasev1.Postgres) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)

	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: postgres.Namespace,
		Name:      postgres.Spec.PostgresSecretRef,
	}, &secret); err != nil {
		log.Error(err, "Failed to fetch secret", "secretName", postgres.Spec.PostgresSecretRef)
		return nil, err
	}

	required_secrets := []string{
		// PostgreSQL Configuration
		"POSTGRES_PASSWORD",
		"POSTGRES_USER",
		"POSTGRES_DB",
		"PGDATA",

		// S3 Configuration
		"S3_BUCKET_NAME",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_ACCESS_KEY_SECRET",
		"S3_REGION",

		// pgBackRest Configuration
		"REPO1_RETENTION_FULL",
		"STANZA_NAME",
		"ARCHIVE_TIMEOUT",
		"MAX_WAL_SENDERS",
		"WAL_KEEP_SIZE",
	}

	for _, secretKeyName := range required_secrets {
		secretValue, ok := secret.Data[secretKeyName]

		if !ok || len(secretValue) == 0 {
			err := fmt.Errorf("missing required variable :%s", secretKeyName)
			log.Error(err, "secretName", postgres.Spec.PostgresSecretRef)

			return nil, err
		}

	}

	log.Info(`Validated that sercret contains all the required variables`)

	return &secret, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.Postgres{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("postgres").
		Complete(r)
}

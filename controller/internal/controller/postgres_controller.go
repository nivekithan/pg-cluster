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
	"bytes"
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
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
	Scheme    *runtime.Scheme
	Config    *rest.Config
	Clientset kubernetes.Interface
}

// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create

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

	socketPVC, err := r.ensureSocketPVC(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure socket PVC")
		return ctrl.Result{}, err
	}

	log.Info("Ensured socket PVC", "pvcName", socketPVC.Name)

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

	// Initialize StanzaCreated condition as false if not already set
	if err := r.ensureStanzaCreatedCondition(ctx, &postgres); err != nil {
		log.Error(err, "Failed to ensure StanzaCreated condition")
		return ctrl.Result{}, err
	}

	// Check if we need to create the pgbackrest stanza
	if err := r.handleStanzaCreation(ctx, &postgres, postgresDeployment); err != nil {
		log.Error(err, "Failed to handle stanza creation")
		return ctrl.Result{}, err
	}

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
								{
									Name:      "postgres-socket",
									MountPath: "/tmp/postgres",
								},
							},
						},
					},
					InitContainers: r.createSidecarContainer(postgres),
					Volumes: []corev1.Volume{
						{
							Name: "postgres-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-pvc", postgres.Name),
								},
							},
						},
						{
							Name: "postgres-socket",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-socket-pvc", postgres.Name),
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

func (r *PostgresReconciler) ensureSocketPVC(ctx context.Context, postgres databasev1.Postgres) (*corev1.PersistentVolumeClaim, error) {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-socket-pvc", postgres.Name)

	var socketPVC corev1.PersistentVolumeClaim

	if err := r.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: pvcName}, &socketPVC); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get socket PVC")
			return nil, err
		}

		log.Info("Socket PVC not found, creating new one")
	}

	if socketPVC.Name != "" {
		log.Info("Socket PVC already exists", "pvcName", socketPVC.Name)
		return &socketPVC, nil
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
		"type":     "socket",
	}

	socketPVC = corev1.PersistentVolumeClaim{
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
					corev1.ResourceStorage: resource.MustParse("16Mi"), // Small size for socket directory
				},
			},
		},
	}

	// Set owner reference for socket PVC
	if err := controllerutil.SetControllerReference(&postgres, &socketPVC, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for socket PVC")
		return nil, err
	}

	// Create socket PVC
	if err := r.Client.Create(ctx, &socketPVC); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create socket PVC")
			return nil, err
		}
	}

	return &socketPVC, nil
}

func (r *PostgresReconciler) createSidecarContainer(postgres databasev1.Postgres) []corev1.Container {
	return []corev1.Container{
		{
			Name:          "pgbackrest-sidecar",
			Image:         "nivekithan/postgres-pgbackrest:latest",
			Args:          []string{"sh", "-c", "while true; do sleep 3600; done"},
			RestartPolicy: (*corev1.ContainerRestartPolicy)(ptr.To(corev1.ContainerRestartPolicyAlways)),
			EnvFrom: []corev1.EnvFromSource{
				{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: postgres.Spec.PostgresSecretRef,
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "postgres-storage",
					MountPath: "/var/lib/postgresql/data",
				},
				{
					Name:      "postgres-socket",
					MountPath: "/tmp/postgres",
				},
			},
		},
	}
}

func (r *PostgresReconciler) handleStanzaCreation(ctx context.Context, postgres *databasev1.Postgres, deployment *appsv1.Deployment) error {
	log := logf.FromContext(ctx)

	// Check if StanzaCreated condition is already true
	for _, condition := range postgres.Status.Conditions {
		if condition.Type == "StanzaCreated" && condition.Status == metav1.ConditionTrue {
			log.Info("Stanza already created")
			return nil
		}
	}

	// Check if deployment is ready
	if deployment.Status.ReadyReplicas == 0 {
		log.Info("Deployment not ready yet, skipping stanza creation")
		return nil
	}

	// Find running pod
	podName, err := r.findRunningPod(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to find running pod")
		return err
	}

	if podName == "" {
		log.Info("No running pod found, skipping stanza creation")
		return nil
	}

	// Get stanza name from secret
	secret, err := r.fetchSecret(ctx, *postgres)
	if err != nil {
		log.Error(err, "Failed to fetch secret for stanza name")
		return err
	}

	stanzaName := string(secret.Data["STANZA_NAME"])
	if stanzaName == "" {
		err := fmt.Errorf("STANZA_NAME not found or empty in secret")
		log.Error(err, "Missing stanza name")
		return err
	}

	// Execute stanza-create command as postgres user with explicit stanza name
	log.Info("Executing pgbackrest stanza-create command as postgres user", "podName", podName, "stanzaName", stanzaName)
	err = r.execInPod(ctx, postgres.Namespace, podName, "postgres", []string{
		"su", "-", "postgres", "-c", fmt.Sprintf("pgbackrest stanza-create --stanza=%s --log-level-console=info", stanzaName),
	})

	// Update condition based on result
	return r.updateStanzaCreatedCondition(ctx, postgres, err)
}

func (r *PostgresReconciler) findRunningPod(ctx context.Context, postgres *databasev1.Postgres) (string, error) {
	log := logf.FromContext(ctx)

	var pods corev1.PodList
	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
	}

	if err := r.List(ctx, &pods, client.InNamespace(postgres.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "Failed to list pods")
		return "", err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Check if postgres container is ready
			for _, container := range pod.Status.ContainerStatuses {
				if container.Name == "postgres" && container.Ready {
					return pod.Name, nil
				}
			}
		}
	}

	return "", nil
}

func (r *PostgresReconciler) execInPod(ctx context.Context, namespace, podName, containerName string, command []string) error {
	log := logf.FromContext(ctx)

	// Create clientset if not available
	if r.Clientset == nil {
		clientset, err := kubernetes.NewForConfig(r.Config)
		if err != nil {
			log.Error(err, "Failed to create clientset")
			return err
		}
		r.Clientset = clientset
	}

	log.Info("Executing command in pod",
		"namespace", namespace,
		"pod", podName,
		"container", containerName,
		"command", strings.Join(command, " "))

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// 1. Build REST API request
	req := r.Clientset.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// 2. Create SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(r.Config, "POST", req.URL())
	if err != nil {
		log.Error(err, "Failed to create executor")
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// 3. Stream command execution
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Log the output
	if stdout.Len() > 0 {
		log.Info("Command stdout", "output", stdout.String())
	}
	if stderr.Len() > 0 {
		log.Info("Command stderr", "output", stderr.String())
	}

	if err != nil {
		log.Error(err, "Command execution failed",
			"stdout", stdout.String(),
			"stderr", stderr.String())
		return fmt.Errorf("command execution failed: %w", err)
	}

	log.Info("Command executed successfully")
	return nil
}

func (r *PostgresReconciler) updateStanzaCreatedCondition(ctx context.Context, postgres *databasev1.Postgres, execErr error) error {
	log := logf.FromContext(ctx)

	var condition metav1.Condition
	if execErr != nil {
		condition = metav1.Condition{
			Type:               "StanzaCreated",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "StanzaCreationFailed",
			Message:            fmt.Sprintf("Failed to create pgBackRest stanza: %v", execErr),
		}
	} else {
		condition = metav1.Condition{
			Type:               "StanzaCreated",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "StanzaCreationSucceeded",
			Message:            "pgBackRest stanza successfully created",
		}
	}

	// Update or replace the condition
	for i, existingCondition := range postgres.Status.Conditions {
		if existingCondition.Type == "StanzaCreated" {
			postgres.Status.Conditions[i] = condition
			break
		}
	}

	// Update status
	if err := r.Status().Update(ctx, postgres); err != nil {
		log.Error(err, "Failed to update StanzaCreated condition")
		return err
	}

	if execErr != nil {
		log.Info("Updated StanzaCreated condition to false")
	} else {
		log.Info("Updated StanzaCreated condition to true")
	}

	return nil
}

func (r *PostgresReconciler) ensureStanzaCreatedCondition(ctx context.Context, postgres *databasev1.Postgres) error {
	log := logf.FromContext(ctx)

	// Check if StanzaCreated condition already exists
	for _, condition := range postgres.Status.Conditions {
		if condition.Type == "StanzaCreated" {
			log.Info("StanzaCreated condition already exists", "status", condition.Status)
			return nil
		}
	}

	// Add StanzaCreated condition with default false status
	condition := metav1.Condition{
		Type:               "StanzaCreated",
		Status:             metav1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
		Reason:             "StanzaNotCreated",
		Message:            "pgBackRest stanza has not been created yet",
	}

	postgres.Status.Conditions = append(postgres.Status.Conditions, condition)

	// Update status
	if err := r.Status().Update(ctx, postgres); err != nil {
		log.Error(err, "Failed to update StanzaCreated condition")
		return err
	}

	log.Info("Added StanzaCreated condition with false status")
	return nil
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
		"PG1_SOCKET_PATH",
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
	// Initialize the Config for exec operations
	r.Config = mgr.GetConfig()

	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.Postgres{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("postgres").
		Complete(r)
}

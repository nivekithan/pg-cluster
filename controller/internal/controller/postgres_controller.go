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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
	"github.com/nivekithan/pg-cluster/internal/domains/backup"
	"github.com/nivekithan/pg-cluster/internal/domains/ipc"
	"github.com/nivekithan/pg-cluster/internal/domains/pki"
	postgresdomain "github.com/nivekithan/pg-cluster/internal/domains/postgres"
	"github.com/nivekithan/pg-cluster/internal/shared"
)

// PostgresReconciler reconciles a Postgres object
type PostgresReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    *rest.Config
	Clientset kubernetes.Interface

	// Domain instances
	PostgresDomain *postgresdomain.Domain
	BackupDomain   *backup.Domain
	PKIDomain      *pki.Domain
	IPCDomain      *ipc.Domain
}

// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.kube.nivekithan.com,resources=postgres/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PostgresReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var postgres databasev1.Postgres

	if err := r.Get(ctx, req.NamespacedName, &postgres); err != nil {
		log.Error(err, "unable to fetch postgres")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Found Postgres instance", "postgres", postgres)

	// Validate PostgreSQL secret
	_, err := r.PostgresDomain.ValidateSecret(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to validate secret")
		return ctrl.Result{}, err
	}

	// Ensure PKI certificates
	caSecret, err := r.PKIDomain.EnsureCASecret(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure CA secret")
		return ctrl.Result{}, err
	}
	log.Info("Ensured CA secret", "secretName", caSecret.Name)

	tlsSecret, err := r.PKIDomain.EnsureTLSSecret(ctx, postgres, caSecret)
	if err != nil {
		log.Error(err, "Failed to ensure pgBackRest TLS secret")
		return ctrl.Result{}, err
	}
	log.Info("Ensured pgBackRest TLS secret", "secretName", tlsSecret.Name)

	// Ensure storage resources
	postgresPVC, err := r.PostgresDomain.EnsureDataPVC(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure PVC")
		return ctrl.Result{}, err
	}
	log.Info("Ensured PVC", "pvcName", postgresPVC.Name)

	socketPVC, err := r.IPCDomain.EnsureSocketPVC(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure socket PVC")
		return ctrl.Result{}, err
	}
	log.Info("Ensured socket PVC", "pvcName", socketPVC.Name)

	// Prepare all volumes and sidecars before creating the deployment
	pgVolumes := r.PostgresDomain.GetDataVolumes(postgres)
	socketVolumes := r.IPCDomain.GetSocketVolumes(postgres)
	tlsVolumes := r.BackupDomain.GetTLSVolumes(postgres)
	allVolumes := append(pgVolumes, socketVolumes...)
	allVolumes = append(allVolumes, tlsVolumes...)

	sidecars := r.BackupDomain.CreateSidecarContainers(postgres)

	// Create the main PostgreSQL deployment with all configurations
	postgresDeployment, err := r.PostgresDomain.EnsureDeployment(ctx, postgres, allVolumes, sidecars)
	if err != nil {
		log.Error(err, "Failed to ensure Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Ensured Deployment", "deploymentName", postgresDeployment.Name)

	// Ensure PostgreSQL service
	postgresService, err := r.PostgresDomain.EnsureService(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure Service")
		return ctrl.Result{}, err
	}
	log.Info("Ensured Service", "serviceName", postgresService.Name)

	// Initialize and handle backup stanza
	if err := r.BackupDomain.EnsureStanzaCondition(ctx, &postgres); err != nil {
		log.Error(err, "Failed to ensure StanzaCreated condition")
		return ctrl.Result{}, err
	}

	if err := r.BackupDomain.HandleStanzaCreation(ctx, &postgres, postgresDeployment); err != nil {
		log.Error(err, "Failed to handle stanza creation")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager and initializes domains.
func (r *PostgresReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the Config for exec operations
	r.Config = mgr.GetConfig()

	// Initialize domain context
	domainCtx := shared.DomainContext{
		Client:    r.Client,
		Clientset: r.Clientset,
		Config:    r.Config,
	}

	// Initialize domain instances
	r.PostgresDomain = postgresdomain.New(domainCtx)
	r.BackupDomain = backup.New(domainCtx)
	r.PKIDomain = pki.New(domainCtx)
	r.IPCDomain = ipc.New(domainCtx)

	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.Postgres{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Named("postgres").
		Complete(r)
}

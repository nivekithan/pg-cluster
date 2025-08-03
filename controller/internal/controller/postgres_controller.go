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

	postgresDeployment, err := r.ensureDeployment(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to ensure Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Ensured Deployment", "deploymentName", postgresDeployment.Name)

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
							Image: "postgres:15",
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRES_DB",
									Value: "postgres",
								},
								{
									Name:  "POSTGRES_USER",
									Value: "postgres",
								},
								{
									Name:  "POSTGRES_PASSWORD",
									Value: "password",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Name:          "postgres",
									Protocol:      corev1.ProtocolTCP,
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

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.Postgres{}).
		Owns(&appsv1.Deployment{}).
		Named("postgres").
		Complete(r)
}

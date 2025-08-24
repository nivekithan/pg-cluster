package postgres

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) EnsureDeployment(ctx context.Context, postgres databasev1.Postgres, volumes []corev1.Volume, sidecars []corev1.Container) (*appsv1.Deployment, error) {
	log := logf.FromContext(ctx)
	deploymentName := fmt.Sprintf("%s-deployment", postgres.Name)

	var postgresDeployment appsv1.Deployment

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: deploymentName}, &postgresDeployment); err != nil {
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
					InitContainers: sidecars,
					Volumes:        volumes,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&postgres, &postgresDeployment, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for Deployment")
		return nil, err
	}

	if err := d.Client.Create(ctx, &postgresDeployment); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create Deployment")
			return nil, err
		}
	}

	return &postgresDeployment, nil
}

// GetDataVolumes returns the postgres data volume configuration
func (d *Domain) GetDataVolumes(postgres databasev1.Postgres) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "postgres-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-pvc", postgres.Name),
				},
			},
		},
	}
}

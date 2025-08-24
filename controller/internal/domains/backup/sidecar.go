package backup

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) CreateSidecarContainers(postgres databasev1.Postgres) []corev1.Container {
	return []corev1.Container{
		{
			Name:          "pgbackrest-sidecar",
			Image:         "nivekithan/postgres-pgbackrest:latest",
			Args:          []string{"sh", "-c", "while true; do sleep 3600; done"},
			RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
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
				{
					Name:      "pgbackrest-tls",
					MountPath: "/etc/pgbackrest/server",
					ReadOnly:  true,
				},
			},
		},
	}
}

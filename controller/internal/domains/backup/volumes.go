package backup

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) GetTLSVolumes(postgres databasev1.Postgres) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "pgbackrest-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-pgbackrest-sidecare-tls", postgres.Name),
					Items: []corev1.KeyToPath{
						{
							Key:  "tls.crt",
							Path: "server-tls.crt",
						},
						{
							Key:  "tls.key",
							Path: "server-tls.key",
							Mode: ptr.To(int32(0600)),
						},
						{
							Key:  "ca.crt",
							Path: "ca.crt",
						},
					},
				},
			},
		},
	}
}

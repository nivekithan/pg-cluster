package ipc

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
	"github.com/nivekithan/pg-cluster/internal/shared"
)

type Domain struct {
	shared.DomainContext
}

func New(ctx shared.DomainContext) *Domain {
	return &Domain{DomainContext: ctx}
}

func (d *Domain) EnsureSocketPVC(ctx context.Context, postgres databasev1.Postgres) (*corev1.PersistentVolumeClaim, error) {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-socket-pvc", postgres.Name)

	var socketPVC corev1.PersistentVolumeClaim

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: pvcName}, &socketPVC); err != nil {
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
					corev1.ResourceStorage: resource.MustParse("16Mi"),
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&postgres, &socketPVC, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for socket PVC")
		return nil, err
	}

	if err := d.Client.Create(ctx, &socketPVC); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create socket PVC")
			return nil, err
		}
	}

	return &socketPVC, nil
}

func (d *Domain) GetSocketVolumes(postgres databasev1.Postgres) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "postgres-socket",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-socket-pvc", postgres.Name),
				},
			},
		},
	}
}

func (d *Domain) GetSocketVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "postgres-socket",
			MountPath: "/tmp/postgres",
		},
	}
}

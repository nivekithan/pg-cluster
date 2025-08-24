package postgres

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) EnsureDataPVC(ctx context.Context, postgres databasev1.Postgres) (*corev1.PersistentVolumeClaim, error) {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-pvc", postgres.Name)

	var postgresPVC corev1.PersistentVolumeClaim

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: pvcName}, &postgresPVC); err != nil {
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

	if err := controllerutil.SetControllerReference(&postgres, &postgresPVC, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for PVC")
		return nil, err
	}

	if err := d.Client.Create(ctx, &postgresPVC); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create PVC")
			return nil, err
		}
	}

	return &postgresPVC, nil
}

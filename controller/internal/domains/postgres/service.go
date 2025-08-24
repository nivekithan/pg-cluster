package postgres

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) EnsureService(ctx context.Context, postgres databasev1.Postgres) (*corev1.Service, error) {
	log := logf.FromContext(ctx)
	serviceName := fmt.Sprintf("%s-service", postgres.Name)

	var postgresService corev1.Service

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: serviceName}, &postgresService); err != nil {
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

	if err := controllerutil.SetControllerReference(&postgres, &postgresService, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for Service")
		return nil, err
	}

	if err := d.Client.Create(ctx, &postgresService); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create Service")
			return nil, err
		}
	}

	return &postgresService, nil
}

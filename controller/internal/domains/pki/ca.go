package pki

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
	"github.com/nivekithan/pg-cluster/internal/pki"
	"github.com/nivekithan/pg-cluster/internal/shared"
)

type Domain struct {
	shared.DomainContext
}

func New(ctx shared.DomainContext) *Domain {
	return &Domain{DomainContext: ctx}
}

func (d *Domain) EnsureCASecret(ctx context.Context, postgres databasev1.Postgres) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)
	secretName := fmt.Sprintf("%s-cluster-ca-cert", postgres.Name)

	var caSecret corev1.Secret

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: secretName}, &caSecret); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get CA secret")
			return nil, err
		}

		log.Info("CA secret not found, creating new one")
	}

	if caSecret.Name != "" {
		log.Info("CA secret already exists", "secretName", caSecret.Name)
		return &caSecret, nil
	}

	privateKey, err := pki.GeneratePrivateKeyForCertificate()
	if err != nil {
		log.Error(err, "Failed to generate private key for CA")
		return nil, fmt.Errorf("failed to generate private key for CA: %w", err)
	}

	certificate, err := pki.GenerateRootCerificate(privateKey)
	if err != nil {
		log.Error(err, "Failed to generate root certificate")
		return nil, fmt.Errorf("failed to generate root certificate: %w", err)
	}

	certPEM, err := pki.MarshalCertificateToPEM(certificate)
	if err != nil {
		log.Error(err, "Failed to marshal certificate to PEM")
		return nil, fmt.Errorf("failed to marshal certificate to PEM: %w", err)
	}

	keyPEM, err := pki.MarshalPrivateKeyToPEM(privateKey)
	if err != nil {
		log.Error(err, "Failed to marshal private key to PEM")
		return nil, fmt.Errorf("failed to marshal private key to PEM: %w", err)
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
		"type":     "ca-cert",
	}

	caSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: postgres.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"root.crt": certPEM,
			"root.key": keyPEM,
		},
	}

	if err := controllerutil.SetControllerReference(&postgres, &caSecret, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for CA secret")
		return nil, err
	}

	if err := d.Client.Create(ctx, &caSecret); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create CA secret")
			return nil, err
		}
	}

	log.Info("Successfully created CA secret", "secretName", secretName)
	return &caSecret, nil
}

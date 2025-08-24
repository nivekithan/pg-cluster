package pki

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
	"github.com/nivekithan/pg-cluster/internal/pki"
)

func (d *Domain) EnsureTLSSecret(ctx context.Context, postgres databasev1.Postgres, caSecret *corev1.Secret) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)
	secretName := fmt.Sprintf("%s-pgbackrest-sidecare-tls", postgres.Name)

	var tlsSecret corev1.Secret

	if err := d.Client.Get(ctx, types.NamespacedName{Namespace: postgres.Namespace, Name: secretName}, &tlsSecret); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get pgBackRest TLS secret")
			return nil, err
		}

		log.Info("pgBackRest TLS secret not found, creating new one")
	}

	if tlsSecret.Name != "" {
		log.Info("pgBackRest TLS secret already exists", "secretName", tlsSecret.Name)
		return &tlsSecret, nil
	}

	caCertPEM := caSecret.Data["root.crt"]
	caKeyPEM := caSecret.Data["root.key"]

	if len(caCertPEM) == 0 || len(caKeyPEM) == 0 {
		err := fmt.Errorf("CA certificate or private key is missing from CA secret")
		log.Error(err, "Invalid CA secret data")
		return nil, err
	}

	caCert, caKey, err := d.parseCACertificateAndKey(caCertPEM, caKeyPEM)
	if err != nil {
		log.Error(err, "Failed to parse CA certificate and key")
		return nil, fmt.Errorf("failed to parse CA certificate and key: %w", err)
	}

	leafPrivateKey, err := pki.GeneratePrivateKeyForCertificate()
	if err != nil {
		log.Error(err, "Failed to generate private key for leaf certificate")
		return nil, fmt.Errorf("failed to generate private key for leaf certificate: %w", err)
	}

	dnsNames := d.generatePgBackRestDNSNames(postgres)
	commonName := dnsNames[0]

	leafCert, err := pki.GenerateLeafCertificate(
		caCert,
		caKey,
		&leafPrivateKey.PublicKey,
		commonName,
		dnsNames,
	)
	if err != nil {
		log.Error(err, "Failed to generate leaf certificate")
		return nil, fmt.Errorf("failed to generate leaf certificate: %w", err)
	}

	leafCertPEM, err := pki.MarshalCertificateToPEM(leafCert)
	if err != nil {
		log.Error(err, "Failed to marshal leaf certificate to PEM")
		return nil, fmt.Errorf("failed to marshal leaf certificate to PEM: %w", err)
	}

	leafKeyPEM, err := pki.MarshalPrivateKeyToPEM(leafPrivateKey)
	if err != nil {
		log.Error(err, "Failed to marshal leaf private key to PEM")
		return nil, fmt.Errorf("failed to marshal leaf private key to PEM: %w", err)
	}

	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
		"type":     "pgbackrest-tls",
	}

	tlsSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: postgres.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": leafCertPEM,
			"tls.key": leafKeyPEM,
			"ca.crt":  caCertPEM,
		},
	}

	if err := controllerutil.SetControllerReference(&postgres, &tlsSecret, d.Client.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference for pgBackRest TLS secret")
		return nil, err
	}

	if err := d.Client.Create(ctx, &tlsSecret); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			log.Error(err, "Failed to create pgBackRest TLS secret")
			return nil, err
		}
	}

	log.Info("Successfully created pgBackRest TLS secret", "secretName", secretName)
	return &tlsSecret, nil
}

func (d *Domain) parseCACertificateAndKey(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA private key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	return caCert, caKey, nil
}

func (d *Domain) generatePgBackRestDNSNames(postgres databasev1.Postgres) []string {
	serviceName := fmt.Sprintf("%s-service", postgres.Name)

	return []string{
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, postgres.Namespace),
		fmt.Sprintf("%s.%s.svc", serviceName, postgres.Namespace),
		fmt.Sprintf("%s.%s", serviceName, postgres.Namespace),
		serviceName,
		"localhost",
		"127.0.0.1",
	}
}

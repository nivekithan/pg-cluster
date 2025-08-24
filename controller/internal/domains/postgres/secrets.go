package postgres

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) ValidateSecret(ctx context.Context, postgres databasev1.Postgres) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)

	var secret corev1.Secret
	if err := d.Client.Get(ctx, types.NamespacedName{
		Namespace: postgres.Namespace,
		Name:      postgres.Spec.PostgresSecretRef,
	}, &secret); err != nil {
		log.Error(err, "Failed to fetch secret", "secretName", postgres.Spec.PostgresSecretRef)
		return nil, err
	}

	required_secrets := []string{
		// PostgreSQL Configuration
		"POSTGRES_PASSWORD",
		"POSTGRES_USER",
		"POSTGRES_DB",
		"PGDATA",

		// S3 Configuration
		"S3_BUCKET_NAME",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_ACCESS_KEY_SECRET",
		"S3_REGION",

		// pgBackRest Configuration
		"REPO1_RETENTION_FULL",
		"STANZA_NAME",
		"ARCHIVE_TIMEOUT",
		"MAX_WAL_SENDERS",
		"WAL_KEEP_SIZE",
		"PG1_SOCKET_PATH",
	}

	for _, secretKeyName := range required_secrets {
		secretValue, ok := secret.Data[secretKeyName]

		if !ok || len(secretValue) == 0 {
			err := fmt.Errorf("missing required variable :%s", secretKeyName)
			log.Error(err, "secretName", postgres.Spec.PostgresSecretRef)

			return nil, err
		}

	}

	log.Info("Validated that secret contains all the required variables")

	return &secret, nil
}

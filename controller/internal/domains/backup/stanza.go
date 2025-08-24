package backup

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

const (
	StanzaCreatedCondition = "StanzaCreated"
)

func (d *Domain) HandleStanzaCreation(ctx context.Context, postgres *databasev1.Postgres, deployment *appsv1.Deployment) error {
	log := logf.FromContext(ctx)

	// Check if StanzaCreated condition is already true
	for _, condition := range postgres.Status.Conditions {
		if condition.Type == StanzaCreatedCondition && condition.Status == metav1.ConditionTrue {
			log.Info("Stanza already created")
			return nil
		}
	}

	// Check if deployment is ready
	if deployment.Status.ReadyReplicas == 0 {
		log.Info("Deployment not ready yet, skipping stanza creation")
		return nil
	}

	// Find running pod
	podName, err := d.findRunningPod(ctx, postgres)
	if err != nil {
		log.Error(err, "Failed to find running pod")
		return err
	}

	if podName == "" {
		log.Info("No running pod found, skipping stanza creation")
		return nil
	}

	// Get stanza name from secret
	secret, err := d.getStanzaSecret(ctx, *postgres)
	if err != nil {
		log.Error(err, "Failed to fetch secret for stanza name")
		return err
	}

	stanzaName := string(secret.Data["STANZA_NAME"])
	if stanzaName == "" {
		err := fmt.Errorf("STANZA_NAME not found or empty in secret")
		log.Error(err, "Missing stanza name")
		return err
	}

	// Execute stanza-create command
	log.Info("Executing pgbackrest stanza-create command as postgres user", "podName", podName, "stanzaName", stanzaName)
	err = d.execInPod(ctx, postgres.Namespace, podName, "postgres", []string{
		"su", "-", "postgres", "-c", fmt.Sprintf("pgbackrest stanza-create --stanza=%s --log-level-console=info", stanzaName),
	})

	// Update condition based on result
	return d.updateStanzaCreatedCondition(ctx, postgres, err)
}

func (d *Domain) EnsureStanzaCondition(ctx context.Context, postgres *databasev1.Postgres) error {
	log := logf.FromContext(ctx)

	// Check if StanzaCreated condition already exists
	for _, condition := range postgres.Status.Conditions {
		if condition.Type == StanzaCreatedCondition {
			log.Info("StanzaCreated condition already exists", "status", condition.Status)
			return nil
		}
	}

	// Add StanzaCreated condition with default false status
	condition := metav1.Condition{
		Type:               StanzaCreatedCondition,
		Status:             metav1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
		Reason:             "StanzaNotCreated",
		Message:            "pgBackRest stanza has not been created yet",
	}

	postgres.Status.Conditions = append(postgres.Status.Conditions, condition)

	// Update status
	if err := d.Client.Status().Update(ctx, postgres); err != nil {
		log.Error(err, "Failed to update StanzaCreated condition")
		return err
	}

	log.Info("Added StanzaCreated condition with false status")
	return nil
}

func (d *Domain) updateStanzaCreatedCondition(ctx context.Context, postgres *databasev1.Postgres, execErr error) error {
	log := logf.FromContext(ctx)

	var condition metav1.Condition
	if execErr != nil {
		condition = metav1.Condition{
			Type:               StanzaCreatedCondition,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "StanzaCreationFailed",
			Message:            fmt.Sprintf("Failed to create pgBackRest stanza: %v", execErr),
		}
	} else {
		condition = metav1.Condition{
			Type:               StanzaCreatedCondition,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "StanzaCreationSucceeded",
			Message:            "pgBackRest stanza successfully created",
		}
	}

	// Update or replace the condition
	for i, existingCondition := range postgres.Status.Conditions {
		if existingCondition.Type == StanzaCreatedCondition {
			postgres.Status.Conditions[i] = condition
			break
		}
	}

	// Update status
	if err := d.Client.Status().Update(ctx, postgres); err != nil {
		log.Error(err, "Failed to update StanzaCreated condition")
		return err
	}

	if execErr != nil {
		log.Info("Updated StanzaCreated condition to false")
	} else {
		log.Info("Updated StanzaCreated condition to true")
	}

	return nil
}

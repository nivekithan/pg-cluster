package backup

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/nivekithan/pg-cluster/api/v1"
)

func (d *Domain) findRunningPod(ctx context.Context, postgres *databasev1.Postgres) (string, error) {
	log := logf.FromContext(ctx)

	var pods corev1.PodList
	labels := map[string]string{
		"app":      "postgres",
		"instance": postgres.Name,
	}

	if err := d.Client.List(ctx, &pods, client.InNamespace(postgres.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "Failed to list pods")
		return "", err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Check if postgres container is ready
			for _, container := range pod.Status.ContainerStatuses {
				if container.Name == "postgres" && container.Ready {
					return pod.Name, nil
				}
			}
		}
	}

	return "", nil
}

func (d *Domain) execInPod(ctx context.Context, namespace, podName, containerName string, command []string) error {
	log := logf.FromContext(ctx)

	// Create clientset if not available
	if d.Clientset == nil {
		clientset, err := kubernetes.NewForConfig(d.Config)
		if err != nil {
			log.Error(err, "Failed to create clientset")
			return err
		}
		d.Clientset = clientset
	}

	log.Info("Executing command in pod",
		"namespace", namespace,
		"pod", podName,
		"container", containerName,
		"command", strings.Join(command, " "))

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// 1. Build REST API request
	req := d.Clientset.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// 2. Create SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(d.Config, "POST", req.URL())
	if err != nil {
		log.Error(err, "Failed to create executor")
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// 3. Stream command execution
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Log the output
	if stdout.Len() > 0 {
		log.Info("Command stdout", "output", stdout.String())
	}
	if stderr.Len() > 0 {
		log.Info("Command stderr", "output", stderr.String())
	}

	if err != nil {
		log.Error(err, "Command execution failed",
			"stdout", stdout.String(),
			"stderr", stderr.String())
		return fmt.Errorf("command execution failed: %w", err)
	}

	log.Info("Command executed successfully")
	return nil
}

func (d *Domain) getStanzaSecret(ctx context.Context, postgres databasev1.Postgres) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)

	var secret corev1.Secret
	if err := d.Client.Get(ctx, types.NamespacedName{
		Namespace: postgres.Namespace,
		Name:      postgres.Spec.PostgresSecretRef,
	}, &secret); err != nil {
		log.Error(err, "Failed to fetch secret", "secretName", postgres.Spec.PostgresSecretRef)
		return nil, err
	}

	return &secret, nil
}

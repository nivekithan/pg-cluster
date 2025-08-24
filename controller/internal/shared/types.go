package shared

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DomainContext holds common dependencies for all domains
type DomainContext struct {
	Client    client.Client
	Clientset kubernetes.Interface
	Config    *rest.Config
}

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresSpec defines the desired state of Postgres
type PostgresSpec struct {
	Size         string   `json:"size"`
	DatabaseList []string `json:"databaseList"`
}

// PostgresStatus defines the observed state of Postgres.
type PostgresStatus struct {
	// Conditions represent the latest available observations of the Postgres state
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// DeploymentName is the name of the PostgreSQL deployment
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// Replicas is the desired number of replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// ServiceName is the name of the PostgreSQL service
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// ServiceType is the type of the service (NodePort, LoadBalancer, etc)
	// +optional
	ServiceType string `json:"serviceType,omitempty"`

	// NodePort is the node port assigned (for NodePort services)
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`

	// ConnectionPort is the port to connect to PostgreSQL
	// +optional
	ConnectionPort int32 `json:"connectionPort,omitempty"`

	// DatabaseName is the default database name
	// +optional
	DatabaseName string `json:"databaseName,omitempty"`

	// Username is the default username
	// +optional
	Username string `json:"username,omitempty"`

	// ManagedDatabaseList contains the list of databases currently managed
	// +optional
	ManagedDatabaseList []string `json:"managedDatabaseList,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Postgres spec
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Postgres is the Schema for the postgres API
type Postgres struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Postgres
	// +required
	Spec PostgresSpec `json:"spec"`

	// status defines the observed state of Postgres
	// +optional
	Status PostgresStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// PostgresList contains a list of Postgres
type PostgresList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Postgres `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Postgres{}, &PostgresList{})
}

/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// ClusterFinalizer allows cleaning up resources associated with
	// VpsieCluster before removing it from the apiserver.
	ClusterFinalizer = "vpsiecluster.infrastructure.cluster.x-k8s.io"
)

// VpsieClusterSpec defines the desired state of VpsieCluster
type VpsieClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Project is the name of the project to deploy the cluster to.
	Project string `json:"project"`

	// The Vpsie Region the cluster lives in.
	// +optional
	DcIdentifier string `json:"region"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// NetworkSpec encapsulates all things related to Vpsie network.
	// +optional
	Network NetworkSpec `json:"network"`
}

// VpsieClusterStatus defines the observed state of VpsieCluster
type VpsieClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready indicates that the cluster is ready.
	// +optional
	// +kubebuilder:default=falctrl.SetupSignalHandler()se
	Ready bool `json:"ready"`

	// Network encapsulates all things related to Vpsie network.
	// +optional
	Network VpsieNetworkResource `json:"network,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VpsieCluster is the Schema for the vpsieclusters API
type VpsieCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VpsieClusterSpec   `json:"spec,omitempty"`
	Status VpsieClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VpsieClusterList contains a list of VpsieCluster
type VpsieClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VpsieCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VpsieCluster{}, &VpsieClusterList{})
}

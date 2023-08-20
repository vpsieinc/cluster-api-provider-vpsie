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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/errors"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// MachineFinalizer allows cleaning up resources associated with
	// VpsieMachine before removing it from the API Server.
	MachineFinalizer = "vpsiemachine.infrastructure.cluster.x-k8s.io"
)

// VpsieMachineSpec defines the desired state of VpsieMachine
type VpsieMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ProviderID is the identifier for the VpsieMachine instance
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// +optional
	VpsiePlan *string `json:"vpsiePlan,omitempty"`

	// +optional
	OsIdentifier *string `json:"osIdentifier,omitempty"`

	// +optional
	Storage []AdditionalStorage `json:"storage,omitempty"`

	// +optional
	SSHKeys []string `json:"sshKeys,omitempty"`
}

// VpsieMachineStatus defines the observed state of VpsieMachine
type VpsieMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready indicates the vpsie infrastructure has been provisioned and is ready
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the associated addresses for the vpsie instance.
	// +optional
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceStatus is the status of the DigitalOcean droplet instance for this machine.
	// +optional
	InstanceStatus *VpsieInstanceStatus `json:"instanceStatus,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VpsieMachine is the Schema for the vpsiemachines API
type VpsieMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VpsieMachineSpec   `json:"spec,omitempty"`
	Status VpsieMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VpsieMachineList contains a list of VpsieMachine
type VpsieMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VpsieMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VpsieMachine{}, &VpsieMachineList{})
}

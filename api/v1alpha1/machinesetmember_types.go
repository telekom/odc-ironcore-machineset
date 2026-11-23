/*
Copyright 2026 Deutsche Telekom AG.

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
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MemberLifecycle is the lifecycle of a member.
type MemberLifecycle string

const (
	// MemberLifecycleActive means the member is requested and should have a Machine.
	MemberLifecycleActive MemberLifecycle = "Active"
	// MemberLifecycleRetained means the member was scaled down with Retain: it keeps its
	// resources without a Machine.
	MemberLifecycleRetained MemberLifecycle = "Retained"
	// MemberLifecycleReclaiming means the member and its owned resources are being deleted.
	MemberLifecycleReclaiming MemberLifecycle = "Reclaiming"
)

// ResourceOwnership tells whether a member created a resource or uses an existing one.
type ResourceOwnership string

const (
	// ResourceOwnershipManaged means the member created the resource and deletes it when the
	// member is reclaimed.
	ResourceOwnershipManaged ResourceOwnership = "Managed"
	// ResourceOwnershipExternal means the resource was referenced; it is released, never
	// deleted.
	ResourceOwnershipExternal ResourceOwnership = "External"
)

// OperationType is the kind of work the controller is doing on a member.
type OperationType string

const (
	// OperationProvisioning creates the member's resources and its first Machine.
	OperationProvisioning OperationType = "Provisioning"
	// OperationPowerChange changes the power state of the existing Machine.
	OperationPowerChange OperationType = "PowerChange"
	// OperationRestoring creates a successor Machine after the previous one was deleted by someone
	// else, for example by eviction, and its deletion completed.
	OperationRestoring OperationType = "Restoring"
	// OperationReactivating creates a Machine for a retained member that is requested again.
	OperationReactivating OperationType = "Reactivating"
	// OperationRetiring removes the Machine of a member scaled down with Retain.
	OperationRetiring OperationType = "Retiring"
	// OperationReclaiming deletes the member's Machine, owned resources and the member.
	OperationReclaiming OperationType = "Reclaiming"
)

// MachineSetMemberSpec defines the desired state of MachineSetMember. It is written by the
// MachineSet controller. Only lifecycle can change after creation.
type MachineSetMemberSpec struct {
	// machineSetRef is the MachineSet the member belongs to, by name and UID.
	// +required
	MachineSetRef LocalUIDReference `json:"machineSetRef"`

	// ordinal is the member's position in the set: 0, 1, ...
	// +required
	Ordinal int32 `json:"ordinal"`

	// lifecycle of the member.
	// +optional
	// +kubebuilder:default=Active
	Lifecycle MemberLifecycle `json:"lifecycle,omitempty"`
}

// MachineSetMemberStatus defines the observed state of MachineSetMember. It holds no keys or
// credentials.
type MachineSetMemberStatus struct {
	// observedGeneration is the generation of the spec the controller last acted on.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// machineRef is the member's current Machine, by name and UID.
	// +optional
	MachineRef *LocalUIDReference `json:"machineRef,omitempty"`

	// machineState is a copy of the current Machine's status.state. It shows, for example, a
	// shutdown from inside the guest without looking up the Machine.
	// +optional
	MachineState computev1alpha1.MachineState `json:"machineState,omitempty"`

	// volumes are the member's Volumes, bound by UID.
	// +optional
	// +listType=map
	// +listMapKey=name
	Volumes []MemberVolumeStatus `json:"volumes,omitempty"`

	// networkInterfaces are the member's NetworkInterfaces, bound by UID.
	// +optional
	// +listType=map
	// +listMapKey=name
	NetworkInterfaces []MemberNetworkInterfaceStatus `json:"networkInterfaces,omitempty"`

	// operation is the work the controller is currently doing on the member.
	// +optional
	Operation *MemberOperation `json:"operation,omitempty"`

	// conditions of the member: ResourcesReady, MachineScheduled, PoolReachable and Ready.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MemberVolumeStatus is the observed state of one of the member's Volumes.
type MemberVolumeStatus struct {
	// name of the volume attachment in the template.
	// +required
	Name string `json:"name"`

	// ownership tells whether the member created the Volume or uses an existing one.
	// +required
	Ownership ResourceOwnership `json:"ownership"`

	// volumeRef is the Volume, by name and UID.
	// +required
	VolumeRef LocalUIDReference `json:"volumeRef"`

	// state is the Volume's status.state.
	// +optional
	State storagev1alpha1.VolumeState `json:"state,omitempty"`

	// size is the Volume's storage capacity.
	// +optional
	Size *resource.Quantity `json:"size,omitempty"`
}

// MemberNetworkInterfaceStatus is the observed state of one of the member's NetworkInterfaces.
type MemberNetworkInterfaceStatus struct {
	// name of the network interface attachment in the template.
	// +required
	Name string `json:"name"`

	// ownership tells whether the member created the NetworkInterface or uses an existing one.
	// +required
	Ownership ResourceOwnership `json:"ownership"`

	// networkInterfaceRef is the NetworkInterface, by name and UID.
	// +required
	NetworkInterfaceRef LocalUIDReference `json:"networkInterfaceRef"`

	// state is the NetworkInterface's status.state.
	// +optional
	State networkingv1alpha1.NetworkInterfaceState `json:"state,omitempty"`

	// ips are the NetworkInterface's IP addresses.
	// +optional
	// +listType=atomic
	IPs []string `json:"ips,omitempty"`

	// prefixes are the NetworkInterface's prefixes.
	// +optional
	// +listType=atomic
	Prefixes []string `json:"prefixes,omitempty"`

	// virtualIP is the NetworkInterface's VirtualIP address.
	// +optional
	VirtualIP *string `json:"virtualIP,omitempty"`
}

// MemberOperation is the work the controller is currently doing on a member.
type MemberOperation struct {
	// type of the operation.
	// +required
	Type OperationType `json:"type"`

	// phase is the current step of the operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// predecessorMachineRef is the Machine being replaced, by name and UID.
	// +optional
	PredecessorMachineRef *LocalUIDReference `json:"predecessorMachineRef,omitempty"`

	// startTime is when the operation started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Ordinal",type=integer,JSONPath=".spec.ordinal"
// +kubebuilder:printcolumn:name="Lifecycle",type=string,JSONPath=".spec.lifecycle"
// +kubebuilder:printcolumn:name="Machine",type=string,JSONPath=".status.machineRef.name"
// +kubebuilder:printcolumn:name="Machine State",type=string,JSONPath=".status.machineState"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// MachineSetMember is the durable record of one member of a MachineSet. It owns the member's
// Volumes and NetworkInterfaces and points to its current, replaceable Machine. It is created by
// the MachineSet controller; users can read and delete it.
type MachineSetMember struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of MachineSetMember
	// +required
	Spec MachineSetMemberSpec `json:"spec"`

	// status defines the observed state of MachineSetMember
	// +optional
	Status MachineSetMemberStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MachineSetMemberList contains a list of MachineSetMember
type MachineSetMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MachineSetMember `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &MachineSetMember{}, &MachineSetMemberList{})
		return nil
	})
}

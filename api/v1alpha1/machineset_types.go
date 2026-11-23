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
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceRetentionPolicyType tells what happens to a member's resources when it is scaled down.
type ResourceRetentionPolicyType string

const (
	// RetentionPolicyRetain removes the member's Machine but keeps the member and its resources.
	RetentionPolicyRetain ResourceRetentionPolicyType = "Retain"
	// RetentionPolicyDelete removes the member's Machine, its owned resources and the member.
	RetentionPolicyDelete ResourceRetentionPolicyType = "Delete"
)

// MachineSetSpec defines the desired state of MachineSet.
//
// Only replicas, template.spec.power and resourceRetentionPolicy can change after creation.
//
// Single-member-only inputs are exclusive to one Machine, so they are accepted only while
// replicas is at most 1, also on a later scale-up. References from attachments to templates must
// resolve, and each template is attached once.
type MachineSetSpec struct {
	// replicas is the number of members. Powered-off members count.
	// +optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// template describes the Machine of every member.
	// +required
	Template MachineSetTemplate `json:"template"`

	// volumeTemplates are named templates for the Volumes created for each member. A Volume
	// created from a template belongs to the member and survives replacement of its Machine.
	// Immutable.
	// +optional
	// +listType=atomic
	VolumeTemplates []VolumeTemplate `json:"volumeTemplates,omitempty"`

	// networkInterfaceTemplates are named templates for the NetworkInterfaces created for each
	// member. A NetworkInterface created from a template belongs to the member and keeps its
	// IPs and VirtualIP across replacement of the member's Machine. Immutable.
	// +optional
	// +listType=atomic
	NetworkInterfaceTemplates []NetworkInterfaceTemplate `json:"networkInterfaceTemplates,omitempty"`

	// resourceRetentionPolicy tells what happens to members' resources on scale-down.
	// +optional
	// +kubebuilder:default={}
	ResourceRetentionPolicy ResourceRetentionPolicy `json:"resourceRetentionPolicy,omitzero"`
}

// ResourceRetentionPolicy tells what happens to members' resources.
type ResourceRetentionPolicy struct {
	// whenScaledDown applies when a member leaves the requested range. Retain keeps the member
	// and its resources without a Machine; Delete removes them. Changing it later does not
	// affect members that are already retained.
	// +optional
	// +kubebuilder:default=Retain
	WhenScaledDown ResourceRetentionPolicyType `json:"whenScaledDown,omitempty"`
}

// MachineSetTemplate describes the Machine of every member.
type MachineSetTemplate struct {
	// metadata is copied onto every Machine.
	// +optional
	Metadata MachineSetTemplateMetadata `json:"metadata,omitzero"`

	// spec is the Machine spec of every member.
	// +required
	Spec MachineSetMachineSpec `json:"spec"`
}

// MachineSetTemplateMetadata is the metadata copied onto every Machine.
type MachineSetTemplateMetadata struct {
	// labels are copied onto every Machine. Labels with the prefix
	// machineset.compute.odc.telekom.com/ are reserved for the controller.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// MachineSetMachineSpec is IronCore's MachineSpec, with attachments that can also reference
// volume and network interface templates. Only power can change after creation.
type MachineSetMachineSpec struct {
	// machineClassRef is the MachineClass of every member's Machine. It cannot change, because
	// changing the class of running members (class rollout) is not supported yet.
	// +required
	MachineClassRef LocalObjectReference `json:"machineClassRef"`

	// machinePoolSelector selects the MachinePools the Machines may run in. Immutable.
	// +optional
	MachinePoolSelector map[string]string `json:"machinePoolSelector,omitempty"`

	// machinePoolRef pins the Machines to one MachinePool. Immutable.
	// +optional
	MachinePoolRef *LocalObjectReference `json:"machinePoolRef,omitempty"`

	// power is the requested power state of every member. Changing it updates the existing
	// Machines and never replaces them.
	// +optional
	// +kubebuilder:default=On
	Power computev1alpha1.Power `json:"power,omitempty"`

	// networkInterfaces are the network interfaces of every Machine. Immutable.
	// +optional
	// +listType=map
	// +listMapKey=name
	NetworkInterfaces []MachineSetNetworkInterface `json:"networkInterfaces,omitempty"`

	// volumes are the volumes of every Machine. Immutable.
	// +optional
	// +listType=map
	// +listMapKey=name
	Volumes []MachineSetVolume `json:"volumes,omitempty"`

	// ignitionRef references a Secret with the ignition (bootstrap configuration) of every
	// Machine. Immutable; the content of the Secret may change and is used by new Machines.
	// +optional
	IgnitionRef *commonv1alpha1.SecretKeySelector `json:"ignitionRef,omitempty"`

	// efiVars are variables passed to EFI while booting. Immutable.
	// +optional
	// +listType=atomic
	EFIVars []computev1alpha1.EFIVar `json:"efiVars,omitempty"`

	// tolerations of every Machine. Immutable.
	// +optional
	// +listType=atomic
	Tolerations []commonv1alpha1.Toleration `json:"tolerations,omitempty"`

	// guestConfig is the guest OS configuration of every Machine. A hostname is only allowed
	// when replicas is at most 1. Immutable.
	// +optional
	GuestConfig *computev1alpha1.MachineGuestConfig `json:"guestConfig,omitempty"`
}

// MachineSetVolume is a volume attachment of a member's Machine. Exactly one source is set.
type MachineSetVolume struct {
	// name of the volume attachment.
	// +required
	Name string `json:"name"`

	// device is the device name where the volume is attached. If empty, a free device name is
	// chosen.
	// +optional
	Device *string `json:"device,omitempty"`

	// volumeTemplateRef creates one Volume per member from the named entry of volumeTemplates.
	// The Volume belongs to the member and survives replacement of the member's Machine.
	// +optional
	VolumeTemplateRef *LocalObjectReference `json:"volumeTemplateRef,omitempty"`

	// volumeRef attaches an existing Volume. Only allowed when replicas is at most 1. The
	// Volume is released, never deleted, by the MachineSet.
	// +optional
	VolumeRef *LocalObjectReference `json:"volumeRef,omitempty"`

	// localDisk is scratch space offered by the MachinePool. It is created empty for every
	// new Machine and does not keep its data across replacement.
	// +optional
	LocalDisk *computev1alpha1.LocalDiskVolumeSource `json:"localDisk,omitempty"`

	// emptyDisk is deprecated in IronCore and rejected. Use localDisk.
	// +optional
	EmptyDisk *computev1alpha1.EmptyDiskVolumeSource `json:"emptyDisk,omitempty"`

	// ephemeral is rejected: IronCore deletes ephemeral Volumes together with their Machine.
	// Use volumeTemplateRef.
	// +optional
	Ephemeral *computev1alpha1.EphemeralVolumeSource `json:"ephemeral,omitempty"`
}

// MachineSetNetworkInterface is a network interface attachment of a member's Machine. Exactly
// one source is set.
type MachineSetNetworkInterface struct {
	// name of the network interface attachment.
	// +required
	Name string `json:"name"`

	// networkInterfaceTemplateRef creates one NetworkInterface per member from the named entry
	// of networkInterfaceTemplates. The NetworkInterface belongs to the member and keeps its IPs
	// and VirtualIP across replacement of the member's Machine.
	// +optional
	NetworkInterfaceTemplateRef *LocalObjectReference `json:"networkInterfaceTemplateRef,omitempty"`

	// networkInterfaceRef attaches an existing NetworkInterface. Only allowed when replicas is
	// at most 1. The NetworkInterface is released, never deleted, by the MachineSet.
	// +optional
	NetworkInterfaceRef *LocalObjectReference `json:"networkInterfaceRef,omitempty"`

	// ephemeral is rejected: IronCore deletes ephemeral NetworkInterfaces together with their
	// Machine. Use networkInterfaceTemplateRef.
	// +optional
	Ephemeral *EphemeralNetworkInterfaceSource `json:"ephemeral,omitempty"`
}

// VolumeTemplate is a named template for the Volume created for each member.
type VolumeTemplate struct {
	// metadata of the template: its name and the labels of the created Volumes.
	// +required
	Metadata TemplateMetadata `json:"metadata"`

	// spec of the created Volumes. Only IronCore sets claimRef and unclaimable, so they are
	// rejected here.
	// +required
	Spec storagev1alpha1.VolumeSpec `json:"spec"`
}

// NetworkInterfaceTemplate is a named template for the NetworkInterface created for each member.
type NetworkInterfaceTemplate struct {
	// metadata of the template: its name and the labels of the created NetworkInterfaces.
	// +required
	Metadata TemplateMetadata `json:"metadata"`

	// spec of the created NetworkInterfaces. It mirrors IronCore's NetworkInterfaceSpec.
	// +required
	Spec NetworkInterfaceSpec `json:"spec"`
}

// MachineSetStatus defines the observed state of MachineSet.
type MachineSetStatus struct {
	// observedGeneration is the generation of the spec the controller last acted on.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// replicas is the number of members that are requested and exist.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// readyReplicas is the number of members whose Ready condition is True.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// pendingReplicas is the number of requested members that are not provisioned yet.
	// +optional
	PendingReplicas int32 `json:"pendingReplicas,omitempty"`

	// retainedReplicas is the number of retained members, which keep their resources without a
	// Machine.
	// +optional
	RetainedReplicas int32 `json:"retainedReplicas,omitempty"`

	// machineClassRef is the MachineClass the controller observed, by name and UID. A
	// MachineClass recreated under the same name has a different UID and is not used.
	// +optional
	MachineClassRef *LocalUIDReference `json:"machineClassRef,omitempty"`

	// lastProgressTime is the last time a member made progress.
	// +optional
	LastProgressTime *metav1.Time `json:"lastProgressTime,omitempty"`

	// conditions of the MachineSet: MachineClassAvailable, Deleting, Ready and Progressing.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// members is a compact list of the set's members. Details are on each MachineSetMember.
	// +optional
	// +listType=map
	// +listMapKey=ordinal
	Members []MachineSetMemberSummary `json:"members,omitempty"`
}

// MachineSetMemberSummary is the compact status of one member.
type MachineSetMemberSummary struct {
	// ordinal of the member.
	// +required
	Ordinal int32 `json:"ordinal"`

	// uid of the MachineSetMember.
	// +optional
	UID string `json:"uid,omitempty"`

	// lifecycle of the member.
	// +optional
	Lifecycle MemberLifecycle `json:"lifecycle,omitempty"`

	// operationPhase is the phase of the member's current operation.
	// +optional
	OperationPhase string `json:"operationPhase,omitempty"`

	// machineState is the state of the member's current Machine.
	// +optional
	MachineState computev1alpha1.MachineState `json:"machineState,omitempty"`

	// blockingReason is the reason that holds the member back, if any.
	// +optional
	BlockingReason string `json:"blockingReason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// MachineSet keeps a number of durable VMs ("members") on IronCore. A member's identity, disks
// and network identity survive when its Machine is replaced.
type MachineSet struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of MachineSet
	// +required
	Spec MachineSetSpec `json:"spec"`

	// status defines the observed state of MachineSet
	// +optional
	Status MachineSetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MachineSetList contains a list of MachineSet
type MachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MachineSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &MachineSet{}, &MachineSetList{})
		return nil
	})
}

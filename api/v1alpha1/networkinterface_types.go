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
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The types in this file mirror IronCore's networking types (networkingv1alpha1.NetworkInterfaceSpec
// and the types below it). They cannot be reused directly: IronCore's IP and IPPrefix types are
// written as strings in JSON, but controller-gen renders them as objects, so the API server would
// reject every fixed IP or prefix. Here these fields are plain strings, checked by the validating
// webhook. The controller converts them with commonv1alpha1.ParseIP and
// commonv1alpha1.ParseIPPrefix.
//
// Fields that IronCore's controllers set (providerID, machineRef, a VirtualIP's targetRef) are left
// out: a template must not set them.

// NetworkInterfaceSpec is IronCore's NetworkInterfaceSpec without the fields set by IronCore.
type NetworkInterfaceSpec struct {
	// networkRef is the Network the NetworkInterface is connected to.
	// +required
	NetworkRef LocalObjectReference `json:"networkRef"`

	// ipFamilies are the IP families of the NetworkInterface.
	// +required
	// +listType=atomic
	IPFamilies []corev1.IPFamily `json:"ipFamilies"`

	// ips are the IPs of the NetworkInterface: fixed values, or ephemeral IPs allocated from a
	// prefix. Fixed values are only allowed when replicas is at most 1.
	// +required
	// +listType=atomic
	IPs []IPSource `json:"ips"`

	// prefixes are additional prefixes routed to the NetworkInterface. Fixed values are only
	// allowed when replicas is at most 1.
	// +optional
	// +listType=atomic
	Prefixes []PrefixSource `json:"prefixes,omitempty"`

	// virtualIP is the VirtualIP of the NetworkInterface.
	// +optional
	VirtualIP *VirtualIPSource `json:"virtualIP,omitempty"`

	// attributes are provider-specific attributes of the NetworkInterface.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

// IPSource is how to obtain an IP. Exactly one of value and ephemeral is set.
type IPSource struct {
	// value is a fixed IP address.
	// +optional
	Value *string `json:"value,omitempty"`

	// ephemeral allocates the IP from a Prefix owned by the NetworkInterface.
	// +optional
	Ephemeral *EphemeralPrefixSource `json:"ephemeral,omitempty"`
}

// PrefixSource is how to obtain a prefix. Exactly one of value and ephemeral is set.
type PrefixSource struct {
	// value is a fixed prefix.
	// +optional
	Value *string `json:"value,omitempty"`

	// ephemeral allocates the prefix as a Prefix owned by the NetworkInterface.
	// +optional
	Ephemeral *EphemeralPrefixSource `json:"ephemeral,omitempty"`
}

// EphemeralPrefixSource allocates a Prefix owned by the NetworkInterface.
type EphemeralPrefixSource struct {
	// prefixTemplate is the template of the Prefix.
	// +required
	PrefixTemplate PrefixTemplate `json:"prefixTemplate"`
}

// PrefixTemplate is the template of an ephemeral Prefix.
type PrefixTemplate struct {
	// metadata of the created Prefix.
	// +optional
	Metadata ObjectTemplateMetadata `json:"metadata,omitzero"`

	// spec of the created Prefix.
	// +required
	Spec PrefixSpec `json:"spec"`
}

// PrefixSpec is IronCore's ipam PrefixSpec.
type PrefixSpec struct {
	// ipFamily of the prefix.
	// +optional
	IPFamily corev1.IPFamily `json:"ipFamily,omitempty"`

	// prefix is a fixed prefix to allocate. Only allowed when replicas is at most 1.
	// +optional
	Prefix *string `json:"prefix,omitempty"`

	// prefixLength is the length of the prefix to allocate.
	// +optional
	PrefixLength int32 `json:"prefixLength,omitempty"`

	// parentRef is the parent Prefix to allocate from.
	// +optional
	ParentRef *LocalObjectReference `json:"parentRef,omitempty"`

	// parentSelector selects the parent Prefix to allocate from.
	// +optional
	ParentSelector *metav1.LabelSelector `json:"parentSelector,omitempty"`
}

// VirtualIPSource is how to obtain a VirtualIP. Exactly one of virtualIPRef and ephemeral is set.
type VirtualIPSource struct {
	// virtualIPRef references an existing VirtualIP. Only allowed when replicas is at most 1.
	// +optional
	VirtualIPRef *LocalObjectReference `json:"virtualIPRef,omitempty"`

	// ephemeral creates a VirtualIP owned by the NetworkInterface.
	// +optional
	Ephemeral *EphemeralVirtualIPSource `json:"ephemeral,omitempty"`
}

// EphemeralVirtualIPSource creates a VirtualIP owned by the NetworkInterface.
type EphemeralVirtualIPSource struct {
	// virtualIPTemplate is the template of the VirtualIP.
	// +required
	VirtualIPTemplate VirtualIPTemplate `json:"virtualIPTemplate"`
}

// VirtualIPTemplate is the template of an ephemeral VirtualIP.
type VirtualIPTemplate struct {
	// metadata of the created VirtualIP.
	// +optional
	Metadata ObjectTemplateMetadata `json:"metadata,omitzero"`

	// spec of the created VirtualIP.
	// +required
	Spec EphemeralVirtualIPSpec `json:"spec"`
}

// EphemeralVirtualIPSpec is IronCore's EphemeralVirtualIPSpec without targetRef, which IronCore
// sets.
type EphemeralVirtualIPSpec struct {
	// type of the VirtualIP.
	// +required
	Type networkingv1alpha1.VirtualIPType `json:"type"`

	// ipFamily of the VirtualIP.
	// +required
	IPFamily corev1.IPFamily `json:"ipFamily"`

	// reclaimPolicy of the VirtualIP. Only Delete is accepted (or leave it unset): a VirtualIP
	// with Retain is not owned by the NetworkInterface and would leak when the member is
	// reclaimed.
	// +optional
	ReclaimPolicy networkingv1alpha1.ReclaimPolicyType `json:"reclaimPolicy,omitempty"`
}

// EphemeralNetworkInterfaceSource is IronCore's ephemeral NetworkInterface source. It exists only
// so that the webhook can reject it with a helpful message.
type EphemeralNetworkInterfaceSource struct {
	// networkInterfaceTemplate is the template of the NetworkInterface.
	// +optional
	NetworkInterfaceTemplate *EphemeralNetworkInterfaceTemplate `json:"networkInterfaceTemplate,omitempty"`
}

// EphemeralNetworkInterfaceTemplate is the template of an ephemeral NetworkInterface.
type EphemeralNetworkInterfaceTemplate struct {
	// metadata of the NetworkInterface.
	// +optional
	Metadata ObjectTemplateMetadata `json:"metadata,omitzero"`

	// spec of the NetworkInterface.
	// +optional
	Spec NetworkInterfaceSpec `json:"spec,omitzero"`
}

// ObjectTemplateMetadata is the metadata of an object created from a template.
type ObjectTemplateMetadata struct {
	// labels of the created object.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// annotations of the created object.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

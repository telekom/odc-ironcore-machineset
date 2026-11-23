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

package validation

import (
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ironcorecomputev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

const (
	rootTemplate    = "root"
	primaryTemplate = "primary"
	brokerValue     = "broker"
	otherValue      = "other"
	machineSetName  = "brokers"
	zoneKey         = "zone"
	invalidValue    = "not valid!"
)

// newMachineSet returns a valid MachineSet like the machineSetName example: 3 members, each with a
// root Volume and a primary NetworkInterface (ephemeral IP and ephemeral public VirtualIP)
// created from templates. Tests change one field of it and check the validation result.
func newMachineSet() *computev1alpha1.MachineSet {
	return &computev1alpha1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{Name: machineSetName, Namespace: "default"},
		Spec: computev1alpha1.MachineSetSpec{
			Replicas: ptr.To[int32](3),
			Template: computev1alpha1.MachineSetTemplate{
				Metadata: computev1alpha1.MachineSetTemplateMetadata{
					Labels: map[string]string{"app": brokerValue},
				},
				Spec: computev1alpha1.MachineSetMachineSpec{
					MachineClassRef: computev1alpha1.LocalObjectReference{Name: "broker-medium"},
					Power:           ironcorecomputev1alpha1.PowerOn,
					Volumes: []computev1alpha1.MachineSetVolume{{
						Name:              rootTemplate,
						VolumeTemplateRef: &computev1alpha1.LocalObjectReference{Name: rootTemplate},
					}},
					NetworkInterfaces: []computev1alpha1.MachineSetNetworkInterface{{
						Name:                        primaryTemplate,
						NetworkInterfaceTemplateRef: &computev1alpha1.LocalObjectReference{Name: primaryTemplate},
					}},
				},
			},
			VolumeTemplates: []computev1alpha1.VolumeTemplate{newVolumeTemplate(rootTemplate)},
			NetworkInterfaceTemplates: []computev1alpha1.NetworkInterfaceTemplate{
				newNetworkInterfaceTemplate(primaryTemplate),
			},
			ResourceRetentionPolicy: computev1alpha1.ResourceRetentionPolicy{
				WhenScaledDown: computev1alpha1.RetentionPolicyRetain,
			},
		},
	}
}

// newSingleMachineSet returns a valid MachineSet with one member.
func newSingleMachineSet() *computev1alpha1.MachineSet {
	machineSet := newMachineSet()
	machineSet.Spec.Replicas = ptr.To[int32](1)
	return machineSet
}

func newVolumeTemplate(name string) computev1alpha1.VolumeTemplate {
	return computev1alpha1.VolumeTemplate{
		Metadata: computev1alpha1.TemplateMetadata{Name: name},
		Spec: storagev1alpha1.VolumeSpec{
			VolumeClassRef: &corev1.LocalObjectReference{Name: "durable-block"},
			Resources: corev1alpha1.ResourceList{
				corev1alpha1.ResourceStorage: resource.MustParse("40Gi"),
			},
			DataSource: storagev1alpha1.VolumeDataSource{
				OSImage: &storagev1alpha1.OSDataSource{Image: "registry.example.org/os/gardenlinux:1877"},
			},
		},
	}
}

func newNetworkInterfaceTemplate(name string) computev1alpha1.NetworkInterfaceTemplate {
	return computev1alpha1.NetworkInterfaceTemplate{
		Metadata: computev1alpha1.TemplateMetadata{Name: name},
		Spec: computev1alpha1.NetworkInterfaceSpec{
			NetworkRef: computev1alpha1.LocalObjectReference{Name: "application-network"},
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			IPs: []computev1alpha1.IPSource{{
				Ephemeral: newEphemeralPrefix(nil),
			}},
			VirtualIP: &computev1alpha1.VirtualIPSource{
				Ephemeral: &computev1alpha1.EphemeralVirtualIPSource{
					VirtualIPTemplate: computev1alpha1.VirtualIPTemplate{
						Spec: computev1alpha1.EphemeralVirtualIPSpec{
							Type:          networkingv1alpha1.VirtualIPTypePublic,
							IPFamily:      corev1.IPv4Protocol,
							ReclaimPolicy: networkingv1alpha1.ReclaimPolicyTypeDelete,
						},
					},
				},
			},
		},
	}
}

// newEphemeralPrefix returns an ephemeral prefix allocated from application-prefix. A non-nil
// fixedPrefix asks for that exact prefix.
func newEphemeralPrefix(fixedPrefix *string) *computev1alpha1.EphemeralPrefixSource {
	return &computev1alpha1.EphemeralPrefixSource{
		PrefixTemplate: computev1alpha1.PrefixTemplate{
			Spec: computev1alpha1.PrefixSpec{
				IPFamily:     corev1.IPv4Protocol,
				Prefix:       fixedPrefix,
				PrefixLength: 32,
				ParentRef:    &computev1alpha1.LocalObjectReference{Name: "application-prefix"},
			},
		},
	}
}

// newMember returns a valid MachineSetMember.
func newMember() *computev1alpha1.MachineSetMember {
	return &computev1alpha1.MachineSetMember{
		ObjectMeta: metav1.ObjectMeta{Name: "brokers-2", Namespace: "default"},
		Spec: computev1alpha1.MachineSetMemberSpec{
			MachineSetRef: computev1alpha1.LocalUIDReference{Name: machineSetName, UID: "uid-1"},
			Ordinal:       2,
			Lifecycle:     computev1alpha1.MemberLifecycleActive,
		},
	}
}

func noChange(*computev1alpha1.MachineSet) {}

func withMachinePoolSelector(ms *computev1alpha1.MachineSet) {
	ms.Spec.Template.Spec.MachinePoolSelector = map[string]string{zoneKey: "a"}
}

func withMachinePoolRef(ms *computev1alpha1.MachineSet) {
	ms.Spec.Template.Spec.MachinePoolRef = &computev1alpha1.LocalObjectReference{Name: "pool-a"}
}

func withTolerations(ms *computev1alpha1.MachineSet) {
	ms.Spec.Template.Spec.Tolerations = []commonv1alpha1.Toleration{{
		Key:      "dedicated",
		Operator: commonv1alpha1.TolerationOpEqual,
		Value:    machineSetName,
		Effect:   commonv1alpha1.TaintEffectNoSchedule,
	}}
}

func withIgnitionRef(ms *computev1alpha1.MachineSet) {
	ms.Spec.Template.Spec.IgnitionRef = &commonv1alpha1.SecretKeySelector{Name: "broker-ignition"}
}

func withEFIVars(ms *computev1alpha1.MachineSet) {
	ms.Spec.Template.Spec.EFIVars = []ironcorecomputev1alpha1.EFIVar{{
		Name:  "SecureBoot",
		UUID:  "8be4df61-93ca-11d2-aa0d-00e098032b8c",
		Value: "1",
	}}
}

// addVolume adds a volume template and attaches it.
func addVolume(ms *computev1alpha1.MachineSet, name string) {
	ms.Spec.VolumeTemplates = append(ms.Spec.VolumeTemplates, newVolumeTemplate(name))
	ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
		Name:              name,
		VolumeTemplateRef: &computev1alpha1.LocalObjectReference{Name: name},
	})
}

// addNetworkInterface adds a network interface template and attaches it.
func addNetworkInterface(ms *computev1alpha1.MachineSet, name string) {
	ms.Spec.NetworkInterfaceTemplates = append(ms.Spec.NetworkInterfaceTemplates, newNetworkInterfaceTemplate(name))
	ms.Spec.Template.Spec.NetworkInterfaces = append(ms.Spec.Template.Spec.NetworkInterfaces,
		computev1alpha1.MachineSetNetworkInterface{
			Name:                        name,
			NetworkInterfaceTemplateRef: &computev1alpha1.LocalObjectReference{Name: name},
		})
}

// firstNIC returns the spec of the first network interface template.
func firstNIC(ms *computev1alpha1.MachineSet) *computev1alpha1.NetworkInterfaceSpec {
	return &ms.Spec.NetworkInterfaceTemplates[0].Spec
}

// ephemeralVIP returns the spec of the ephemeral VirtualIP of the first network interface template.
func ephemeralVIP(ms *computev1alpha1.MachineSet) *computev1alpha1.EphemeralVirtualIPSpec {
	return &firstNIC(ms).VirtualIP.Ephemeral.VirtualIPTemplate.Spec
}

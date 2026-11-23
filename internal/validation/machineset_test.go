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
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

var _ = Describe("ValidateMachineSet", func() {
	DescribeTable("on create",
		func(change func(*computev1alpha1.MachineSet), wantErr string) {
			machineSet := newMachineSet()
			change(machineSet)
			expectErrors(ValidateMachineSet(machineSet), wantErr)
		},

		Entry("accepts the brokers example", noChange, ""),
		Entry("accepts empty defaulted fields", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = nil
			ms.Spec.Template.Spec.Power = ""
			ms.Spec.ResourceRetentionPolicy.WhenScaledDown = ""
		}, ""),

		// replicas, power and retention policy
		Entry("accepts 0 replicas", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](0)
		}, ""),
		Entry("accepts many replicas (there is no upper limit)", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](5000)
		}, ""),
		Entry("rejects negative replicas", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](-1)
		}, "spec.replicas"),
		Entry("rejects an unknown power state", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Power = "Sleeping"
		}, "spec.template.spec.power: Unsupported value"),
		Entry("rejects an unknown retention policy", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.ResourceRetentionPolicy.WhenScaledDown = "Archive"
		}, "spec.resourceRetentionPolicy.whenScaledDown: Unsupported value"),

		// references
		Entry("rejects a missing MachineClass", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachineClassRef.Name = ""
		}, "spec.template.spec.machineClassRef.name: Required value"),
		Entry("rejects a MachineClass name that is not a valid object name", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachineClassRef.Name = "Broker_Medium"
		}, "spec.template.spec.machineClassRef.name: Invalid value"),
		Entry("rejects an invalid machinePoolSelector", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachinePoolSelector = map[string]string{zoneKey: invalidValue}
		}, "spec.template.spec.machinePoolSelector"),
		Entry("rejects an ignitionRef without a name", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.IgnitionRef = &commonv1alpha1.SecretKeySelector{}
		}, "spec.template.spec.ignitionRef.name: Required value"),

		// unique template names
		Entry("accepts two volume templates with different names", func(ms *computev1alpha1.MachineSet) {
			addVolume(ms, "data")
		}, ""),
		Entry("rejects two volume templates with the same name", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates = append(ms.Spec.VolumeTemplates, newVolumeTemplate(rootTemplate))
		}, "spec.volumeTemplates[1].metadata.name: Duplicate value"),
		Entry("accepts two network interface templates with different names", func(ms *computev1alpha1.MachineSet) {
			addNetworkInterface(ms, "secondary")
		}, ""),
		Entry("rejects two network interface templates with the same name", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.NetworkInterfaceTemplates = append(ms.Spec.NetworkInterfaceTemplates,
				newNetworkInterfaceTemplate(primaryTemplate))
		}, "spec.networkInterfaceTemplates[1].metadata.name: Duplicate value"),
		Entry("rejects a template name that is not a DNS label", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Metadata.Name = "Root_Disk"
			ms.Spec.Template.Spec.Volumes[0].VolumeTemplateRef.Name = "Root_Disk"
		}, "spec.volumeTemplates[0].metadata.name"),

		// template references resolve, and each template is attached once
		Entry("rejects a volumeTemplateRef without a template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes[0].VolumeTemplateRef.Name = "missing"
		}, msgUnknownVolumeTemplate),
		Entry("rejects a networkInterfaceTemplateRef without a template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces[0].NetworkInterfaceTemplateRef.Name = "missing"
		}, msgUnknownNetworkInterfaceTemplate),
		Entry("rejects attaching a volume template twice", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
				Name:              "root-again",
				VolumeTemplateRef: &computev1alpha1.LocalObjectReference{Name: rootTemplate},
			})
		}, msgVolumeTemplateTwice),
		Entry("rejects attaching a network interface template twice", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces = append(ms.Spec.Template.Spec.NetworkInterfaces,
				computev1alpha1.MachineSetNetworkInterface{
					Name:                        "primary-again",
					NetworkInterfaceTemplateRef: &computev1alpha1.LocalObjectReference{Name: primaryTemplate},
				})
		}, msgNetworkInterfaceTemplateTwice),

		// attachment names
		Entry("rejects two volumes with the same name", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
				Name:      rootTemplate,
				LocalDisk: &ironcorecomputev1alpha1.LocalDiskVolumeSource{},
			})
		}, "spec.template.spec.volumes[1].name: Duplicate value"),
		Entry("rejects a network interface name that is not a DNS label", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces[0].Name = "Primary"
		}, "spec.template.spec.networkInterfaces[0].name: Invalid value"),

		// exactly one source per attachment
		Entry("accepts a localDisk volume (scratch space, not durable)", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
				Name:      "scratch",
				LocalDisk: &ironcorecomputev1alpha1.LocalDiskVolumeSource{SizeLimit: ptr.To(resource.MustParse("10Gi"))},
			})
		}, ""),
		Entry("rejects a volume without a source", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes[0].VolumeTemplateRef = nil
		}, msgVolumeSource),
		Entry("rejects a volume with two sources", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes[0].LocalDisk = &ironcorecomputev1alpha1.LocalDiskVolumeSource{}
		}, msgVolumeSource),
		Entry("rejects a network interface without a source", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces[0].NetworkInterfaceTemplateRef = nil
		}, msgNetworkInterfaceSource),
		Entry("rejects a network interface with two sources", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			ms.Spec.Template.Spec.NetworkInterfaces[0].NetworkInterfaceRef = &computev1alpha1.LocalObjectReference{Name: "nic"}
		}, msgNetworkInterfaceSource),
		Entry("rejects an IP with both a value and an ephemeral prefix", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			firstNIC(ms).IPs[0].Value = ptr.To("10.0.0.1")
		}, msgValueOrEphemeral),
		Entry("rejects a prefix with neither a value nor an ephemeral prefix", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).Prefixes = []computev1alpha1.PrefixSource{{}}
		}, msgValueOrEphemeral),
		Entry("rejects a VirtualIP with both a reference and an ephemeral source", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			firstNIC(ms).VirtualIP.VirtualIPRef = &computev1alpha1.LocalObjectReference{Name: "vip"}
		}, msgVirtualIPSource),

		// sources that would not survive Machine replacement or would have no owner
		Entry("rejects an ephemeral volume and points to volumeTemplateRef", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes[0].VolumeTemplateRef = nil
			ms.Spec.Template.Spec.Volumes[0].Ephemeral = &ironcorecomputev1alpha1.EphemeralVolumeSource{
				VolumeTemplate: &storagev1alpha1.VolumeTemplateSpec{},
			}
		}, "use volumeTemplateRef"),
		Entry("rejects an emptyDisk volume and points to localDisk", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes[0].VolumeTemplateRef = nil
			ms.Spec.Template.Spec.Volumes[0].EmptyDisk = &ironcorecomputev1alpha1.EmptyDiskVolumeSource{}
		}, "use localDisk"),
		Entry("rejects an ephemeral network interface and points to networkInterfaceTemplateRef",
			func(ms *computev1alpha1.MachineSet) {
				ms.Spec.Template.Spec.NetworkInterfaces[0].NetworkInterfaceTemplateRef = nil
				ms.Spec.Template.Spec.NetworkInterfaces[0].Ephemeral = &computev1alpha1.EphemeralNetworkInterfaceSource{}
			}, "use networkInterfaceTemplateRef"),
		Entry("accepts an ephemeral VirtualIP without a reclaimPolicy", func(ms *computev1alpha1.MachineSet) {
			ephemeralVIP(ms).ReclaimPolicy = ""
		}, ""),
		Entry("rejects an ephemeral VirtualIP with reclaimPolicy Retain", func(ms *computev1alpha1.MachineSet) {
			ephemeralVIP(ms).ReclaimPolicy = networkingv1alpha1.ReclaimPolicyTypeRetain
		}, msgVirtualIPRetain),
		Entry("rejects an ephemeral VirtualIP that is not public", func(ms *computev1alpha1.MachineSet) {
			ephemeralVIP(ms).Type = "Private"
		}, "virtualIPTemplate.spec.type: Unsupported value"),
		Entry("rejects an ephemeral VirtualIP without an IP family", func(ms *computev1alpha1.MachineSet) {
			ephemeralVIP(ms).IPFamily = ""
		}, "virtualIPTemplate.spec.ipFamily: Required value"),

		// volume templates
		Entry("rejects a volume template with a class but no storage", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.Resources = nil
		}, msgStorageRequired),
		Entry("accepts a volume template without a class and without storage", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.VolumeClassRef = nil
			ms.Spec.VolumeTemplates[0].Spec.Resources = nil
		}, ""),
		Entry("rejects a volume template with zero storage", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.Resources[corev1alpha1.ResourceStorage] = resource.MustParse("0")
		}, msgStoragePositive),
		Entry("rejects a volume template with negative storage", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.Resources[corev1alpha1.ResourceStorage] = resource.MustParse("-1Gi")
		}, msgStoragePositive),
		Entry("rejects a volume template with two data sources", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.DataSource.VolumeSnapshotRef = &corev1.LocalObjectReference{Name: "snap"}
		}, msgOneDataSource),
		Entry("rejects an osImage without an image", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.DataSource.OSImage.Image = ""
		}, "spec.volumeTemplates[0].spec.dataSource.osImage.image: Required value"),
		Entry("rejects a volume template with a claimRef", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.ClaimRef = &commonv1alpha1.LocalUIDReference{Name: "machine", UID: "uid"}
		}, "spec.volumeTemplates[0].spec.claimRef: Forbidden"),
		Entry("rejects an unclaimable volume template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.Unclaimable = true
		}, "spec.volumeTemplates[0].spec.unclaimable: Forbidden"),

		// labels
		Entry("rejects a reserved label in template.metadata", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Metadata.Labels[computev1alpha1.LabelMachineSetName] = otherValue
		}, reservedMessage),
		Entry("rejects an invalid label in template.metadata", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Metadata.Labels["app"] = invalidValue
		}, "spec.template.metadata.labels: Invalid value"),
		Entry("rejects a reserved label in a volume template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Metadata.Labels = map[string]string{computev1alpha1.LabelOrdinal: "0"}
		}, reservedMessage),
		Entry("accepts other labels in a network interface template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.NetworkInterfaceTemplates[0].Metadata.Labels = map[string]string{"tier": brokerValue}
		}, ""),
		Entry("rejects a reserved label in a network interface template", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.NetworkInterfaceTemplates[0].Metadata.Labels = map[string]string{computev1alpha1.LabelTemplate: "x"}
		}, reservedMessage),
		Entry("rejects an invalid annotation on an ephemeral prefix", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0].Ephemeral.PrefixTemplate.Metadata.Annotations = map[string]string{invalidValue: "x"}
		}, "prefixTemplate.metadata.annotations: Invalid value"),

		// IP families
		Entry("accepts dual stack", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
		}, ""),
		Entry("rejects a network interface without IP families", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPFamilies = nil
		}, "spec.networkInterfaceTemplates[0].spec.ipFamilies: Required value"),
		Entry("rejects an unknown IP family", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPFamilies = []corev1.IPFamily{"IPv5"}
		}, "spec.networkInterfaceTemplates[0].spec.ipFamilies[0]: Unsupported value"),
		Entry("rejects a repeated IP family", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv4Protocol}
		}, "spec.networkInterfaceTemplates[0].spec.ipFamilies[1]: Duplicate value"),
		Entry("rejects more than two IP families", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol, corev1.IPv4Protocol}
		}, "spec.networkInterfaceTemplates[0].spec.ipFamilies: Too many"),

		// IPs and prefixes
		Entry("rejects a network interface without IPs", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs = nil
		}, "spec.networkInterfaceTemplates[0].spec.ips: Required value"),
		Entry("rejects a fixed IP that is not an IP address", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			firstNIC(ms).IPs[0] = computev1alpha1.IPSource{Value: ptr.To("10.0.0.300")}
		}, "must be a valid IP address"),
		Entry("rejects a fixed prefix that is not in CIDR notation", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			firstNIC(ms).Prefixes = []computev1alpha1.PrefixSource{{Value: ptr.To("10.0.0.0")}}
		}, "must be a valid CIDR value"),
		Entry("rejects an ephemeral prefix with an unknown IP family", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0].Ephemeral.PrefixTemplate.Spec.IPFamily = "IPv5"
		}, "prefixTemplate.spec.ipFamily: Unsupported value"),
		Entry("rejects an ephemeral prefix length above 128", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0].Ephemeral.PrefixTemplate.Spec.PrefixLength = 129
		}, "prefixTemplate.spec.prefixLength: Invalid value"),
		Entry("rejects an invalid parent selector", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0].Ephemeral.PrefixTemplate.Spec.ParentSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{Key: zoneKey, Operator: "Near"}},
			}
		}, "prefixTemplate.spec.parentSelector"),
		Entry("accepts ephemeral prefixes allocated by length for multi-member sets",
			func(ms *computev1alpha1.MachineSet) {
				firstNIC(ms).Prefixes = []computev1alpha1.PrefixSource{{Ephemeral: newEphemeralPrefix(nil)}}
			}, ""),

		// hostname
		Entry("rejects a hostname that is not a DNS label", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			ms.Spec.Template.Spec.GuestConfig = &ironcorecomputev1alpha1.MachineGuestConfig{Hostname: "Broker_1"}
		}, "spec.template.spec.guestConfig.hostname: Invalid value"),
	)

	// Single-member-only inputs are exclusive to one Machine. Each is accepted with
	// replicas 1, rejected with replicas 2, and also rejected on a later scale-up from 1 to 2.
	DescribeTable("single-member-only inputs",
		func(change func(*computev1alpha1.MachineSet), wantErr string) {
			By("accepting the input with replicas 1")
			oldMachineSet := newSingleMachineSet()
			change(oldMachineSet)
			expectErrors(ValidateMachineSet(oldMachineSet), "")

			By("rejecting a scale-up to 2")
			newMachineSet := oldMachineSet.DeepCopy()
			newMachineSet.Spec.Replicas = ptr.To[int32](2)
			expectErrors(ValidateMachineSetUpdate(newMachineSet, oldMachineSet), wantErr)

			By("rejecting the input with replicas 2")
			machineSet := newMachineSet.DeepCopy()
			expectErrors(ValidateMachineSet(machineSet), wantErr)
		},
		Entry("guestConfig.hostname", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.GuestConfig = &ironcorecomputev1alpha1.MachineGuestConfig{Hostname: brokerValue}
		}, msgHostname),
		Entry("a reference to an existing Volume", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
				Name:      "data",
				VolumeRef: &computev1alpha1.LocalObjectReference{Name: "existing-volume"},
			})
		}, msgVolumeRef),
		Entry("a reference to an existing NetworkInterface", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces = append(ms.Spec.Template.Spec.NetworkInterfaces,
				computev1alpha1.MachineSetNetworkInterface{
					Name:                "management",
					NetworkInterfaceRef: &computev1alpha1.LocalObjectReference{Name: "existing-nic"},
				})
		}, msgNetworkInterface),
		Entry("a fixed IP", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0] = computev1alpha1.IPSource{Value: ptr.To("10.0.0.1")}
		}, msgFixedAddress),
		Entry("a fixed IP from an ephemeral prefix", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).IPs[0].Ephemeral = newEphemeralPrefix(ptr.To("10.0.0.1/32"))
		}, msgFixedAddress),
		Entry("a fixed prefix", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).Prefixes = []computev1alpha1.PrefixSource{{Value: ptr.To("10.1.0.0/24")}}
		}, msgFixedAddress),
		Entry("a fixed ephemeral prefix", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).Prefixes = []computev1alpha1.PrefixSource{{Ephemeral: newEphemeralPrefix(ptr.To("10.1.0.0/24"))}}
		}, msgFixedAddress),
		Entry("a reference to an existing VirtualIP", func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).VirtualIP = &computev1alpha1.VirtualIPSource{
				VirtualIPRef: &computev1alpha1.LocalObjectReference{Name: "existing-vip"},
			}
		}, msgVirtualIPRef),
	)
})

var _ = Describe("ValidateMachineSetUpdate", func() {
	// Only replicas, template.spec.power and resourceRetentionPolicy can change. Each
	// entry builds the old MachineSet with prepare, then changes a copy with change.
	DescribeTable("on update",
		func(prepare, change func(*computev1alpha1.MachineSet), wantErr string) {
			oldMachineSet := newMachineSet()
			prepare(oldMachineSet)
			expectErrors(ValidateMachineSet(oldMachineSet), "")
			newMachineSet := oldMachineSet.DeepCopy()
			change(newMachineSet)
			expectErrors(ValidateMachineSetUpdate(newMachineSet, oldMachineSet), wantErr)
		},

		// mutable fields
		Entry("accepts no change", noChange, noChange, ""),
		Entry("accepts a change of replicas", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](5)
		}, ""),
		Entry("accepts a change of power", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Power = ironcorecomputev1alpha1.PowerOff
		}, ""),
		Entry("accepts a change of the retention policy", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.ResourceRetentionPolicy.WhenScaledDown = computev1alpha1.RetentionPolicyDelete
		}, ""),
		Entry("accepts an empty list instead of a missing one", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Tolerations = []commonv1alpha1.Toleration{}
		}, ""),

		// the MachineClass
		Entry("rejects a change of the MachineClass and says that class rollout is not supported", noChange,
			func(ms *computev1alpha1.MachineSet) {
				ms.Spec.Template.Spec.MachineClassRef.Name = "broker-large"
			}, "class rollout, which is not supported"),

		// template metadata
		Entry("rejects a change of template labels", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Metadata.Labels["app"] = otherValue
		}, "spec.template.metadata: Forbidden: field is immutable"),
		Entry("rejects removing all template labels", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Metadata.Labels = nil
		}, "spec.template.metadata: Forbidden: field is immutable"),

		// placement
		Entry("rejects a change of machinePoolSelector", withMachinePoolSelector, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachinePoolSelector[zoneKey] = "b"
		}, "spec.template.spec.machinePoolSelector: Forbidden"),
		Entry("rejects adding machinePoolSelector", noChange, withMachinePoolSelector,
			"spec.template.spec.machinePoolSelector: Forbidden"),
		Entry("rejects a change of machinePoolRef", withMachinePoolRef, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachinePoolRef.Name = "pool-b"
		}, "spec.template.spec.machinePoolRef: Forbidden"),
		Entry("rejects removing machinePoolRef", withMachinePoolRef, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.MachinePoolRef = nil
		}, "spec.template.spec.machinePoolRef: Forbidden"),
		Entry("rejects adding tolerations", noChange, withTolerations, "spec.template.spec.tolerations: Forbidden"),
		Entry("rejects a change of tolerations", withTolerations, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Tolerations[0].Value = otherValue
		}, "spec.template.spec.tolerations: Forbidden"),

		// attachment layout
		Entry("rejects adding a volume", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = append(ms.Spec.Template.Spec.Volumes, computev1alpha1.MachineSetVolume{
				Name:      "scratch",
				LocalDisk: &ironcorecomputev1alpha1.LocalDiskVolumeSource{},
			})
		}, "spec.template.spec.volumes: Forbidden"),
		Entry("rejects removing all volumes", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = nil
		}, "spec.template.spec.volumes: Forbidden"),
		Entry("rejects a change of a network interface", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces[0].Name = "eth0"
		}, "spec.template.spec.networkInterfaces: Forbidden"),
		Entry("rejects removing all network interfaces", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces = nil
		}, "spec.template.spec.networkInterfaces: Forbidden"),

		// image, storage capacity and network in the templates
		Entry("rejects a change of the image", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.DataSource.OSImage.Image = "registry.example.org/os/other:1"
		}, "spec.volumeTemplates: Forbidden"),
		Entry("rejects a change of the storage capacity", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates[0].Spec.Resources[corev1alpha1.ResourceStorage] = resource.MustParse("80Gi")
		}, "spec.volumeTemplates: Forbidden"),
		Entry("rejects removing all volume templates", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.Volumes = nil
		}, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.VolumeTemplates = nil
		}, "spec.volumeTemplates: Forbidden"),
		Entry("rejects a change of the network", noChange, func(ms *computev1alpha1.MachineSet) {
			firstNIC(ms).NetworkRef.Name = "other-network"
		}, "spec.networkInterfaceTemplates: Forbidden"),
		Entry("rejects a change of network interface template labels", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.NetworkInterfaceTemplates[0].Metadata.Labels = map[string]string{"tier": brokerValue}
		}, "spec.networkInterfaceTemplates: Forbidden"),
		Entry("rejects removing all network interface templates", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.NetworkInterfaces = nil
		}, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.NetworkInterfaceTemplates = nil
		}, "spec.networkInterfaceTemplates: Forbidden"),

		// bootstrap
		Entry("rejects a change of ignitionRef", withIgnitionRef, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.IgnitionRef.Name = "other-ignition"
		}, "spec.template.spec.ignitionRef: Forbidden"),
		Entry("rejects adding ignitionRef", noChange, withIgnitionRef, "spec.template.spec.ignitionRef: Forbidden"),
		Entry("rejects a change of efiVars", withEFIVars, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.EFIVars[0].Value = otherValue
		}, "spec.template.spec.efiVars: Forbidden"),
		Entry("rejects removing efiVars", withEFIVars, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.EFIVars = nil
		}, "spec.template.spec.efiVars: Forbidden"),
		Entry("rejects a change of guestConfig", func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Replicas = ptr.To[int32](1)
			ms.Spec.Template.Spec.GuestConfig = &ironcorecomputev1alpha1.MachineGuestConfig{Hostname: brokerValue}
		}, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.GuestConfig.Hostname = otherValue
		}, "spec.template.spec.guestConfig: Forbidden"),
		Entry("rejects adding guestConfig", noChange, func(ms *computev1alpha1.MachineSet) {
			ms.Spec.Template.Spec.GuestConfig = &ironcorecomputev1alpha1.MachineGuestConfig{}
		}, "spec.template.spec.guestConfig: Forbidden"),
	)
})

var _ = Describe("ValidateMachineSetStatus", func() {
	DescribeTable("member summaries",
		func(member computev1alpha1.MachineSetMemberSummary, wantErr string) {
			machineSet := newMachineSet()
			machineSet.Status.Members = []computev1alpha1.MachineSetMemberSummary{member}
			expectErrors(ValidateMachineSetStatus(machineSet), wantErr)
		},
		Entry("accepts a running member", computev1alpha1.MachineSetMemberSummary{
			Ordinal:      0,
			Lifecycle:    computev1alpha1.MemberLifecycleActive,
			MachineState: ironcorecomputev1alpha1.MachineStateRunning,
		}, ""),
		Entry("rejects a negative ordinal", computev1alpha1.MachineSetMemberSummary{Ordinal: -1},
			"status.members[0].ordinal"),
		Entry("rejects an unknown lifecycle", computev1alpha1.MachineSetMemberSummary{Lifecycle: "Paused"},
			"status.members[0].lifecycle: Unsupported value"),
		Entry("rejects an unknown machine state", computev1alpha1.MachineSetMemberSummary{MachineState: "Asleep"},
			"status.members[0].machineState: Unsupported value"),
	)
})

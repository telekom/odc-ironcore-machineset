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
	ironcorecomputev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

// Messages that the tests and users rely on. They explain why an input is rejected and what to
// use instead.
const (
	msgVolumeSource           = "exactly one volume source must be set: volumeTemplateRef, volumeRef or localDisk"
	msgEphemeralVolume        = "ephemeral volumes are deleted with their Machine and do not survive replacement; use volumeTemplateRef"
	msgEmptyDisk              = "emptyDisk is deprecated in IronCore; use localDisk"
	msgUnknownVolumeTemplate  = "every volumeTemplateRef must name an entry of volumeTemplates"
	msgVolumeTemplateTwice    = "each volume template can be attached only once"
	msgNetworkInterfaceSource = "exactly one network interface source must be set: networkInterfaceTemplateRef or " +
		"networkInterfaceRef"
	msgEphemeralNetworkInterface = "ephemeral network interfaces are deleted with their Machine and do not survive " +
		"replacement; use networkInterfaceTemplateRef"
	msgUnknownNetworkInterfaceTemplate = "every networkInterfaceTemplateRef must name an entry of networkInterfaceTemplates"
	msgNetworkInterfaceTemplateTwice   = "each network interface template can be attached only once"
	msgValueOrEphemeral                = "exactly one of value and ephemeral must be set"
	msgVirtualIPSource                 = "exactly one of virtualIPRef and ephemeral must be set"
	msgVirtualIPRetain                 = "ephemeral VirtualIPs with reclaimPolicy Retain have no owner and would leak; " +
		"use reclaimPolicy Delete or leave it unset"
	msgStorageRequired  = "resources.storage is required when volumeClassRef is set"
	msgStoragePositive  = "resources.storage must be greater than zero"
	msgOneDataSource    = "at most one of volumeSnapshotRef and osImage may be set"
	msgClaimRef         = "must not be set: the controller attaches each Volume to its Machine"
	msgUnclaimable      = "must not be set: Volumes created from a template must be attachable to their Machine"
	msgHostname         = "template.spec.guestConfig.hostname is only allowed when replicas is at most 1"
	msgVolumeRef        = "volumeRef (an existing Volume) is only allowed when replicas is at most 1; use volumeTemplateRef"
	msgNetworkInterface = "networkInterfaceRef (an existing NetworkInterface) is only allowed when replicas is at " +
		"most 1; use networkInterfaceTemplateRef"
	msgFixedAddress = "fixed IPs and prefixes are only allowed when replicas is at most 1"
	msgVirtualIPRef = "virtualIPRef (an existing VirtualIP) is only allowed when replicas is at most 1; " +
		"use an ephemeral VirtualIP"
	msgMachineClassImmutable = "field is immutable: changing the MachineClass needs class rollout, which is not supported"
	msgOnlyPowerChanges      = "field is immutable: only template.spec.power can change"
	maxIPFamilies            = 2
	maxPrefixLength          = 128
)

// ValidateMachineSet checks a MachineSet on create.
func ValidateMachineSet(machineSet *computev1alpha1.MachineSet) field.ErrorList {
	return validateMachineSetSpec(&machineSet.Spec, field.NewPath("spec"))
}

// ValidateMachineSetUpdate checks a MachineSet on update. Besides the create rules, only
// spec.replicas, spec.template.spec.power and spec.resourceRetentionPolicy may change.
func ValidateMachineSetUpdate(newMachineSet, oldMachineSet *computev1alpha1.MachineSet) field.ErrorList {
	allErrs := ValidateMachineSet(newMachineSet)
	allErrs = append(allErrs, validateMachineSetSpecUpdate(&newMachineSet.Spec, &oldMachineSet.Spec,
		field.NewPath("spec"))...)
	return allErrs
}

// ValidateMachineSetStatus checks the status of a MachineSet, which only the controller writes.
func ValidateMachineSetStatus(machineSet *computev1alpha1.MachineSet) field.ErrorList {
	allErrs := make(field.ErrorList, 0, len(machineSet.Status.Members))
	membersPath := field.NewPath("status", "members")
	for i, member := range machineSet.Status.Members {
		memberPath := membersPath.Index(i)
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(member.Ordinal),
			memberPath.Child("ordinal"))...)
		allErrs = append(allErrs, validateEnum(member.Lifecycle, true, memberPath.Child("lifecycle"),
			memberLifecycles...)...)
		allErrs = append(allErrs, validateEnum(member.MachineState, true, memberPath.Child("machineState"),
			machineStates...)...)
	}
	return allErrs
}

// desiredReplicas returns spec.replicas. The API server sets the default 1 before the webhook
// runs, so nil only happens when the function is called directly, for example in unit tests.
func desiredReplicas(spec *computev1alpha1.MachineSetSpec) int32 {
	if spec.Replicas == nil {
		return 1
	}
	return *spec.Replicas
}

func validateMachineSetSpec(spec *computev1alpha1.MachineSetSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Replicas != nil {
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(*spec.Replicas),
			fldPath.Child("replicas"))...)
	}
	// Single-member-only inputs, such as a hostname or a fixed IP, belong to exactly one
	// Machine and are rejected when the set has more than one member.
	multiMember := desiredReplicas(spec) > 1

	volumeTemplateNames, errs := validateVolumeTemplates(spec.VolumeTemplates, fldPath.Child("volumeTemplates"))
	allErrs = append(allErrs, errs...)
	networkInterfaceTemplateNames, errs := validateNetworkInterfaceTemplates(spec.NetworkInterfaceTemplates,
		multiMember, fldPath.Child("networkInterfaceTemplates"))
	allErrs = append(allErrs, errs...)

	templatePath := fldPath.Child("template")
	allErrs = append(allErrs, validateUserLabels(spec.Template.Metadata.Labels,
		templatePath.Child("metadata", "labels"))...)
	allErrs = append(allErrs, validateMachineSpec(&spec.Template.Spec, volumeTemplateNames,
		networkInterfaceTemplateNames, multiMember, templatePath.Child("spec"))...)

	allErrs = append(allErrs, validateEnum(spec.ResourceRetentionPolicy.WhenScaledDown, true,
		fldPath.Child("resourceRetentionPolicy", "whenScaledDown"),
		computev1alpha1.RetentionPolicyRetain, computev1alpha1.RetentionPolicyDelete)...)
	return allErrs
}

func validateMachineSpec(
	spec *computev1alpha1.MachineSetMachineSpec,
	volumeTemplateNames, networkInterfaceTemplateNames sets.Set[string],
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	allErrs := validateRefName(spec.MachineClassRef.Name, fldPath.Child("machineClassRef", "name"))
	allErrs = append(allErrs, metav1validation.ValidateLabels(spec.MachinePoolSelector,
		fldPath.Child("machinePoolSelector"))...)
	if spec.MachinePoolRef != nil {
		allErrs = append(allErrs, validateRefName(spec.MachinePoolRef.Name, fldPath.Child("machinePoolRef", "name"))...)
	}
	allErrs = append(allErrs, validateEnum(spec.Power, true, fldPath.Child("power"),
		ironcorecomputev1alpha1.PowerOn, ironcorecomputev1alpha1.PowerOff)...)
	if spec.IgnitionRef != nil {
		allErrs = append(allErrs, validateRefName(spec.IgnitionRef.Name, fldPath.Child("ignitionRef", "name"))...)
	}
	if spec.GuestConfig != nil && spec.GuestConfig.Hostname != "" {
		hostnamePath := fldPath.Child("guestConfig", "hostname")
		allErrs = append(allErrs, validateDNSLabel(spec.GuestConfig.Hostname, hostnamePath)...)
		if multiMember {
			allErrs = append(allErrs, field.Forbidden(hostnamePath, msgHostname))
		}
	}
	allErrs = append(allErrs, validateVolumes(spec.Volumes, volumeTemplateNames, multiMember,
		fldPath.Child("volumes"))...)
	allErrs = append(allErrs, validateNetworkInterfaces(spec.NetworkInterfaces, networkInterfaceTemplateNames,
		multiMember, fldPath.Child("networkInterfaces"))...)
	return allErrs
}

// validateVolumes checks the volume attachments of the Machine template.
func validateVolumes(
	volumes []computev1alpha1.MachineSetVolume,
	templateNames sets.Set[string],
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	names := sets.New[string]()
	attachedTemplates := sets.New[string]()
	for i, volume := range volumes {
		volumePath := fldPath.Index(i)
		allErrs = append(allErrs, validateDNSLabel(volume.Name, volumePath.Child("name"))...)
		if names.Has(volume.Name) {
			allErrs = append(allErrs, field.Duplicate(volumePath.Child("name"), volume.Name))
		}
		names.Insert(volume.Name)

		allErrs = append(allErrs, validateExactlyOne(volumePath, msgVolumeSource,
			volume.VolumeTemplateRef != nil, volume.VolumeRef != nil, volume.LocalDisk != nil,
			volume.EmptyDisk != nil, volume.Ephemeral != nil)...)
		if volume.Ephemeral != nil {
			allErrs = append(allErrs, field.Forbidden(volumePath.Child("ephemeral"), msgEphemeralVolume))
		}
		if volume.EmptyDisk != nil {
			allErrs = append(allErrs, field.Forbidden(volumePath.Child("emptyDisk"), msgEmptyDisk))
		}
		if ref := volume.VolumeTemplateRef; ref != nil {
			refPath := volumePath.Child("volumeTemplateRef", "name")
			switch {
			case !templateNames.Has(ref.Name):
				allErrs = append(allErrs, field.Invalid(refPath, ref.Name, msgUnknownVolumeTemplate))
			case attachedTemplates.Has(ref.Name):
				allErrs = append(allErrs, field.Invalid(refPath, ref.Name, msgVolumeTemplateTwice))
			}
			attachedTemplates.Insert(ref.Name)
		}
		if ref := volume.VolumeRef; ref != nil {
			allErrs = append(allErrs, validateRefName(ref.Name, volumePath.Child("volumeRef", "name"))...)
			if multiMember {
				allErrs = append(allErrs, field.Forbidden(volumePath.Child("volumeRef"), msgVolumeRef))
			}
		}
	}
	return allErrs
}

// validateNetworkInterfaces checks the network interface attachments of the Machine template.
func validateNetworkInterfaces(
	networkInterfaces []computev1alpha1.MachineSetNetworkInterface,
	templateNames sets.Set[string],
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList
	names := sets.New[string]()
	attachedTemplates := sets.New[string]()
	for i, networkInterface := range networkInterfaces {
		nicPath := fldPath.Index(i)
		allErrs = append(allErrs, validateDNSLabel(networkInterface.Name, nicPath.Child("name"))...)
		if names.Has(networkInterface.Name) {
			allErrs = append(allErrs, field.Duplicate(nicPath.Child("name"), networkInterface.Name))
		}
		names.Insert(networkInterface.Name)

		allErrs = append(allErrs, validateExactlyOne(nicPath, msgNetworkInterfaceSource,
			networkInterface.NetworkInterfaceTemplateRef != nil, networkInterface.NetworkInterfaceRef != nil,
			networkInterface.Ephemeral != nil)...)
		if networkInterface.Ephemeral != nil {
			allErrs = append(allErrs, field.Forbidden(nicPath.Child("ephemeral"), msgEphemeralNetworkInterface))
		}
		if ref := networkInterface.NetworkInterfaceTemplateRef; ref != nil {
			refPath := nicPath.Child("networkInterfaceTemplateRef", "name")
			switch {
			case !templateNames.Has(ref.Name):
				allErrs = append(allErrs, field.Invalid(refPath, ref.Name, msgUnknownNetworkInterfaceTemplate))
			case attachedTemplates.Has(ref.Name):
				allErrs = append(allErrs, field.Invalid(refPath, ref.Name, msgNetworkInterfaceTemplateTwice))
			}
			attachedTemplates.Insert(ref.Name)
		}
		if ref := networkInterface.NetworkInterfaceRef; ref != nil {
			allErrs = append(allErrs, validateRefName(ref.Name, nicPath.Child("networkInterfaceRef", "name"))...)
			if multiMember {
				allErrs = append(allErrs, field.Forbidden(nicPath.Child("networkInterfaceRef"), msgNetworkInterface))
			}
		}
	}
	return allErrs
}

// validateTemplateMetadata checks the name and labels of a volume or network interface template
// and records the name in names. It reports a duplicate name.
func validateTemplateMetadata(
	metadata computev1alpha1.TemplateMetadata,
	names sets.Set[string],
	fldPath *field.Path,
) field.ErrorList {
	allErrs := validateDNSLabel(metadata.Name, fldPath.Child("name"))
	if names.Has(metadata.Name) {
		allErrs = append(allErrs, field.Duplicate(fldPath.Child("name"), metadata.Name))
	}
	names.Insert(metadata.Name)
	allErrs = append(allErrs, validateUserLabels(metadata.Labels, fldPath.Child("labels"))...)
	return allErrs
}

// validateVolumeTemplates checks the volume templates and returns their names.
func validateVolumeTemplates(
	templates []computev1alpha1.VolumeTemplate,
	fldPath *field.Path,
) (sets.Set[string], field.ErrorList) {
	allErrs := make(field.ErrorList, 0, len(templates))
	names := sets.New[string]()
	for i := range templates {
		templatePath := fldPath.Index(i)
		allErrs = append(allErrs, validateTemplateMetadata(templates[i].Metadata, names,
			templatePath.Child("metadata"))...)
		allErrs = append(allErrs, validateVolumeSpec(&templates[i].Spec, templatePath.Child("spec"))...)
	}
	return names, allErrs
}

// validateVolumeSpec checks the IronCore VolumeSpec of a volume template. It covers the rules
// that matter for the MachineSet; IronCore's API server checks the rest when the controller
// creates the Volume.
func validateVolumeSpec(spec *storagev1alpha1.VolumeSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	storagePath := fldPath.Child("resources").Key(string(corev1alpha1.ResourceStorage))
	storage, hasStorage := spec.Resources[corev1alpha1.ResourceStorage]
	if spec.VolumeClassRef != nil {
		allErrs = append(allErrs, validateRefName(spec.VolumeClassRef.Name, fldPath.Child("volumeClassRef", "name"))...)
		if !hasStorage {
			allErrs = append(allErrs, field.Required(storagePath, msgStorageRequired))
		}
	}
	if hasStorage && storage.Sign() <= 0 {
		allErrs = append(allErrs, field.Invalid(storagePath, storage.String(), msgStoragePositive))
	}
	if spec.VolumePoolRef != nil {
		allErrs = append(allErrs, validateRefName(spec.VolumePoolRef.Name, fldPath.Child("volumePoolRef", "name"))...)
	}
	allErrs = append(allErrs, metav1validation.ValidateLabels(spec.VolumePoolSelector,
		fldPath.Child("volumePoolSelector"))...)
	if spec.ClaimRef != nil {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("claimRef"), msgClaimRef))
	}
	if spec.Unclaimable {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("unclaimable"), msgUnclaimable))
	}
	if spec.Encryption != nil {
		allErrs = append(allErrs, validateRefName(spec.Encryption.SecretRef.Name,
			fldPath.Child("encryption", "secretRef", "name"))...)
	}

	dataSourcePath := fldPath.Child("dataSource")
	dataSource := spec.DataSource
	if dataSource.VolumeSnapshotRef != nil && dataSource.OSImage != nil {
		allErrs = append(allErrs, field.Invalid(dataSourcePath, "volumeSnapshotRef and osImage", msgOneDataSource))
	}
	if dataSource.VolumeSnapshotRef != nil {
		allErrs = append(allErrs, validateRefName(dataSource.VolumeSnapshotRef.Name,
			dataSourcePath.Child("volumeSnapshotRef", "name"))...)
	}
	if dataSource.OSImage != nil && dataSource.OSImage.Image == "" {
		allErrs = append(allErrs, field.Required(dataSourcePath.Child("osImage", "image"), ""))
	}
	return allErrs
}

// validateNetworkInterfaceTemplates checks the network interface templates and returns their names.
func validateNetworkInterfaceTemplates(
	templates []computev1alpha1.NetworkInterfaceTemplate,
	multiMember bool,
	fldPath *field.Path,
) (sets.Set[string], field.ErrorList) {
	allErrs := make(field.ErrorList, 0, len(templates))
	names := sets.New[string]()
	for i := range templates {
		templatePath := fldPath.Index(i)
		allErrs = append(allErrs, validateTemplateMetadata(templates[i].Metadata, names,
			templatePath.Child("metadata"))...)
		allErrs = append(allErrs, validateNetworkInterfaceSpec(&templates[i].Spec, multiMember,
			templatePath.Child("spec"))...)
	}
	return names, allErrs
}

func validateMachineSetSpecUpdate(newSpec, oldSpec *computev1alpha1.MachineSetSpec, fldPath *field.Path) field.ErrorList {
	templatePath := fldPath.Child("template")
	allErrs := validateImmutable(newSpec.Template.Metadata, oldSpec.Template.Metadata,
		templatePath.Child("metadata"), immutableMessage)

	specPath := templatePath.Child("spec")
	newMachine, oldMachine := &newSpec.Template.Spec, &oldSpec.Template.Spec
	machineErrs := validateImmutable(newMachine.MachineClassRef, oldMachine.MachineClassRef,
		specPath.Child("machineClassRef"), msgMachineClassImmutable)
	for _, immutableField := range []struct {
		name             string
		newValue, oldVal any
	}{
		{"machinePoolSelector", newMachine.MachinePoolSelector, oldMachine.MachinePoolSelector},
		{"machinePoolRef", newMachine.MachinePoolRef, oldMachine.MachinePoolRef},
		{"networkInterfaces", newMachine.NetworkInterfaces, oldMachine.NetworkInterfaces},
		{"volumes", newMachine.Volumes, oldMachine.Volumes},
		{"ignitionRef", newMachine.IgnitionRef, oldMachine.IgnitionRef},
		{"efiVars", newMachine.EFIVars, oldMachine.EFIVars},
		{"tolerations", newMachine.Tolerations, oldMachine.Tolerations},
		{"guestConfig", newMachine.GuestConfig, oldMachine.GuestConfig},
	} {
		machineErrs = append(machineErrs, validateImmutable(immutableField.newValue, immutableField.oldVal,
			specPath.Child(immutableField.name), immutableMessage)...)
	}
	if len(machineErrs) == 0 {
		// Catches fields added to the Machine template later and not listed above.
		newWithOldPower := newMachine.DeepCopy()
		newWithOldPower.Power = oldMachine.Power
		machineErrs = validateImmutable(newWithOldPower, oldMachine, specPath, msgOnlyPowerChanges)
	}
	allErrs = append(allErrs, machineErrs...)

	allErrs = append(allErrs, validateImmutable(newSpec.VolumeTemplates, oldSpec.VolumeTemplates,
		fldPath.Child("volumeTemplates"), immutableMessage)...)
	allErrs = append(allErrs, validateImmutable(newSpec.NetworkInterfaceTemplates, oldSpec.NetworkInterfaceTemplates,
		fldPath.Child("networkInterfaceTemplates"), immutableMessage)...)
	return allErrs
}

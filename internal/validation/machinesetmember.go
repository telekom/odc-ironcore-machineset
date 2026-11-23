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
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

const msgTriggerOnlyForRestoring = "trigger is only set for a Restoring operation"

var (
	memberLifecycles = []computev1alpha1.MemberLifecycle{
		computev1alpha1.MemberLifecycleActive,
		computev1alpha1.MemberLifecycleRetained,
		computev1alpha1.MemberLifecycleReclaiming,
	}
	machineStates = []ironcorecomputev1alpha1.MachineState{
		ironcorecomputev1alpha1.MachineStatePending,
		ironcorecomputev1alpha1.MachineStateRunning,
		ironcorecomputev1alpha1.MachineStateShutdown,
		ironcorecomputev1alpha1.MachineStateTerminating,
		ironcorecomputev1alpha1.MachineStateTerminated,
	}
	ownerships = []computev1alpha1.ResourceOwnership{
		computev1alpha1.ResourceOwnershipManaged,
		computev1alpha1.ResourceOwnershipExternal,
	}
	operationTypes = []computev1alpha1.OperationType{
		computev1alpha1.OperationProvisioning,
		computev1alpha1.OperationPowerChange,
		computev1alpha1.OperationRestoring,
		computev1alpha1.OperationReactivating,
		computev1alpha1.OperationRetiring,
		computev1alpha1.OperationReclaiming,
	}
	restorationTriggers = []computev1alpha1.RestorationTrigger{
		computev1alpha1.RestorationTriggerRelinquished,
		computev1alpha1.RestorationTriggerFenced,
		computev1alpha1.RestorationTriggerExternalDeletion,
		computev1alpha1.RestorationTriggerEviction,
	}
)

// ValidateMachineSetMember checks a MachineSetMember on create.
func ValidateMachineSetMember(member *computev1alpha1.MachineSetMember) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := validateUIDRef(member.Spec.MachineSetRef, specPath.Child("machineSetRef"))
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(member.Spec.Ordinal),
		specPath.Child("ordinal"))...)
	allErrs = append(allErrs, validateEnum(member.Spec.Lifecycle, true, specPath.Child("lifecycle"),
		memberLifecycles...)...)
	return allErrs
}

// ValidateMachineSetMemberUpdate checks a MachineSetMember on update. A member always belongs to
// the same MachineSet and keeps its ordinal.
func ValidateMachineSetMemberUpdate(newMember, oldMember *computev1alpha1.MachineSetMember) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := ValidateMachineSetMember(newMember)
	allErrs = append(allErrs, validateImmutable(newMember.Spec.MachineSetRef, oldMember.Spec.MachineSetRef,
		specPath.Child("machineSetRef"), immutableMessage)...)
	allErrs = append(allErrs, validateImmutable(newMember.Spec.Ordinal, oldMember.Spec.Ordinal,
		specPath.Child("ordinal"), immutableMessage)...)
	return allErrs
}

// ValidateMachineSetMemberStatus checks the status of a MachineSetMember, which only the
// controller writes.
func ValidateMachineSetMemberStatus(member *computev1alpha1.MachineSetMember) field.ErrorList {
	statusPath := field.NewPath("status")
	status := &member.Status
	var allErrs field.ErrorList
	if status.MachineRef != nil {
		allErrs = append(allErrs, validateUIDRef(*status.MachineRef, statusPath.Child("machineRef"))...)
	}
	allErrs = append(allErrs, validateEnum(status.MachineState, true, statusPath.Child("machineState"),
		machineStates...)...)

	volumesPath := statusPath.Child("volumes")
	for i, volume := range status.Volumes {
		volumePath := volumesPath.Index(i)
		allErrs = append(allErrs, validateDNSLabel(volume.Name, volumePath.Child("name"))...)
		allErrs = append(allErrs, validateEnum(volume.Ownership, false, volumePath.Child("ownership"), ownerships...)...)
		allErrs = append(allErrs, validateUIDRef(volume.VolumeRef, volumePath.Child("volumeRef"))...)
	}

	nicsPath := statusPath.Child("networkInterfaces")
	for i, nic := range status.NetworkInterfaces {
		nicPath := nicsPath.Index(i)
		allErrs = append(allErrs, validateDNSLabel(nic.Name, nicPath.Child("name"))...)
		allErrs = append(allErrs, validateEnum(nic.Ownership, false, nicPath.Child("ownership"), ownerships...)...)
		allErrs = append(allErrs, validateUIDRef(nic.NetworkInterfaceRef, nicPath.Child("networkInterfaceRef"))...)
		for j, ip := range nic.IPs {
			allErrs = append(allErrs, validation.IsValidIP(nicPath.Child("ips").Index(j), ip)...)
		}
		for j, prefix := range nic.Prefixes {
			allErrs = append(allErrs, validation.IsValidCIDR(nicPath.Child("prefixes").Index(j), prefix)...)
		}
		if nic.VirtualIP != nil {
			allErrs = append(allErrs, validation.IsValidIP(nicPath.Child("virtualIP"), *nic.VirtualIP)...)
		}
	}

	if operation := status.Operation; operation != nil {
		operationPath := statusPath.Child("operation")
		allErrs = append(allErrs, validateEnum(operation.Type, false, operationPath.Child("type"), operationTypes...)...)
		triggerPath := operationPath.Child("trigger")
		allErrs = append(allErrs, validateEnum(operation.Trigger, true, triggerPath, restorationTriggers...)...)
		if operation.Trigger != "" && operation.Type != computev1alpha1.OperationRestoring {
			allErrs = append(allErrs, field.Forbidden(triggerPath, msgTriggerOnlyForRestoring))
		}
		if operation.PredecessorMachineRef != nil {
			allErrs = append(allErrs, validateUIDRef(*operation.PredecessorMachineRef,
				operationPath.Child("predecessorMachineRef"))...)
		}
	}
	return allErrs
}

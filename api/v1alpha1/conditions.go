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

// Condition types and reasons. The IEP only names Ready and Progressing; the other types and all
// reasons are a proposal of this project (see issue #11).

// MachineSetMember condition types.
const (
	// MemberConditionResourcesReady tells whether the member's Volumes and NetworkInterfaces exist
	// and are bound to the member.
	MemberConditionResourcesReady = "ResourcesReady"
	// MemberConditionMachineScheduled tells whether the member has a Machine that is placed on a
	// MachinePool.
	MemberConditionMachineScheduled = "MachineScheduled"
	// MemberConditionPoolReachable tells whether the MachinePool of the current Machine is
	// reachable. It is absent while the member has no Machine.
	MemberConditionPoolReachable = "PoolReachable"
	// MemberConditionReady is the summary condition of a member.
	MemberConditionReady = "Ready"
)

// Reasons of the MachineSetMember ResourcesReady condition.
const (
	// ReasonBound (True): all resources exist and are bound to the member by UID.
	ReasonBound = "Bound"
	// ReasonProvisioning (False): resources are being created.
	ReasonProvisioning = "Provisioning"
	// ReasonResourceMissing (False): a bound or referenced resource does not exist.
	ReasonResourceMissing = "ResourceMissing"
	// ReasonResourceConflict (False): a resource exists but belongs to another owner or UID.
	ReasonResourceConflict = "ResourceConflict"
	// ReasonResourceError (False): a resource reports an error.
	ReasonResourceError = "ResourceError"
)

// Reasons of the MachineSetMember MachineScheduled condition.
const (
	// ReasonScheduled (True): the Machine is placed on a MachinePool.
	ReasonScheduled = "Scheduled"
	// ReasonCreating (False): the Machine is being created.
	ReasonCreating = "Creating"
	// ReasonWaitingForRelease (False): a predecessor Machine still holds a resource of the member.
	ReasonWaitingForRelease = "WaitingForRelease"
	// ReasonWaitingForCapacity (False): the Machine has no MachinePool assigned yet.
	ReasonWaitingForCapacity = "WaitingForCapacity"
	// ReasonMachineClassUnavailable (False): the MachineClass is missing or was recreated.
	ReasonMachineClassUnavailable = "MachineClassUnavailable"
	// ReasonNoMachine (False): the member has no Machine, for example while retained.
	ReasonNoMachine = "NoMachine"
)

// Reasons of the MachineSetMember PoolReachable condition.
const (
	// ReasonPoolReady (True): the MachinePool is Ready.
	ReasonPoolReady = "PoolReady"
	// ReasonPoolNotReady (False): the MachinePool reports not Ready.
	ReasonPoolNotReady = "PoolNotReady"
	// ReasonPoolUnreachable (Unknown): the MachinePool's Ready condition is Unknown.
	// Also used by the member's Ready condition.
	ReasonPoolUnreachable = "PoolUnreachable"
)

// Reasons of the MachineSetMember Ready condition. When several apply, the first one in
// MemberReadyReasonPriority wins.
const (
	// ReasonRunning (True): the Machine runs and power is On.
	ReasonRunning = "Running"
	// ReasonPoweredOff (True): the Machine is shut down and power is Off.
	ReasonPoweredOff = "PoweredOff"
	// ReasonStarting (False): the Machine is starting.
	ReasonStarting = "Starting"
	// ReasonGuestShutdown (False): the guest shut down although power is On.
	ReasonGuestShutdown = "GuestShutdown"
	// ReasonTerminated (False): the Machine is terminated.
	ReasonTerminated = "Terminated"
	// ReasonResourcesNotReady (False): ResourcesReady is not True.
	ReasonResourcesNotReady = "ResourcesNotReady"
	// ReasonNotScheduled (False): MachineScheduled is not True.
	ReasonNotScheduled = "NotScheduled"
	// ReasonRestoring (False): the member's Machine is being restored.
	// Also used by the MachineSet's Progressing condition.
	ReasonRestoring = "Restoring"
	// ReasonRetained (False): the member is retained and has no Machine.
	ReasonRetained = "Retained"
	// ReasonReclaiming (False): the member and its owned resources are being deleted.
	ReasonReclaiming = "Reclaiming"
)

// MemberReadyReasonPriority lists the non-True reasons of the member's Ready condition, highest
// priority first.
var MemberReadyReasonPriority = []string{
	ReasonReclaiming,
	ReasonRetained,
	ReasonPoolUnreachable,
	ReasonResourcesNotReady,
	ReasonRestoring,
	ReasonNotScheduled,
	ReasonTerminated,
	ReasonGuestShutdown,
	ReasonStarting,
}

// MachineSet condition types.
const (
	// MachineSetConditionMachineClassAvailable tells whether the MachineClass of the template
	// exists with the UID the set observed.
	MachineSetConditionMachineClassAvailable = "MachineClassAvailable"
	// MachineSetConditionDeleting is True while the set is being deleted; absent otherwise.
	MachineSetConditionDeleting = "Deleting"
	// MachineSetConditionReady is the summary condition of the set's members.
	MachineSetConditionReady = "Ready"
	// MachineSetConditionProgressing tells whether the set is changing its members.
	MachineSetConditionProgressing = "Progressing"
)

// Reasons of the MachineSet conditions.
const (
	// ReasonFound (MachineClassAvailable=True): the MachineClass exists.
	ReasonFound = "Found"
	// ReasonMachineClassNotFound (MachineClassAvailable=False): the MachineClass does not exist.
	ReasonMachineClassNotFound = "MachineClassNotFound"
	// ReasonMachineClassRecreated (MachineClassAvailable=False): the MachineClass was deleted and
	// created again with a different UID.
	ReasonMachineClassRecreated = "MachineClassRecreated"

	// ReasonDeleting (Deleting=True): the set is being deleted.
	ReasonDeleting = "Deleting"

	// ReasonAllMembersReady (Ready=True): all requested members are Ready.
	ReasonAllMembersReady = "AllMembersReady"
	// ReasonMembersNotReady (Ready=False): at least one requested member is not Ready.
	ReasonMembersNotReady = "MembersNotReady"
	// ReasonMembersUnknown (Ready=Unknown): the readiness of at least one member is Unknown.
	ReasonMembersUnknown = "MembersUnknown"

	// ReasonScalingUp (Progressing=True): members are being added.
	ReasonScalingUp = "ScalingUp"
	// ReasonScalingDown (Progressing=True): members are being removed.
	ReasonScalingDown = "ScalingDown"
	// ReasonStable (Progressing=False): there is nothing to do.
	ReasonStable = "Stable"
	// ReasonBlocked (Progressing=False): work is needed but cannot continue.
	ReasonBlocked = "Blocked"
)

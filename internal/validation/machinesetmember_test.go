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
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

var _ = Describe("ValidateMachineSetMember", func() {
	DescribeTable("on create",
		func(change func(*computev1alpha1.MachineSetMember), wantErr string) {
			member := newMember()
			change(member)
			expectErrors(ValidateMachineSetMember(member), wantErr)
		},
		Entry("accepts a valid member", func(*computev1alpha1.MachineSetMember) {}, ""),
		Entry("accepts an empty, defaulted lifecycle", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Lifecycle = ""
		}, ""),
		Entry("accepts a large ordinal (there is no upper limit)", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Ordinal = 5000
		}, ""),
		Entry("rejects a negative ordinal", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Ordinal = -1
		}, "spec.ordinal"),
		Entry("rejects an unknown lifecycle", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Lifecycle = "Paused"
		}, "spec.lifecycle: Unsupported value"),
		Entry("rejects a machineSetRef without a name", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.MachineSetRef.Name = ""
		}, "spec.machineSetRef.name: Required value"),
		Entry("rejects a machineSetRef without a UID", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.MachineSetRef.UID = ""
		}, "spec.machineSetRef.uid: Required value"),
	)

	DescribeTable("on update",
		func(change func(*computev1alpha1.MachineSetMember), wantErr string) {
			oldMember := newMember()
			newMember := oldMember.DeepCopy()
			change(newMember)
			expectErrors(ValidateMachineSetMemberUpdate(newMember, oldMember), wantErr)
		},
		Entry("accepts a change of lifecycle", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Lifecycle = computev1alpha1.MemberLifecycleRetained
		}, ""),
		Entry("rejects a change of machineSetRef", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.MachineSetRef.UID = "uid-2"
		}, "spec.machineSetRef: Forbidden: field is immutable"),
		Entry("rejects a change of ordinal", func(m *computev1alpha1.MachineSetMember) {
			m.Spec.Ordinal = 3
		}, "spec.ordinal: Forbidden: field is immutable"),
	)

	DescribeTable("status",
		func(change func(*computev1alpha1.MachineSetMemberStatus), wantErr string) {
			member := newMember()
			change(&member.Status)
			expectErrors(ValidateMachineSetMemberStatus(member), wantErr)
		},
		Entry("accepts a full status", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.MachineRef = &computev1alpha1.LocalUIDReference{Name: "brokers-2-abcde", UID: "uid-m"}
			s.MachineState = ironcorecomputev1alpha1.MachineStateRunning
			s.Volumes = []computev1alpha1.MemberVolumeStatus{{
				Name:      rootTemplate,
				Ownership: computev1alpha1.ResourceOwnershipManaged,
				VolumeRef: computev1alpha1.LocalUIDReference{Name: "brokers-2-root", UID: "uid-v"},
			}}
			s.NetworkInterfaces = []computev1alpha1.MemberNetworkInterfaceStatus{{
				Name:                primaryTemplate,
				Ownership:           computev1alpha1.ResourceOwnershipManaged,
				NetworkInterfaceRef: computev1alpha1.LocalUIDReference{Name: "brokers-2-primary", UID: "uid-n"},
				IPs:                 []string{"10.0.0.5", "2001:db8::5"},
				Prefixes:            []string{"10.1.0.0/24"},
				VirtualIP:           ptr.To("192.0.2.10"),
			}}
		}, ""),
		Entry("rejects an unknown machine state", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.MachineState = "Asleep"
		}, "status.machineState: Unsupported value"),
		Entry("rejects a volume without ownership", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.Volumes = []computev1alpha1.MemberVolumeStatus{{
				Name:      rootTemplate,
				VolumeRef: computev1alpha1.LocalUIDReference{Name: "v", UID: "u"},
			}}
		}, "status.volumes[0].ownership: Required value"),
		Entry("rejects an invalid IP", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.NetworkInterfaces = []computev1alpha1.MemberNetworkInterfaceStatus{{
				Name:                primaryTemplate,
				Ownership:           computev1alpha1.ResourceOwnershipExternal,
				NetworkInterfaceRef: computev1alpha1.LocalUIDReference{Name: "n", UID: "u"},
				IPs:                 []string{"10.0.0.300"},
			}}
		}, "status.networkInterfaces[0].ips[0]"),
		Entry("rejects an invalid prefix", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.NetworkInterfaces = []computev1alpha1.MemberNetworkInterfaceStatus{{
				Name:                primaryTemplate,
				Ownership:           computev1alpha1.ResourceOwnershipExternal,
				NetworkInterfaceRef: computev1alpha1.LocalUIDReference{Name: "n", UID: "u"},
				Prefixes:            []string{"10.1.0.0"},
			}}
		}, "status.networkInterfaces[0].prefixes[0]"),
		Entry("accepts a trigger for a Restoring operation", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.Operation = &computev1alpha1.MemberOperation{
				Type:    computev1alpha1.OperationRestoring,
				Trigger: computev1alpha1.RestorationTriggerFenced,
			}
		}, ""),
		Entry("rejects a trigger for another operation", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.Operation = &computev1alpha1.MemberOperation{
				Type:    computev1alpha1.OperationProvisioning,
				Trigger: computev1alpha1.RestorationTriggerFenced,
			}
		}, msgTriggerOnlyForRestoring),
		Entry("rejects an unknown trigger", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.Operation = &computev1alpha1.MemberOperation{Type: computev1alpha1.OperationRestoring, Trigger: "Timeout"}
		}, "status.operation.trigger: Unsupported value"),
		Entry("rejects an operation without a type", func(s *computev1alpha1.MachineSetMemberStatus) {
			s.Operation = &computev1alpha1.MemberOperation{}
		}, "status.operation.type: Required value"),
	)
})

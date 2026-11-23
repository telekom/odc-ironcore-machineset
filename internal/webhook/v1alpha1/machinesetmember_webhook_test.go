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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

func newMember() *computev1alpha1.MachineSetMember {
	return &computev1alpha1.MachineSetMember{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "brokers-", Namespace: testNamespace},
		Spec: computev1alpha1.MachineSetMemberSpec{
			MachineSetRef: computev1alpha1.LocalUIDReference{Name: "brokers", UID: "uid-1"},
			Ordinal:       2,
		},
	}
}

var _ = Describe("MachineSetMember webhook", func() {
	It("should default the lifecycle to Active", func() {
		member := newMember()
		create(member)
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(member), member)).To(Succeed())
		Expect(member.Spec.Lifecycle).To(Equal(computev1alpha1.MemberLifecycleActive))
	})

	It("should have printer columns for ordinal, lifecycle, machine and machine state", func() {
		Expect(printerColumns("machinesetmembers.compute.odc.telekom.com")).To(Equal(map[string]string{
			"Ordinal":       ".spec.ordinal",
			"Lifecycle":     ".spec.lifecycle",
			"Machine":       ".status.machineRef.name",
			"Machine State": ".status.machineState",
			"Age":           ".metadata.creationTimestamp",
		}))
	})

	It("should reject an invalid member on create", func() {
		member := newMember()
		member.Spec.Ordinal = -1
		expectInvalid(k8sClient.Create(ctx, member), "spec.ordinal")
	})

	It("should accept a lifecycle change and reject an ordinal change on update", func() {
		member := newMember()
		create(member)

		member.Spec.Lifecycle = computev1alpha1.MemberLifecycleRetained
		Expect(k8sClient.Update(ctx, member)).To(Succeed())

		member.Spec.Ordinal = 3
		expectInvalid(k8sClient.Update(ctx, member), "spec.ordinal: Forbidden: field is immutable")
	})

	It("should validate status updates", func() {
		member := newMember()
		create(member)

		member.Status.Operation = &computev1alpha1.MemberOperation{Type: computev1alpha1.OperationRestoring}
		Expect(k8sClient.Status().Update(ctx, member)).To(Succeed())

		member.Status.Operation.Type = "Migrating"
		expectInvalid(k8sClient.Status().Update(ctx, member), "status.operation.type: Unsupported value")
	})
})

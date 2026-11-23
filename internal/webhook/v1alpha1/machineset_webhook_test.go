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
	"os"
	"path/filepath"

	ironcorecomputev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

const testNamespace = "default"

// newSampleMachineSet returns the sample from config/samples with a unique name.
func newSampleMachineSet() *computev1alpha1.MachineSet {
	GinkgoHelper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "samples", "compute_v1alpha1_machineset.yaml"))
	Expect(err).NotTo(HaveOccurred())
	machineSet := &computev1alpha1.MachineSet{}
	Expect(yaml.UnmarshalStrict(data, machineSet)).To(Succeed())
	machineSet.Name = ""
	machineSet.GenerateName = "sample-"
	machineSet.Namespace = testNamespace
	return machineSet
}

// create creates obj and deletes it when the test ends.
func create(obj client.Object) {
	GinkgoHelper()
	Expect(k8sClient.Create(ctx, obj)).To(Succeed())
	DeferCleanup(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, obj))).To(Succeed())
	})
}

// expectInvalid checks that the API server rejected a request with an "Invalid" error that
// contains wantErr.
func expectInvalid(err error, wantErr string) {
	GinkgoHelper()
	Expect(apierrors.IsInvalid(err)).To(BeTrue(), "expected an Invalid error, got: %v", err)
	Expect(err.Error()).To(ContainSubstring(wantErr))
}

// printerColumns returns the printer columns of the CRD's first version, by name.
func printerColumns(crdName string) map[string]string {
	GinkgoHelper()
	crd := &apiextensionsv1.CustomResourceDefinition{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crdName}, crd)).To(Succeed())
	columns := map[string]string{}
	for _, column := range crd.Spec.Versions[0].AdditionalPrinterColumns {
		columns[column.Name] = column.JSONPath
	}
	return columns
}

var _ = Describe("MachineSet webhook", func() {
	It("should accept the sample", func() {
		create(newSampleMachineSet())
	})

	It("should apply the defaults before the webhook runs", func() {
		machineSet := newSampleMachineSet()
		machineSet.Spec.Replicas = nil
		machineSet.Spec.Template.Spec.Power = ""
		machineSet.Spec.ResourceRetentionPolicy.WhenScaledDown = ""
		create(machineSet)

		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(machineSet), machineSet)).To(Succeed())
		Expect(machineSet.Spec.Replicas).To(Equal(ptr.To[int32](1)))
		Expect(machineSet.Spec.Template.Spec.Power).To(Equal(ironcorecomputev1alpha1.PowerOn))
		Expect(machineSet.Spec.ResourceRetentionPolicy.WhenScaledDown).To(Equal(computev1alpha1.RetentionPolicyRetain))
	})

	It("should have printer columns for replicas and ready members", func() {
		Expect(printerColumns("machinesets.compute.odc.telekom.com")).To(Equal(map[string]string{
			"Replicas": ".spec.replicas",
			"Ready":    ".status.readyReplicas",
			"Age":      ".metadata.creationTimestamp",
		}))
	})

	It("should reject an invalid MachineSet on create", func() {
		machineSet := newSampleMachineSet()
		machineSet.Spec.Replicas = ptr.To[int32](-1)
		expectInvalid(k8sClient.Create(ctx, machineSet), "spec.replicas")
	})

	It("should accept a scale change and reject a MachineClass change on update", func() {
		machineSet := newSampleMachineSet()
		create(machineSet)

		machineSet.Spec.Replicas = ptr.To[int32](5)
		Expect(k8sClient.Update(ctx, machineSet)).To(Succeed())

		machineSet.Spec.Template.Spec.MachineClassRef.Name = "other-class"
		expectInvalid(k8sClient.Update(ctx, machineSet), "class rollout, which is not supported")
	})

	It("should validate status updates", func() {
		machineSet := newSampleMachineSet()
		create(machineSet)

		machineSet.Status.Members = []computev1alpha1.MachineSetMemberSummary{{
			Ordinal:   0,
			Lifecycle: computev1alpha1.MemberLifecycleActive,
		}}
		Expect(k8sClient.Status().Update(ctx, machineSet)).To(Succeed())

		machineSet.Status.Members[0].Lifecycle = "Paused"
		expectInvalid(k8sClient.Status().Update(ctx, machineSet), "status.members[0].lifecycle")
	})
})

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
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
	"github.com/telekom/odc-ironcore-machineset/internal/validation"
)

// SetupMachineSetMemberWebhookWithManager registers the webhook for MachineSetMember in the manager.
func SetupMachineSetMemberWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &computev1alpha1.MachineSetMember{}).
		WithValidator(&MachineSetMemberCustomValidator{}).
		Complete()
}

// The webhook also receives writes to the status subresource, so that the controller cannot store
// an invalid status either.
// +kubebuilder:webhook:path=/validate-compute-odc-telekom-com-v1alpha1-machinesetmember,mutating=false,failurePolicy=fail,sideEffects=None,groups=compute.odc.telekom.com,resources=machinesetmembers;machinesetmembers/status,verbs=create;update,versions=v1alpha1,name=vmachinesetmember-v1alpha1.kb.io,admissionReviewVersions=v1

// MachineSetMemberCustomValidator validates MachineSetMembers when they are created or updated.
type MachineSetMemberCustomValidator struct{}

// ValidateCreate validates a new MachineSetMember.
func (v *MachineSetMemberCustomValidator) ValidateCreate(
	_ context.Context,
	obj *computev1alpha1.MachineSetMember,
) (admission.Warnings, error) {
	return nil, invalidError("MachineSetMember", obj.Name, validation.ValidateMachineSetMember(obj))
}

// ValidateUpdate validates a change of a MachineSetMember. A write to the status subresource can
// only change the status, so only the status is checked then.
func (v *MachineSetMemberCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj *computev1alpha1.MachineSetMember,
) (admission.Warnings, error) {
	if isStatusRequest(ctx) {
		return nil, invalidError("MachineSetMember", newObj.Name, validation.ValidateMachineSetMemberStatus(newObj))
	}
	return nil, invalidError("MachineSetMember", newObj.Name,
		validation.ValidateMachineSetMemberUpdate(newObj, oldObj))
}

// ValidateDelete is not called: the webhook is registered only for create and update.
func (v *MachineSetMemberCustomValidator) ValidateDelete(
	_ context.Context,
	_ *computev1alpha1.MachineSetMember,
) (admission.Warnings, error) {
	return nil, nil
}

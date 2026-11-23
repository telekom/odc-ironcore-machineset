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

// Package v1alpha1 holds the validating admission webhooks of the compute.odc.telekom.com/v1alpha1
// API. A validating webhook is an HTTPS endpoint in the operator that the Kubernetes API server
// calls before it stores a create or update. If the webhook answers with an error, the API server
// rejects the request and shows the error to the user.
//
// The rules themselves live in internal/validation; this package only connects them to the API
// server.
package v1alpha1

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
	"github.com/telekom/odc-ironcore-machineset/internal/validation"
)

// SetupMachineSetWebhookWithManager registers the webhook for MachineSet in the manager.
func SetupMachineSetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &computev1alpha1.MachineSet{}).
		WithValidator(&MachineSetCustomValidator{}).
		Complete()
}

// The webhook also receives writes to the status subresource, so that the controller cannot store
// an invalid status either.
// +kubebuilder:webhook:path=/validate-compute-odc-telekom-com-v1alpha1-machineset,mutating=false,failurePolicy=fail,sideEffects=None,groups=compute.odc.telekom.com,resources=machinesets;machinesets/status,verbs=create;update,versions=v1alpha1,name=vmachineset-v1alpha1.kb.io,admissionReviewVersions=v1

// MachineSetCustomValidator validates MachineSets when they are created or updated.
type MachineSetCustomValidator struct{}

// ValidateCreate validates a new MachineSet.
func (v *MachineSetCustomValidator) ValidateCreate(
	_ context.Context,
	obj *computev1alpha1.MachineSet,
) (admission.Warnings, error) {
	return nil, invalidError("MachineSet", obj.Name, validation.ValidateMachineSet(obj))
}

// ValidateUpdate validates a change of a MachineSet. A write to the status subresource can only
// change the status, so only the status is checked then.
func (v *MachineSetCustomValidator) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj *computev1alpha1.MachineSet,
) (admission.Warnings, error) {
	if isStatusRequest(ctx) {
		return nil, invalidError("MachineSet", newObj.Name, validation.ValidateMachineSetStatus(newObj))
	}
	return nil, invalidError("MachineSet", newObj.Name, validation.ValidateMachineSetUpdate(newObj, oldObj))
}

// ValidateDelete is not called: the webhook is registered only for create and update.
func (v *MachineSetCustomValidator) ValidateDelete(
	_ context.Context,
	_ *computev1alpha1.MachineSet,
) (admission.Warnings, error) {
	return nil, nil
}

// invalidError turns validation errors into the "Invalid" API error that kubectl shows, for
// example: MachineSet "brokers" is invalid: spec.replicas: Invalid value: -1: ...
// It returns nil if there are no errors.
func invalidError(kind, name string, errs field.ErrorList) error {
	if len(errs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: computev1alpha1.GroupName, Kind: kind}, name, errs)
}

// isStatusRequest reports whether the request writes the status subresource.
func isStatusRequest(ctx context.Context) bool {
	req, err := admission.RequestFromContext(ctx)
	return err == nil && req.SubResource == "status"
}

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

// Package validation holds the validation rules of the MachineSet API as plain Go functions.
//
// Each function checks an object and returns a field.ErrorList: one entry per problem, with the
// path of the bad field (for example "spec.template.spec.volumes[0]") and a message. An empty list
// means the object is valid. The validating webhook in internal/webhook calls these functions.
//
// The functions do not depend on the webhook, so they can be unit-tested without a cluster and
// can later move into IronCore's API server, which validates its types in the same way.
package validation

import (
	"fmt"
	"strings"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

const (
	immutableMessage = "field is immutable"
	reservedMessage  = "labels must not use the reserved prefix " + computev1alpha1.LabelPrefix
)

// validateRefName checks the name in a reference to another object. Kubernetes object names are
// DNS subdomains: lowercase letters, digits, '-' and '.', at most 253 characters.
func validateRefName(name string, fldPath *field.Path) field.ErrorList {
	if name == "" {
		return field.ErrorList{field.Required(fldPath, "")}
	}
	var allErrs field.ErrorList
	for _, msg := range apivalidation.NameIsDNSSubdomain(name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath, name, msg))
	}
	return allErrs
}

// validateDNSLabel checks a name that must be a DNS label: lowercase letters, digits and '-', at
// most 63 characters. Template and attachment names must be DNS labels because they become part
// of the names of the created objects.
func validateDNSLabel(name string, fldPath *field.Path) field.ErrorList {
	if name == "" {
		return field.ErrorList{field.Required(fldPath, "")}
	}
	var allErrs field.ErrorList
	for _, msg := range apivalidation.NameIsDNSLabel(name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath, name, msg))
	}
	return allErrs
}

// validateUserLabels checks labels that the user writes into a template. They must be valid
// Kubernetes labels and must not use the prefix that the controller reserves for its own labels.
func validateUserLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := metav1validation.ValidateLabels(labels, fldPath)
	for key := range labels {
		if strings.HasPrefix(key, computev1alpha1.LabelPrefix) {
			allErrs = append(allErrs, field.Invalid(fldPath.Key(key), key, reservedMessage))
		}
	}
	return allErrs
}

// validateObjectTemplateMetadata checks the labels and annotations of a nested object template.
func validateObjectTemplateMetadata(
	metadata computev1alpha1.ObjectTemplateMetadata,
	fldPath *field.Path,
) field.ErrorList {
	allErrs := metav1validation.ValidateLabels(metadata.Labels, fldPath.Child("labels"))
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(metadata.Annotations, fldPath.Child("annotations"))...)
	return allErrs
}

// validateExactlyOne checks that exactly one of the given sources is set.
func validateExactlyOne(fldPath *field.Path, message string, sources ...bool) field.ErrorList {
	count := 0
	for _, isSet := range sources {
		if isSet {
			count++
		}
	}
	if count != 1 {
		return field.ErrorList{field.Invalid(fldPath, fmt.Sprintf("%d sources set", count), message)}
	}
	return nil
}

// validateEnum checks that value is one of the allowed values. An empty value is accepted only
// if allowEmpty is true; this is used for fields that the API server fills with a default.
func validateEnum[T ~string](value T, allowEmpty bool, fldPath *field.Path, allowed ...T) field.ErrorList {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return field.ErrorList{field.Required(fldPath, "")}
	}
	if !sets.New(allowed...).Has(value) {
		return field.ErrorList{field.NotSupported(fldPath, value, allowed)}
	}
	return nil
}

// validateImmutable returns an error if the field changed. equality.Semantic treats an empty list
// or map the same as a missing one, so clients that drop empty fields are not rejected.
func validateImmutable(newValue, oldValue any, fldPath *field.Path, message string) field.ErrorList {
	if apiequality.Semantic.DeepEqual(newValue, oldValue) {
		return nil
	}
	return field.ErrorList{field.Forbidden(fldPath, message)}
}

// validateUIDRef checks a reference by name and UID.
func validateUIDRef(ref computev1alpha1.LocalUIDReference, fldPath *field.Path) field.ErrorList {
	allErrs := validateRefName(ref.Name, fldPath.Child("name"))
	if ref.UID == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("uid"), ""))
	}
	return allErrs
}

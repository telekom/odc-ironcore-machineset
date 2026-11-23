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
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
)

var ipFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}

// validateNetworkInterfaceSpec checks the spec of a network interface template. With multiMember,
// it also rejects inputs that can belong to one Machine only: fixed IPs and prefixes and a
// reference to an existing VirtualIP.
func validateNetworkInterfaceSpec(
	spec *computev1alpha1.NetworkInterfaceSpec,
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	allErrs := validateRefName(spec.NetworkRef.Name, fldPath.Child("networkRef", "name"))
	allErrs = append(allErrs, validateIPFamilies(spec.IPFamilies, fldPath.Child("ipFamilies"))...)

	ipsPath := fldPath.Child("ips")
	if len(spec.IPs) == 0 {
		allErrs = append(allErrs, field.Required(ipsPath, "at least one IP must be set"))
	}
	for i, ip := range spec.IPs {
		allErrs = append(allErrs, validateAddressSource(ip.Value, ip.Ephemeral, validation.IsValidIP, multiMember,
			ipsPath.Index(i))...)
	}
	prefixesPath := fldPath.Child("prefixes")
	for i, prefix := range spec.Prefixes {
		allErrs = append(allErrs, validateAddressSource(prefix.Value, prefix.Ephemeral, validation.IsValidCIDR,
			multiMember, prefixesPath.Index(i))...)
	}
	if spec.VirtualIP != nil {
		allErrs = append(allErrs, validateVirtualIPSource(spec.VirtualIP, multiMember, fldPath.Child("virtualIP"))...)
	}
	return allErrs
}

func validateIPFamilies(families []corev1.IPFamily, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	switch {
	case len(families) == 0:
		allErrs = append(allErrs, field.Required(fldPath, ""))
	case len(families) > maxIPFamilies:
		allErrs = append(allErrs, field.TooMany(fldPath, len(families), maxIPFamilies))
	}
	seen := sets.New[corev1.IPFamily]()
	for i, family := range families {
		allErrs = append(allErrs, validateEnum(family, false, fldPath.Index(i), ipFamilies...)...)
		if seen.Has(family) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(i), family))
		}
		seen.Insert(family)
	}
	return allErrs
}

// validateAddressSource checks an IP or prefix that is either a fixed value or allocated from an
// ephemeral Prefix. isValid checks the format of a fixed value (IP address or CIDR).
func validateAddressSource(
	value *string,
	ephemeral *computev1alpha1.EphemeralPrefixSource,
	isValid func(*field.Path, string) field.ErrorList,
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	allErrs := validateExactlyOne(fldPath, msgValueOrEphemeral, value != nil, ephemeral != nil)
	if value != nil {
		allErrs = append(allErrs, isValid(fldPath.Child("value"), *value)...)
		if multiMember {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("value"), msgFixedAddress))
		}
	}
	if ephemeral != nil {
		templatePath := fldPath.Child("ephemeral", "prefixTemplate")
		allErrs = append(allErrs, validatePrefixTemplate(&ephemeral.PrefixTemplate, templatePath)...)
		if multiMember && ephemeral.PrefixTemplate.Spec.Prefix != nil {
			allErrs = append(allErrs, field.Forbidden(templatePath.Child("spec", "prefix"), msgFixedAddress))
		}
	}
	return allErrs
}

func validatePrefixTemplate(template *computev1alpha1.PrefixTemplate, fldPath *field.Path) field.ErrorList {
	allErrs := validateObjectTemplateMetadata(template.Metadata, fldPath.Child("metadata"))
	specPath := fldPath.Child("spec")
	spec := template.Spec
	allErrs = append(allErrs, validateEnum(spec.IPFamily, true, specPath.Child("ipFamily"), ipFamilies...)...)
	if spec.Prefix != nil {
		allErrs = append(allErrs, validation.IsValidCIDR(specPath.Child("prefix"), *spec.Prefix)...)
	}
	if spec.PrefixLength < 0 || spec.PrefixLength > maxPrefixLength {
		allErrs = append(allErrs, field.Invalid(specPath.Child("prefixLength"), spec.PrefixLength,
			"must be between 0 and 128"))
	}
	if spec.ParentRef != nil {
		allErrs = append(allErrs, validateRefName(spec.ParentRef.Name, specPath.Child("parentRef", "name"))...)
	}
	if spec.ParentSelector != nil {
		allErrs = append(allErrs, metav1validation.ValidateLabelSelector(spec.ParentSelector,
			metav1validation.LabelSelectorValidationOptions{}, specPath.Child("parentSelector"))...)
	}
	return allErrs
}

func validateVirtualIPSource(
	source *computev1alpha1.VirtualIPSource,
	multiMember bool,
	fldPath *field.Path,
) field.ErrorList {
	allErrs := validateExactlyOne(fldPath, msgVirtualIPSource, source.VirtualIPRef != nil, source.Ephemeral != nil)
	if source.VirtualIPRef != nil {
		allErrs = append(allErrs, validateRefName(source.VirtualIPRef.Name, fldPath.Child("virtualIPRef", "name"))...)
		if multiMember {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("virtualIPRef"), msgVirtualIPRef))
		}
	}
	if source.Ephemeral != nil {
		templatePath := fldPath.Child("ephemeral", "virtualIPTemplate")
		template := source.Ephemeral.VirtualIPTemplate
		allErrs = append(allErrs, validateObjectTemplateMetadata(template.Metadata, templatePath.Child("metadata"))...)
		specPath := templatePath.Child("spec")
		allErrs = append(allErrs, validateEnum(template.Spec.Type, false, specPath.Child("type"),
			networkingv1alpha1.VirtualIPTypePublic)...)
		allErrs = append(allErrs, validateEnum(template.Spec.IPFamily, false, specPath.Child("ipFamily"),
			ipFamilies...)...)
		reclaimPolicyPath := specPath.Child("reclaimPolicy")
		if template.Spec.ReclaimPolicy == networkingv1alpha1.ReclaimPolicyTypeRetain {
			allErrs = append(allErrs, field.Forbidden(reclaimPolicyPath, msgVirtualIPRetain))
		} else {
			allErrs = append(allErrs, validateEnum(template.Spec.ReclaimPolicy, true, reclaimPolicyPath,
				networkingv1alpha1.ReclaimPolicyTypeDelete)...)
		}
	}
	return allErrs
}

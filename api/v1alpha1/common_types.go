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
	"k8s.io/apimachinery/pkg/types"
)

// The validation rules of this API are Go functions in internal/validation, run by a validating
// webhook. The markers here only describe the shape of the objects and their defaults.

// LocalObjectReference references an object in the same namespace by name.
type LocalObjectReference struct {
	// name of the referenced object.
	// +required
	Name string `json:"name"`
}

// LocalUIDReference references an object in the same namespace by name and UID. The UID makes
// sure that an object recreated under the same name is not mistaken for the original.
type LocalUIDReference struct {
	// name of the referenced object.
	// +required
	Name string `json:"name"`

	// uid of the referenced object.
	// +required
	UID types.UID `json:"uid"`
}

// TemplateMetadata is the metadata of a volume or network interface template.
type TemplateMetadata struct {
	// name of the template. Attachments reference the template by this name, and it becomes
	// part of the names of the objects created from it.
	// +required
	Name string `json:"name"`

	// labels are copied onto every object created from the template. Labels with the prefix
	// machineset.compute.odc.telekom.com/ are reserved for the controller.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

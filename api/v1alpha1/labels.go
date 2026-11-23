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

// Labels the controller sets on the objects it creates for a member. IronCore's NetworkPolicy and
// LoadBalancer select NetworkInterfaces only by label, so these labels make members selectable.
//
// Labels from the templates are copied onto the created objects; these labels win on conflict.
// Externally referenced resources are never labeled.
const (
	// LabelPrefix is the prefix of all labels owned by the MachineSet controller.
	// Template labels must not use it.
	LabelPrefix = "machineset." + GroupName + "/"

	// LabelMachineSetName holds the name of the MachineSet.
	// Set on MachineSetMembers, Machines, Volumes and NetworkInterfaces.
	LabelMachineSetName = LabelPrefix + "name"
	// LabelMachineSetUID holds the UID of the MachineSet.
	// Set on MachineSetMembers, Machines, Volumes and NetworkInterfaces.
	LabelMachineSetUID = LabelPrefix + "uid"
	// LabelOrdinal holds the member's ordinal ("0", "1", ...).
	// Set on MachineSetMembers, Machines, Volumes and NetworkInterfaces.
	LabelOrdinal = LabelPrefix + "ordinal"
	// LabelTemplate holds the name of the volume or network interface template an object was
	// created from, for example "primary". Set on Volumes and NetworkInterfaces.
	LabelTemplate = LabelPrefix + "template"
)

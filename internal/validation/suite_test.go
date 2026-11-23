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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// These are plain unit tests: they call the validation functions directly, without a cluster.
func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

// expectErrors checks the result of a validation function. An empty wantErr means the object
// must be valid; otherwise one of the errors must contain wantErr. An error reads like
// "spec.replicas: Invalid value: -1: must be greater than or equal to 0", so wantErr can name the
// field path, the message, or both.
func expectErrors(errs field.ErrorList, wantErr string) {
	GinkgoHelper()
	if wantErr == "" {
		Expect(errs).To(BeEmpty())
		return
	}
	Expect(errs).NotTo(BeEmpty(), "expected an error containing %q", wantErr)
	Expect(errs.ToAggregate().Error()).To(ContainSubstring(wantErr))
}

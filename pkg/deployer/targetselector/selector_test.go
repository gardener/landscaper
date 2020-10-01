// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package targetselector_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/selection"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/targetselector"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Target Selectors test suite")
}

var _ = Describe("Selectors", func() {

	var target *lsv1alpha1.Target

	BeforeEach(func() {
		target = &lsv1alpha1.Target{}
	})

	Context("Annotations", func() {
		It("should pass if all annotations match", func() {
			target.Annotations = map[string]string{
				"ann1": "val1",
				"ann2": "val2",
			}
			req := []lsv1alpha1.Requirement{
				{
					Key:      "ann1",
					Operator: selection.Equals,
					Values:   []string{"val1"},
				},
				{
					Key:      "ann2",
					Operator: selection.Equals,
					Values:   []string{"val2"},
				},
			}
			ok, err := targetselector.MatchAnnotations(target, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should pass if a target does not have a given annotation", func() {
			target.Annotations = map[string]string{
				"ann1": "key1",
			}
			req := []lsv1alpha1.Requirement{
				{
					Key:      "deployer.landscaper.cloud/class",
					Operator: selection.DoesNotExist,
				},
			}
			ok, err := targetselector.MatchAnnotations(target, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should forbid if one annotations does not match", func() {
			target.Annotations = map[string]string{
				"ann1": "key1",
				"ann2": "key2",
			}
			req := []lsv1alpha1.Requirement{
				{
					Key:      "ann1",
					Operator: selection.Equals,
					Values:   []string{"val1"},
				},
			}
			ok, err := targetselector.MatchAnnotations(target, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})

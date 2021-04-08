// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetselector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/selection"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
)

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
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Annotations: req})
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
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Annotations: req})
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
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Annotations: req})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Context("Labels", func() {
		It("should pass if all labels match", func() {
			target.Labels = map[string]string{
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
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Labels: req})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should pass if a target does not have a given labels", func() {
			target.Labels = map[string]string{
				"ann1": "key1",
			}
			req := []lsv1alpha1.Requirement{
				{
					Key:      "deployer.landscaper.cloud/class",
					Operator: selection.DoesNotExist,
				},
			}
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Labels: req})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should forbid if one label does not match", func() {
			target.Labels = map[string]string{
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
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Labels: req})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})

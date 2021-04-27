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

	Context("ObjectReference", func() {
		It("should pass if a reference name and namespace matches", func() {
			target.Name = "mytarget"
			target.Namespace = "test"
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Targets: []lsv1alpha1.ObjectReference{
				{
					Name:      "mytarget",
					Namespace: "test",
				},
			}})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should pass if a reference name matches and no namespace is provided", func() {
			target.Name = "mytarget"
			target.Namespace = "test"
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Targets: []lsv1alpha1.ObjectReference{
				{
					Name: "mytarget",
				},
			}})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should not pass if a reference namespace do not match", func() {
			target.Name = "mytarget"
			target.Namespace = "test"
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Targets: []lsv1alpha1.ObjectReference{
				{
					Name:      "mytarget",
					Namespace: "te",
				},
			}})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("should not pass if a reference name do not match", func() {
			target.Name = "mytarget"
			target.Namespace = "test"
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Targets: []lsv1alpha1.ObjectReference{
				{
					Name:      "my",
					Namespace: "test",
				},
			}})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

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

		It("should forbid if a target does not have a given annotation", func() {
			target.Annotations = map[string]string{
				"deployer.landscaper.cloud/class": "key1",
			}
			req := []lsv1alpha1.Requirement{
				{
					Key:      "deployer.landscaper.cloud/class",
					Operator: selection.DoesNotExist,
				},
			}
			ok, err := targetselector.MatchSelector(target, lsv1alpha1.TargetSelector{Annotations: req})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
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

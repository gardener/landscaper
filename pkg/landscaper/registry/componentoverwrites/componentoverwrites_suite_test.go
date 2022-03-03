// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Version Overwrites Test Suite")
}

var _ = Describe("ComponentVersionOverwrites", func() {

	var (
		cdRef *lsv1alpha1.ComponentDescriptorReference
	)

	BeforeEach(func() {
		cdRef = &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: testutils.ExampleRepositoryContext(),
			ComponentName:     "component.example.com",
			Version:           "v1.0.0",
		}
	})

	Context("Matcher", func() {

		It("should not match if the component name is different", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName: "different.example.com",
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeFalse())
		})

		It("should not match if the version is different", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						Version: "v0.1.0",
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeFalse())
		})

		It("should not match if the repository context is different", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: testutils.DefaultRepositoryContext("different.com"),
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeFalse())
		})

		It("should not match if the name fits but the version differs", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName: cdRef.ComponentName,
						Version:       "v0.1.0",
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeFalse())
		})

		It("should match with only the component name given", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName: cdRef.ComponentName,
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
		})

		It("should match with only the version given", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						Version: cdRef.Version,
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
		})

		It("should match with only the repository context given", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: cdRef.RepositoryContext.DeepCopy(),
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
		})

		It("should match with name and version given", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName: cdRef.ComponentName,
						Version:       cdRef.Version,
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
		})

		It("should match with name, version and repository context given", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName:     cdRef.ComponentName,
						Version:           cdRef.Version,
						RepositoryContext: cdRef.RepositoryContext.DeepCopy(),
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
		})

	})

	Context("Overwriter", func() {

		It("should replace only the name if only the name is specified in replacement ref", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
						ComponentName: "overwritten",
					},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
			Expect(cdRef).To(PointTo(Equal(lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: cdRef.RepositoryContext,
				ComponentName:     "overwritten",
				Version:           cdRef.Version,
			})))
		})

		It("should replace only the version if only the version is specified in replacement ref", func() {
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
						Version: "overwritten",
					},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
			Expect(cdRef).To(PointTo(Equal(lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: cdRef.RepositoryContext,
				ComponentName:     cdRef.ComponentName,
				Version:           "overwritten",
			})))
		})

		It("should replace only the repository context if only the repository context is specified in replacement ref", func() {
			repoCtx := testutils.DefaultRepositoryContext("foo.bar.com")
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: repoCtx,
					},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
			Expect(cdRef).To(PointTo(Equal(lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repoCtx,
				ComponentName:     cdRef.ComponentName,
				Version:           cdRef.Version,
			})))
		})

		It("should replace everything if everything is specified in replacement ref", func() {
			repoCtx := testutils.DefaultRepositoryContext("foo.bar.com")
			subs := componentoverwrites.NewSubstitutionManager([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: repoCtx,
						ComponentName:     "overwritten",
						Version:           "v2.0.0",
					},
				},
			})
			Expect(subs.Replace(cdRef)).To(BeTrue())
			Expect(cdRef).To(PointTo(Equal(lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repoCtx,
				ComponentName:     "overwritten",
				Version:           "v2.0.0",
			})))
		})

	})

})

// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
)

var _ = Describe("Reconcile", func() {

	Context("HandleComponenReference", func() {
		It("should default the repository context", func() {
			type custom struct {
				cdv2.ObjectType
				BaseURL string `json:"baseUrl"`
			}

			repoCtx, err := cdv2.NewUnstructured(&custom{
				ObjectType: cdv2.ObjectType{
					Type: "mycustom",
				},
				BaseURL: "test",
			})
			Expect(err).ToNot(HaveOccurred())
			c := &installationsctl.Controller{
				LsConfig: &config.LandscaperConfiguration{
					RepositoryContext: &repoCtx,
				},
			}
			inst := &lsv1alpha1.Installation{}
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{}
			inst.Spec.ComponentDescriptor.Reference = &lsv1alpha1.ComponentDescriptorReference{}

			Expect(c.HandleComponentReference(inst)).To(Succeed())
			Expect(inst.Spec.ComponentDescriptor.Reference.RepositoryContext.Object).To(Equal(repoCtx.Object))
		})
	})

})

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Context", func() {

	var state *envtest.State

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(state.CleanupState(context.TODO(), testenv.Client, nil)).To(Succeed())
	})

	Context("GetContext", func() {
		It("should default the repository context", func() {
			ctx := context.Background()
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
			lsCtx := &lsv1alpha1.Context{}
			lsCtx.Name = "cc"
			lsCtx.Namespace = state.Namespace
			lsCtx.RepositoryContext = &repoCtx
			Expect(state.Create(ctx, testenv.Client, lsCtx)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Namespace = state.Namespace
			inst.Spec.Context = "cc"
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{}
			inst.Spec.ComponentDescriptor.Reference = &lsv1alpha1.ComponentDescriptorReference{}

			res, err := installations.GetContext(ctx, testenv.Client, inst, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.RepositoryContext.Object).To(Equal(repoCtx.Object))
			Expect(inst.Spec.ComponentDescriptor.Reference.RepositoryContext).ToNot(BeNil())
		})
	})
})

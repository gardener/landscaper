// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	mock_componentrepository "github.com/gardener/landscaper/pkg/landscaper/registry/components/mock"
)

var _ = Describe("Resolve", func() {
	var (
		ctrl     *gomock.Controller
		cdClient *mock_componentrepository.MockComponentResolver

		repoCtx = []cdv2.RepositoryContext{
			{Type: cdv2.OCIRegistryType, BaseURL: "example.com"},
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		cdClient = mock_componentrepository.NewMockComponentResolver(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should resolve 2 direct transitive components", func() {
		ctx := context.Background()
		defer ctx.Done()

		l11_CD := cdv2.ComponentDescriptor{}
		l11_CD.RepositoryContexts = repoCtx
		l12_CD := cdv2.ComponentDescriptor{}
		l12_CD.RepositoryContexts = repoCtx

		cd := cdv2.ComponentDescriptor{}
		cd.RepositoryContexts = repoCtx
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name:          "l11",
				ComponentName: "l11",
				Version:       "0.0.1",
			},
			{
				Name:          "l12",
				ComponentName: "l12",
				Version:       "0.0.1",
			},
		}

		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[0].Name, cd.ComponentReferences[0].Version).Return(&l11_CD, nil, nil)
		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[1].Name, cd.ComponentReferences[1].Version).Return(&l12_CD, nil, nil)

		res, err := cdutils.ResolveEffectiveComponentDescriptor(ctx, cdClient, cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.ComponentReferences).To(HaveKey("l11"))
		Expect(res.ComponentReferences).To(HaveKey("l12"))
	})

	It("should recursively resolve transitive components", func() {
		ctx := context.Background()
		defer ctx.Done()

		l111_CD := cdv2.ComponentDescriptor{}
		l111_CD.RepositoryContexts = repoCtx

		l11_CD := cdv2.ComponentDescriptor{}
		l11_CD.RepositoryContexts = repoCtx
		l11_CD.ComponentReferences = []cdv2.ComponentReference{
			{
				Name:          "l111",
				ComponentName: "l111",
				Version:       "0.0.1",
			},
		}

		cd := cdv2.ComponentDescriptor{}
		cd.RepositoryContexts = repoCtx
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name:          "l11",
				ComponentName: "l11",
				Version:       "0.0.1",
			},
		}

		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[0].Name, cd.ComponentReferences[0].Version).Return(&l11_CD, nil, nil)
		cdClient.EXPECT().Resolve(ctx, repoCtx[0], l11_CD.ComponentReferences[0].Name, l11_CD.ComponentReferences[0].Version).Return(&l111_CD, nil, nil)

		res, err := cdutils.ResolveEffectiveComponentDescriptor(ctx, cdClient, cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.ComponentReferences).To(HaveKey("l11"))
		Expect(res.ComponentReferences["l11"].ComponentReferences).To(HaveKey("l111"))
	})
})

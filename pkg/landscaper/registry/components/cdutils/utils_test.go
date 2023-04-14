// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	testutils "github.com/gardener/landscaper/test/utils"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

var (
	fakeCompRepo ctf.ComponentResolver
	repoCtx      *cdv2.UnstructuredTypedObject
	ctx          context.Context
	err          error
)

var _ = BeforeSuite(func() {
	fakeCompRepo, err = componentsregistry.NewLocalClient("../../testdata/registry")
	repoCtx = testutils.LocalRepositoryContext("../../testdata/registry")
	Expect(err).ToNot(HaveOccurred())
})

type componentIdentifier struct {
	Name    string
	Version string
}

func componentIdentifierFromCD(cd cdv2.ComponentDescriptor) componentIdentifier {
	return componentIdentifier{Name: cd.Name, Version: cd.Version}
}

var _ = Describe("cdutils Tests", func() {
	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should return each referenced component only once when resolving component references", func() {
		cd, err := cdutils.ResolveWithOverwriter(ctx, fakeCompRepo, repoCtx, "example.com/root", "v1.0.0", nil)
		testutils.ExpectNoError(err)
		Expect(cd).ToNot(BeNil())
		cdList, err := cdutils.ResolveToComponentDescriptorList(ctx, fakeCompRepo, *cd, repoCtx, nil)
		testutils.ExpectNoError(err)
		fetchedComponents := []componentIdentifier{}
		for _, fcd := range cdList.Components {
			fetchedComponents = append(fetchedComponents, componentIdentifierFromCD(fcd))
		}
		Expect(fetchedComponents).To(ConsistOf(
			componentIdentifier{
				Name:    "example.com/root",
				Version: "v1.0.0",
			},
			componentIdentifier{
				Name:    "example.com/a",
				Version: "v1.0.0",
			},
			componentIdentifier{
				Name:    "example.com/b",
				Version: "v1.0.0",
			},
			componentIdentifier{
				Name:    "example.com/ab",
				Version: "v1.0.0",
			},
		))
	})

})

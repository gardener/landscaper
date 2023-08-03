// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/config"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

var (
	registryAccess    model.RegistryAccess
	repositoryContext types.UnstructuredTypedObject
	ctx               context.Context
	err               error
)

var _ = BeforeSuite(func() {
	localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "./testdata/registry"}
	registryAccess, err = GetFactory().NewRegistryAccess(context.Background(), nil, nil, nil, localregistryconfig, nil, nil)
	Expect(err).ToNot(HaveOccurred())

	Expect(repositoryContext.UnmarshalJSON([]byte(`{"type":"local"}`))).To(Succeed())
})

var _ = Describe("cdutils Tests", func() {
	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should return each referenced component only once when resolving component references", func() {
		componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repositoryContext,
			ComponentName:     "example.com/root",
			Version:           "v1.0.0",
		})
		Expect(err).NotTo(HaveOccurred())

		componentVersionList, err := model.GetTransitiveComponentReferences(ctx, componentVersion, &repositoryContext, nil)
		Expect(err).NotTo(HaveOccurred())

		type componentKey struct {
			Name    string
			Version string
		}

		fetchedComponentKeys := make([]componentKey, len(componentVersionList.Components))
		for i, cv := range componentVersionList.Components {
			fetchedComponentKeys[i] = componentKey{Name: cv.GetName(), Version: cv.GetVersion()}
		}

		Expect(fetchedComponentKeys).To(ConsistOf(
			componentKey{
				Name:    "example.com/root",
				Version: "v1.0.0",
			},
			componentKey{
				Name:    "example.com/a",
				Version: "v1.0.0",
			},
			componentKey{
				Name:    "example.com/b",
				Version: "v1.0.0",
			},
			componentKey{
				Name:    "example.com/ab",
				Version: "v1.0.0",
			},
		))
	})

})

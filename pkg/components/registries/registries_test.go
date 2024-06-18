// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

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
	octx              ocm.Context
	err               error
)

var _ = Describe("cdutils Tests", func() {
	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	It("should return each referenced component only once when resolving component references", func() {
		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "./testdata/registry"}
		registryAccess, err = GetFactory().NewRegistryAccess(ctx, nil, nil, nil,
			localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(repositoryContext.UnmarshalJSON([]byte(`{"type":"local"}`))).To(Succeed())

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

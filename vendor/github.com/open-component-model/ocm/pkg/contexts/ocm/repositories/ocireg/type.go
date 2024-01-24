// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocireg

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
	"github.com/open-component-model/ocm/pkg/utils"
)

// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
// to OCI Image References.
type ComponentNameMapping = genericocireg.ComponentNameMapping

const (
	Type   = ocireg.Type
	TypeV1 = ocireg.TypeV1

	OCIRegistryURLPathMapping ComponentNameMapping = "urlPath"
	OCIRegistryDigestMapping  ComponentNameMapping = "sha256-digest"
)

// ComponentRepositoryMeta describes config special for a mapping of
// a component repository to an oci registry.
type ComponentRepositoryMeta = genericocireg.ComponentRepositoryMeta

// RepositorySpec describes a component repository backed by a oci registry.
type RepositorySpec = genericocireg.RepositorySpec

// NewRepositorySpec creates a new RepositorySpec.
// If no ocm meta is given, the subPath part is extracted from the base URL.
// Otherwise, the given URL is used as OCI registry URL as it is.
func NewRepositorySpec(baseURL string, metas ...*ComponentRepositoryMeta) *RepositorySpec {
	meta := utils.Optional(metas...)
	if meta == nil {
		scheme := ""
		if idx := strings.Index(baseURL, "://"); idx > 0 {
			scheme = baseURL[:idx+3]
			baseURL = baseURL[idx+3:]
		}
		if idx := strings.Index(baseURL, "/"); idx > 0 {
			meta = NewComponentRepositoryMeta(baseURL[idx+1:])
			baseURL = scheme + baseURL[:idx]
		}
	}

	return genericocireg.NewRepositorySpec(ocireg.NewRepositorySpec(baseURL), meta)
}

func NewComponentRepositoryMeta(subPath string, mapping ...ComponentNameMapping) *ComponentRepositoryMeta {
	return genericocireg.NewComponentRepositoryMeta(subPath, utils.OptionalDefaulted(OCIRegistryURLPathMapping, mapping...))
}

func NewRepository(ctx cpi.ContextProvider, baseURL string, metas ...*ComponentRepositoryMeta) (cpi.Repository, error) {
	spec := NewRepositorySpec(baseURL, metas...)
	return ctx.OCMContext().RepositoryForSpec(spec)
}

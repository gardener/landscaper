// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocireg

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
)

// ComponentNameMapping describes the method that is used to map the "Component Name", "Component Version"-tuples
// to OCI Image References.
type ComponentNameMapping = genericocireg.ComponentNameMapping

const (
	Type   = genericocireg.Type
	TypeV1 = genericocireg.TypeV1

	OCIRegistryURLPathMapping ComponentNameMapping = "urlPath"
	OCIRegistryDigestMapping  ComponentNameMapping = "sha256-digest"
)

// ComponentRepositoryMeta describes config special for a mapping of
// a component repository to an oci registry.
type ComponentRepositoryMeta = genericocireg.ComponentRepositoryMeta

// RepositorySpec describes a component repository backed by a oci registry.
type RepositorySpec = genericocireg.RepositorySpec

// NewRepositorySpec creates a new RepositorySpec.
func NewRepositorySpec(baseURL string, meta *ComponentRepositoryMeta) *RepositorySpec {
	return genericocireg.NewRepositorySpec(ocireg.NewRepositorySpec(baseURL), meta)
}

func NewComponentRepositoryMeta(subPath string, mapping ComponentNameMapping) *ComponentRepositoryMeta {
	return genericocireg.NewComponentRepositoryMeta(subPath, mapping)
}

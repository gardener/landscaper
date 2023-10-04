// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// DefaultComponent applies defaults to a component.
func DefaultComponent(component *ComponentDescriptor) *ComponentDescriptor {
	if component.RepositoryContexts == nil {
		component.RepositoryContexts = make([]*runtime.UnstructuredTypedObject, 0)
	}
	if component.Sources == nil {
		component.Sources = make([]Source, 0)
	}
	if component.References == nil {
		component.References = make([]ComponentReference, 0)
	}
	if component.Resources == nil {
		component.Resources = make([]Resource, 0)
	}

	if component.Metadata.ConfiguredVersion == "" {
		component.Metadata.ConfiguredVersion = DefaultSchemeVersion
	}
	DefaultResources(component)
	return component
}

// DefaultResources defaults a list of resources.
// The version of the component is defaulted for local resources that do not contain a version.
// adds the version as identity if the resource identity would clash otherwise.
func DefaultResources(component *ComponentDescriptor) {
	for i, res := range component.Resources {
		if res.Relation == v1.LocalRelation && len(res.Version) == 0 {
			component.Resources[i].Version = component.GetVersion()
		}

		id := res.GetIdentity(component.Resources)
		if v, ok := id[SystemIdentityVersion]; ok {
			if res.ExtraIdentity == nil {
				res.ExtraIdentity = v1.Identity{
					SystemIdentityVersion: v,
				}
			} else {
				if _, ok := res.ExtraIdentity[SystemIdentityVersion]; !ok {
					res.ExtraIdentity[SystemIdentityVersion] = v
				}
			}
		}
	}
}

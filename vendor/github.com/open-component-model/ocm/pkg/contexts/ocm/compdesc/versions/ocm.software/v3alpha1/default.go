// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v3alpha1

import (
	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Default applies defaults to a component.
func (cd *ComponentDescriptor) Default() error {
	if cd.RepositoryContexts == nil {
		cd.RepositoryContexts = make([]*runtime.UnstructuredTypedObject, 0)
	}
	if cd.Spec.Sources == nil {
		cd.Spec.Sources = make([]Source, 0)
	}
	if cd.Spec.References == nil {
		cd.Spec.References = make([]Reference, 0)
	}
	if cd.Spec.Resources == nil {
		cd.Spec.Resources = make([]Resource, 0)
	}

	DefaultResources(cd)
	return nil
}

// DefaultResources defaults a list of resources.
// The version of the component is defaulted for local resources that do not contain a version.
// adds the version as identity if the resource identity would clash otherwise.
func DefaultResources(component *ComponentDescriptor) {
	for i, res := range component.Spec.Resources {
		if res.Relation == v1.LocalRelation && len(res.Version) == 0 {
			component.Spec.Resources[i].Version = component.GetVersion()
		}

		id := res.GetIdentity(component.Spec.Resources)
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

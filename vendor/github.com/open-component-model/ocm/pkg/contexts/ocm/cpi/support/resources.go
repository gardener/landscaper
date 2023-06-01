// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

// GetResourcesByIdentitySelectors returns resources that match the given identity selectors.
func (s *ComponentVersionAccess) GetResourcesByIdentitySelectors(selectors ...compdesc.IdentitySelector) ([]cpi.ResourceAccess, error) {
	return s.GetResourcesBySelectors(selectors, nil)
}

// GetResourcesByResourceSelectors returns resources that match the given resource selectors.
func (s *ComponentVersionAccess) GetResourcesByResourceSelectors(selectors ...compdesc.ResourceSelector) ([]cpi.ResourceAccess, error) {
	return s.GetResourcesBySelectors(nil, selectors)
}

// GetResourcesBySelectors returns resources that match the given selector.
func (s *ComponentVersionAccess) GetResourcesBySelectors(selectors []compdesc.IdentitySelector, resourceSelectors []compdesc.ResourceSelector) ([]cpi.ResourceAccess, error) {
	resources := make([]cpi.ResourceAccess, 0)
	rscs := s.GetDescriptor().Resources
	for i := range rscs {
		selctx := compdesc.NewResourceSelectionContext(i, rscs)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := compdesc.MatchResourceByResourceSelector(selctx, resourceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		r, err := s.GetResourceByIndex(i)
		if err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	if len(resources) == 0 {
		return resources, compdesc.NotFound
	}
	return resources, nil
}

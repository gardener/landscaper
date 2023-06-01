// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/utils/selector"
)

// GetReferencesByIdentitySelectors returns references that match the given identity selectors.
func (s *ComponentVersionAccess) GetReferencesByIdentitySelectors(selectors ...compdesc.IdentitySelector) (compdesc.References, error) {
	return s.GetReferencesBySelectors(selectors, nil)
}

// GetReferencesByReferenceSelectors returns references that match the given resource selectors.
func (s *ComponentVersionAccess) GetReferencesByReferenceSelectors(selectors ...compdesc.ReferenceSelector) (compdesc.References, error) {
	return s.GetReferencesBySelectors(nil, selectors)
}

// GetReferencesBySelectors returns references that match the given selector.
func (s *ComponentVersionAccess) GetReferencesBySelectors(selectors []compdesc.IdentitySelector, referenceSelectors []compdesc.ReferenceSelector) (compdesc.References, error) {
	references := make(compdesc.References, 0)
	refs := s.GetDescriptor().References
	for i := range refs {
		selctx := compdesc.NewReferenceSelectionContext(i, refs)
		if len(selectors) > 0 {
			ok, err := selector.MatchSelectors(selctx.Identity(), selectors...)
			if err != nil {
				return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
			}
			if !ok {
				continue
			}
		}
		ok, err := compdesc.MatchReferencesByReferenceSelector(selctx, referenceSelectors...)
		if err != nil {
			return nil, fmt.Errorf("unable to match selector for resource %s: %w", selctx.Name, err)
		}
		if !ok {
			continue
		}
		references = append(references, *selctx.ComponentReference)
	}
	if len(references) == 0 {
		return references, compdesc.NotFound
	}
	return references, nil
}

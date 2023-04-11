// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/signing"
)

// CDExcludes describes the fields relevant for Signing
// ATTENTION: if changed, please adapt the HashEqual Functions
// in the generic part, accordingly.
var CDExcludes = signing.MapExcludes{
	"component": signing.MapExcludes{
		"labels": signing.ExcludeEmpty{signing.DynamicArrayExcludes{
			ValueChecker: signing.IgnoreLabelsWithoutSignature,
			Continue:     signing.NoExcludes{},
		}},
		"repositoryContexts": nil,
		"resources": signing.DynamicArrayExcludes{
			ValueChecker: signing.IgnoreResourcesWithNoneAccess,
			Continue: signing.MapExcludes{
				"access": nil,
				"srcRef": nil,
				"labels": signing.ExcludeEmpty{signing.DynamicArrayExcludes{
					ValueChecker: signing.IgnoreLabelsWithoutSignature,
					Continue:     signing.NoExcludes{},
				}},
			},
		},
		"sources": nil,
		"componentReferences": signing.ArrayExcludes{
			signing.MapExcludes{
				"labels": signing.ExcludeEmpty{signing.DynamicArrayExcludes{
					ValueChecker: signing.IgnoreLabelsWithoutSignature,
					Continue:     signing.NoExcludes{},
				}},
			},
		},
	},
	"signatures": nil,
}

func (cd *ComponentDescriptor) Normalize(normAlgo string) ([]byte, error) {
	if normAlgo != compdesc.JsonNormalisationV1 {
		return nil, fmt.Errorf("unsupported cd normalization %q", normAlgo)
	}
	data, err := signing.Normalize(cd, CDExcludes)
	logrus.Debugf("**** normalized:\n %s\n", string(data))
	return data, err
}

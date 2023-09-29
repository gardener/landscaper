// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/normalizations/rules"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/norm/entry"
)

func providerMapper(v interface{}) interface{} {
	var provider map[string]interface{}
	err := json.Unmarshal([]byte(v.(string)), &provider)
	if err == nil {
		return provider
	}
	return v
}

// CDExcludes describes the fields relevant for Signing
// ATTENTION: if changed, please adapt the HashEqual Functions
// in the generic part, accordingly.
var CDExcludes = signing.MapExcludes{
	"component": signing.MapExcludes{
		"provider": signing.MapValue{
			Mapping: providerMapper,
			Continue: signing.MapExcludes{
				"labels": rules.LabelExcludes,
			},
		},
		"labels":             rules.LabelExcludes,
		"repositoryContexts": nil,
		"resources": signing.DefaultedMapFields{
			Next: signing.DynamicArrayExcludes{
				ValueMapper: rules.MapResourcesWithNoneAccess,
				Continue: signing.MapExcludes{
					"access": nil,
					"srcRef": nil,
					"labels": rules.LabelExcludes,
				},
			},
		}.EnforceNull("extraIdentity"),
		"sources": nil,
		"componentReferences": signing.DefaultedMapFields{
			Next: signing.ArrayExcludes{
				signing.MapExcludes{
					"labels": rules.LabelExcludes,
				},
			},
		}.EnforceNull("extraIdentity"),
	},
	"signatures":    nil,
	"nestedDigests": nil,
}

func (cd *ComponentDescriptor) Normalize(normAlgo string) ([]byte, error) {
	if normAlgo != compdesc.JsonNormalisationV1 {
		return nil, fmt.Errorf("unsupported cd normalization %q", normAlgo)
	}
	data, err := signing.Normalize(entry.Type, cd, CDExcludes)
	return data, err
}

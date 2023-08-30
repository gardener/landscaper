// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package jsonv2 provides a normalization which is completely based on the
// abstract (internal) version of the component descriptor and is therefore
// agnostic of the final serialization format. Signatures using this algorithm
// can be transferred among different schema versions, as long as is able to
// handle the complete information using for the normalization.
// Older format might omit some info, therefore the signatures cannot be
// validated for such representations, if the original component descriptor
// has used such parts.
package jsonv2

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/norm/jcs"
)

const Algorithm = compdesc.JsonNormalisationV2

func init() {
	compdesc.Normalizations.Register(Algorithm, normalization{})
}

type normalization struct{}

func (m normalization) Normalize(cd *compdesc.ComponentDescriptor) ([]byte, error) {
	data, err := signing.Normalize(jcs.Type, cd, CDExcludes)
	return data, err
}

// CDExcludes describes the fields relevant for Signing
// ATTENTION: if changed, please adapt the HashEqual Functions
// in the generic part, accordingly.
var CDExcludes = signing.MapExcludes{
	"meta": nil,
	"component": signing.MapExcludes{
		"repositoryContexts": nil,
		"provider": signing.MapExcludes{
			"labels": signing.LabelExcludes,
		},
		"labels": signing.LabelExcludes,
		"resources": signing.DynamicArrayExcludes{
			ValueChecker: signing.IgnoreResourcesWithNoneAccess,
			Continue: signing.MapExcludes{
				"access": nil,
				"srcRef": nil,
				"labels": signing.LabelExcludes,
			},
		},
		"sources": signing.DynamicArrayExcludes{
			ValueChecker: signing.IgnoreResourcesWithNoneAccess,
			Continue: signing.MapExcludes{
				"access": nil,
				"labels": signing.LabelExcludes,
			},
		},
		"references": signing.ArrayExcludes{
			signing.MapExcludes{
				"labels": signing.LabelExcludes,
			},
		},
	},
	"signatures":    nil,
	"nestedDigests": nil,
}

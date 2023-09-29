// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v3alpha1

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/norm/entry"
)

// CDExcludes describes the fields relevant for Signing
// ATTENTION: if changed, please adapt the HashEqual Functions
// in the generic part, accordingly.
var CDExcludes = signing.MapExcludes{
	"repositoryContexts": nil,
	"metadata": signing.MapExcludes{
		"labels": signing.LabelExcludes,
	},
	"spec": signing.MapExcludes{
		"provider": signing.MapExcludes{
			"labels": signing.LabelExcludes,
		},
		"resources": signing.ArrayExcludes{
			Continue: signing.MapExcludes{
				"access": nil,
				"srcRef": nil,
				"labels": signing.LabelExcludes,
			},
		},
		"sources": signing.ArrayExcludes{
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

func (cd *ComponentDescriptor) Normalize(normAlgo string) ([]byte, error) {
	if normAlgo != compdesc.JsonNormalisationV1 {
		return nil, fmt.Errorf("unsupported cd normalization %q", normAlgo)
	}
	data, err := signing.Normalize(entry.Type, cd, CDExcludes)
	logrus.Debugf("**** normalized:\n %s\n", string(data))
	return data, err
}

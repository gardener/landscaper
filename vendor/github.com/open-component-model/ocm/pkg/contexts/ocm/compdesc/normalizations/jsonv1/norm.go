// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package jsonv1 provides a normalization which uses schema specific
// normalizations.
// It creates the requested schema for the component descriptor
// and just forwards the normalization to this version.
package jsonv1

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

const Algorithm = compdesc.JsonNormalisationV1

func init() {
	compdesc.Normalizations.Register(Algorithm, normalization{})
}

type normalization struct{}

func (m normalization) Normalize(cd *compdesc.ComponentDescriptor) ([]byte, error) {
	cv := compdesc.DefaultSchemes[cd.SchemaVersion()]
	if cv == nil {
		if cv == nil {
			return nil, errors.ErrNotSupported(errors.KIND_SCHEMAVERSION, cd.SchemaVersion())
		}
	}
	v, err := cv.ConvertFrom(cd)
	if err != nil {
		return nil, err
	}
	return v.Normalize(Algorithm)
}

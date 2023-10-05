// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"github.com/open-component-model/ocm/pkg/errors"
)

type Validater interface {
	Validate() error
}

func Validate(o interface{}) error {
	if t, ok := o.(TypedObject); ok {
		if t.GetType() == "" {
			return errors.New("type missing")
		}
	}
	if v, ok := o.(Validater); ok {
		return v.Validate()
	}
	return nil
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/modern-go/reflect2"
)

// Validatable is an optional interface for DataAccess
// implementations or any other object, which might reach
// an error state. The error can then be queried with
// the method Validatable.Validate.
// This is used to support objects with access methods not
// returning an error. If the object is not valid,
// those methods return an unknown/default state, but
// the object should be queryable for its state.
type Validatable interface {
	Validate() error
}

// ValidateObject checks whether an object
// is in error state. If yes, an appropriate
// error is returned.
func ValidateObject(o interface{}) error {
	if reflect2.IsNil(o) {
		return nil
	}
	if p, ok := o.(Validatable); ok {
		return p.Validate()
	}
	return nil
}

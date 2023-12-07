// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package optionutils

import (
	"github.com/open-component-model/ocm/pkg/generics"
)

/////////////////////////////////////////////////////////////////////////////(//
// if the option target is an interface, it is easily possible to
// provide new targets with more options just by extending the
// target interface. The option consumer the accepts options for
// the target interface.
// To be able to reuse options from the base target interface
// a wrapper option implementation is required which implements
// the extended option interface and maps it to the base option
// interface.
// The following mechanism requires option targets W and B to be
// interface types.

type targetInterfaceWrapper[B any, W any /*B*/] struct {
	option Option[B]
}

func (w *targetInterfaceWrapper[B, W]) ApplyTo(opts W) {
	w.option.ApplyTo(generics.As[B](opts))
}

// MapOptionTarget maps the option target interface from
// B to W, hereby, W must be a subtype of B, which cannot be
// expressed with Go generics (Type constraint should be W B).
// If this constraint is not met, there will be a runtime error.
func MapOptionTarget[W, B any](opt Option[B]) Option[W] {
	return &targetInterfaceWrapper[B, W]{
		opt,
	}
}

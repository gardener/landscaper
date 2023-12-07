// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package optionutils

// NestedOptionsProvider is the interface for a
// more specific options object to provide
// access to a nested options object of type T.
// T must be a pointer type.
type NestedOptionsProvider[T any] interface {
	NestedOptions() T
}

// OptionTargetProvider is helper interface
// to declare a pointer type (*O) for an options
// object providing access to a nested
// options object of type N (must be a pointer type).
type OptionTargetProvider[N, O any] interface {
	NestedOptionsProvider[N]
	*O
}

// OptionWrapper genericly wraps a nested option of type Option[N] to
// an option of type Option[*O], assuming that the nested option source
// N implements NestedOptionsProvider[N].
// P is only a helper type parameter for Go and doesn't need to be given.
// It is the pointer type for O (P = *O).
//
// create a wrap option function for all wrappable options with
//
//	 func WrapXXX[O any, P optionutils.OptionTargetProvider[*Options, O]](args...any) optionutils.Option[P] {
//			return optionutils.OptionWrapper[*Options, O, P](WithXXX(args...))
//	 }
//
// where *Options is the type of the pointer type to the options object to be nested.
//
// The outer option functions wrapping the nested one can then be defined as
//
//	func WithXXX(h string) Option {
//		return optionutils.WrapXXX[Options](h)
//	}
//
// For an example see package github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes/rpi.
func OptionWrapper[N, O any, P OptionTargetProvider[N, O]](o Option[N]) Option[P] {
	return optionWrapper[N, O, P]{o}
}

type optionWrapper[N, O any, P OptionTargetProvider[N, O]] struct {
	opt Option[N]
}

func (w optionWrapper[N, O, P]) ApplyTo(opts P) {
	w.opt.ApplyTo(opts.NestedOptions())
}

/////////////////////////////////////////////////////////////////////////////(//

func OptionWrapperFunc[N, O any](o Option[N], nested func(outer O) N) Option[O] {
	return optionWrapperFunc[N, O]{o, nested}
}

type optionWrapperFunc[N, O any] struct {
	opt    Option[N]
	nested func(O) N
}

func (w optionWrapperFunc[N, O]) ApplyTo(opts O) {
	w.opt.ApplyTo(w.nested(opts))
}

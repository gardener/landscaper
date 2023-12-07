// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package optionutils

func PointerTo[T any](v T) *T {
	temp := v
	return &temp
}

func AsValue[T any](p *T) T {
	var r T
	if p != nil {
		r = *p
	}
	return r
}

func ApplyOption[T any](opt *T, tgt **T) {
	if opt != nil {
		*tgt = opt
	}
}

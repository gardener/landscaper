// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

import (
	"golang.org/x/exp/slices"
)

func AppendedSlice[E any](slice []E, elems ...E) []E {
	return append(slices.Clone(slice), elems...)
}

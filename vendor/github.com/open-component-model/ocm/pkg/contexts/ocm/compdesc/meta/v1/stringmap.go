// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

type StringMap map[string]string

// Copy copies map.
func (l StringMap) Copy() StringMap {
	n := StringMap{}
	for k, v := range l {
		n[k] = v
	}
	return n
}

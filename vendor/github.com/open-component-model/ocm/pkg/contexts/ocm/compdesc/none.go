// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

// Type is the access type for no blob.
const (
	NoneType       = "none"
	NoneLegacyType = "None"
)

func IsNoneAccessKind(kind string) bool {
	return kind == NoneType || kind == NoneLegacyType
}

func IsNoneAccess(a AccessSpec) bool {
	if a == nil {
		return false
	}
	return IsNoneAccessKind(a.GetKind())
}

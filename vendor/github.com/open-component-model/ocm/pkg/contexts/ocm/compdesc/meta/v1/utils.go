// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"regexp"
	"unicode"
)

const IdentityKeyValidationErrMsg string = "a identity label or name must consist of lower case alphanumeric characters, '-', '_' or '+', and must start and end with an alphanumeric character"

var identityKeyValidationRegexp = regexp.MustCompile("^[a-z0-9]([-_+a-z0-9]*[a-z0-9])?$")

func IsIdentity(s string) bool {
	return identityKeyValidationRegexp.Match([]byte(s))
}

// IsAscii checks whether a string only contains ascii characters.
func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

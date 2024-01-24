// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

func RepositoryUsage(scheme RepositoryTypeScheme) string {
	s := `
The following list describes the supported credential providers
(credential repositories), their specification versions
and formats. Because of the extensible nature of the OCM model,
credential consum
`
	return s + scheme.Describe()
}

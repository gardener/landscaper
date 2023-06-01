// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
)

type StringList []string

func (s *StringList) Add(n string) {
	for _, e := range *s {
		if n == e {
			return
		}
	}
	*s = append(*s, n)
}

func FilterByNamespacePrefix(prefix string, list []string) []string {
	result := []string{}
	sub := prefix
	if prefix != "" && !strings.HasSuffix(prefix, grammar.RepositorySeparator) {
		sub = prefix + grammar.RepositorySeparator
	}
	for _, k := range list {
		if k == prefix || strings.HasPrefix(k, sub) {
			result = append(result, k)
		}
	}
	return result
}

func FilterChildren(closure bool, list []string) []string {
	if closure {
		return list
	}
	set := map[string]bool{}
	for _, n := range list {
		i := strings.Index(n, grammar.RepositorySeparator)
		if i < 0 {
			set[n] = true
		} else {
			set[n[:i]] = true
		}
	}
	result := make([]string, 0, len(set))
	for n := range set {
		result = append(result, n)
	}
	return result
}

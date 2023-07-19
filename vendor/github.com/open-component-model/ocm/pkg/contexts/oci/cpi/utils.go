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

func FilterChildren(closure bool, prefix string, list []string) []string {
	if closure {
		return FilterByNamespacePrefix(prefix, list)
	}
	sub := prefix
	if prefix != "" && !strings.HasSuffix(prefix, grammar.RepositorySeparator) {
		sub = prefix + grammar.RepositorySeparator
	}
	set := map[string]bool{}
	for _, n := range list {
		if n == prefix {
			set[n] = true
		} else if strings.HasPrefix(n, sub) {
			rest := n[len(sub):]
			i := strings.Index(rest, grammar.RepositorySeparator)
			if i < 0 {
				set[n] = true
			} else {
				set[n[:i+len(sub)]] = true
			}
		}
	}
	result := make([]string, 0, len(set))
	for _, n := range list {
		if set[n] {
			result = append(result, n)
		}
	}
	return result
}

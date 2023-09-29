// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"sort"
	"strings"
)

type DescriptionProvider interface {
	GetDescription() string
}

type KeyInfo interface {
	DescriptionProvider
	GetKey() string
}

func FormatKey(k string) string {
	return strings.ReplaceAll(k, "<", "&lt;")
}

func FormatList(def string, elems ...KeyInfo) string {
	names := ""
	for _, n := range elems {
		add := ""
		if n.GetKey() == def {
			add = " (default)"
		}
		names = fmt.Sprintf("%s\n  - <code>%s</code>:%s %s", names, FormatKey(n.GetKey()), add, n.GetDescription())
	}
	return names
}

func FormatMap[T DescriptionProvider](def string, elems map[string]T) string {
	keys := StringMapKeys(elems)
	sort.Strings(keys)
	names := ""
	for _, k := range keys {
		e := elems[k]
		add := ""
		if k == def {
			add = " (default)"
		}
		names = fmt.Sprintf("%s\n  - <code>%s</code>:%s %s", names, FormatKey(k), add, e.GetDescription())
	}
	return names
}

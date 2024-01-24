// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package simplelistmerge

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/hpi"
)

const ALGORITHM = "simpleListMerge"

func init() {
	hpi.Register(New())
}

type (
	// Value is the minimal structure of values usable with the merge algorithm.
	Value = []Entry
	Entry = interface{}
)

func New() hpi.Handler {
	return hpi.New(ALGORITHM, desc, merge)
}

var desc = `
This handler merges simple list labels values.

It supports the following config structure:
- *<code>overwrite</code>* *string* (optional) determines how to handle conflicts.

`

func merge(ctx hpi.Context, c *Config, lv Value, tv *Value) (bool, error) {
	modified := false
outer:
	for _, le := range lv {
		for _, te := range *tv {
			if equal(c, le, te) {
				continue outer
			}
		}
		*tv = append(*tv, le)
		modified = true
	}
	return modified, nil
}

func equal(c *Config, le, te Entry) bool {
	if c == nil || len(c.IgnoredFields) == 0 {
		return reflect.DeepEqual(le, te)
	}

	if lm, ok := le.(map[string]interface{}); ok {
		if tm, ok := te.(map[string]interface{}); ok {
			for _, n := range c.IgnoredFields {
				delete(lm, n)
				delete(tm, n)
			}
			return reflect.DeepEqual(lm, tm)
		}
	}
	return reflect.DeepEqual(le, te)
}

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

// Value is the minimal structure of values usable with the merge algorithm.
type Value = []Entry
type Entry = interface{}

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
			if reflect.DeepEqual(le, te) {
				continue outer
			}
		}
		*tv = append(*tv, le)
		modified = true
	}
	return modified, nil
}

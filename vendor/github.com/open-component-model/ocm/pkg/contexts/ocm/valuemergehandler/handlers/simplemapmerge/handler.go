// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package simplemapmerge

import (
	"fmt"
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/hpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

const ALGORITHM = "simpleMapMerge"

func init() {
	hpi.Register(New())
}

type (
	// Value is the minimal structure of values usable with the merge algorithm.
	Value = map[string]Entry
	Entry = interface{}
)

func New() hpi.Handler {
	return hpi.New(ALGORITHM, desc, merge)
}

var desc = `
This handler merges simple map labels values.

It supports the following config structure:
- *<code>overwrite</code>* *string* (optional) determines how to handle conflicts.

  - <code>none</code> (default) no change possible, if entry differs the merge is rejected.
  - <code>local</code> the local value is preserved.
  - <code>inbound</code> the inbound value overwrites the local one.

- *<code>entries</code> *merge spec* (optional)

  The merge specification (<code>algorithm</code> and <code>config</code>) used to merge conflicting
  changes in map entries.
`

func merge(ctx hpi.Context, c *Config, lv Value, tv *Value) (bool, error) {
	var err error

	subm := false
	modified := false
	for lk, le := range lv {
		if te, ok := (*tv)[lk]; ok {
			if !reflect.DeepEqual(le, te) {
				switch c.Overwrite {
				case MODE_DEFAULT:
					if c.Entries != nil {
						hpi.Log.Trace("different entry found in target -> merge it", "name", lk, "entries", c.Entries)
						subm, te, err = hpi.GenericMerge(ctx, c.Entries, "", le, te)
						if err != nil {
							return false, errors.Wrapf(err, "map key %q", lk)
						}
						if subm {
							(*tv)[lk] = te
							modified = true
							hpi.Log.Trace("entry merge result", "result", (*tv))
						} else {
							hpi.Log.Trace("not modified")
						}
						break
					}
					fallthrough
				case MODE_NONE:
					hpi.Log.Trace("different entry found in target -> fail", "name", lk)
					return false, fmt.Errorf("target value for %q changed", lk)
				case MODE_LOCAL:
					(*tv)[lk] = le
					hpi.Log.Trace("different entry found in target -> use local", "name", lk, "result", (*tv))
					modified = true
				}
			} else {
				hpi.Log.Trace("entry found in target", "name", lk)
			}
		} else {
			(*tv)[lk] = le
			hpi.Log.Trace("entry not found in target -> append it", "name", lk, "result", (*tv))
			modified = true
		}
	}
	hpi.Log.Trace("merge result", "modified", modified, "result", (*tv))
	return modified, nil
}

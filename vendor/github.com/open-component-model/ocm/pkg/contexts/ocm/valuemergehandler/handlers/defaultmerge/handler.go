// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package defaultmerge

import (
	"fmt"
	"reflect"

	// special case to resolve dependency cycles.
	hpi "github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/internal"
)

const ALGORITHM = "default"

func init() {
	hpi.Register(New())
}

// LabelValue is the minimal structure of values usable with the merge algorithm.
type LabelValue interface{}

func New() hpi.Handler {
	return hpi.New(ALGORITHM, desc, merge)
}

var desc = `
This handler merges arbitrary label values by deciding for
one or none side.

It supports the following config structure:
- *<code>overwrite</code>* *string* (optional) determines how to handle conflicts.

  - <code>none</code> no change possible, if entry differs the merge is rejected.
  - <code>local</code> the local value is preserved.
  - <code>inbound</code> (default) the inbound value overwrites the local one.
`

func merge(ctx hpi.Context, c *Config, lv LabelValue, tv *LabelValue) (bool, error) {
	modified := false
	if !reflect.DeepEqual(lv, tv) {
		switch c.Overwrite {
		// default = INBOUND: keep precalculated tarted = inbound CD
		case MODE_LOCAL:
			*tv = lv
			modified = true
		case MODE_NONE:
			return false, fmt.Errorf("target value changed")
		}
	}
	return modified, nil
}

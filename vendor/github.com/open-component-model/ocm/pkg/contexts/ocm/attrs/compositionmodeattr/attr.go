// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compositionmodeattr

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// UseCompositionMode enables the support of the new Composition mode for
// Component versions. It disabled the direct write-through and update-on-close
// to the underlying repository. Instead, an explicit call to AddVersion call
// s required to persist a composed change on a new as well as queried
// component version object.
const UseCompositionMode = false

const (
	ATTR_KEY   = "ocm.software/compositionmode"
	ATTR_SHORT = "compositionmode"
)

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{}, ATTR_SHORT)
}

type AttributeType struct{}

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
*bool* (default: ` + fmt.Sprintf("%t", UseCompositionMode) + `
Composition mode decouples a component version provided by a repository
implemention from the backened persistence. Added local blobs will
and other changes witll not be forwarded to the backend repository until
an AddVersion is called on the component.
If composition mode is disabled blobs will directly be forwarded to
the backend and descriptor updated will be persisted on AddVersion
or closing a provided existing component version.
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(bool); !ok {
		return nil, fmt.Errorf("boolean required")
	}
	return marshaller.Marshal(v)
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value bool
	err := unmarshaller.Unmarshal(data, &value)
	return value, err
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx datacontext.Context) bool {
	a := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return UseCompositionMode
	}
	return a.(bool)
}

func Set(ctx datacontext.Context, flag bool) error {
	return ctx.GetAttributes().SetAttribute(ATTR_KEY, flag)
}

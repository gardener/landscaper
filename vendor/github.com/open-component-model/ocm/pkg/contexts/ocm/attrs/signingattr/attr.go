// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signingattr

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	ocm "github.com/open-component-model/ocm/pkg/contexts/ocm/context"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/signing"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/ocm/signing"
	ATTR_SHORT = "signing"
)

type (
	Context         = ocm.Context
	ContextProvider = ocm.ContextProvider
)

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{})
}

type AttributeType struct{}

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
*JSON*
Public and private Key settings given as JSON document with the following
format:

<pre>
{
  "publicKeys"": [
     "&lt;provider>": {
       "data": ""&lt;base64>"
     }
  ],
  "privateKeys"": [
     "&lt;provider>": {
       "path": ""&lt;file path>"
     }
  ]
</pre>

One of following data fields are possible:
- <code>data</code>:       base64 encoded binary data
- <code>stringdata</code>: plain text data
- <code>path</code>:       a file path to read the data from
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(signing.Registry); ok {
		return nil, nil
	}
	return nil, errors.ErrNotSupported("encoding of key registry")
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value Config
	err := unmarshaller.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	value.SetType(ConfigType)
	registry := signing.NewRegistry(signing.DefaultHandlerRegistry(), signing.DefaultKeyRegistry())
	value.ApplyToRegistry(registry)
	return registry, err
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx ContextProvider) signing.Registry {
	a := ctx.OCMContext().GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return signing.DefaultRegistry()
	}
	return a.(signing.Registry)
}

func SetKeyRegistry(ctx ContextProvider, registry signing.KeyRegistry) error {
	old := Get(ctx)
	r := signing.NewRegistry(old.HandlerRegistry(), registry)
	return ctx.OCMContext().GetAttributes().SetAttribute(ATTR_KEY, r)
}

func SetHandlerRegistry(ctx ContextProvider, registry signing.HandlerRegistry) error {
	old := Get(ctx)
	r := signing.NewRegistry(registry, old.KeyRegistry())
	return ctx.OCMContext().GetAttributes().SetAttribute(ATTR_KEY, r)
}

func Set(ctx ContextProvider, registry signing.Registry) error {
	return ctx.OCMContext().GetAttributes().SetAttribute(ATTR_KEY, registry)
}

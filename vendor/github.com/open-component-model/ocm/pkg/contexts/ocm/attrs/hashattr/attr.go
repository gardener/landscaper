// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hashattr

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/ocm/hasher"
	ATTR_SHORT = "hasher"
)

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{})
}

type AttributeType struct{}

var (
	_ datacontext.AttributeType = (*AttributeType)(nil)
	_ datacontext.Converter     = (*AttributeType)(nil)
)

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
*JSON*
Preferred hash algorithm to calculate resource digests. The following
digesters are supported:
` + listformat.FormatList(sha256.Algorithm, signing.DefaultRegistry().HasherNames()...)
}

func (a AttributeType) Convert(v interface{}) (interface{}, error) {
	switch s := v.(type) {
	case string:
		return &Attribute{
			signing.DefaultRegistry(),
			s,
		}, nil
	case *Attribute:
		return v, nil
	}
	return nil, fmt.Errorf("digest algorithm name or hash attribute required")
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	switch s := v.(type) {
	case string:
		return []byte(s), nil
	case *Attribute:
		return []byte(s.DefaultHasher), nil
	}
	return nil, fmt.Errorf("digest algorithm name required")
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value string
	err := unmarshaller.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return &Attribute{
		signing.DefaultRegistry(),
		value,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type Attribute struct {
	Provider      internal.HasherProvider
	DefaultHasher string
}

func (a *Attribute) GetProvider(ctx datacontext.Context) internal.HasherProvider {
	if a.Provider != nil {
		return a.Provider
	}
	return signingattr.Get(ctx)
}

func (a *Attribute) GetHasher(names ...string) internal.Hasher {
	name := utils.Optional(names...)
	if name != "" {
		return a.Provider.GetHasher(name)
	}
	return a.Provider.GetHasher(a.DefaultHasher)
}

func Get(ctx datacontext.Context) *Attribute {
	a := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return &Attribute{
			signingattr.Get(ctx),
			sha256.Algorithm,
		}
	}
	return a.(*Attribute)
}

func Set(ctx datacontext.Context, registry signing.KeyRegistry) error {
	if _, ok := registry.(signing.Registry); !ok {
		registry = signing.NewRegistry(signing.DefaultHandlerRegistry(), registry)
	}
	return ctx.GetAttributes().SetAttribute(ATTR_KEY, registry)
}

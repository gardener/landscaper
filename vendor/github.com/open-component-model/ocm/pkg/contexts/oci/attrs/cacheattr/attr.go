// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cacheattr

import (
	"fmt"
	"os"
	"strings"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/oci/cache"
	ATTR_SHORT = "cache"
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
*string*
Filesystem folder to use for caching OCI blobs
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(accessio.BlobCache); !ok {
		return nil, fmt.Errorf("accessio.BlobCache required")
	}
	return nil, nil
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value string
	err := unmarshaller.Unmarshal(data, &value)
	if value != "" {
		if strings.HasPrefix(value, "~"+string(os.PathSeparator)) {
			home := os.Getenv("HOME")
			if home == "" {
				panic("HOME not set")
			}
			value = home + value[1:]
		}
		// TODO: This should use the virtual filesystem.
		err = os.MkdirAll(value, 0o700)
		if err == nil {
			return accessio.NewStaticBlobCache(value)
		}
	} else if err == nil {
		err = errors.Newf("file path missing")
	}
	return value, err
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx datacontext.Context) accessio.BlobCache {
	a := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return nil
	}
	return a.(accessio.BlobCache)
}

func Set(ctx datacontext.Context, cache accessio.BlobCache) error {
	return ctx.GetAttributes().SetAttribute(ATTR_KEY, cache)
}

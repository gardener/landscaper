// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logforward

import (
	"encoding/json"
	"fmt"

	logcfg "github.com/mandelsoft/logging/config"
	"sigs.k8s.io/yaml"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/logforward"
	ATTR_SHORT = "logfwd"
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
*logconfig* Logging config structure used for config forwarding
This attribute is used to specify a logging configuration intended
to be forwarded to other tools.
(For example: TOI passes this config to the executor)
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(*logcfg.Config); !ok {
		return nil, fmt.Errorf("logging config required")
	}
	return json.Marshal(v)
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var c logcfg.Config
	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx datacontext.Context) *logcfg.Config {
	v := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if v == nil {
		return nil
	}
	return v.(*logcfg.Config)
}

func Set(ctx datacontext.Context, c *logcfg.Config) {
	ctx.GetAttributes().SetAttribute(ATTR_KEY, c)
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registrations

import (
	"encoding/json"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Decoder func(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error)

func DecodeDefaultedConfig[T any](config interface{}, d ...Decoder) (*T, error) {
	if config == nil {
		var cfg T
		return &cfg, nil
	}
	return DecodeConfig[T](config, d...)
}

func DecodeConfig[T any](config interface{}, d ...Decoder) (*T, error) {
	var err error

	if config == nil {
		return nil, nil
	}

	var cfg *T
	switch a := config.(type) {
	case string:
		cfg, err = decodeConfig[T]([]byte(a), d...)
	case json.RawMessage:
		cfg, err = decodeConfig[T](a, d...)
	case []byte:
		cfg, err = decodeConfig[T](a, d...)
	case *T:
		cfg = a
	case T:
		cfg = &a
	default:
		var data []byte
		data, err = json.Marshal(a)
		if err != nil {
			return nil, err
		}
		cfg, err = decodeConfig[T](data, d...)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal config")
	}
	return cfg, nil
}

func decodeConfig[T any](data []byte, dec ...Decoder) (*T, error) {
	if d := utils.Optional(dec...); d != nil {
		r, err := d(data, runtime.DefaultYAMLEncoding)
		if err != nil {
			return nil, err
		}
		if eff, ok := r.(*T); ok {
			return eff, nil
		}
		return nil, errors.Newf("invalid decoded type %T ", r)
	}

	var c T
	err := runtime.DefaultYAMLEncoding.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func DecodeAnyConfig(config interface{}) (json.RawMessage, error) {
	var attr json.RawMessage
	switch a := config.(type) {
	case json.RawMessage:
		attr = a
	case []byte:
		err := runtime.DefaultYAMLEncoding.Unmarshal(a, &attr)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid target specification")
		}
		attr = a
	default:
		data, err := json.Marshal(config)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid target specification")
		}
		attr = data
	}
	return attr, nil
}

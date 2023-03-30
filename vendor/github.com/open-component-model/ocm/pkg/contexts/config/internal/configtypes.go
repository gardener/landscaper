// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type ConfigType interface {
	runtime.TypedObjectDecoder
	runtime.VersionedTypedObject
	Usage() string
}

type ConfigTypeScheme interface {
	runtime.Scheme
	AddKnownTypes(s ConfigTypeScheme)

	GetConfigType(name string) ConfigType
	Register(name string, atype ConfigType)

	DecodeConfig(data []byte, unmarshaler runtime.Unmarshaler) (Config, error)
	CreateConfig(obj runtime.TypedObject) (Config, error)

	Usage() string
}

type configTypeScheme struct {
	runtime.SchemeBase
}

func NewConfigTypeScheme(defaultRepoDecoder runtime.TypedObjectDecoder, base ...ConfigTypeScheme) ConfigTypeScheme {
	var rt Config
	scheme := runtime.MustNewDefaultScheme(&rt, &GenericConfig{}, true, defaultRepoDecoder, utils.Optional(base...))
	return &configTypeScheme{scheme}
}

func (t *configTypeScheme) AddKnownTypes(s ConfigTypeScheme) {
	t.SchemeBase.AddKnownTypes(s)
}

func (t *configTypeScheme) GetConfigType(name string) ConfigType {
	d := t.GetDecoder(name)
	if d == nil {
		return nil
	}
	return d.(ConfigType)
}

func (t *configTypeScheme) RegisterByDecoder(name string, decoder runtime.TypedObjectDecoder) error {
	if _, ok := decoder.(ConfigType); !ok {
		return errors.ErrInvalid("type", reflect.TypeOf(decoder).String())
	}
	return t.SchemeBase.RegisterByDecoder(name, decoder)
}

func (t *configTypeScheme) Register(name string, rtype ConfigType) {
	t.SchemeBase.RegisterByDecoder(name, rtype)
}

func (t *configTypeScheme) DecodeConfig(data []byte, unmarshaler runtime.Unmarshaler) (Config, error) {
	obj, err := t.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	if spec, ok := obj.(Config); ok {
		return spec, nil
	}
	return nil, fmt.Errorf("invalid object type: yield %T instead of Config", obj)
}

func (t *configTypeScheme) CreateConfig(obj runtime.TypedObject) (Config, error) {
	if s, ok := obj.(Config); ok {
		return s, nil
	}
	if u, ok := obj.(*runtime.UnstructuredTypedObject); ok {
		raw, err := u.GetRaw()
		if err != nil {
			return nil, err
		}
		return t.DecodeConfig(raw, runtime.DefaultJSONEncoding)
	}
	return nil, fmt.Errorf("invalid object type %T for repository specs", obj)
}

func (t *configTypeScheme) Usage() string {
	found := map[string]bool{}

	s := "\nThe following configuration types are supported:\n"
	for _, n := range t.KnownTypeNames() {
		ct := t.GetConfigType(n)
		u := ct.Usage()
		if strings.TrimSpace(u) == "" || found[u] {
			continue
		}
		found[u] = true
		for strings.HasSuffix(u, "\n") {
			u = u[:len(u)-1]
		}
		s = fmt.Sprintf("%s\n- <code>%s</code>\n%s", s, ct.GetKind(), utils.IndentLines(u, "  "))
	}
	return s + "\n"
}

// DefaultConfigTypeScheme contains all globally known access serializer.
var DefaultConfigTypeScheme = NewConfigTypeScheme(nil)

////////////////////////////////////////////////////////////////////////////////

type Evaluator interface {
	Evaluate(ctx Context) (Config, error)
}

type GenericConfig struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

func IsGeneric(cfg Config) bool {
	_, ok := cfg.(*GenericConfig)
	return ok
}

func NewGenericConfig(data []byte, unmarshaler runtime.Unmarshaler) (Config, error) {
	unstr := &runtime.UnstructuredVersionedTypedObject{}
	if unmarshaler == nil {
		unmarshaler = runtime.DefaultYAMLEncoding
	}
	err := unmarshaler.Unmarshal(data, unstr)
	if err != nil {
		return nil, err
	}
	return &GenericConfig{*unstr}, nil
}

func ToGenericConfig(c Config) (*GenericConfig, error) {
	if reflect2.IsNil(c) {
		return nil, nil
	}
	if g, ok := c.(*GenericConfig); ok {
		return g, nil
	}
	u, err := runtime.ToUnstructuredVersionedTypedObject(c)
	if err != nil {
		return nil, err
	}
	return &GenericConfig{*u}, nil
}

func (s *GenericConfig) Evaluate(ctx Context) (Config, error) {
	raw, err := s.GetRaw()
	if err != nil {
		return nil, err
	}
	cfg, err := ctx.ConfigTypes().DecodeConfig(raw, runtime.DefaultJSONEncoding)
	if err != nil {
		return nil, err
	}
	if IsGeneric(cfg) {
		return nil, errors.ErrUnknown(KIND_CONFIGTYPE, s.GetType())
	}
	return cfg, nil
}

func (s *GenericConfig) ApplyTo(ctx Context, target interface{}) error {
	spec, err := s.Evaluate(ctx)
	if err != nil {
		return err
	}
	return spec.ApplyTo(ctx, target)
}

var _ Config = &GenericConfig{}

////////////////////////////////////////////////////////////////////////////////

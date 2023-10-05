// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"encoding/json"
	"reflect"

	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/open-component-model/ocm/pkg/errors"
)

type YAMLValue[T any] struct {
	addr *T
}

func NewYAMLValue[T any](val T, p *T) *YAMLValue[T] {
	*p = val
	return &YAMLValue[T]{p}
}

func (i *YAMLValue[T]) String() string {
	v := reflect.ValueOf(i.addr)

	if v.Elem().IsZero() {
		return ""
	}

	data, err := json.Marshal(*i.addr)
	if err != nil {
		return "error " + err.Error()
	}
	return string(data)
}

func (i *YAMLValue[T]) Set(s string) error {
	err := yaml.Unmarshal([]byte(s), i.addr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse YAML: %q", s)
	}
	return nil
}

func (i *YAMLValue[T]) Type() string {
	return "YAML"
}

func YAMLVarP[T any](flags *pflag.FlagSet, p *T, name, shorthand string, value T, usage string) {
	flags.VarP(NewYAMLValue(value, p), name, shorthand, usage)
}

func YAMLVarPF[T any](flags *pflag.FlagSet, p *T, name, shorthand string, value T, usage string) *pflag.Flag {
	return flags.VarPF(NewYAMLValue(value, p), name, shorthand, usage)
}

func parseValue(s string) (interface{}, error) {
	var v interface{}
	err := yaml.Unmarshal([]byte(s), &v)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse YAML: %q", s)
	}
	return v, nil
}

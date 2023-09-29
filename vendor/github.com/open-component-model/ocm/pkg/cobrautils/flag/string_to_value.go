// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

type valueStringToValue struct {
	value   *map[string]interface{}
	changed bool
}

func newStringToValueValue(val map[string]interface{}, p *map[string]interface{}) *valueStringToValue {
	ssv := new(valueStringToValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *valueStringToValue) Set(val string) error {
	k, v, err := parseAssignment(val)
	if err != nil {
		return err
	}
	y, err := parseValue(v)
	if err != nil {
		return err
	}
	if !s.changed {
		*s.value = map[string]interface{}{k: y}
	} else {
		if *s.value == nil {
			*s.value = map[string]interface{}{}
		}
		(*s.value)[k] = y
	}
	s.changed = true
	return nil
}

func (s *valueStringToValue) Type() string {
	return "<name>=<YAML>"
}

func (s *valueStringToValue) String() string {
	if *s.value == nil {
		return ""
	}
	var list []string
	for k, v := range *s.value {
		//nolint: errchkjson // initialized by unmarshal
		s, _ := json.Marshal(v)
		list = append(list, fmt.Sprintf("%s=%s", k, string(s)))
	}
	return "[" + strings.Join(list, ", ") + "]"
}

func (s *valueStringToValue) GetMap() map[string]interface{} {
	return *s.value
}

func StringToValueVar(f *pflag.FlagSet, p *map[string]interface{}, name string, value map[string]interface{}, usage string) {
	f.VarP(newStringToValueValue(value, p), name, "", usage)
}

func StringToValueVarP(f *pflag.FlagSet, p *map[string]interface{}, name, shorthand string, value map[string]interface{}, usage string) {
	f.VarP(newStringToValueValue(value, p), name, shorthand, usage)
}

func StringToValueVarPF(f *pflag.FlagSet, p *map[string]interface{}, name, shorthand string, value map[string]interface{}, usage string) *pflag.Flag {
	return f.VarPF(newStringToValueValue(value, p), name, shorthand, usage)
}

func StringToValue(f *pflag.FlagSet, name string, value map[string]interface{}, usage string) *map[string]interface{} {
	p := map[string]interface{}{}
	StringToValueVarP(f, &p, name, "", value, usage)
	return &p
}

func StringToValueP(f *pflag.FlagSet, name, shorthand string, value map[string]interface{}, usage string) *map[string]interface{} {
	p := map[string]interface{}{}
	StringToValueVarP(f, &p, name, shorthand, value, usage)
	return &p
}

func parseAssignment(s string) (string, string, error) {
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return "", "", fmt.Errorf("expected <name>=<value>")
	}
	return s[:idx], s[idx+1:], nil
}

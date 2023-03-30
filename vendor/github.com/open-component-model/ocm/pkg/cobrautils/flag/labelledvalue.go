// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

type LabelledValue struct {
	Name  string
	Value interface{}
}

type LabelledValueValue LabelledValue

func NewLabelledValueValue(val LabelledValue, p *LabelledValue) *LabelledValueValue {
	*p = val
	return (*LabelledValueValue)(p)
}

func (i *LabelledValueValue) String() string {
	if i.Name == "" {
		return ""
	}
	data, err := json.Marshal(i.Value)
	if err != nil {
		return "error " + err.Error()
	}
	return i.Name + "=" + string(data)
}

func (i *LabelledValueValue) Set(s string) error {
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return fmt.Errorf("expected <name>=<value>")
	}
	i.Name = s[:idx]
	var v interface{}
	err := yaml.Unmarshal([]byte(s[idx+1:]), &v)
	if err != nil {
		return err
	}
	i.Value = v
	return nil
}

func (i *LabelledValueValue) Type() string {
	return "<name>=<YAML>"
}

func LabelledValueVarP(flags *pflag.FlagSet, p *LabelledValue, name, shorthand string, value LabelledValue, usage string) {
	flags.VarP(NewLabelledValueValue(value, p), name, shorthand, usage)
}

func LabelledValueVarPF(flags *pflag.FlagSet, p *LabelledValue, name, shorthand string, value LabelledValue, usage string) *pflag.Flag {
	return flags.VarPF(NewLabelledValueValue(value, p), name, shorthand, usage)
}

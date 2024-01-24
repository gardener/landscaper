// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

type LabelledString struct {
	Name  string
	Value string
}

type labelledStringValue LabelledString

func NewLabelledStringValue(val LabelledString, p *LabelledString) *labelledStringValue {
	*p = val
	return (*labelledStringValue)(p)
}

func (i *labelledStringValue) String() string {
	if i.Name == "" {
		return ""
	}
	return i.Name + "=" + i.Value
}

func (i *labelledStringValue) Set(s string) error {
	name, value, found := strings.Cut(s, "=")
	if !found {
		return fmt.Errorf("expected <name>=<value>")
	}
	i.Name = name
	i.Value = value
	return nil
}

func (i *labelledStringValue) Type() string {
	return "LabelledString"
}

func LabelledStringVarP(flags *pflag.FlagSet, p *LabelledString, name, shorthand string, value LabelledString, usage string) {
	flags.VarP(NewLabelledStringValue(value, p), name, shorthand, usage)
}

func LabelledStringVarPF(flags *pflag.FlagSet, p *LabelledString, name, shorthand string, value LabelledString, usage string) *pflag.Flag {
	return flags.VarPF(NewLabelledStringValue(value, p), name, shorthand, usage)
}

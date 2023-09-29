// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/pflag"
)

type constaintsValue struct {
	value   *[]*semver.Constraints
	changed bool
}

func newConstraintsValue(val []*semver.Constraints, p *[]*semver.Constraints) *constaintsValue {
	ssv := new(constaintsValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *constaintsValue) Set(val string) error {
	c, err := semver.NewConstraint(val)
	if err != nil {
		return err
	}
	if !s.changed {
		*s.value = []*semver.Constraints{c}
	} else {
		*s.value = append(*s.value, c)
	}
	s.changed = true
	return nil
}

func (s *constaintsValue) Type() string {
	return "constraints"
}

func (s *constaintsValue) String() string {
	if *s.value == nil {
		return ""
	}
	var list []string
	for _, v := range *s.value {
		list = append(list, v.String())
	}
	return "[" + strings.Join(list, ", ") + "]"
}

func SemverConstraintsVar(f *pflag.FlagSet, p *[]*semver.Constraints, name string, value []*semver.Constraints, usage string) {
	f.VarP(newConstraintsValue(value, p), name, "", usage)
}

func SemverConstraintsVarP(f *pflag.FlagSet, p *[]*semver.Constraints, name, shorthand string, value []*semver.Constraints, usage string) {
	f.VarP(newConstraintsValue(value, p), name, shorthand, usage)
}

func SemverConstraintsVarPF(f *pflag.FlagSet, p *[]*semver.Constraints, name, shorthand string, value []*semver.Constraints, usage string) *pflag.Flag {
	return f.VarPF(newConstraintsValue(value, p), name, shorthand, usage)
}

func SemverConstraints(f *pflag.FlagSet, name string, value []*semver.Constraints, usage string) *[]*semver.Constraints {
	p := []*semver.Constraints{}
	SemverConstraintsVarP(f, &p, name, "", value, usage)
	return &p
}

func SemverConstraintsP(f *pflag.FlagSet, name, shorthand string, value []*semver.Constraints, usage string) *[]*semver.Constraints {
	p := []*semver.Constraints{}
	SemverConstraintsVarP(f, &p, name, shorthand, value, usage)
	return &p
}

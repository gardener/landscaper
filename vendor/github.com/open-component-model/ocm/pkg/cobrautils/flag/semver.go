// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/pflag"
)

type semverValue struct {
	value   *[]*semver.Version
	changed bool
}

func newSemverValue(val []*semver.Version, p *[]*semver.Version) *semverValue {
	ssv := new(semverValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *semverValue) Set(val string) error {
	c, err := semver.NewVersion(val)
	if err != nil {
		return err
	}
	if !s.changed {
		*s.value = []*semver.Version{c}
	} else {
		*s.value = append(*s.value, c)
	}
	s.changed = true
	return nil
}

func (s *semverValue) Type() string {
	return "[]semver"
}

func (s *semverValue) String() string {
	if *s.value == nil {
		return ""
	}
	var list []string
	for _, v := range *s.value {
		list = append(list, v.String())
	}
	return "[" + strings.Join(list, ", ") + "]"
}

func SemverVar(f *pflag.FlagSet, p *[]*semver.Version, name string, value []*semver.Version, usage string) {
	f.VarP(newSemverValue(value, p), name, "", usage)
}

func SemverVarP(f *pflag.FlagSet, p *[]*semver.Version, name, shorthand string, value []*semver.Version, usage string) {
	f.VarP(newSemverValue(value, p), name, shorthand, usage)
}

func SemverVarPF(f *pflag.FlagSet, p *[]*semver.Version, name, shorthand string, value []*semver.Version, usage string) *pflag.Flag {
	return f.VarPF(newSemverValue(value, p), name, shorthand, usage)
}

func Semver(f *pflag.FlagSet, name string, value []*semver.Version, usage string) *[]*semver.Version {
	p := []*semver.Version{}
	SemverVarP(f, &p, name, "", value, usage)
	return &p
}

func SemverP(f *pflag.FlagSet, name, shorthand string, value []*semver.Version, usage string) *[]*semver.Version {
	p := []*semver.Version{}
	SemverVarP(f, &p, name, shorthand, value, usage)
	return &p
}

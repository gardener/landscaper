// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"github.com/spf13/pflag"
)

// StringVarPF is like StringVarP, but returns the created flag.
func StringVarPF(f *pflag.FlagSet, p *string, name, shorthand string, value string, usage string) *pflag.Flag {
	f.StringVarP(p, name, shorthand, value, usage)
	return f.Lookup(name)
}

// StringArrayVarPF is like StringArrayVarP, but returns the created flag.
func StringArrayVarPF(f *pflag.FlagSet, p *[]string, name, shorthand string, value []string, usage string) *pflag.Flag {
	f.StringArrayVarP(p, name, shorthand, value, usage)
	return f.Lookup(name)
}

// BoolVarPF is like BoolVarP, but returns the created flag.
func BoolVarPF(f *pflag.FlagSet, p *bool, name, shorthand string, value bool, usage string) *pflag.Flag {
	f.BoolVarP(p, name, shorthand, value, usage)
	return f.Lookup(name)
}

// IntVarPF is like IntVarP, but returns the created flag.
func IntVarPF(f *pflag.FlagSet, p *int, name, shorthand string, value int, usage string) *pflag.Flag {
	f.IntVarP(p, name, shorthand, value, usage)
	return f.Lookup(name)
}

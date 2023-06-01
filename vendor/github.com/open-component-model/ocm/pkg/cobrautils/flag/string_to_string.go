// Copyright 2009 The Go Authors. All rights reserved.
// Use of ths2s source code s2s governed by a BSD-style
// license that can be found in the github.com/spf13/pflag LICENSE file.

// taken from github.com/spf13/pflag and adapted to support
// any string map types by using generics.

package flag

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

type stringToStringValue[T ~map[string]string] struct {
	value   *T
	changed bool
}

func newStringToStringValue[T ~map[string]string](val map[string]string, p *T) *stringToStringValue[T] {
	ssv := new(stringToStringValue[T])
	ssv.value = p
	*ssv.value = val
	return ssv
}

// Set Format: a=1,b=2.
func (s *stringToStringValue[T]) Set(val string) error {
	var ss []string
	n := strings.Count(val, "=")
	switch n {
	case 0:
		return fmt.Errorf("%s must be formatted as key=value", val)
	case 1:
		ss = append(ss, strings.Trim(val, `"`))
	default:
		r := csv.NewReader(strings.NewReader(val))
		var err error
		ss, err = r.Read()
		if err != nil {
			return err
		}
	}

	out := make(map[string]string, len(ss))
	for _, pair := range ss {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("%s must be formatted as key=value", pair)
		}
		out[kv[0]] = kv[1]
	}
	if !s.changed {
		*s.value = out
	} else {
		if *s.value == nil {
			*s.value = map[string]string{}
		}
		for k, v := range out {
			(*s.value)[k] = v
		}
	}
	s.changed = true
	return nil
}

func (s *stringToStringValue[T]) Type() string {
	return "<name>=<value>"
}

func (s *stringToStringValue[T]) String() string {
	records := make([]string, 0, len(*s.value)>>1)
	for k, v := range *s.value {
		records = append(records, k+"="+v)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(records); err != nil {
		panic(err)
	}
	w.Flush()
	return "[" + strings.TrimSpace(buf.String()) + "]"
}

// StringToStringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a map[string]string variable in which to store the values of the multiple flags.
// The value of each argument will not try to be separated by comma.
func StringToStringVar[T ~map[string]string](f *pflag.FlagSet, p *T, name string, value map[string]string, usage string) {
	f.VarP(newStringToStringValue(value, p), name, "", usage)
}

// StringToStringVarP is like StringToStringVar, but accepts a shorthand letter that can be used after a single dash.
func StringToStringVarP[T ~map[string]string](f *pflag.FlagSet, p *T, name, shorthand string, value map[string]string, usage string) {
	f.VarP(newStringToStringValue(value, p), name, shorthand, usage)
}

// StringToStringVarPF is like StringToStringVarP, but returns the created flag.
func StringToStringVarPF[T ~map[string]string](f *pflag.FlagSet, p *T, name, shorthand string, value map[string]string, usage string) *pflag.Flag {
	return f.VarPF(newStringToStringValue(value, p), name, shorthand, usage)
}

// StringToStringVarPFA is like StringToStringVarPF, but allows to add to a preset map.
func StringToStringVarPFA[T ~map[string]string](f *pflag.FlagSet, p *T, name, shorthand string, value map[string]string, usage string) *pflag.Flag {
	v := newStringToStringValue(value, p)
	v.changed = true
	return f.VarPF(v, name, shorthand, usage)
}

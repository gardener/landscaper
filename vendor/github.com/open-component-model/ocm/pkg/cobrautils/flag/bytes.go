// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"encoding/base64"
	"strings"

	"github.com/spf13/pflag"
)

// BytesBase64 adapts []byte for use as a flag. Value of flag is Base64 encoded.
// If the given string starts with an '!', the rest is used as string byte sequence.
type bytesBase64Value []byte

// String implements pflag.Value.String.
func (bytesBase64 bytesBase64Value) String() string {
	return base64.StdEncoding.EncodeToString([]byte(bytesBase64))
}

// Set implements pflag.Value.Set.
func (bytesBase64 *bytesBase64Value) Set(value string) error {
	if !strings.HasPrefix(value, "!") {
		bin, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		*bytesBase64 = bin
		return nil
	}
	*bytesBase64 = []byte(value[1:])
	return nil
}

// Type implements pflag.Value.Type.
func (*bytesBase64Value) Type() string {
	return "!bytesBase64"
}

func newBytesBase64Value(val []byte, p *[]byte) *bytesBase64Value {
	*p = val
	return (*bytesBase64Value)(p)
}

// BytesBase64VarP is like BytesBase64Var, but accepts a shorthand letter that can be used after a single dash.
func BytesBase64VarP(f *pflag.FlagSet, p *[]byte, name, shorthand string, value []byte, usage string) {
	f.VarP(newBytesBase64Value(value, p), name, shorthand, usage)
}

// BytesBase64Var defines an []byte flag with specified name, default value, and usage string.
// The return value is the address of an []byte variable that stores the value of the flag.
func BytesBase64Var(f *pflag.FlagSet, name string, value []byte, usage string) *[]byte {
	p := new([]byte)
	BytesBase64VarP(f, p, name, "", value, usage)
	return p
}

func BytesBase64VarPF(f *pflag.FlagSet, p *[]byte, name, shorthand string, value []byte, usage string) *pflag.Flag {
	return f.VarPF(newBytesBase64Value(value, p), name, shorthand, usage)
}

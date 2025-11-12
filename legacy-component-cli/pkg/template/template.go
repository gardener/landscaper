// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"strings"

	"github.com/drone/envsubst"
)

// Options defines the options for component-cli templating
type Options struct {
	Vars map[string]string
}

// Usage prints out the usage for templating
func (o *Options) Usage() string {
	return `
Templating:
All yaml/json defined resources can be templated using simple envsubst syntax.
Variables are specified after a "--" and follow the syntax "<name>=<value>".

Note: Variable names are case-sensitive.

Example:
<pre>
<command> [args] [--flags] -- MY_VAL=test
</pre>

<pre>

key:
  subkey: "abc ${MY_VAL}"

</pre>

`
}

// Parse parses commandline argument variables.
// it returns all non variable arguments
func (o *Options) Parse(args []string) []string {
	o.Vars = make(map[string]string)
	var addArgs []string
	for _, arg := range args {
		if i := strings.Index(arg, "="); i > 0 {
			value := arg[i+1:]
			name := arg[0:i]
			o.Vars[name] = value
			continue
		}
		addArgs = append(addArgs, arg)
	}
	return addArgs
}

// Template templates a string with the parsed vars.
func (o *Options) Template(data string) (string, error) {
	return envsubst.Eval(data, o.mapping)
}

// mapping is a helper function for the envsubst to provide the value for a variable name.
// It returns an emtpy string if the variable is not defined.
func (o *Options) mapping(variable string) string {
	if o.Vars == nil {
		return ""
	}
	// todo: maybe use os.getenv as backup.
	return o.Vars[variable]
}

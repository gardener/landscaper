// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

func AccessUsage(scheme AccessTypeScheme, cli bool) string {
	s := `
The following list describes the supported access methods, their versions
and specification formats.
Typically there is special support for the CLI artifact add commands.
The access method specification can be put below the <code>access</code> field.
If always requires the field <code>type</code> describing the kind and version
shown below.
`
	type method struct {
		desc     string
		versions map[string]string
		options  flagsets.ConfigOptionTypeSetHandler
	}

	descs := map[string]*method{}

	// gather info for kinds and versions
	for _, n := range scheme.KnownTypeNames() {
		kind, vers := runtime.KindVersion(n)

		info := descs[kind]
		if info == nil {
			info = &method{versions: map[string]string{}}
			descs[kind] = info
		}

		if vers == "" {
			vers = "v1"
		}
		if _, ok := info.versions[vers]; !ok {
			info.versions[vers] = ""
		}

		t := scheme.GetType(n)

		if t.ConfigOptionTypeSetHandler() != nil {
			info.options = t.ConfigOptionTypeSetHandler()
		}
		desc := t.Description()
		if desc != "" {
			info.desc = desc
		}

		desc = t.Format()
		if desc != "" {
			info.versions[vers] = desc
		}
	}

	for _, t := range utils.StringMapKeys(descs) {
		info := descs[t]
		desc := strings.Trim(info.desc, "\n")
		if desc != "" {
			s = fmt.Sprintf("%s\n- Access type <code>%s</code>\n\n%s\n\n", s, t, utils.IndentLines(desc, "  "))

			format := ""
			for _, f := range utils.StringMapKeys(info.versions) {
				desc = strings.Trim(info.versions[f], "\n")
				if desc != "" {
					format = fmt.Sprintf("%s\n- Version <code>%s</code>\n\n%s\n", format, f, utils.IndentLines(desc, "  "))
				}
			}
			if format != "" {
				s += fmt.Sprintf("  The following versions are supported:\n%s\n", strings.Trim(utils.IndentLines(format, "  "), "\n"))
			}
		}
		s += utils.IndentLines(flagsets.FormatConfigOptions(info.options), "  ")
	}
	return s
}

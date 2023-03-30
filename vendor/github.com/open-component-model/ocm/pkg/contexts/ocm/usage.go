// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"strings"

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
	versions := map[string]map[string]string{}
	descs := map[string]string{}

	// gather info for kinds and versions
	for _, n := range scheme.KnownTypeNames() {
		kind, vers := runtime.KindVersion(n)

		if _, ok := descs[kind]; !ok {
			descs[kind] = ""
		}
		var set map[string]string
		if set = versions[kind]; set == nil {
			set = map[string]string{}
			versions[kind] = set
		}
		if vers == "" {
			vers = "v1"
		}
		if _, ok := set[vers]; !ok {
			set[vers] = ""
		}

		t := scheme.GetAccessType(n)

		desc := t.Description()
		if desc != "" {
			descs[kind] = desc
		}

		desc = t.Format(cli)
		if desc != "" {
			set[vers] = desc
		}
	}

	for _, t := range utils.StringMapKeys(descs) {
		desc := strings.Trim(descs[t], "\n")
		if desc != "" {
			s = fmt.Sprintf("%s\n- Access type <code>%s</code>\n\n%s\n\n", s, t, utils.IndentLines(desc, "  "))

			format := ""
			for _, f := range utils.StringMapKeys(versions[t]) {
				desc = strings.Trim(versions[t][f], "\n")
				if desc != "" {
					format = fmt.Sprintf("%s\n- Version <code>%s</code>\n\n%s\n", format, f, utils.IndentLines(desc, "  "))
				}
			}
			if format != "" {
				s += fmt.Sprintf("  The following versions are supported:\n%s\n", strings.Trim(utils.IndentLines(format, "  "), "\n"))
			}
		}
	}
	return s
}

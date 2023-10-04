// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/open-component-model/ocm/pkg/utils"
)

const FlagGroupAnnotation = "flag-group-annotation"

// FlagUsagesWrapped returns a string containing the usage information
// for all flags in the FlagSet. Wrapped to `cols` columns (0 for no
// wrapping)
// It groups flags according to group annotation.
func FlagUsagesWrapped(f *pflag.FlagSet, cols int) string {
	lines := DetermineGroups(f, cols)

	sep := ""
	buf := new(bytes.Buffer)
	for _, g := range utils.StringMapKeys(lines) {
		if g != "" {
			fmt.Fprintln(buf, sep+"  "+g+":")
		}
		sep = "\n"
		for _, line := range lines[g] {
			fmt.Fprintln(buf, line)
		}
	}

	return buf.String()
}

type UsageGroup struct {
	Title  string
	Usages string
}

func GroupedFlagUsagesWrapped(f *pflag.FlagSet, cols int) []UsageGroup {
	lines := DetermineGroups(f, cols)

	var groups []UsageGroup
	for _, g := range utils.StringMapKeys(lines) {
		buf := new(bytes.Buffer)
		for _, line := range lines[g] {
			fmt.Fprintln(buf, line)
		}
		groups = append(groups, UsageGroup{
			Title:  g,
			Usages: buf.String(),
		})
	}

	return groups
}

func DetermineGroups(f *pflag.FlagSet, cols int) map[string][]string {
	lines := map[string][]string{}
	for _, line := range strings.Split(f.FlagUsagesWrapped(cols), "\n") {
		i := strings.Index(line, "--")
		if i < 0 {
			continue
		}

		name := line[i+2 : i+strings.Index(line[i:], " ")]
		flag := f.Lookup(name)
		groups := []string{""}
		if flag.Annotations != nil {
			g := flag.Annotations[FlagGroupAnnotation]
			if len(g) > 0 {
				groups = g
			}
		}
		for _, g := range groups {
			lines[g] = append(lines[g], line)
		}
	}
	return lines
}

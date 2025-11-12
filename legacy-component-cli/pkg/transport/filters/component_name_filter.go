// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	"fmt"
	"regexp"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

type ComponentNameFilterSpec struct {
	IncludeComponentNames []string
}

type componentNameFilter struct {
	includeComponentNames []*regexp.Regexp
}

func (f componentNameFilter) Matches(cd cdv2.ComponentDescriptor, r cdv2.Resource) bool {
	for _, icn := range f.includeComponentNames {
		if icn.MatchString(cd.Name) {
			return true
		}
	}
	return false
}

// NewComponentNameFilter creates a new componentNameFilter
func NewComponentNameFilter(spec ComponentNameFilterSpec) (Filter, error) {
	if len(spec.IncludeComponentNames) == 0 {
		return nil, fmt.Errorf("includeComponentNames must not be empty")
	}

	icnRegexps := []*regexp.Regexp{}
	for _, icn := range spec.IncludeComponentNames {
		icnRegexp, err := regexp.Compile(icn)
		if err != nil {
			return nil, fmt.Errorf("unable to parse regexp %s: %w", icn, err)
		}
		icnRegexps = append(icnRegexps, icnRegexp)
	}

	filter := componentNameFilter{
		includeComponentNames: icnRegexps,
	}

	return &filter, nil
}

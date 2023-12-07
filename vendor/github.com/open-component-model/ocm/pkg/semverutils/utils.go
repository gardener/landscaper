// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package semverutils

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"

	"github.com/open-component-model/ocm/pkg/errors"
)

// MatchVersionStrings returns an ordered list of versions filtered by the given
// constraints. If no constraints a given the complete list is returned.
// If one given version is no semver version it is ignored for the matching
// and an additional error describing the parsing errors is returned.
func MatchVersionStrings(vers []string, constraints ...*semver.Constraints) (semver.Collection, error) {
	var versions semver.Collection
	list := errors.ErrListf("invalid semver versions")
	for _, vn := range vers {
		v, err := semver.NewVersion(vn)
		if err == nil {
			versions = append(versions, v)
		} else {
			list.Add(fmt.Errorf("%s", vn))
		}
	}
	return MatchVersions(versions, constraints...), list.Result()
}

func MatchVersions(versions semver.Collection, constraints ...*semver.Constraints) semver.Collection {
	// Filter by patterns
	if len(constraints) > 0 {
	next:
		for i := 0; i < len(versions); i++ {
			for _, c := range constraints {
				if c.Check(versions[i]) {
					continue next
				}
			}
			versions = append(versions[:i], versions[i+1:]...)
			i--
		}
	}
	sort.Sort(versions)
	return versions
}

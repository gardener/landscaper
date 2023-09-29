// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"sort"

	"github.com/Masterminds/semver/v3"
)

type VersionCache map[string]*semver.Version

func (c VersionCache) Get(v string) (*semver.Version, error) {
	if s := c[v]; s != nil {
		return s, nil
	}
	s, err := semver.NewVersion(v)
	if err != nil {
		return nil, err
	}
	c[v] = s
	return s, nil
}

func SortVersions(vers []string) error {
	cache := VersionCache{}
	for _, v := range vers {
		_, err := cache.Get(v)
		if err != nil {
			return err
		}
	}

	sort.Slice(vers, func(a, b int) bool {
		va, _ := cache.Get(vers[a])
		vb, _ := cache.Get(vers[b])
		return va.Compare(vb) < 0
	})
	return nil
}

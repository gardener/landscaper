// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

func init() {
	cpi.RegisterRepositorySpecHandler(&repospechandler{}, "*")
	cpi.RegisterRefParseHandler(Type, HandleRef)
}

type repospechandler struct{}

func (h *repospechandler) MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	var meta *ComponentRepositoryMeta
	host := u.Host
	subp := u.SubPath

	if u.Type == Type {
		if u.Info != "" && u.SubPath == "" {
			idx := strings.Index(u.Info, grammar.RepositorySeparator)
			if idx > 0 {
				host = u.Info[:idx]
				subp = u.Info[idx+1:]
			} else {
				host = u.Info
			}
		} else if u.Host == "" {
			return nil, fmt.Errorf("host required for OCI based OCM reference")
		}
	} else {
		if u.Type != "" || u.Info != "" || u.Host == "" {
			return nil, nil
		}
		host = u.Host
	}
	if subp != "" {
		meta = NewComponentRepositoryMeta(subp, "")
	}
	if compatattr.Get(ctx) {
		return NewRepositorySpec(ocireg.NewLegacyRepositorySpec(host), meta), nil
	}
	return NewRepositorySpec(ocireg.NewRepositorySpec(host), meta), nil
}

func HandleRef(u *cpi.UniformRepositorySpec) error {
	if u.Host == "" && u.Info != "" && u.SubPath == "" {
		host := ""
		subp := ""
		idx := strings.Index(u.Info, grammar.RepositorySeparator)
		if idx > 0 {
			host = u.Info[:idx]
			subp = u.Info[idx+1:]
		} else {
			host = u.Info
		}
		if grammar.HostPortRegexp.MatchString(host) || grammar.DomainPortRegexp.MatchString(host) {
			u.Host = host
			u.SubPath = subp
			u.Info = ""
		}
	}
	return nil
}

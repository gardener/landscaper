// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

func init() {
	cpi.RegisterRepositorySpecHandler(&repospechandler{}, Type)
}

type repospechandler struct{}

func (h *repospechandler) MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	host := u.Host
	if u.Scheme != "" && host != "" {
		host = u.Scheme + "://" + u.Host
	}
	if u.Info != "" {
		if u.Info == "default" {
			host = ""
		} else if host == "" {
			host = u.Info
		}
	}
	return NewRepositorySpec(host), nil
}

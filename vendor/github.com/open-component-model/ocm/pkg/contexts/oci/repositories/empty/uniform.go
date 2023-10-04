// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package empty

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

func init() {
	cpi.RegisterRepositorySpecHandler(&repospechandler{}, Type)
}

type repospechandler struct{}

func (h *repospechandler) MapReference(ctx cpi.Context, u *cpi.UniformRepositorySpec) (cpi.RepositorySpec, error) {
	if u.Info != "" || u.Host == "" {
		return nil, nil
	}

	return NewRepositorySpec(), nil
}

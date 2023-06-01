// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/errors"
)

func GetOCIArtifactRef(ctx ocm.Context, r ocm.ResourceAccess) (string, error) {
	acc, err := r.Access()
	if err != nil {
		return "", err
	}

	if localblob.Is(acc) {
		g := acc.(*localblob.AccessSpec).GlobalAccess
		if g != nil {
			acc, err = ctx.AccessSpecForSpec(g)
			if err != nil {
				return "", errors.Wrapf(err, "global access spec")
			}
		}
	}
	if ociartifact.Is(acc) {
		return acc.(*ociartifact.AccessSpec).ImageReference, nil
	}
	return "", errors.Newf("cannot map access to external image reference")
}

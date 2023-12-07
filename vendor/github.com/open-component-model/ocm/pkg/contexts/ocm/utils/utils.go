// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

func GetOCIArtifactRef(ctxp ocm.ContextProvider, r ocm.ResourceAccess) (string, error) {
	ctx := ctxp.OCMContext()

	acc, err := r.Access()
	if err != nil || acc == nil {
		return "", err
	}

	var cv cpi.ComponentVersionAccess
	if p, ok := r.(cpi.ComponentVersionProvider); ok {
		cv, err = p.GetComponentVersion()
		if err != nil {
			return "", errors.Wrapf(err, "cannot access component version for re/source")
		}
		defer cv.Close()
	}

	return ociartifact.GetOCIArtifactReference(ctx, acc, cv)
}

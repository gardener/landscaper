// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci"
)

// TODO: add view concept to OCI context

type nonClosing struct {
	oci.Repository
}

func (n *nonClosing) Close() error {
	return nil
}

func view(repo oci.Repository) oci.Repository {
	return &nonClosing{repo}
}

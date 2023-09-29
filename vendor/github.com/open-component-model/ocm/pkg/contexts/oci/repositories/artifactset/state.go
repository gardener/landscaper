// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

// NewStateHandler implements the factory interface for the artifact set
// state descriptor handling
// Basically this is an index state.
func NewStateHandler(fs vfs.FileSystem) accessobj.StateHandler {
	return &cpi.IndexStateHandler{}
}

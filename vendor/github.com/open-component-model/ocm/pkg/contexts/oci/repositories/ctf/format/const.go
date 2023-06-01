// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
)

const (
	DirMode  = accessobj.DirMode
	FileMode = accessobj.FileMode
)

var ModTime = accessobj.ModTime

const (
	// BlobsDirectoryName is the name of the directory holding the artifact archives.
	BlobsDirectoryName = artifactset.BlobsDirectoryName
	// ArtifactIndexFileName is the artifact index descriptor name for CommanTransportFormat.
	ArtifactIndexFileName = "artifact-index.json"
)

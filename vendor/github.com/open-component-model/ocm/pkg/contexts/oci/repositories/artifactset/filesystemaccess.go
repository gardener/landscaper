// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi/support"
)

type FileSystemBlobAccess struct {
	*accessobj.FileSystemBlobAccess
}

func NewFileSystemBlobAccess(access *accessobj.AccessObject) *FileSystemBlobAccess {
	return &FileSystemBlobAccess{accessobj.NewFileSystemBlobAccess(access)}
}

func (i *FileSystemBlobAccess) GetArtifact(access support.NamespaceAccessImpl, digest digest.Digest) (acc cpi.ArtifactAccess, err error) {
	v, err := access.View()
	if err != nil {
		return nil, err
	}
	defer v.Close()
	_, data, err := i.GetBlobData(digest)
	if err == nil {
		blob := accessio.BlobAccessForDataAccess("", -1, "", data)
		acc, err = support.NewArtifactForBlob(access, blob)
	}
	return
}

func (i *FileSystemBlobAccess) AddArtifactBlob(artifact cpi.Artifact) (cpi.BlobAccess, error) {
	blob, err := artifact.Blob()
	if err != nil {
		return nil, err
	}

	err = i.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

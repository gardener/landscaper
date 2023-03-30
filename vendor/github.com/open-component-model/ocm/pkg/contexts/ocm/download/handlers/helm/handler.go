// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blob

import (
	"io"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/mime"
)

const TYPE = resourcetypes.HELM_CHART

type Handler struct{}

func init() {
	download.RegisterForArtifactType(TYPE, &Handler{})
}

func (h Handler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	meth, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	defer meth.Close()
	if mime.BaseType(meth.MimeType()) != mime.BaseType(artdesc.MediaTypeImageManifest) {
		return false, "", nil
	}
	rd, err := meth.Reader()
	if err != nil {
		return true, "", err
	}
	defer rd.Close()
	set, err := artifactset.Open(accessobj.ACC_READONLY, "", 0, accessio.Reader(rd))
	if err != nil {
		return true, "", err
	}
	art, err := set.GetArtifact(set.GetMain().String())
	if err != nil {
		return true, "", err
	}
	m := art.ManifestAccess()
	if m == nil {
		return true, "", errors.Newf("artifact is no image manifest")
	}
	if len(m.GetDescriptor().Layers) < 1 {
		return true, "", errors.Newf("no layers found")
	}
	if !strings.HasSuffix(path, ".tgz") {
		path += ".tgz"
	}
	blob, err := m.GetBlob(m.GetDescriptor().Layers[0].Digest)
	if err != nil {
		return true, "", err
	}
	err = h.write(p, blob, path, fs)
	if err != nil {
		return true, "", err
	}
	if len(m.GetDescriptor().Layers) > 1 {
		path = path[:len(path)-3] + "prov"
		blob, err := m.GetBlob(m.GetDescriptor().Layers[1].Digest)
		if err != nil {
			return true, "", err
		}
		err = h.write(p, blob, path, fs)
		if err != nil {
			return true, "", err
		}
	}
	return true, path, nil
}

func (_ Handler) write(p common.Printer, blob accessio.BlobAccess, path string, fs vfs.FileSystem) error {
	cr, err := blob.Reader()
	if err != nil {
		return err
	}
	defer cr.Close()
	file, err := fs.OpenFile(path, vfs.O_TRUNC|vfs.O_CREATE|vfs.O_WRONLY, 0o660)
	if err != nil {
		return err
	}
	defer file.Close()
	n, err := io.Copy(file, cr)
	if err == nil {
		p.Printf("%s: %d byte(s) written\n", path, n)
	}
	return nil
}

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
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	registry "github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/mime"
)

const TYPE = resourcetypes.HELM_CHART

type Handler struct{}

func init() {
	registry.Register(&Handler{}, registry.ForArtifactType(TYPE))
}

func (h Handler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (_ bool, _ string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&err, "downloading helm chart")

	meth, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	finalize.Close(meth)
	if mime.BaseType(meth.MimeType()) != mime.BaseType(artdesc.MediaTypeImageManifest) {
		return false, "", nil
	}
	rd, err := meth.Reader()
	if err != nil {
		return true, "", err
	}
	finalize.Close(rd)
	set, err := artifactset.Open(accessobj.ACC_READONLY, "", 0, accessio.Reader(rd))
	if err != nil {
		return true, "", err
	}
	finalize.Close(set)
	art, err := set.GetArtifact(set.GetMain().String())
	if err != nil {
		return true, "", err
	}
	finalize.Close(art)
	if path == "" {
		path = racc.Meta().GetName()
	}
	_, _, err = download(p, art, path, fs)
	if err != nil {
		return true, "", err
	}
	return true, "", nil
}

func download(p common.Printer, art oci.ArtifactAccess, path string, fs vfs.FileSystem) (chart, prov string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	m := art.ManifestAccess()
	if m == nil {
		return "", "", errors.Newf("artifact is no image manifest")
	}
	if len(m.GetDescriptor().Layers) < 1 {
		return "", "", errors.Newf("no layers found")
	}
	chart = path
	if !strings.HasSuffix(chart, ".tgz") {
		chart += ".tgz"
	}
	blob, err := m.GetBlob(m.GetDescriptor().Layers[0].Digest)
	if err != nil {
		return "", "", err
	}
	finalize.Close(blob)
	err = write(p, blob, chart, fs)
	if err != nil {
		return "", "", err
	}
	if len(m.GetDescriptor().Layers) > 1 {
		prov = chart[:len(chart)-3] + "prov"
		blob, err := m.GetBlob(m.GetDescriptor().Layers[1].Digest)
		if err != nil {
			return "", "", err
		}
		err = write(p, blob, path, fs)
		if err != nil {
			return "", "", err
		}
	}
	return chart, prov, err
}

func write(p common.Printer, blob accessio.BlobAccess, path string, fs vfs.FileSystem) (err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	cr, err := blob.Reader()
	if err != nil {
		return err
	}
	finalize.Close(cr)
	file, err := fs.OpenFile(path, vfs.O_TRUNC|vfs.O_CREATE|vfs.O_WRONLY, 0o660)
	if err != nil {
		return err
	}
	finalize.Close(file)
	n, err := io.Copy(file, cr)
	if err == nil {
		p.Printf("%s: %d byte(s) written\n", path, n)
	}
	return nil
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprint

import (
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

const (
	BLUEPRINT_MIMETYPE_LEGACY = "application/vnd.gardener.landscaper.blueprint.layer.v1.tar"
	BLUEPRINT_MIMETYPE        = "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"
)

func ExtractArchive(pr common.Printer, _ *Handler, access accessio.DataAccess, path string, fs vfs.FileSystem) (_ bool, rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&rerr, "extracting archived (and compressed) blueprint")

	rawReader, err := access.Reader()
	if err != nil {
		return true, err
	}
	finalize.Close(rawReader)

	reader, _, err := compression.AutoDecompress(rawReader)
	if err != nil {
		return true, err
	}
	finalize.Close(reader)

	err = fs.MkdirAll(path, 0o700)
	if err != nil {
		return true, err
	}

	pfs, err := projectionfs.New(fs, path)
	if err != nil {
		return true, err
	}
	fcnt, bcnt, err := tarutils.ExtractTarToFsWithInfo(pfs, reader)
	if err != nil {
		return true, err
	}
	pr.Printf("%s: %d file(s) with %d byte(s) written\n", path, fcnt, bcnt)
	return true, nil
}

func ExtractArtifact(pr common.Printer, handler *Handler, access accessio.DataAccess, path string, fs vfs.FileSystem) (_ bool, rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&rerr, "extracting oci artifact containing a blueprint")

	rd, err := access.Reader()
	if err != nil {
		return true, err
	}
	finalize.Close(rd)

	set, err := artifactset.Open(accessobj.ACC_READONLY, "", 0, accessio.Reader(rd))
	if err != nil {
		return true, err
	}
	finalize.Close(set)

	art, err := set.GetArtifact(set.GetMain().String())
	if err != nil {
		return true, err
	}
	finalize.Close(art)

	desc := art.ManifestAccess().GetDescriptor().Layers[0]
	if !handler.ociConfigMimeTypes.Contains(art.ManifestAccess().GetDescriptor().Config.MediaType) {
		if desc.MediaType != BLUEPRINT_MIMETYPE && desc.MediaType != BLUEPRINT_MIMETYPE_LEGACY {
			return false, nil
		}
	}

	blob, err := art.GetBlob(desc.Digest)
	if err != nil {
		return true, err
	}
	return ExtractArchive(pr, handler, blob, path, fs)
}

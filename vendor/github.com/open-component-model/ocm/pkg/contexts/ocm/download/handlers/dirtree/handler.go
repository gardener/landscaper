// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dirtree

import (
	"fmt"
	"io"
	"os"

	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/utils"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

var (
	MimeOCIImageArtifactArchive = artifactset.MediaType(artdesc.MediaTypeImageManifest)
	MimeOCIImageArtifact        = artdesc.ToContentMediaType(artdesc.MediaTypeImageManifest)
)

var (
	supportedMimeTypes   = []string{MimeOCIImageArtifactArchive, mime.MIME_TGZ, mime.MIME_TGZ_ALT, mime.MIME_TAR}
	defaultArtifactTypes = []string{resourcetypes.DIRECTORY_TREE, resourcetypes.FILESYSTEM_LEGACY}
)

func SupportedMimeTypes() []string {
	return slices.Clone(supportedMimeTypes)
}

type Handler struct {
	ociConfigtypes generics.Set[string]
	archive        bool
}

func New(mimetypes ...string) *Handler {
	if len(mimetypes) == 0 || utils.Optional(mimetypes...) == "" {
		mimetypes = []string{artdesc.MediaTypeImageConfig}
	}
	return &Handler{
		ociConfigtypes: generics.NewSet[string](mimetypes...),
	}
}

var DefaultHandler = New()

func init() {
	for _, t := range defaultArtifactTypes {
		for _, m := range supportedMimeTypes {
			download.Register(DefaultHandler, download.ForCombi(t, m))
		}
	}
}

func (h *Handler) SetArchiveMode(b bool) *Handler {
	h.archive = b
	return h
}

func (h *Handler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	lfs, r, err := h.GetForResource(racc)
	if err != nil || (lfs == nil && r == nil) {
		return err != nil, "", err
	}
	if path == "" {
		path = racc.Meta().GetName()
	}
	return h.download(p, fs, path, lfs, r)
}

func (h *Handler) DownloadFromArtifactSet(pr common.Printer, set *artifactset.ArtifactSet, path string, fs vfs.FileSystem) (bool, string, error) {
	lfs, r, err := h.GetForArtifactSet(set)
	if err != nil || (lfs == nil && r != nil) {
		return err != nil, "", err
	}
	if path == "" {
		path = set.GetMain().String()
	}
	return h.download(common.NewPrinter(nil), fs, path, lfs, r)
}

func (h *Handler) download(pr common.Printer, fs vfs.FileSystem, path string, lfs vfs.FileSystem, r io.ReadCloser) (ok bool, dest string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	if r != nil {
		finalize.Close(r)
	}
	if lfs != nil {
		finalize.With(func() error { return vfs.Cleanup(lfs) })
	}
	if h.archive {
		w, err := fs.OpenFile(path, vfs.O_TRUNC|vfs.O_CREATE|vfs.O_WRONLY, 0o600)
		if err != nil {
			return true, "", errors.Wrapf(err, "cannot write target archive %s", path)
		}
		finalize.Close(w)
		if r != nil {
			n, err := io.Copy(w, r)
			if err != nil {
				return true, "", errors.Wrapf(err, "cannot copy to archive %s", path)
			}
			pr.Printf("%s: %d byte(s) written\n", path, n)
			return true, path, nil
		} else {
			cw := accessio.NewCountingWriter(w)
			err := tarutils.PackFsIntoTar(lfs, "", cw, tarutils.TarFileSystemOptions{})
			if err == nil {
				pr.Printf("%s: %d byte(s) written\n", path, cw.Size())
			}
			return true, path, err
		}
	} else {
		err := fs.MkdirAll(path, 0o700)
		if err != nil {
			return true, "", errors.Wrapf(err, "cannot create target directory")
		}

		var fcnt, size int64
		if r != nil {
			var p vfs.FileSystem
			p, err = projectionfs.New(fs, path)
			if err != nil {
				return true, "", err
			}
			fcnt, size, err = tarutils.ExtractTarToFsWithInfo(p, r)
		} else {
			fcnt, size, err = CopyDir(lfs, "/", fs, path)
		}
		if err == nil {
			pr.Printf("%s: %d file(s) with %d byte(s) written\n", path, fcnt, size)
		}
		return true, path, err
	}
}

// GetForResource provides a virtual filesystem for an OCi image manifest
// provided by the given resource matching the configured config types.
// It returns nil without error, if the OCI artifact does not match the requirement.
func (h *Handler) GetForResource(racc cpi.ResourceAccess) (fs vfs.FileSystem, reader io.ReadCloser, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	meth, err := racc.AccessMethod()
	if err != nil {
		return nil, nil, err
	}
	finalize.Close(meth)

	media := mime.BaseType(meth.MimeType())

	switch media {
	case mime.MIME_TGZ, mime.MIME_TAR:
	case MimeOCIImageArtifact:
	default:
		return nil, nil, nil
	}

	r, err := meth.Reader()
	if err != nil {
		return nil, nil, err
	}
	if media != MimeOCIImageArtifact {
		r, _, err = compression.AutoDecompress(r)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot determine compression for filesystem blob")
		}
		return nil, finalize.BindToReader(r), nil
	}
	finalize.Close(r)
	set, err := artifactset.Open(accessobj.ACC_READONLY, "", 0, accessio.Reader(r))
	if err != nil {
		return nil, nil, err
	}
	finalize.Close(set)
	return h.getForArtifactSet(&finalize, set)
}

// GetForArtifactSet provides a virtual filesystem for an OCi image manifest
// provided by the given artifact set matching the configured config types.
// It returns nil without error, if the OCI artifact does not match the requirement.
func (h *Handler) GetForArtifactSet(set *artifactset.ArtifactSet) (fs vfs.FileSystem, reader io.ReadCloser, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)
	return h.getForArtifactSet(&finalize, set)
}

func (h *Handler) getForArtifactSet(finalize *finalizer.Finalizer, set *artifactset.ArtifactSet) (fs vfs.FileSystem, reader io.ReadCloser, err error) {
	m, err := set.GetArtifact(set.GetMain().String())
	if err != nil {
		return nil, nil, err
	}
	if !m.IsManifest() {
		return nil, nil, fmt.Errorf("oci artifact is no image manifest")
	}
	finalize.Close(m)
	macc := m.ManifestAccess()
	if !h.ociConfigtypes.Contains(macc.GetDescriptor().Config.MediaType) {
		return nil, nil, nil
	}

	var cfs vfs.FileSystem
	finalize.With(func() error {
		return vfs.Cleanup(cfs)
	})

	// setup layered filesystem from manifest layers
	for i, l := range macc.GetDescriptor().Layers {
		nested := finalize.Nested()

		blob, err := macc.GetBlob(l.Digest)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot get blob for layer %d", i)
		}
		nested.Close(blob)
		r, err := blob.Reader()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot get reader for layer blob %d", i)
		}
		nested.Close(r)
		r, _, err = compression.AutoDecompress(r)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot determine compression for layer blob %d", i)
		}

		if len(macc.GetDescriptor().Layers) == 1 {
			// return archive reader to enable optimized handling bay caller
			return nil, finalize.BindToReader(r), nil
		}

		nested.Close(r)

		fslayer, err := osfs.NewTempFileSystem()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot create filesystem for layer %d", i)
		}
		nested.With(func() error {
			return vfs.Cleanup(fslayer)
		})
		err = tarutils.ExtractTarToFs(fslayer, r)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot unpack layer blob %d", i)
		}

		if cfs == nil {
			cfs = fslayer
		} else {
			cfs = layerfs.New(fslayer, cfs)
		}
		fslayer = nil // don't cleanup used layer
		nested.Finalize()
	}
	fs = cfs
	cfs = nil // don't cleanup used filesystem
	return fs, nil, nil
}

// TODO: to be moved to vfs

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory may exist.
// Symlinks are ignored and skipped.
func CopyDir(srcfs vfs.FileSystem, src string, dstfs vfs.FileSystem, dst string) (int64, int64, error) {
	var fcnt, bcnt int64
	var n, b int64

	src = vfs.Trim(srcfs, src)
	dst = vfs.Trim(dstfs, dst)

	si, err := srcfs.Stat(src)
	if err != nil {
		return 0, 0, err
	}
	if !si.IsDir() {
		return 0, 0, vfs.NewPathError("CopyDir", src, vfs.ErrNotDir)
	}

	di, err := dstfs.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return 0, 0, err
	}
	if err == nil && !di.IsDir() {
		return 0, 0, vfs.NewPathError("CopyDir", dst, vfs.ErrNotDir)
	}

	err = dstfs.MkdirAll(dst, si.Mode())
	if err != nil {
		return 0, 0, err
	}

	entries, err := vfs.ReadDir(srcfs, src)
	if err != nil {
		return 0, 0, err
	}

	for _, entry := range entries {
		srcPath := vfs.Join(srcfs, src, entry.Name())
		dstPath := vfs.Join(dstfs, dst, entry.Name())

		if entry.IsDir() {
			n, b, err = CopyDir(srcfs, srcPath, dstfs, dstPath)
			fcnt += n
			bcnt += b
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				var old string
				old, err = srcfs.Readlink(srcPath)
				if err == nil {
					err = dstfs.Symlink(old, dstPath)
				}
				if err == nil {
					fcnt++
					err = os.Chmod(dst, entry.Mode())
				}
			} else {
				err = vfs.CopyFile(srcfs, srcPath, dstfs, dstPath)
				if err == nil {
					bcnt += entry.Size()
					fcnt++
				}
			}
		}
		if err != nil {
			return fcnt, bcnt, err
		}
	}
	return fcnt, bcnt, nil
}

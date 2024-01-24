package dirtree

import (
	"compress/gzip"
	"fmt"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/blobaccess/bpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/utils"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

func DataAccessForDirTree(path string, opts ...Option) (bpi.DataAccess, error) {
	blobAccess, err := BlobAccessForDirTree(path, opts...)
	if err != nil {
		return nil, err
	}
	return blobAccess, nil
}

func BlobAccessForDirTree(path string, opts ...Option) (_ bpi.BlobAccess, rerr error) {
	eff := optionutils.EvalOptions(opts...)
	fs := utils.FileSystem(eff.FileSystem)

	ok, err := vfs.IsDir(fs, path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%q is no directory", path)
	}

	taropts := tarutils.TarFileSystemOptions{
		IncludeFiles:   eff.IncludeFiles,
		ExcludeFiles:   eff.ExcludeFiles,
		PreserveDir:    utils.AsBool(eff.PreserveDir),
		FollowSymlinks: utils.AsBool(eff.FollowSymlinks),
	}

	temp, err := blobaccess.NewTempFile(fs.FSTempDir(), "resourceblob*.tgz", fs)
	if err != nil {
		return nil, err
	}
	defer errors.PropagateError(&rerr, temp.Close)

	if utils.AsBool(eff.CompressWithGzip) {
		if eff.MimeType == "" {
			eff.MimeType = mime.MIME_TGZ
		}
		gw := gzip.NewWriter(temp.Writer())
		if err := tarutils.PackFsIntoTar(fs, path, gw, taropts); err != nil {
			return nil, fmt.Errorf("unable to tar input artifact: %w", err)
		}
		if err := gw.Close(); err != nil {
			return nil, fmt.Errorf("unable to close gzip writer: %w", err)
		}
	} else {
		if eff.MimeType == "" {
			eff.MimeType = mime.MIME_TAR
		}
		if err := tarutils.PackFsIntoTar(fs, path, temp.Writer(), taropts); err != nil {
			return nil, fmt.Errorf("unable to tar input artifact: %w", err)
		}
	}
	return temp.AsBlob(eff.MimeType), nil
}

func BlobAccessProviderForDirTree(path string, opts ...Option) bpi.BlobAccessProvider {
	return bpi.BlobAccessProviderFunction(func() (bpi.BlobAccess, error) {
		return BlobAccessForDirTree(path, opts...)
	})
}

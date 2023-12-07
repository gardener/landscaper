// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blob

import (
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Handler struct{}

func init() {
	download.Register(&Handler{}, download.ForArtifactType(download.ALL))
}

func wrapErr(err error, racc cpi.ResourceAccess) error {
	if err == nil {
		return nil
	}
	m := racc.Meta()
	return errors.Wrapf(err, "resource %s/%s%s", m.GetName(), m.GetVersion(), m.ExtraIdentity.String())
}

func (_ Handler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	rd, err := cpi.GetResourceReader(racc)
	if err != nil {
		return true, "", wrapErr(err, racc)
	}
	defer rd.Close()
	if path == "" {
		path = racc.Meta().GetName()
	}
	file, err := fs.OpenFile(path, vfs.O_TRUNC|vfs.O_CREATE|vfs.O_WRONLY, 0o660)
	if err != nil {
		return true, "", wrapErr(errors.Wrapf(err, "creating target file %q", path), racc)
	}
	defer file.Close()
	n, err := io.Copy(file, rd)
	if err == nil {
		p.Printf("%s: %d byte(s) written\n", path, n)
	}
	return true, path, wrapErr(err, racc)
}

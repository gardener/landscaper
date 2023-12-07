// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
)

func Download(p common.Printer, ctx oci.Context, ref string, path string, fs vfs.FileSystem, creds ...credentials.CredentialsSource) error {
	_, _, _, err := Download2(p, ctx, ref, path, fs, false, creds...)
	return err
}

func Download2(p common.Printer, ctx oci.Context, ref string, path string, fs vfs.FileSystem, asartifact bool, creds ...credentials.CredentialsSource) (chart, prov string, aset string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&err, "downloading helm chart %q", ref)

	r, err := oci.ParseRef(ref)
	if err != nil {
		return
	}

	spec, err := ctx.MapUniformRepositorySpec(&r.UniformRepositorySpec)
	if err != nil {
		return
	}

	repo, err := ctx.RepositoryForSpec(spec, creds...)
	if err != nil {
		return
	}
	finalize.Close(repo)

	art, err := repo.LookupArtifact(r.Repository, r.Version())
	if err != nil {
		return
	}
	finalize.Close(art)

	if asartifact {
		aset = strings.TrimSuffix(path, ".tgz") + ".ctf"
		ctf, err := artifactset.Open(accessobj.ACC_CREATE|accessobj.ACC_WRITABLE, aset, 0o600, accessio.FormatTGZ, accessio.PathFileSystem(fs))
		if err != nil {
			return "", "", "", errors.Wrapf(err, "cannot create artifact set")
		}
		err = artifactset.TransferArtifact(art, ctf)
		if err == nil {
			ctf.Annotate(artifactset.MAINARTIFACT_ANNOTATION, art.Digest().String())
		}
		ctf.Close()
		if err != nil {
			fs.Remove(aset)
			return "", "", "", errors.Wrapf(err, "cannot transfer helm OCI artifact")
		}
	}
	chart, prov, err = download(p, art, path, fs)
	return chart, prov, aset, err
}

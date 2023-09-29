// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocirepo

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/oci/transfer"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/ociuploadattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

type handler struct {
	spec *ociuploadattr.Attribute
}

func New(repospec ...*ociuploadattr.Attribute) download.Handler {
	return &handler{spec: utils.Optional(repospec...)}
}

func (h *handler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (accepted bool, target string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&err, "upload to OCI registry")

	ctx := racc.ComponentVersion().GetContext()
	m, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	finalize.Close(m)

	mediaType := m.MimeType()

	if !artdesc.IsOCIMediaType(mediaType) || (!strings.HasSuffix(mediaType, "+tar") && !strings.HasSuffix(mediaType, "+tar+gzip")) {
		return false, "", nil
	}

	log := download.Logger(ctx).WithName("ocireg")

	var repo oci.Repository

	var namespace string
	var version string = "latest"

	aspec := m.AccessSpec()
	if hp, ok := aspec.(cpi.HintProvider); ok {
		namespace = hp.GetReferenceHint(racc.ComponentVersion())
	} else if l, ok := aspec.(*localblob.AccessSpec); ok {
		namespace = l.ReferenceName
	}

	i := strings.LastIndex(namespace, ":")
	if i > 0 {
		version = namespace[i:]
		version = version[1:] // remove colon
		namespace = namespace[:i]
	}

	ocictx := ctx.OCIContext()

	var artspec oci.ArtSpec
	var prefix string
	var result oci.RefSpec

	if h.spec == nil {
		log.Debug("no config set")
		if path == "" {
			return false, "", fmt.Errorf("path required as target repo specification")
		}
		ref, err := oci.ParseRef(path)
		if err != nil {
			return true, "", err
		}
		result.UniformRepositorySpec = ref.UniformRepositorySpec
		repospec, err := ocictx.MapUniformRepositorySpec(&ref.UniformRepositorySpec)
		if err != nil {
			return true, "", err
		}
		repo, err = ocictx.RepositoryForSpec(repospec)
		if err != nil {
			return true, "", err
		}
		finalize.Close(repo)
		artspec = ref.ArtSpec
	} else {
		log.Debug("evaluating config")
		if path != "" {
			artspec, err = oci.ParseArt(path)
			if err != nil {
				return true, "", err
			}
		}
		var us *oci.UniformRepositorySpec
		repo, us, prefix, err = h.spec.GetInfo(ctx)
		if err != nil {
			return true, "", err
		}
		result.UniformRepositorySpec = *us
	}
	log.Debug("using artifact spec", "spec", artspec.String())
	if artspec.Digest != nil {
		return true, "", fmt.Errorf("digest no possible for target")
	}

	if artspec.Repository != "" {
		namespace = artspec.Repository
	}
	if artspec.Reference() != "" {
		version = artspec.Reference()
	}

	if prefix != "" && namespace != "" {
		namespace = prefix + grammar.RepositorySeparator + namespace
	}
	if version == "" || version == "latest" {
		version = racc.Meta().GetVersion()
	}
	log.Debug("using final target", "namespace", namespace, "version", version)
	if namespace == "" {
		return true, "", fmt.Errorf("no OCI namespace")
	}

	var art oci.ArtifactAccess

	cand := m
	if local, ok := aspec.(*localblob.AccessSpec); ok {
		if local.GlobalAccess != nil {
			s, err := ctx.AccessSpecForSpec(local.GlobalAccess)
			if err == nil {
				_ = s
				// c, err := s.AccessMethod()  // TODO: try global access for direct artifact access
				// set cand to oci access method
			}
		}
	}
	if ocimeth, ok := cand.(ociartifact.AccessMethod); ok {
		// prepare for optimized point to point implementation
		art, _, err = ocimeth.GetArtifact(&finalize)
		if err != nil {
			return true, "", errors.Wrapf(err, "cannot access source artifact")
		}
		finalize.Close(art)
	}

	ns, err := repo.LookupNamespace(namespace)
	if err != nil {
		return true, "", err
	}
	finalize.Close(ns)

	if art == nil {
		log.Debug("using artifact set transfer mode")
		set, err := artifactset.OpenFromDataAccess(accessobj.ACC_READONLY, m.MimeType(), m)
		if err != nil {
			return true, "", errors.Wrapf(err, "opening resource blob as artifact set")
		}
		finalize.Close(set)
		art, err = set.GetArtifact(set.GetMain().String())
		if err != nil {
			return true, "", errors.Wrapf(err, "get artifact from blob")
		}
		finalize.Close(art)
	} else {
		log.Debug("using direct transfer mode")
	}

	p.Printf("uploading resource %s to %s[%s:%s]...\n", racc.Meta().GetName(), repo.GetSpecification().UniformRepositorySpec(), namespace, version)
	err = transfer.TransferArtifact(art, ns, oci.AsTags(version)...)
	if err != nil {
		return true, "", errors.Wrapf(err, "transfer artifact")
	}

	result.Repository = namespace
	result.Tag = &version
	return true, result.String(), nil
}

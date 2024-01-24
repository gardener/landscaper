// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocirepo

import (
	"fmt"
	"path"
	"strings"

	. "github.com/open-component-model/ocm/pkg/finalizer"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/oci/transfer"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/keepblobattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/mapocirepoattr"
	storagecontext "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/handlers/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
)

func init() {
	for _, mime := range artdesc.ArchiveBlobTypes() {
		cpi.RegisterBlobHandler(NewArtifactHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.Type),
			cpi.ForMimeType(mime))
		cpi.RegisterBlobHandler(NewArtifactHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.LegacyType),
			cpi.ForMimeType(mime))
		cpi.RegisterBlobHandler(NewArtifactHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.ShortType),
			cpi.ForMimeType(mime))
	}
	/*
		cpi.RegisterBlobHandler(NewBlobHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.Type))
		cpi.RegisterBlobHandler(NewBlobHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.LegacyType))
		cpi.RegisterBlobHandler(NewBlobHandler(OCIRegBaseFunction), cpi.ForRepo(oci.CONTEXT_TYPE, ocireg.ShortType))
	*/
}

////////////////////////////////////////////////////////////////////////////////

type BaseFunction func(ctx *storagecontext.StorageContext) string

func OCIRegBaseFunction(ctx *storagecontext.StorageContext) string {
	i, err := ocireg.GetRepositoryImplementation(ctx.Repository)
	if err != nil {
		panic("ocireg implementation mismatch")
	}
	return i.GetBaseURL()
}

// blobHandler is the default handling to store local blobs as local blobs but with an additional
// globally accessible OCIBlob access method.
type blobHandler struct {
	base BaseFunction
}

func (h *blobHandler) GetBaseURL(ctx *storagecontext.StorageContext) string {
	if h.base == nil {
		return ""
	}
	return h.base(ctx)
}

func NewBlobHandler(base BaseFunction) cpi.BlobHandler {
	return &blobHandler{base}
}

func (b *blobHandler) StoreBlob(blob cpi.BlobAccess, artType, hint string, global cpi.AccessSpec, ctx cpi.StorageContext) (cpi.AccessSpec, error) {
	ocictx, ok := ctx.(*storagecontext.StorageContext)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to storagecontext.StorageContext", ctx)
	}

	values := []interface{}{
		"arttype", artType,
		"mediatype", blob.MimeType(),
		"hint", hint,
	}
	if m, ok := blob.(blobaccess.AnnotatedBlobAccess[accspeccpi.AccessMethodView]); ok {
		cpi.BlobHandlerLogger(ctx.GetContext()).Debug("oci blob handler with ocm access source",
			generics.AppendedSlice[any](values, "sourcetype", m.Source().AccessSpec().GetType())...,
		)
	} else {
		cpi.BlobHandlerLogger(ctx.GetContext()).Debug("oci blob handler", values...)
	}

	err := ocictx.Manifest.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	err = ocictx.AssureLayer(blob)
	if err != nil {
		return nil, err
	}
	if compatattr.Get(ctx.GetContext()) {
		return localociblob.New(blob.Digest()), nil
	} else {
		if global == nil {
			base := b.GetBaseURL(ocictx)
			if base != "" {
				global = ociblob.New(path.Join(base, ocictx.Namespace.GetNamespace()), blob.Digest(), blob.MimeType(), blob.Size())
			}
		}
		return localblob.New(blob.Digest().String(), "", blob.MimeType(), global), nil
	}
}

////////////////////////////////////////////////////////////////////////////////

// artifactHandler stores artifact blobs as OCIArtifacts.
type artifactHandler struct {
	blobHandler
}

func NewArtifactHandler(base BaseFunction) cpi.BlobHandler {
	return &artifactHandler{blobHandler{base}}
}

func (b *artifactHandler) StoreBlob(blob cpi.BlobAccess, artType, hint string, global cpi.AccessSpec, ctx cpi.StorageContext) (cpi.AccessSpec, error) {
	mediaType := blob.MimeType()

	if !artdesc.IsOCIMediaType(mediaType) || (!strings.HasSuffix(mediaType, "+tar") && !strings.HasSuffix(mediaType, "+tar+gzip")) {
		return nil, nil
	}

	errhint := "[" + hint + "]"
	log := cpi.BlobHandlerLogger(ctx.GetContext())

	values := []interface{}{
		"arttype", artType,
		"mediatype", mediaType,
		"hint", hint,
	}

	var art oci.ArtifactAccess
	var err error
	var finalizer Finalizer
	defer finalizer.Finalize()

	keep := keepblobattr.Get(ctx.GetContext())

	if m, ok := blob.(blobaccess.AnnotatedBlobAccess[accspeccpi.AccessMethodView]); ok {
		// prepare for optimized point to point implementation
		log.Debug("oci artifact handler with ocm access source",
			generics.AppendedSlice[any](values, "sourcetype", m.Source().AccessSpec().GetType())...,
		)
		if ocimeth, ok := m.Source().Unwrap().(ociartifact.AccessMethodImpl); !keep && ok {
			art, _, err = ocimeth.GetArtifact()
			if err != nil {
				return nil, errors.Wrapf(err, "cannot access source artifact")
			}
			if art != nil {
				defer art.Close()
			}
		}
	} else {
		log.Debug("oci artifact handler", values...)
	}

	var namespace oci.NamespaceAccess
	var version string
	var name string
	var tag string
	var digest digest.Digest

	ocictx, ok := ctx.(*storagecontext.StorageContext)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to storagecontext.StorageContext", ctx)
	}
	base := b.GetBaseURL(ocictx)
	if hint == "" {
		namespace = ocictx.Namespace
	} else {
		prefix := cpi.RepositoryPrefix(ctx.TargetComponentRepository().GetSpecification())
		i := strings.LastIndex(hint, "@")
		if i >= 0 {
			hint = hint[:i] // remove digest
		}
		i = strings.LastIndex(hint, ":")
		if i > 0 {
			version = hint[i:]
			tag = version[1:] // remove colon
			name = hint[:i]
		} else {
			name = hint
		}

		hash := mapocirepoattr.Get(ctx.GetContext())
		if hash.Prefix != nil {
			prefix = *hash.Prefix
		}
		orig := name
		mapped := hash.Map(name)
		name = path.Join(prefix, mapped)
		if mapped == orig {
			log.Debug("namespace derived from hint",
				generics.AppendedSlice[any](values, "namespace", name),
			)
		} else {
			log.Debug("mapped namespace derived from hint",
				generics.AppendedSlice[any](values, "namespace", name),
			)
		}

		namespace, err = ocictx.Repository.LookupNamespace(name)
		if err != nil {
			return nil, err
		}
		defer namespace.Close()
	}

	errhint += " namespace " + namespace.GetNamespace()

	if art == nil {
		log.Debug("using artifact set transfer mode")
		set, err := artifactset.OpenFromBlob(accessobj.ACC_READONLY, blob)
		if err != nil {
			return nil, wrap(err, errhint, "open blob")
		}
		defer set.Close()
		digest = set.GetMain()
		art, err = set.GetArtifact(digest.String())
		if err != nil {
			return nil, wrap(err, errhint, "get artifact from blob")
		}
		defer art.Close()
	} else {
		log.Debug("using direct transfer mode")
		digest = art.Digest()
	}

	if version == "" {
		version = "@" + digest.String()
	}

	err = transfer.TransferArtifact(art, namespace, oci.AsTags(tag)...)
	if err != nil {
		return nil, wrap(err, errhint, "transfer artifact")
	}

	ref := path.Join(base, namespace.GetNamespace()) + version
	return ociartifact.New(ref), nil
}

func wrap(err error, msg string, args ...interface{}) error {
	for _, a := range args {
		msg = fmt.Sprintf("%s: %s", msg, a)
	}
	return errors.Wrapf(err, "exploding OCI artifact resource blob (%s)", msg)
}
